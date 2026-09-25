package audiomorph

import (
	"bytes"
	"fmt"
	"io"
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

// toIntSlice converts normalized float32 samples in [-1.0, 1.0] to integer
// PCM samples at the given bit depth, clamping to the representable range.
func toIntSlice(data []float32, bitDepth int) []int {
	maxVal := float64(int64(1) << uint(bitDepth-1))
	minVal := -int64(1) << uint(bitDepth-1)
	maxSample := int64(1)<<uint(bitDepth-1) - 1

	out := make([]int, len(data))
	for i, v := range data {
		s := int64(float64(v) * maxVal)
		if s < minVal {
			s = minVal
		} else if s > maxSample {
			s = maxSample
		}
		out[i] = int(s)
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

	numSamples := len(audio.Data)
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

	// Resample the mono signal
	inData := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		inData[i] = float64(audio.Data[i])
	}

	outData, err := interpolators.Interpolate(inData, newNumSamples, interpType)
	if err != nil {
		return fmt.Errorf("failed to interpolate audio: %w", err)
	}

	newData := make([]float32, newNumSamples)
	for i := 0; i < newNumSamples; i++ {
		newData[i] = float32(outData[i])
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

	// Data is normalized to [-1.0, 1.0], so the sample values themselves do
	// not change with bit depth; only the quantization applied on encode
	// differs. Validate and record the new depth.
	audio.BitDepth = targetBitDepth

	return nil
}

// Encode encodes the Audio to the format given by its Format property
// ("wav", "aiff", "mp3", or "flac") and returns the encoded bytes as an
// io.ReadSeeker, mirroring the stream-based Decode API.
func Encode(a *Audio, options ...Option) (io.ReadSeeker, error) {
	// Apply options
	for _, option := range options {
		option(a)
	}

	format := strings.ToLower(strings.TrimSpace(a.Format))
	if format == "" {
		return nil, fmt.Errorf("format not specified: set Audio.Format (e.g. \"wav\") before encoding")
	}

	// For MP3 files, ensure the sample rate is supported
	if format == "mp3" {
		// If a target sample rate was specified, adjust it to nearest supported rate
		if a.targetSampleRate > 0 {
			a.targetSampleRate = findNearestSupportedMP3SampleRate(a.targetSampleRate)
		} else {
			// If no target sample rate, but current rate is unsupported, adjust to nearest
			supportedRate := findNearestSupportedMP3SampleRate(a.SampleRate)
			if supportedRate != a.SampleRate {
				a.targetSampleRate = supportedRate
			}
		}
	}

	// Apply sample rate conversion if specified
	if a.targetSampleRate > 0 && a.targetSampleRate != a.SampleRate {
		if err := convertSampleRate(a, a.targetSampleRate, a.interpolationMethod); err != nil {
			return nil, fmt.Errorf("failed to convert sample rate: %w", err)
		}
	}

	// Apply bit depth conversion if specified
	if a.targetBitDepth > 0 && a.targetBitDepth != a.BitDepth {
		if err := convertBitDepth(a, a.targetBitDepth); err != nil {
			return nil, fmt.Errorf("failed to convert bit depth: %w", err)
		}
	}

	out := &seekableBuffer{}
	var err error

	switch format {
	case "wav":
		err = encodeWAV(a, out)
	case "aiff":
		err = encodeAIFF(a, out)
	case "mp3":
		err = encodeMP3(a, out)
	case "flac":
		err = encodeFLAC(a, out)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(out.buf), nil
}

// EncodeFile encodes an Audio struct to a file based on the filename extension.
// The extension is written into Audio.Format and encoding is delegated to Encode.
func EncodeFile(a *Audio, filename string, options ...Option) error {
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
	switch format {
	case "aif":
		format = "aiff"
	case "":
		return fmt.Errorf("cannot determine format: filename %q has no extension", filename)
	}
	a.Format = format

	r, err := Encode(a, options...)
	if err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

// seekableBuffer is a minimal in-memory io.WriteSeeker. It is needed because
// container encoders such as WAV/AIFF patch header fields after writing data.
type seekableBuffer struct {
	buf []byte
	pos int
}

func (b *seekableBuffer) Write(p []byte) (int, error) {
	end := b.pos + len(p)
	if end > len(b.buf) {
		b.buf = append(b.buf, make([]byte, end-len(b.buf))...)
	}
	copy(b.buf[b.pos:end], p)
	b.pos = end
	return len(p), nil
}

func (b *seekableBuffer) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = int64(b.pos) + offset
	case io.SeekEnd:
		newPos = int64(len(b.buf)) + offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}
	if newPos < 0 {
		return 0, fmt.Errorf("negative seek position: %d", newPos)
	}
	b.pos = int(newPos)
	return newPos, nil
}

// encodeWAV encodes audio data to a WAV stream
func encodeWAV(audio *Audio, w io.WriteSeeker) error {
	// Create WAV encoder (mono output)
	encoder := wav.NewEncoder(w, audio.SampleRate, audio.BitDepth, 1, 1)

	// Create PCM buffer
	buf := &goaudio.IntBuffer{
		Format: &goaudio.Format{
			NumChannels: 1,
			SampleRate:  audio.SampleRate,
		},
		Data:           toIntSlice(audio.Data, audio.BitDepth),
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

// encodeAIFF encodes audio data to an AIFF stream
func encodeAIFF(audio *Audio, w io.WriteSeeker) error {
	// Create AIFF encoder (mono output)
	encoder := aiff.NewEncoder(w, audio.SampleRate, audio.BitDepth, 1)

	// Create PCM buffer
	buf := &goaudio.IntBuffer{
		Format: &goaudio.Format{
			NumChannels: 1,
			SampleRate:  audio.SampleRate,
		},
		Data:           toIntSlice(audio.Data, audio.BitDepth),
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

// encodeMP3 encodes audio data to an MP3 stream
func encodeMP3(audio *Audio, w io.Writer) error {
	// Create MP3 encoder (mono output) - sample rate should already be converted
	// to a supported rate by Encode
	encoder := mp3.NewEncoder(audio.SampleRate, 1)

	// Convert and scale normalized samples to int16 range
	int16Data := make([]int16, len(audio.Data))
	for i, sample := range audio.Data {
		s := int64(float64(sample) * 32768)
		if s < -32768 {
			s = -32768
		} else if s > 32767 {
			s = 32767
		}
		int16Data[i] = int16(s)
	}

	// Write MP3 data
	if err := encoder.Write(w, int16Data); err != nil {
		return fmt.Errorf("failed to write MP3 data: %w", err)
	}

	return nil
}

// encodeFLAC encodes audio data to a FLAC stream
func encodeFLAC(audio *Audio, w io.Writer) error {
	// Create FLAC encoder (mono output)
	encoder, err := goflac.NewEncoder(w, uint32(audio.SampleRate), 1, uint8(audio.BitDepth))
	if err != nil {
		return fmt.Errorf("failed to create FLAC encoder: %w", err)
	}

	numSamples := len(audio.Data)

	// Build the single-channel sample array for the FLAC encoder, scaling
	// normalized samples back to the target bit depth
	maxVal := float64(int64(1) << uint(audio.BitDepth-1))
	minVal := -int64(1) << uint(audio.BitDepth-1)
	maxSample := int64(1)<<uint(audio.BitDepth-1) - 1
	samples := make([][]int32, 1)
	samples[0] = make([]int32, numSamples)
	for i := 0; i < numSamples; i++ {
		s := int64(float64(audio.Data[i]) * maxVal)
		if s < minVal {
			s = minVal
		} else if s > maxSample {
			s = maxSample
		}
		samples[0][i] = int32(s)
	}

	// Encode samples
	if err := encoder.Encode(samples); err != nil {
		return fmt.Errorf("failed to encode FLAC data: %w", err)
	}

	return nil
}
