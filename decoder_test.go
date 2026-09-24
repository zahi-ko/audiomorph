package audiomorph

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeWAV(t *testing.T) {
	filename := filepath.Join("data", "wilhelm.wav")

	audio, err := DecodeFile(filename)
	if err != nil {
		t.Fatalf("Failed to decode WAV file: %v", err)
	}

	// Verify that we got some data
	if audio == nil {
		t.Fatal("Audio is nil")
	}

	// Check that basic fields are populated
	if audio.NumChannels <= 0 {
		t.Errorf("Expected NumChannels > 0, got %d", audio.NumChannels)
	}
	if audio.SampleRate <= 0 {
		t.Errorf("Expected SampleRate > 0, got %d", audio.SampleRate)
	}
	if audio.BitDepth <= 0 {
		t.Errorf("Expected BitDepth > 0, got %d", audio.BitDepth)
	}
	if len(audio.Data) == 0 {
		t.Error("Expected audio Data to be non-empty")
	}
	if len(audio.Data)%audio.NumChannels != 0 {
		t.Errorf("Expected interleaved Data length %d to be a multiple of %d channels", len(audio.Data), audio.NumChannels)
	}
	if audio.Duration <= 0 {
		t.Errorf("Expected Duration > 0, got %f", audio.Duration)
	}

	t.Logf("WAV Audio Info:")
	t.Logf("  NumChannels: %d", audio.NumChannels)
	t.Logf("  SampleRate: %d", audio.SampleRate)
	t.Logf("  BitDepth: %d", audio.BitDepth)
	t.Logf("  Data length: %d interleaved samples", len(audio.Data))
	t.Logf("  Duration: %.2f seconds", audio.Duration)
}

func TestDecodeAIFF(t *testing.T) {
	filename := filepath.Join("data", "wilhelm.aiff")

	audio, err := DecodeFile(filename)
	if err != nil {
		t.Fatalf("Failed to decode AIFF file: %v", err)
	}

	// Verify that we got some data
	if audio == nil {
		t.Fatal("Audio is nil")
	}

	// Check that basic fields are populated
	if audio.NumChannels <= 0 {
		t.Errorf("Expected NumChannels > 0, got %d", audio.NumChannels)
	}
	if audio.SampleRate <= 0 {
		t.Errorf("Expected SampleRate > 0, got %d", audio.SampleRate)
	}
	if audio.BitDepth <= 0 {
		t.Errorf("Expected BitDepth > 0, got %d", audio.BitDepth)
	}
	if len(audio.Data) == 0 {
		t.Error("Expected audio Data to be non-empty")
	}
	if len(audio.Data)%audio.NumChannels != 0 {
		t.Errorf("Expected interleaved Data length %d to be a multiple of %d channels", len(audio.Data), audio.NumChannels)
	}
	if audio.Duration <= 0 {
		t.Errorf("Expected Duration > 0, got %f", audio.Duration)
	}

	t.Logf("AIFF Audio Info:")
	t.Logf("  NumChannels: %d", audio.NumChannels)
	t.Logf("  SampleRate: %d", audio.SampleRate)
	t.Logf("  BitDepth: %d", audio.BitDepth)
	t.Logf("  Data length: %d interleaved samples", len(audio.Data))
	t.Logf("  Duration: %.2f seconds", audio.Duration)
}

func TestDecodeMP3(t *testing.T) {
	filename := filepath.Join("data", "wilhelm.mp3")

	audio, err := DecodeFile(filename)
	if err != nil {
		t.Fatalf("Failed to decode MP3 file: %v", err)
	}

	// Verify that we got some data
	if audio == nil {
		t.Fatal("Audio is nil")
	}

	// Check that basic fields are populated
	if audio.NumChannels <= 0 {
		t.Errorf("Expected NumChannels > 0, got %d", audio.NumChannels)
	}
	if audio.SampleRate <= 0 {
		t.Errorf("Expected SampleRate > 0, got %d", audio.SampleRate)
	}
	if audio.BitDepth <= 0 {
		t.Errorf("Expected BitDepth > 0, got %d", audio.BitDepth)
	}
	if len(audio.Data) == 0 {
		t.Error("Expected audio Data to be non-empty")
	}
	if len(audio.Data)%audio.NumChannels != 0 {
		t.Errorf("Expected interleaved Data length %d to be a multiple of %d channels", len(audio.Data), audio.NumChannels)
	}
	if audio.Duration <= 0 {
		t.Errorf("Expected Duration > 0, got %f", audio.Duration)
	}

	t.Logf("MP3 Audio Info:")
	t.Logf("  NumChannels: %d", audio.NumChannels)
	t.Logf("  SampleRate: %d", audio.SampleRate)
	t.Logf("  BitDepth: %d", audio.BitDepth)
	t.Logf("  Data length: %d interleaved samples", len(audio.Data))
	t.Logf("  Duration: %.2f seconds", audio.Duration)
}

func TestDecodeOGG(t *testing.T) {
	filename := filepath.Join("data", "wilhelm.ogg")

	audio, err := DecodeFile(filename)
	if err != nil {
		t.Fatalf("Failed to decode OGG file: %v", err)
	}

	// Verify that we got some data
	if audio == nil {
		t.Fatal("Audio is nil")
	}

	// Check that basic fields are populated
	if audio.NumChannels <= 0 {
		t.Errorf("Expected NumChannels > 0, got %d", audio.NumChannels)
	}
	if audio.SampleRate <= 0 {
		t.Errorf("Expected SampleRate > 0, got %d", audio.SampleRate)
	}
	if audio.BitDepth <= 0 {
		t.Errorf("Expected BitDepth > 0, got %d", audio.BitDepth)
	}
	if len(audio.Data) == 0 {
		t.Error("Expected audio Data to be non-empty")
	}
	if len(audio.Data)%audio.NumChannels != 0 {
		t.Errorf("Expected interleaved Data length %d to be a multiple of %d channels", len(audio.Data), audio.NumChannels)
	}
	if audio.Duration <= 0 {
		t.Errorf("Expected Duration > 0, got %f", audio.Duration)
	}

	t.Logf("OGG Audio Info:")
	t.Logf("  NumChannels: %d", audio.NumChannels)
	t.Logf("  SampleRate: %d", audio.SampleRate)
	t.Logf("  BitDepth: %d", audio.BitDepth)
	t.Logf("  Data length: %d interleaved samples", len(audio.Data))
	t.Logf("  Duration: %.2f seconds", audio.Duration)
}

func TestDecodeFLAC(t *testing.T) {
	filename := filepath.Join("data", "wilhelm.flac")

	audio, err := DecodeFile(filename)
	if err != nil {
		t.Fatalf("Failed to decode FLAC file: %v", err)
	}

	// Verify that we got some data
	if audio == nil {
		t.Fatal("Audio is nil")
	}

	// Check that basic fields are populated
	if audio.NumChannels <= 0 {
		t.Errorf("Expected NumChannels > 0, got %d", audio.NumChannels)
	}
	if audio.SampleRate <= 0 {
		t.Errorf("Expected SampleRate > 0, got %d", audio.SampleRate)
	}
	if audio.BitDepth <= 0 {
		t.Errorf("Expected BitDepth > 0, got %d", audio.BitDepth)
	}
	if len(audio.Data) == 0 {
		t.Error("Expected audio Data to be non-empty")
	}
	if len(audio.Data)%audio.NumChannels != 0 {
		t.Errorf("Expected interleaved Data length %d to be a multiple of %d channels", len(audio.Data), audio.NumChannels)
	}
	if audio.Duration <= 0 {
		t.Errorf("Expected Duration > 0, got %f", audio.Duration)
	}

	t.Logf("FLAC Audio Info:")
	t.Logf("  NumChannels: %d", audio.NumChannels)
	t.Logf("  SampleRate: %d", audio.SampleRate)
	t.Logf("  BitDepth: %d", audio.BitDepth)
	t.Logf("  Data length: %d interleaved samples", len(audio.Data))
	t.Logf("  Duration: %.2f seconds", audio.Duration)
}

func TestDecodeUnsupportedFormat(t *testing.T) {
	filename := filepath.Join("data", "wilhelm.unknown")

	_, err := DecodeFile(filename)
	if err == nil {
		t.Fatal("Expected error for unsupported format, got nil")
	}
}

func TestDetectFormat(t *testing.T) {
	testCases := []struct {
		filename string
		want     string
	}{
		{"wilhelm.wav", "wav"},
		{"wilhelm.aiff", "aiff"},
		{"wilhelm.mp3", "mp3"},
		{"wilhelm.ogg", "ogg"},
		{"wilhelm.flac", "flac"},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			f, err := os.Open(filepath.Join("data", tc.filename))
			if err != nil {
				t.Fatalf("Failed to open file: %v", err)
			}
			defer f.Close()

			got, err := DetectFormat(f)
			if err != nil {
				t.Fatalf("Failed to detect format: %v", err)
			}
			if got != tc.want {
				t.Errorf("Format mismatch: expected %s, got %s", tc.want, got)
			}
		})
	}
}

func TestDecodeAudioFromReader(t *testing.T) {
	testCases := []struct {
		filename string
		want     string
	}{
		{"wilhelm.wav", "wav"},
		{"wilhelm.aiff", "aiff"},
		{"wilhelm.mp3", "mp3"},
		{"wilhelm.ogg", "ogg"},
		{"wilhelm.flac", "flac"},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			fileData, err := os.ReadFile(filepath.Join("data", tc.filename))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			audio, err := Decode(bytes.NewReader(fileData))
			if err != nil {
				t.Fatalf("Failed to decode from reader: %v", err)
			}

			if audio.Format != tc.want {
				t.Errorf("Format mismatch: expected %s, got %s", tc.want, audio.Format)
			}
			if audio.NumChannels <= 0 || audio.SampleRate <= 0 {
				t.Errorf("Expected valid format info, got channels=%d rate=%d", audio.NumChannels, audio.SampleRate)
			}
			if len(audio.Data) == 0 {
				t.Error("Expected audio Data to be non-empty")
			}
			if audio.Duration <= 0 {
				t.Errorf("Expected Duration > 0, got %f", audio.Duration)
			}
		})
	}
}

func TestDecodeFileSetsFormat(t *testing.T) {
	audio, err := DecodeFile(filepath.Join("data", "wilhelm.ogg"))
	if err != nil {
		t.Fatalf("Failed to decode OGG file: %v", err)
	}
	if audio.Format != "ogg" {
		t.Errorf("Expected Format \"ogg\", got %q", audio.Format)
	}
}
