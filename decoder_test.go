package audiomorph

import (
	"bytes"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"

	mp3enc "github.com/braheezy/shine-mp3/pkg/mp3"
	goaudio "github.com/go-audio/audio"
	"github.com/go-audio/aiff"
	"github.com/go-audio/wav"
	"github.com/schollz/goflac"
)

const (
	fixtureSampleRate = 44100
	fixtureBitDepth   = 16
	fixtureDuration   = 0.5 // seconds
)

var fixtureNumSamples = int(fixtureSampleRate * fixtureDuration)

// fixtureLeft/fixtureRight generate deterministic 440 Hz / 880 Hz sine
// samples for synthetic fixtures.
func fixtureLeft(i int) int {
	return int(12000 * math.Sin(2*math.Pi*440*float64(i)/fixtureSampleRate))
}

func fixtureRight(i int) int {
	return int(8000 * math.Sin(2*math.Pi*880*float64(i)/fixtureSampleRate))
}

// requireFixture skips the test when a real sample file is not present.
// Used for formats without an in-process encoder (e.g. OGG).
func requireFixture(t *testing.T, name string) string {
	t.Helper()
	filename := filepath.Join("data", name)
	if _, err := os.Stat(filename); err != nil {
		t.Skipf("fixture %s not available, skipping", filename)
	}
	return filename
}

func intBuffer(data []int, channels int) *goaudio.IntBuffer {
	return &goaudio.IntBuffer{
		Format:         &goaudio.Format{NumChannels: channels, SampleRate: fixtureSampleRate},
		Data:           data,
		SourceBitDepth: fixtureBitDepth,
	}
}

func interleavedFixture(channels int) []int {
	data := make([]int, 0, fixtureNumSamples*channels)
	for i := 0; i < fixtureNumSamples; i++ {
		l, r := fixtureLeft(i), fixtureRight(i)
		if channels == 1 {
			data = append(data, l)
		} else {
			data = append(data, l, r)
		}
	}
	return data
}

// writeWAV writes a synthetic WAV file with the given channel count.
func writeWAV(t *testing.T, path string, channels int) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create WAV fixture: %v", err)
	}
	defer f.Close()

	enc := wav.NewEncoder(f, fixtureSampleRate, fixtureBitDepth, channels, 1)
	if err := enc.Write(intBuffer(interleavedFixture(channels), channels)); err != nil {
		t.Fatalf("failed to write WAV fixture: %v", err)
	}
	if err := enc.Close(); err != nil {
		t.Fatalf("failed to close WAV fixture: %v", err)
	}
}

// writeAIFF writes a synthetic AIFF file with the given channel count.
func writeAIFF(t *testing.T, path string, channels int) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create AIFF fixture: %v", err)
	}
	defer f.Close()

	enc := aiff.NewEncoder(f, fixtureSampleRate, fixtureBitDepth, channels)
	if err := enc.Write(intBuffer(interleavedFixture(channels), channels)); err != nil {
		t.Fatalf("failed to write AIFF fixture: %v", err)
	}
	if err := enc.Close(); err != nil {
		t.Fatalf("failed to close AIFF fixture: %v", err)
	}
}

// writeMP3 writes a synthetic mono MP3 file.
func writeMP3(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create MP3 fixture: %v", err)
	}
	defer f.Close()

	samples := make([]int16, fixtureNumSamples)
	for i := range samples {
		samples[i] = int16(fixtureLeft(i))
	}
	enc := mp3enc.NewEncoder(fixtureSampleRate, 1)
	if err := enc.Write(f, samples); err != nil {
		t.Fatalf("failed to write MP3 fixture: %v", err)
	}
}

// writeFLAC writes a synthetic mono FLAC file.
func writeFLAC(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create FLAC fixture: %v", err)
	}
	defer f.Close()

	samples := make([]int32, fixtureNumSamples)
	for i := range samples {
		samples[i] = int32(fixtureLeft(i))
	}
	enc, err := goflac.NewEncoder(f, fixtureSampleRate, 1, fixtureBitDepth)
	if err != nil {
		t.Fatalf("failed to create FLAC encoder: %v", err)
	}
	if err := enc.Encode([][]int32{samples}); err != nil {
		t.Fatalf("failed to write FLAC fixture: %v", err)
	}
}

func TestDecodeWAV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stereo.wav")
	writeWAV(t, path, 2)

	audio, err := DecodeFile(path)
	if err != nil {
		t.Fatalf("Failed to decode WAV file: %v", err)
	}

	if audio.SampleRate != fixtureSampleRate {
		t.Errorf("SampleRate mismatch: expected %d, got %d", fixtureSampleRate, audio.SampleRate)
	}
	if audio.BitDepth != fixtureBitDepth {
		t.Errorf("BitDepth mismatch: expected %d, got %d", fixtureBitDepth, audio.BitDepth)
	}
	if audio.Format != "wav" {
		t.Errorf("Format mismatch: expected \"wav\", got %q", audio.Format)
	}
	if len(audio.Data) != fixtureNumSamples {
		t.Fatalf("Expected mono downmix to %d samples, got %d", fixtureNumSamples, len(audio.Data))
	}
	for i := range audio.Data {
		want := (float32(fixtureLeft(i))/32768 + float32(fixtureRight(i))/32768) / 2
		if audio.Data[i] != want {
			t.Errorf("Downmix mismatch at %d: expected %v, got %v", i, want, audio.Data[i])
			break
		}
	}
	if math.Abs(audio.Duration-fixtureDuration) > 0.01 {
		t.Errorf("Duration mismatch: expected ~%.2f, got %.2f", fixtureDuration, audio.Duration)
	}
}

func TestDecodeAIFF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stereo.aiff")
	writeAIFF(t, path, 2)

	audio, err := DecodeFile(path)
	if err != nil {
		t.Fatalf("Failed to decode AIFF file: %v", err)
	}

	if audio.SampleRate != fixtureSampleRate {
		t.Errorf("SampleRate mismatch: expected %d, got %d", fixtureSampleRate, audio.SampleRate)
	}
	if audio.BitDepth != fixtureBitDepth {
		t.Errorf("BitDepth mismatch: expected %d, got %d", fixtureBitDepth, audio.BitDepth)
	}
	if audio.Format != "aiff" {
		t.Errorf("Format mismatch: expected \"aiff\", got %q", audio.Format)
	}
	if len(audio.Data) != fixtureNumSamples {
		t.Fatalf("Expected mono downmix to %d samples, got %d", fixtureNumSamples, len(audio.Data))
	}
	for i := range audio.Data {
		want := (float32(fixtureLeft(i))/32768 + float32(fixtureRight(i))/32768) / 2
		if audio.Data[i] != want {
			t.Errorf("Downmix mismatch at %d: expected %v, got %v", i, want, audio.Data[i])
			break
		}
	}
	if math.Abs(audio.Duration-fixtureDuration) > 0.01 {
		t.Errorf("Duration mismatch: expected ~%.2f, got %.2f", fixtureDuration, audio.Duration)
	}
}

func TestDecodeMP3(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mono.mp3")
	writeMP3(t, path)

	audio, err := DecodeFile(path)
	if err != nil {
		t.Fatalf("Failed to decode MP3 file: %v", err)
	}

	if audio.SampleRate != fixtureSampleRate {
		t.Errorf("SampleRate mismatch: expected %d, got %d", fixtureSampleRate, audio.SampleRate)
	}
	if audio.Format != "mp3" {
		t.Errorf("Format mismatch: expected \"mp3\", got %q", audio.Format)
	}
	if len(audio.Data) == 0 {
		t.Error("Expected audio Data to be non-empty")
	}
	if audio.Duration <= 0 {
		t.Errorf("Expected Duration > 0, got %f", audio.Duration)
	}
}

func TestDecodeOGG(t *testing.T) {
	filename := requireFixture(t, "wilhelm.ogg")

	audio, err := DecodeFile(filename)
	if err != nil {
		t.Fatalf("Failed to decode OGG file: %v", err)
	}

	if audio.Format != "ogg" {
		t.Errorf("Format mismatch: expected \"ogg\", got %q", audio.Format)
	}
	if len(audio.Data) == 0 {
		t.Error("Expected audio Data to be non-empty")
	}
}

func TestDecodeFLAC(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mono.flac")
	writeFLAC(t, path)

	audio, err := DecodeFile(path)
	if err != nil {
		t.Fatalf("Failed to decode FLAC file: %v", err)
	}

	if audio.SampleRate != fixtureSampleRate {
		t.Errorf("SampleRate mismatch: expected %d, got %d", fixtureSampleRate, audio.SampleRate)
	}
	if audio.BitDepth != fixtureBitDepth {
		t.Errorf("BitDepth mismatch: expected %d, got %d", fixtureBitDepth, audio.BitDepth)
	}
	if audio.Format != "flac" {
		t.Errorf("Format mismatch: expected \"flac\", got %q", audio.Format)
	}
	if len(audio.Data) != fixtureNumSamples {
		t.Fatalf("Expected %d samples, got %d", fixtureNumSamples, len(audio.Data))
	}
	for i := range audio.Data {
		if audio.Data[i] != float32(fixtureLeft(i))/32768 {
			t.Errorf("Sample mismatch at %d: expected %v, got %v", i, float32(fixtureLeft(i))/32768, audio.Data[i])
			break
		}
	}
}

func TestDecodeUnsupportedFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "junk.unknown")
	if err := os.WriteFile(path, []byte("this is not audio"), 0o644); err != nil {
		t.Fatalf("Failed to write junk file: %v", err)
	}

	if _, err := DecodeFile(path); err == nil {
		t.Fatal("Expected error for unsupported format, got nil")
	}
}

func TestDetectFormat(t *testing.T) {
	dir := t.TempDir()
	writeWAV(t, filepath.Join(dir, "test.wav"), 2)
	writeAIFF(t, filepath.Join(dir, "test.aiff"), 2)
	writeMP3(t, filepath.Join(dir, "test.mp3"))
	writeFLAC(t, filepath.Join(dir, "test.flac"))

	testCases := []struct {
		filename string
		want     string
	}{
		{"test.wav", "wav"},
		{"test.aiff", "aiff"},
		{"test.mp3", "mp3"},
		{"test.flac", "flac"},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			f, err := os.Open(filepath.Join(dir, tc.filename))
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

	t.Run("wilhelm.ogg", func(t *testing.T) {
		filename := requireFixture(t, "wilhelm.ogg")
		f, err := os.Open(filename)
		if err != nil {
			t.Skipf("Failed to open file: %v", err)
		}
		defer f.Close()

		got, err := DetectFormat(f)
		if err != nil {
			t.Fatalf("Failed to detect format: %v", err)
		}
		if got != "ogg" {
			t.Errorf("Format mismatch: expected ogg, got %s", got)
		}
	})
}

func TestDecodeAudioFromReader(t *testing.T) {
	dir := t.TempDir()
	writeWAV(t, filepath.Join(dir, "test.wav"), 2)
	writeAIFF(t, filepath.Join(dir, "test.aiff"), 2)
	writeMP3(t, filepath.Join(dir, "test.mp3"))
	writeFLAC(t, filepath.Join(dir, "test.flac"))

	testCases := []struct {
		filename string
		want     string
	}{
		{"test.wav", "wav"},
		{"test.aiff", "aiff"},
		{"test.mp3", "mp3"},
		{"test.flac", "flac"},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			fileData, err := os.ReadFile(filepath.Join(dir, tc.filename))
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
			if audio.SampleRate <= 0 {
				t.Errorf("Expected SampleRate > 0, got %d", audio.SampleRate)
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
	path := filepath.Join(t.TempDir(), "test.wav")
	writeWAV(t, path, 2)

	audio, err := DecodeFile(path)
	if err != nil {
		t.Fatalf("Failed to decode WAV file: %v", err)
	}
	if audio.Format != "wav" {
		t.Errorf("Expected Format \"wav\", got %q", audio.Format)
	}
}

// TestReaderPositionRestored verifies that stream-consuming functions restore
// the read position of the io.ReadSeeker, so the same reader can be reused
// for subsequent calls.
func TestReaderPositionRestored(t *testing.T) {
	dir := t.TempDir()
	writeWAV(t, filepath.Join(dir, "test.wav"), 2)
	writeMP3(t, filepath.Join(dir, "test.mp3"))

	testCases := []struct {
		filename string
		want     string
	}{
		{"test.wav", "wav"},
		{"test.mp3", "mp3"},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			fileData, err := os.ReadFile(filepath.Join(dir, tc.filename))
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			// DetectFormat must not consume the stream.
			r := bytes.NewReader(fileData)
			if _, err := r.Seek(7, io.SeekStart); err != nil {
				t.Fatalf("Seek failed: %v", err)
			}
			format, err := DetectFormat(r)
			if err != nil {
				t.Fatalf("DetectFormat failed: %v", err)
			}
			if format != tc.want {
				t.Errorf("Format mismatch: expected %s, got %s", tc.want, format)
			}
			if pos, err := r.Seek(0, io.SeekCurrent); err != nil || pos != 7 {
				t.Errorf("DetectFormat did not restore position: pos = %d, err = %v", pos, err)
			}

			// Decode must leave the reader reusable: decode twice from the
			// same reader.
			r = bytes.NewReader(fileData)
			for i := 0; i < 2; i++ {
				audio, err := Decode(r)
				if err != nil {
					t.Fatalf("Decode call %d failed: %v", i+1, err)
				}
				if audio.Format != tc.want {
					t.Errorf("Decode call %d: Format mismatch: expected %s, got %s", i+1, tc.want, audio.Format)
				}
			}

			// DecodeMetadata must leave the reader reusable too.
			r = bytes.NewReader(fileData)
			for i := 0; i < 2; i++ {
				meta, err := DecodeMetadata(r)
				if err != nil {
					t.Fatalf("DecodeMetadata call %d failed: %v", i+1, err)
				}
				if meta.Format != tc.want {
					t.Errorf("DecodeMetadata call %d: Format mismatch: expected %s, got %s", i+1, tc.want, meta.Format)
				}
			}
		})
	}
}
