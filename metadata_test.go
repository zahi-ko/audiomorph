package audiomorph

import (
	"math"
	"path/filepath"
	"testing"
)

func TestDecodeMetadataWAV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stereo.wav")
	writeWAV(t, path, 2)

	meta, err := DecodeMetadataFile(path)
	if err != nil {
		t.Fatalf("Failed to read WAV metadata: %v", err)
	}
	if meta.Data != nil {
		t.Error("Expected Data to be nil for metadata-only decode")
	}
	if meta.SampleRate != fixtureSampleRate || meta.BitDepth != fixtureBitDepth {
		t.Errorf("Unexpected metadata: %+v", meta)
	}
	if math.Abs(meta.Duration-fixtureDuration) > 0.01 {
		t.Errorf("Expected Duration ~%.2f, got %.2f", fixtureDuration, meta.Duration)
	}

	decoded, err := DecodeFile(path)
	if err != nil {
		t.Fatalf("Failed to decode WAV file: %v", err)
	}
	if meta.SampleRate != decoded.SampleRate ||
		meta.BitDepth != decoded.BitDepth ||
		math.Abs(meta.Duration-decoded.Duration) > 0.01 {
		t.Errorf("Metadata %+v inconsistent with decoded %+v", meta, decoded)
	}
}

func TestDecodeMetadataAIFF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stereo.aiff")
	writeAIFF(t, path, 2)

	meta, err := DecodeMetadataFile(path)
	if err != nil {
		t.Fatalf("Failed to read AIFF metadata: %v", err)
	}
	if meta.Data != nil {
		t.Error("Expected Data to be nil for metadata-only decode")
	}
	if meta.SampleRate != fixtureSampleRate || meta.BitDepth != fixtureBitDepth {
		t.Errorf("Unexpected metadata: %+v", meta)
	}
	if math.Abs(meta.Duration-fixtureDuration) > 0.01 {
		t.Errorf("Expected Duration ~%.2f, got %.2f", fixtureDuration, meta.Duration)
	}
}

func TestDecodeMetadataMP3(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mono.mp3")
	writeMP3(t, path)

	meta, err := DecodeMetadataFile(path)
	if err != nil {
		t.Fatalf("Failed to read MP3 metadata: %v", err)
	}
	if meta.Data != nil {
		t.Error("Expected Data to be nil for metadata-only decode")
	}
	if meta.SampleRate != fixtureSampleRate {
		t.Errorf("Expected SampleRate %d, got %d", fixtureSampleRate, meta.SampleRate)
	}
	if meta.Duration <= 0 {
		t.Errorf("Expected Duration > 0, got %f", meta.Duration)
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
	path := filepath.Join(t.TempDir(), "mono.flac")
	writeFLAC(t, path)

	meta, err := DecodeMetadataFile(path)
	if err != nil {
		t.Fatalf("Failed to read FLAC metadata: %v", err)
	}
	if meta.Data != nil {
		t.Error("Expected Data to be nil for metadata-only decode")
	}
	if meta.SampleRate != fixtureSampleRate || meta.BitDepth != fixtureBitDepth {
		t.Errorf("Unexpected metadata: %+v", meta)
	}
	// Note: streaming FLAC encoders may leave STREAMINFO.NSamples at zero,
	// so Duration is intentionally not asserted here.
}
