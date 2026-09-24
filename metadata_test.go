package audiomorph

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// requireFixture skips the test when the sample file is not present.
func requireFixture(t *testing.T, name string) string {
	t.Helper()
	filename := filepath.Join("data", name)
	if _, err := os.Stat(filename); err != nil {
		t.Skipf("fixture %s not available, skipping", filename)
	}
	return filename
}

func TestDecodeMetadataWAV(t *testing.T) {
	filename := requireFixture(t, "wilhelm.wav")

	meta, err := DecodeMetadataFile(filename)
	if err != nil {
		t.Fatalf("Failed to read WAV metadata: %v", err)
	}
	if meta.Data != nil {
		t.Error("Expected Data to be nil for metadata-only decode")
	}
	if meta.NumChannels <= 0 || meta.SampleRate <= 0 || meta.BitDepth <= 0 {
		t.Errorf("Unexpected metadata: %+v", meta)
	}
	if meta.Duration <= 0 {
		t.Errorf("Expected Duration > 0, got %f", meta.Duration)
	}

	decoded, err := DecodeFile(filename)
	if err != nil {
		t.Fatalf("Failed to decode WAV file: %v", err)
	}
	if meta.NumChannels != decoded.NumChannels ||
		meta.SampleRate != decoded.SampleRate ||
		meta.BitDepth != decoded.BitDepth ||
		math.Abs(meta.Duration-decoded.Duration) > 0.01 {
		t.Errorf("Metadata %+v inconsistent with decoded %+v", meta, decoded)
	}
}

func TestDecodeMetadataAIFF(t *testing.T) {
	filename := requireFixture(t, "wilhelm.aiff")

	meta, err := DecodeMetadataFile(filename)
	if err != nil {
		t.Fatalf("Failed to read AIFF metadata: %v", err)
	}
	if meta.Data != nil {
		t.Error("Expected Data to be nil for metadata-only decode")
	}
	if meta.NumChannels <= 0 || meta.SampleRate <= 0 || meta.BitDepth <= 0 || meta.Duration <= 0 {
		t.Errorf("Unexpected metadata: %+v", meta)
	}
}

func TestDecodeMetadataMP3(t *testing.T) {
	filename := requireFixture(t, "wilhelm.mp3")

	meta, err := DecodeMetadataFile(filename)
	if err != nil {
		t.Fatalf("Failed to read MP3 metadata: %v", err)
	}
	if meta.Data != nil {
		t.Error("Expected Data to be nil for metadata-only decode")
	}
	if meta.SampleRate <= 0 || meta.Duration <= 0 {
		t.Errorf("Unexpected metadata: %+v", meta)
	}
}

func TestDecodeMetadataOGG(t *testing.T) {
	filename := requireFixture(t, "wilhelm.ogg")

	meta, err := DecodeMetadataFile(filename)
	if err != nil {
		t.Fatalf("Failed to read OGG metadata: %v", err)
	}
	if meta.Data != nil {
		t.Error("Expected Data to be nil for metadata-only decode")
	}
	if meta.SampleRate <= 0 || meta.Duration <= 0 {
		t.Errorf("Unexpected metadata: %+v", meta)
	}
}

func TestDecodeMetadataFLAC(t *testing.T) {
	filename := requireFixture(t, "wilhelm.flac")

	meta, err := DecodeMetadataFile(filename)
	if err != nil {
		t.Fatalf("Failed to read FLAC metadata: %v", err)
	}
	if meta.Data != nil {
		t.Error("Expected Data to be nil for metadata-only decode")
	}
	if meta.NumChannels <= 0 || meta.SampleRate <= 0 || meta.BitDepth <= 0 || meta.Duration <= 0 {
		t.Errorf("Unexpected metadata: %+v", meta)
	}
}
