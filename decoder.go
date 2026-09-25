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

// Decode decodes audio from an io.ReadSeeker and returns an Audio struct.
// The format is detected automatically by sniffing the leading bytes.
func Decode(r io.ReadSeeker) (*Audio, error) {
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
// format is not recognized. The read position is restored to the start.
func DetectFormat(r io.ReadSeeker) (string, error) {
	header := make([]byte, 12)
	if _, err := io.ReadFull(r, header); err != nil {
		return "", fmt.Errorf("failed to read file header: %w", err)
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to rewind stream: %w", err)
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

// monoDownmix averages the channels of interleaved PCM data into mono.
func monoDownmix(data []int, numChannels int) []int {
	if numChannels <= 1 {
		return data
	}

	numSamples := len(data) / numChannels
	out := make([]int, 0, numSamples)
	for i := 0; i < numSamples; i++ {
		sum := 0
		for ch := 0; ch < numChannels; ch++ {
			sum += data[i*numChannels+ch]
		}
		out = append(out, sum/numChannels)
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

	return &Audio{
		SampleRate: int(format.SampleRate),
		BitDepth:   int(decoder.BitDepth),
		Format:     "wav",
		Data:       monoDownmix(buf.Data, int(format.NumChannels)),
		Duration:   float64(len(buf.Data)/int(format.NumChannels)) / float64(format.SampleRate),
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

	return &Audio{
		SampleRate: int(format.SampleRate),
		BitDepth:   int(decoder.BitDepth),
		Format:     "aiff",
		Data:       monoDownmix(buf.Data, int(format.NumChannels)),
		Duration:   float64(len(buf.Data)/int(format.NumChannels)) / float64(format.SampleRate),
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

	data := make([]int, 0, totalSamples)

	// Read all frames
	for {
		frame, err := stream.ParseNext()
		if err != nil {
			break
		}

		// Average channels into mono
		for i := 0; i < len(frame.Subframes[0].Samples); i++ {
			sum := 0
			for ch := 0; ch < numChannels; ch++ {
				sum += int(frame.Subframes[ch].Samples[i])
			}
			data = append(data, sum/numChannels)
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
	maxVal := float64(int64(1) << uint(bitDepth-1))

	data := make([]int, 0, length)

	bufSize := 512
	buf := make([][2]float64, bufSize)
	totalRead := 0

	for totalRead < length {
		n, ok := streamer.Stream(buf)
		if !ok && n == 0 {
			break
		}

		for i := 0; i < n; i++ {
			// Convert float64 [-1, 1] back to integer PCM, averaging channels
			sum := 0
			for ch := 0; ch < numChannels; ch++ {
				sum += int(buf[i][ch] * maxVal)
			}
			data = append(data, sum/numChannels)
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
