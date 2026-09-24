package audiomorph

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/braheezy/shine-mp3/pkg/mp3"
	"github.com/go-audio/aiff"
	goaudio "github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/schollz/goflac"
	interpolators "github.com/schollz/interpolation"
)

// OptionUseChannels specifies which channels to use when encoding audio.
func OptionUseChannels(channels []int) Option {
	return func(a *Audio) {
		a.useChannels = channels
	}
}

// OptionSampleRate specifies the target sample rate for encoding audio.
func OptionSampleRate(sampleRate int) Option {
	return func(a *Audio) {
		a.targetSampleRate = sampleRate
	}
}

// OptionInterpolationMethod specifies the interpolation method to use for sample rate conversion.
// Valid methods are: "linear", "cubic", "hermite", "lanczos2", "lanczos3", "bspline3", "bspline5", "monotonic"
func OptionInterpolationMethod(method string) Option {
	return func(a *Audio) {
		a.interpolationMethod = method
	}
}

// OptionBitDepth specifies the target bit depth for encoding audio.
func OptionBitDepth(bitDepth int) Option {
	return func(a *Audio) {
		a.targetBitDepth = bitDepth
	}
}

// extractChannels selects channels from interleaved PCM data.
// If useChannels is empty, the original data is returned unchanged; otherwise
// the output contains len(useChannels) channels in the given order.
func extractChannels(data []int, numChannels int, useChannels []int) []int {
	if len(useChannels) == 0 || numChannels <= 0 {
		return data
	}

	numSamples := len(data) / numChannels
	out := make([]int, 0, numSamples*len(useChannels))
	for i := 0; i < numSamples; i++ {
		for _, ch := range useChannels {
			out = append(out, data[i*numChannels+ch])
		}
	}
	return out
}

// convertSampleRate converts audio data to a different sample rate using interpolation
func convertSampleRate(audio *Audio, targetSampleRate int, method string) error {
	// If no target sample rate is specified or it matches current, no conversion needed
	if targetSampleRate == 0 || targetSampleRate == audio.SampleRate {
		return nil
	}

	// Default to linear interpolation if not specified
	if method == "" {
		method = "linear"
	}

	numChannels := audio.NumChannels
	numSamples := len(audio.Data) / numChannels
	ratio := float64(targetSampleRate) / float64(audio.SampleRate)
	newNumSamples := int(float64(numSamples) * ratio)

	// Map method string to interpolator type
	var interpType interpolators.InterpolatorType
	switch method {
	case "linear":
		interpType = interpolators.Linear
	case "cubic":
		interpType = interpolators.CubicSpline
	case "hermite":
		interpType = interpolators.Hermite4
	case "lanczos2":
		interpType = interpolators.Lanczos2
	case "lanczos3":
		interpType = interpolators.Lanczos3
	case "bspline3":
		interpType = interpolators.BSpline3
	case "bspline5":
		interpType = interpolators.BSpline5
	case "monotonic":
		interpType = interpolators.MonotonicCubic
	default:
		return fmt.Errorf("unsupported interpolation method: %s", method)
	}

	// Resample each channel, then re-interleave
	newData := make([]int, 0, newNumSamples*numChannels)
	for ch := 0; ch < numChannels; ch++ {
		// Deinterleave the channel for interpolation
		inData := make([]float64, numSamples)
		for i := 0; i < numSamples; i++ {
			inData[i] = float64(audio.Data[i*numChannels+ch])
		}

		outData, err := interpolators.Interpolate(inData, newNumSamples, interpType)
		if err != nil {
			return fmt.Errorf("failed to interpolate channel %d: %w", ch, err)
		}

		for i := 0; i < newNumSamples; i++ {
			newData = append(newData, int(outData[i]))
		}
	}

	// Update the audio struct with resampled data
	audio.Data = newData
	audio.SampleRate = targetSampleRate
	audio.Duration = float64(newNumSamples) / float64(targetSampleRate)

	return nil
}

// convertBitDepth converts audio data to a different bit depth
func convertBitDepth(audio *Audio, targetBitDepth int) error {
	// If no target bit depth is specified or it matches current, no conversion needed
	if targetBitDepth == 0 || targetBitDepth == audio.BitDepth {
		return nil
	}

	// Validate target bit depth
	if targetBitDepth != 8 && targetBitDepth != 16 && targetBitDepth != 24 && targetBitDepth != 32 {
		return fmt.Errorf("unsupported bit depth: %d (must be 8, 16, 24, or 32)", targetBitDepth)
	}

	sourceBitDepth := audio.BitDepth

	// Calculate scaling factor
	// When converting bit depth, we need to scale the sample values
	// For example, 16-bit samples range from -32768 to 32767
	// and 24-bit samples range from -8388608 to 8388607
	var scale float64
	if targetBitDepth > sourceBitDepth {
		// Upscaling: multiply by 2^(targetBitDepth - sourceBitDepth)
		scale = float64(int64(1) << uint(targetBitDepth-sourceBitDepth))
	} else {
		// Downscaling: divide by 2^(sourceBitDepth - targetBitDepth)
		scale = 1.0 / float64(int64(1)<<uint(sourceBitDepth-targetBitDepth))
	}

	// Scale every sample in the interleaved data
	maxVal := int64(1)<<uint(targetBitDepth-1) - 1
	minVal := -int64(1) << uint(targetBitDepth-1)
	for i := range audio.Data {
		scaledValue := float64(audio.Data[i]) * scale

		if scaledValue > float64(maxVal) {
			audio.Data[i] = int(maxVal)
		} else if scaledValue < float64(minVal) {
			audio.Data[i] = int(minVal)
		} else {
			audio.Data[i] = int(scaledValue)
		}
	}

	// Update the bit depth
	audio.BitDepth = targetBitDepth

	return nil
}

// EncodeFile encodes an Audio struct to a file based on the filename extension
func EncodeFile(audio *Audio, filename string, options ...Option) error {
	// Apply options
	for _, option := range options {
		option(audio)
	}

	ext := strings.ToLower(filepath.Ext(filename))

	// For MP3 files, ensure the sample rate is supported
	if ext == ".mp3" {
		// If a target sample rate was specified, adjust it to nearest supported rate
		if audio.targetSampleRate > 0 {
			audio.targetSampleRate = findNearestSupportedMP3SampleRate(audio.targetSampleRate)
		} else {
			// If no target sample rate, but current rate is unsupported, adjust to nearest
			supportedRate := findNearestSupportedMP3SampleRate(audio.SampleRate)
			if supportedRate != audio.SampleRate {
				audio.targetSampleRate = supportedRate
			}
		}
	}

	// Apply sample rate conversion if specified
	if audio.targetSampleRate > 0 && audio.targetSampleRate != audio.SampleRate {
		if err := convertSampleRate(audio, audio.targetSampleRate, audio.interpolationMethod); err != nil {
			return fmt.Errorf("failed to convert sample rate: %w", err)
		}
	}

	// Apply bit depth conversion if specified
	if audio.targetBitDepth > 0 && audio.targetBitDepth != audio.BitDepth {
		if err := convertBitDepth(audio, audio.targetBitDepth); err != nil {
			return fmt.Errorf("failed to convert bit depth: %w", err)
		}
	}

	switch ext {
	case ".wav":
		return encodeWAV(audio, filename)
	case ".aif", ".aiff":
		return encodeAIFF(audio, filename)
	case ".mp3":
		return encodeMP3(audio, filename)
	case ".flac":
		return encodeFLAC(audio, filename)
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}
}

// encodeWAV encodes audio data to a WAV file
func encodeWAV(audio *Audio, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create WAV file: %w", err)
	}
	defer f.Close()

	// Determine number of channels (channel selection if requested)
	numChannels := audio.NumChannels
	if len(audio.useChannels) > 0 {
		numChannels = len(audio.useChannels)
	}

	// Create WAV encoder
	encoder := wav.NewEncoder(f, audio.SampleRate, audio.BitDepth, numChannels, 1)

	data := extractChannels(audio.Data, audio.NumChannels, audio.useChannels)

	// Create PCM buffer
	buf := &goaudio.IntBuffer{
		Format: &goaudio.Format{
			NumChannels: numChannels,
			SampleRate:  audio.SampleRate,
		},
		Data:           data,
		SourceBitDepth: audio.BitDepth,
	}

	// Write the buffer
	if err := encoder.Write(buf); err != nil {
		return fmt.Errorf("failed to write WAV data: %w", err)
	}

	// Close encoder
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("failed to close WAV encoder: %w", err)
	}

	return nil
}

// encodeAIFF encodes audio data to an AIFF file
func encodeAIFF(audio *Audio, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create AIFF file: %w", err)
	}
	defer f.Close()

	// Determine number of channels (channel selection if requested)
	numChannels := audio.NumChannels
	if len(audio.useChannels) > 0 {
		numChannels = len(audio.useChannels)
	}

	// Create AIFF encoder
	encoder := aiff.NewEncoder(f, audio.SampleRate, audio.BitDepth, numChannels)

	data := extractChannels(audio.Data, audio.NumChannels, audio.useChannels)

	// Create PCM buffer
	buf := &goaudio.IntBuffer{
		Format: &goaudio.Format{
			NumChannels: numChannels,
			SampleRate:  audio.SampleRate,
		},
		Data:           data,
		SourceBitDepth: audio.BitDepth,
	}

	// Write the buffer
	if err := encoder.Write(buf); err != nil {
		return fmt.Errorf("failed to write AIFF data: %w", err)
	}

	// Close encoder
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("failed to close AIFF encoder: %w", err)
	}

	return nil
}

// supportedMP3SampleRates lists all sample rates supported by the MP3 encoder
var supportedMP3SampleRates = []int{
	44100, 48000, 32000, // MPEG-1
	22050, 24000, 16000, // MPEG-2
	11025, 12000, 8000, // MPEG-2.5
}

// findNearestSupportedMP3SampleRate returns the nearest supported MP3 sample rate
func findNearestSupportedMP3SampleRate(sampleRate int) int {
	// Check if already supported
	for _, supported := range supportedMP3SampleRates {
		if sampleRate == supported {
			return sampleRate
		}
	}

	// Find the nearest supported rate
	nearestRate := supportedMP3SampleRates[0]
	minDiff := abs(sampleRate - nearestRate)

	for _, supported := range supportedMP3SampleRates[1:] {
		diff := abs(sampleRate - supported)
		if diff < minDiff {
			minDiff = diff
			nearestRate = supported
		}
	}

	return nearestRate
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// encodeMP3 encodes audio data to an MP3 file
func encodeMP3(audio *Audio, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create MP3 file: %w", err)
	}
	defer f.Close()

	// Determine number of channels (channel selection if requested)
	numChannels := audio.NumChannels
	if len(audio.useChannels) > 0 {
		numChannels = len(audio.useChannels)
	}

	// Create MP3 encoder - sample rate should already be converted to a supported rate by EncodeFile
	encoder := mp3.NewEncoder(audio.SampleRate, numChannels)

	data := extractChannels(audio.Data, audio.NumChannels, audio.useChannels)

	// Convert and scale samples to int16 range (interleaved)
	scale := float64(1<<15) / float64(int64(1)<<uint(audio.BitDepth-1))
	int16Data := make([]int16, len(data))
	for i, sample := range data {
		int16Data[i] = int16(float64(sample) * scale)
	}

	// Write MP3 data
	if err := encoder.Write(f, int16Data); err != nil {
		return fmt.Errorf("failed to write MP3 data: %w", err)
	}

	return nil
}

// encodeFLAC encodes audio data to a FLAC file
func encodeFLAC(audio *Audio, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create FLAC file: %w", err)
	}
	defer f.Close()

	// Determine number of channels (channel selection if requested)
	numChannels := audio.NumChannels
	if len(audio.useChannels) > 0 {
		numChannels = len(audio.useChannels)
	}

	// Create FLAC encoder
	encoder, err := goflac.NewEncoder(f, uint32(audio.SampleRate), uint8(numChannels), uint8(audio.BitDepth))
	if err != nil {
		return fmt.Errorf("failed to create FLAC encoder: %w", err)
	}

	data := extractChannels(audio.Data, audio.NumChannels, audio.useChannels)
	numSamples := len(data) / numChannels

	// Deinterleave data into [][]int32 for the FLAC encoder
	samples := make([][]int32, numChannels)
	for ch := 0; ch < numChannels; ch++ {
		samples[ch] = make([]int32, numSamples)
		for i := 0; i < numSamples; i++ {
			samples[ch][i] = int32(data[i*numChannels+ch])
		}
	}

	// Encode samples
	if err := encoder.Encode(samples); err != nil {
		return fmt.Errorf("failed to encode FLAC data: %w", err)
	}

	return nil
}
