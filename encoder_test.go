package audiomorph

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestEncodeWAV(t *testing.T) {
	srcFilename := filepath.Join(t.TempDir(), "stereo.wav")
	writeWAV(t, srcFilename, 2)

	audio, err := DecodeFile(srcFilename)
	if err != nil {
		t.Fatalf("Failed to decode source file: %v", err)
	}

	dstFilename := filepath.Join(t.TempDir(), "test_output.wav")
	if err := EncodeFile(audio, dstFilename); err != nil {
		t.Fatalf("Failed to encode WAV file: %v", err)
	}

	decodedAudio, err := DecodeFile(dstFilename)
	if err != nil {
		t.Fatalf("Failed to decode encoded WAV file: %v", err)
	}

	if decodedAudio.SampleRate != audio.SampleRate {
		t.Errorf("SampleRate mismatch: expected %d, got %d", audio.SampleRate, decodedAudio.SampleRate)
	}
	if decodedAudio.BitDepth != audio.BitDepth {
		t.Errorf("BitDepth mismatch: expected %d, got %d", audio.BitDepth, decodedAudio.BitDepth)
	}
	if len(decodedAudio.Data) != len(audio.Data) {
		t.Errorf("Sample count mismatch: expected %d, got %d", len(audio.Data), len(decodedAudio.Data))
	}
	for i := range audio.Data {
		if decodedAudio.Data[i] != audio.Data[i] {
			t.Errorf("Sample mismatch at %d: expected %d, got %d", i, audio.Data[i], decodedAudio.Data[i])
			break
		}
	}
}

func TestEncodeAIFF(t *testing.T) {
	srcFilename := filepath.Join(t.TempDir(), "stereo.wav")
	writeWAV(t, srcFilename, 2)

	audio, err := DecodeFile(srcFilename)
	if err != nil {
		t.Fatalf("Failed to decode source file: %v", err)
	}

	dstFilename := filepath.Join(t.TempDir(), "test_output.aiff")
	if err := EncodeFile(audio, dstFilename); err != nil {
		t.Fatalf("Failed to encode AIFF file: %v", err)
	}

	decodedAudio, err := DecodeFile(dstFilename)
	if err != nil {
		t.Fatalf("Failed to decode encoded AIFF file: %v", err)
	}

	if decodedAudio.SampleRate != audio.SampleRate {
		t.Errorf("SampleRate mismatch: expected %d, got %d", audio.SampleRate, decodedAudio.SampleRate)
	}
	if decodedAudio.BitDepth != audio.BitDepth {
		t.Errorf("BitDepth mismatch: expected %d, got %d", audio.BitDepth, decodedAudio.BitDepth)
	}
	if len(decodedAudio.Data) != len(audio.Data) {
		t.Errorf("Sample count mismatch: expected %d, got %d", len(audio.Data), len(decodedAudio.Data))
	}
}

func TestEncodeMP3(t *testing.T) {
	srcFilename := filepath.Join(t.TempDir(), "stereo.wav")
	writeWAV(t, srcFilename, 2)

	audio, err := DecodeFile(srcFilename)
	if err != nil {
		t.Fatalf("Failed to decode source file: %v", err)
	}

	dstFilename := filepath.Join(t.TempDir(), "test_output.mp3")
	if err := EncodeFile(audio, dstFilename, OptionSampleRate(22050)); err != nil {
		t.Fatalf("Failed to encode MP3 file: %v", err)
	}

	decodedAudio, err := DecodeFile(dstFilename)
	if err != nil {
		t.Fatalf("Failed to decode encoded MP3 file: %v", err)
	}

	if decodedAudio.SampleRate != 22050 {
		t.Errorf("SampleRate mismatch: expected 22050, got %d", decodedAudio.SampleRate)
	}
	if len(decodedAudio.Data) == 0 {
		t.Error("Expected audio Data to be non-empty")
	}
}

func TestEncodeFLAC(t *testing.T) {
	srcFilename := filepath.Join(t.TempDir(), "stereo.wav")
	writeWAV(t, srcFilename, 2)

	audio, err := DecodeFile(srcFilename)
	if err != nil {
		t.Fatalf("Failed to decode source file: %v", err)
	}

	dstFilename := filepath.Join(t.TempDir(), "test_output.flac")
	if err := EncodeFile(audio, dstFilename); err != nil {
		t.Fatalf("Failed to encode FLAC file: %v", err)
	}

	decodedAudio, err := DecodeFile(dstFilename)
	if err != nil {
		t.Fatalf("Failed to decode encoded FLAC file: %v", err)
	}

	if decodedAudio.SampleRate != audio.SampleRate {
		t.Errorf("SampleRate mismatch: expected %d, got %d", audio.SampleRate, decodedAudio.SampleRate)
	}
	if decodedAudio.BitDepth != audio.BitDepth {
		t.Errorf("BitDepth mismatch: expected %d, got %d", audio.BitDepth, decodedAudio.BitDepth)
	}
	if len(decodedAudio.Data) != len(audio.Data) {
		t.Errorf("Sample count mismatch: expected %d, got %d", len(audio.Data), len(decodedAudio.Data))
	}
}

func TestEncodeUnsupportedFormat(t *testing.T) {
	audio := &Audio{
		SampleRate: 44100,
		BitDepth:   16,
		Data:       []int{0, 1, 2, 3, 4, 5},
		Duration:   0.001,
	}

	dstFilename := filepath.Join(t.TempDir(), "test_output.unknown")
	err := EncodeFile(audio, dstFilename)
	if err == nil {
		t.Fatal("Expected error for unsupported format, got nil")
	}
}

func TestSampleRateConversion(t *testing.T) {
	srcFilename := filepath.Join(t.TempDir(), "stereo.wav")
	writeWAV(t, srcFilename, 2)

	audio, err := DecodeFile(srcFilename)
	if err != nil {
		t.Fatalf("Failed to decode source file: %v", err)
	}

	originalSampleRate := audio.SampleRate
	originalSamples := len(audio.Data)

	testCases := []struct {
		name                string
		targetSampleRate    int
		interpolationMethod string
	}{
		{"Upsample to 48kHz with linear", 48000, "linear"},
		{"Downsample to 22.05kHz with linear", 22050, "linear"},
		{"Upsample to 48kHz with cubic", 48000, "cubic"},
		{"Downsample to 22.05kHz with hermite", 22050, "hermite"},
		{"Upsample to 48kHz with lanczos3", 48000, "lanczos3"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			audio, err := DecodeFile(srcFilename)
			if err != nil {
				t.Fatalf("Failed to decode source file: %v", err)
			}

			dstFilename := filepath.Join(t.TempDir(), "test_sample_rate.wav")
			if err := EncodeFile(audio, dstFilename,
				OptionSampleRate(tc.targetSampleRate),
				OptionInterpolationMethod(tc.interpolationMethod)); err != nil {
				t.Fatalf("Failed to encode with sample rate conversion: %v", err)
			}

			decodedAudio, err := DecodeFile(dstFilename)
			if err != nil {
				t.Fatalf("Failed to decode encoded file: %v", err)
			}

			if decodedAudio.SampleRate != tc.targetSampleRate {
				t.Errorf("Sample rate mismatch: expected %d, got %d", tc.targetSampleRate, decodedAudio.SampleRate)
			}

			expectedSamples := int(float64(originalSamples) * float64(tc.targetSampleRate) / float64(originalSampleRate))
			actualSamples := len(decodedAudio.Data)
			tolerance := int(float64(expectedSamples) * 0.01) // 1% tolerance
			if actualSamples < expectedSamples-tolerance || actualSamples > expectedSamples+tolerance {
				t.Errorf("Sample count mismatch: expected ~%d, got %d", expectedSamples, actualSamples)
			}
		})
	}
}

func TestInvalidInterpolationMethod(t *testing.T) {
	audio := &Audio{
		SampleRate: 44100,
		BitDepth:   16,
		Data:       []int{0, 100, 200, 300, 400},
		Duration:   0.0001,
	}

	err := convertSampleRate(audio, 48000, "invalid_method")
	if err == nil {
		t.Fatal("Expected error for invalid interpolation method, got nil")
	}
}

func TestBitDepthConversion(t *testing.T) {
	srcFilename := filepath.Join(t.TempDir(), "stereo.wav")
	writeWAV(t, srcFilename, 2)

	audio, err := DecodeFile(srcFilename)
	if err != nil {
		t.Fatalf("Failed to decode source file: %v", err)
	}

	originalSamples := len(audio.Data)

	testCases := []struct {
		name           string
		targetBitDepth int
	}{
		{"Convert to 8-bit", 8},
		{"Convert to 24-bit", 24},
		{"Convert to 32-bit", 32},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			audio, err := DecodeFile(srcFilename)
			if err != nil {
				t.Fatalf("Failed to decode source file: %v", err)
			}

			dstFilename := filepath.Join(t.TempDir(), fmt.Sprintf("test_bitdepth_%d.wav", tc.targetBitDepth))
			if err := EncodeFile(audio, dstFilename, OptionBitDepth(tc.targetBitDepth)); err != nil {
				t.Fatalf("Failed to encode with bit depth conversion: %v", err)
			}

			decodedAudio, err := DecodeFile(dstFilename)
			if err != nil {
				t.Fatalf("Failed to decode encoded file: %v", err)
			}

			if decodedAudio.BitDepth != tc.targetBitDepth {
				t.Errorf("Bit depth mismatch: expected %d, got %d", tc.targetBitDepth, decodedAudio.BitDepth)
			}
			if len(decodedAudio.Data) != originalSamples {
				t.Errorf("Sample count mismatch: expected %d, got %d", originalSamples, len(decodedAudio.Data))
			}
		})
	}
}

func TestInvalidBitDepth(t *testing.T) {
	audio := &Audio{
		SampleRate: 44100,
		BitDepth:   16,
		Data:       []int{0, 100, 200, 300, 400},
		Duration:   0.0001,
	}

	err := convertBitDepth(audio, 12)
	if err == nil {
		t.Fatal("Expected error for invalid bit depth, got nil")
	}
}

func TestBitDepthAndSampleRateConversion(t *testing.T) {
	srcFilename := filepath.Join(t.TempDir(), "stereo.wav")
	writeWAV(t, srcFilename, 2)

	audio, err := DecodeFile(srcFilename)
	if err != nil {
		t.Fatalf("Failed to decode source file: %v", err)
	}

	dstFilename := filepath.Join(t.TempDir(), "test_combined_conversion.wav")

	targetBitDepth := 24
	targetSampleRate := 48000
	if err := EncodeFile(audio, dstFilename,
		OptionBitDepth(targetBitDepth),
		OptionSampleRate(targetSampleRate)); err != nil {
		t.Fatalf("Failed to encode with combined conversion: %v", err)
	}

	decodedAudio, err := DecodeFile(dstFilename)
	if err != nil {
		t.Fatalf("Failed to decode encoded file: %v", err)
	}

	if decodedAudio.BitDepth != targetBitDepth {
		t.Errorf("Bit depth mismatch: expected %d, got %d", targetBitDepth, decodedAudio.BitDepth)
	}
	if decodedAudio.SampleRate != targetSampleRate {
		t.Errorf("Sample rate mismatch: expected %d, got %d", targetSampleRate, decodedAudio.SampleRate)
	}

	t.Logf("Combined conversion test passed: %d Hz @ %d-bit",
		decodedAudio.SampleRate, decodedAudio.BitDepth)
}
