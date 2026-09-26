package audiomorph

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/vorbis"
	"github.com/go-audio/aiff"
	"github.com/go-audio/wav"
	"github.com/mewkiz/flac"
)

// DecodeFile opens the file at filename and decodes it.
// The format is detected automatically by content sniffing.
func DecodeFile(filename string) (*Audio, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	return Decode(f)
}

// restoreSeeker records the current position of r and returns a function that
// seeks back to it, so stream-consuming functions can restore the caller's
// read position when they finish (successfully or not).
func restoreSeeker(r io.ReadSeeker) func() {
	pos, err := r.Seek(0, io.SeekCurrent)
	return func() {
		if err == nil {
			_, _ = r.Seek(pos, io.SeekStart)
		}
	}
}

// Decode decodes audio from an io.ReadSeeker and returns an Audio struct.
// The format is detected automatically by sniffing the leading bytes.
// The read position of r is restored to where it was on entry when Decode
// returns, so the same reader remains usable for subsequent calls.
func Decode(r io.ReadSeeker) (*Audio, error) {
	restore := restoreSeeker(r)
	defer restore()

	format, err := DetectFormat(r)
	if err != nil {
		return nil, err
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to start: %w", err)
	}

	switch format {
	case "wav":
		return decodeWAV(r)
	case "aiff":
		return decodeAIFF(r)
	case "mp3":
		return decodeMP3(r)
	case "ogg":
		return decodeOGG(r)
	case "flac":
		return decodeFLAC(r)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// DetectFormat sniffs the leading bytes of r to identify the audio format.
// It returns one of "wav", "aiff", "mp3", "ogg", "flac", or an error if the
// format is not recognized. Sniffing always starts from the beginning of the
// stream; the read position of r is restored to where it was on entry when
// DetectFormat returns.
func DetectFormat(r io.ReadSeeker) (string, error) {
	restore := restoreSeeker(r)
	defer restore()

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to rewind stream: %w", err)
	}

	header := make([]byte, 12)
	if _, err := io.ReadFull(r, header); err != nil {
		return "", fmt.Errorf("failed to read file header: %w", err)
	}

	switch {
	case bytes.Equal(header[0:4], []byte("RIFF")) && bytes.Equal(header[8:12], []byte("WAVE")):
		return "wav", nil
	case bytes.Equal(header[0:4], []byte("FORM")) &&
		(bytes.Equal(header[8:12], []byte("AIFF")) || bytes.Equal(header[8:12], []byte("AIFC"))):
		return "aiff", nil
	case bytes.Equal(header[0:4], []byte("OggS")):
		return "ogg", nil
	case bytes.Equal(header[0:4], []byte("fLaC")):
		return "flac", nil
	case bytes.Equal(header[0:3], []byte("ID3")),
		header[0] == 0xFF && header[1]&0xE0 == 0xE0: // MPEG frame sync
		return "mp3", nil
	default:
		return "", fmt.Errorf("unsupported or unrecognized audio format")
	}
}

// normalize converts integer PCM samples of the given bit depth to float32
// in the normalized range [-1.0, 1.0].
func normalize(data []int, bitDepth int) []float32 {
	maxVal := float32(int64(1) << uint(bitDepth-1))
	out := make([]float32, len(data))
	for i, v := range data {
		out[i] = float32(v) / maxVal
	}
	return out
}

// monoDownmix averages the channels of interleaved float32 PCM data into mono.
func monoDownmix(data []float32, numChannels int) []float32 {
	if numChannels <= 1 {
		return data
	}

	numSamples := len(data) / numChannels
	out := make([]float32, 0, numSamples)
	for i := range numSamples {
		sum := float32(0)
		for ch := range numChannels {
			sum += data[i*numChannels+ch]
		}
		out = append(out, sum/float32(numChannels))
	}
	return out
}

// decodeWAV decodes a WAV stream, downmixing multi-channel data to mono.
func decodeWAV(r io.ReadSeeker) (*Audio, error) {
	decoder := wav.NewDecoder(r)
	if !decoder.IsValidFile() {
		return nil, fmt.Errorf("invalid WAV file")
	}

	// Forward to the PCM data section and read the format information
	if err := decoder.FwdToPCM(); err != nil {
		return nil, fmt.Errorf("failed to forward to PCM data: %w", err)
	}
	format := decoder.Format()

	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to read PCM buffer: %w", err)
	}

	numChannels := int(format.NumChannels)

	return &Audio{
		SampleRate: int(format.SampleRate),
		BitDepth:   int(decoder.BitDepth),
		Format:     "wav",
		Data:       monoDownmix(normalize(buf.Data, int(decoder.BitDepth)), numChannels),
		Duration:   float64(len(buf.Data)/numChannels) / float64(format.SampleRate),
	}, nil
}

// decodeAIFF decodes an AIFF/AIFC stream, downmixing multi-channel data to mono.
func decodeAIFF(r io.ReadSeeker) (*Audio, error) {
	decoder := aiff.NewDecoder(r)
	if !decoder.IsValidFile() {
		return nil, fmt.Errorf("invalid AIFF file")
	}

	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to read PCM buffer: %w", err)
	}

	format := buf.Format
	numChannels := int(format.NumChannels)

	return &Audio{
		SampleRate: int(format.SampleRate),
		BitDepth:   int(decoder.BitDepth),
		Format:     "aiff",
		Data:       monoDownmix(normalize(buf.Data, int(decoder.BitDepth)), numChannels),
		Duration:   float64(len(buf.Data)/numChannels) / float64(format.SampleRate),
	}, nil
}

// decodeMP3 decodes an MP3 stream, downmixing multi-channel data to mono.
func decodeMP3(r io.ReadSeeker) (*Audio, error) {
	streamer, format, err := mp3.Decode(nopReadSeekCloser{r})
	if err != nil {
		return nil, fmt.Errorf("failed to decode MP3 file: %w", err)
	}
	defer streamer.Close()

	return streamToAudio(streamer, format, "mp3")
}

// decodeOGG decodes an OGG Vorbis stream, downmixing multi-channel data to mono.
func decodeOGG(r io.ReadSeeker) (*Audio, error) {
	streamer, format, err := vorbis.Decode(nopReadSeekCloser{r})
	if err != nil {
		return nil, fmt.Errorf("failed to decode OGG file: %w", err)
	}
	defer streamer.Close()

	return streamToAudio(streamer, format, "ogg")
}

// decodeFLAC decodes a FLAC stream, downmixing multi-channel data to mono.
func decodeFLAC(r io.Reader) (*Audio, error) {
	stream, err := flac.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse FLAC file: %w", err)
	}
	defer stream.Close()

	// Get metadata
	info := stream.Info
	numChannels := int(info.NChannels)
	sampleRate := int(info.SampleRate)
	bitDepth := int(info.BitsPerSample)
	totalSamples := int(info.NSamples)

	data := make([]float32, 0, totalSamples)
	maxVal := float32(int64(1) << uint(bitDepth-1))

	// Read all frames
	for {
		frame, err := stream.ParseNext()
		if err != nil {
			break
		}

		// Normalize, then average channels into mono
		for i := 0; i < len(frame.Subframes[0].Samples); i++ {
			sum := float32(0)
			for ch := 0; ch < numChannels; ch++ {
				sum += float32(frame.Subframes[ch].Samples[i]) / maxVal
			}
			data = append(data, sum/float32(numChannels))
		}
	}

	return &Audio{
		SampleRate: sampleRate,
		BitDepth:   bitDepth,
		Format:     "flac",
		Data:       data,
		Duration:   float64(len(data)) / float64(sampleRate),
	}, nil
}

// streamToAudio converts a beep.StreamSeekCloser to an Audio struct with mono
// data, averaging the streamer's channels when it is multi-channel.
func streamToAudio(streamer beep.StreamSeekCloser, format beep.Format, formatName string) (*Audio, error) {
	length := streamer.Len()
	numChannels := format.NumChannels
	bitDepth := format.Precision * 8

	data := make([]float32, 0, length)

	bufSize := 512
	buf := make([][2]float64, bufSize)
	totalRead := 0

	for totalRead < length {
		n, ok := streamer.Stream(buf)
		if !ok && n == 0 {
			break
		}

		for i := 0; i < n; i++ {
			// beep streams are already normalized to [-1, 1]; average channels
			sum := 0.0
			for ch := 0; ch < numChannels; ch++ {
				sum += buf[i][ch]
			}
			data = append(data, float32(sum/float64(numChannels)))
		}
		totalRead += n

		if streamer.Err() != nil {
			return nil, fmt.Errorf("error streaming audio: %w", streamer.Err())
		}
	}

	return &Audio{
		SampleRate: int(format.SampleRate),
		BitDepth:   bitDepth,
		Format:     formatName,
		Data:       data,
		Duration:   float64(len(data)) / float64(format.SampleRate),
	}, nil
}

// nopReadSeekCloser adapts an io.ReadSeeker to io.ReadCloser with a no-op
// Close, so beep decoders can consume streams owned by the caller.
type nopReadSeekCloser struct{ io.ReadSeeker }

func (nopReadSeekCloser) Close() error { return nil }
