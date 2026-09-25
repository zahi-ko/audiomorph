package audiomorph

import (
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

// DecodeMetadataFile opens the file at filename and reads only its
// container/codec headers. The returned Audio carries format metadata
// (SampleRate, BitDepth, Format, Duration) with Data left nil,
// so no PCM samples are decoded.
func DecodeMetadataFile(filename string) (*Audio, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	return DecodeMetadata(f)
}

// DecodeMetadata reads audio metadata from an io.ReadSeeker without decoding
// any PCM data. The format is detected automatically by sniffing the leading
// bytes. The returned Audio has all metadata fields populated and Data nil.
func DecodeMetadata(r io.ReadSeeker) (*Audio, error) {
	format, err := DetectFormat(r)
	if err != nil {
		return nil, err
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to start: %w", err)
	}

	switch format {
	case "wav":
		return metadataWAV(r)
	case "aiff":
		return metadataAIFF(r)
	case "mp3":
		return metadataMP3(r)
	case "ogg":
		return metadataOGG(r)
	case "flac":
		return metadataFLAC(r)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// metadataWAV parses the RIFF header of a WAV stream. Duration is derived
// from the data chunk size and byte rate, without reading any PCM samples.
func metadataWAV(r io.ReadSeeker) (*Audio, error) {
	decoder := wav.NewDecoder(r)
	if !decoder.IsValidFile() {
		return nil, fmt.Errorf("invalid WAV file")
	}

	format := decoder.Format()
	dur, err := decoder.Duration()
	if err != nil {
		return nil, fmt.Errorf("failed to compute WAV duration: %w", err)
	}

	return &Audio{
		SampleRate: int(format.SampleRate),
		BitDepth:   int(decoder.BitDepth),
		Format:     "wav",
		Duration:   dur.Seconds(),
	}, nil
}

// metadataAIFF parses the COMM chunk of an AIFF stream. Duration is derived
// from NumSampleFrames and SampleRate, without reading any PCM samples.
func metadataAIFF(r io.ReadSeeker) (*Audio, error) {
	decoder := aiff.NewDecoder(r)
	if !decoder.IsValidFile() {
		return nil, fmt.Errorf("invalid AIFF file")
	}

	dur, err := decoder.Duration()
	if err != nil {
		return nil, fmt.Errorf("failed to compute AIFF duration: %w", err)
	}

	return &Audio{
		SampleRate: int(decoder.SampleRate),
		BitDepth:   int(decoder.BitDepth),
		Format:     "aiff",
		Duration:   dur.Seconds(),
	}, nil
}

// metadataMP3 parses MP3 frame headers only. The beep mp3 decoder is lazy:
// constructing it reads header information, and Len() reports the estimated
// total sample count without decoding audio.
func metadataMP3(r io.ReadSeeker) (*Audio, error) {
	streamer, format, err := mp3.Decode(nopReadSeekCloser{r})
	if err != nil {
		return nil, fmt.Errorf("failed to parse MP3 file: %w", err)
	}
	defer streamer.Close()

	return streamToMetadata(streamer, format, "mp3")
}

// metadataOGG parses the Ogg/Vorbis identification and setup headers. The
// reader locates the last granule position to report duration without
// decoding any audio packets.
func metadataOGG(r io.ReadSeeker) (*Audio, error) {
	streamer, format, err := vorbis.Decode(nopReadSeekCloser{r})
	if err != nil {
		return nil, fmt.Errorf("failed to parse OGG file: %w", err)
	}
	defer streamer.Close()

	return streamToMetadata(streamer, format, "ogg")
}

// metadataFLAC parses the STREAMINFO metadata block of a FLAC stream.
// Duration is taken from NSamples, without decoding any audio frames. If the
// stream does not declare a total sample count (e.g. files written by a
// streaming encoder that leaves STREAMINFO.NSamples at zero), Duration is 0.
func metadataFLAC(r io.Reader) (*Audio, error) {
	stream, err := flac.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse FLAC file: %w", err)
	}
	defer stream.Close()

	info := stream.Info
	if info.SampleRate == 0 {
		return nil, fmt.Errorf("invalid FLAC stream: zero sample rate")
	}

	duration := 0.0
	if info.NSamples > 0 {
		duration = float64(info.NSamples) / float64(info.SampleRate)
	}

	return &Audio{
		SampleRate: int(info.SampleRate),
		BitDepth:   int(info.BitsPerSample),
		Format:     "flac",
		Duration:   duration,
	}, nil
}

// streamToMetadata builds a metadata-only Audio from a lazily constructed
// beep streamer, using the same reported values as full decoding so that
// DecodeMetadata stays consistent with Decode.
func streamToMetadata(streamer beep.StreamSeekCloser, format beep.Format, formatName string) (*Audio, error) {
	if format.SampleRate == 0 {
		return nil, fmt.Errorf("invalid %s stream: zero sample rate", formatName)
	}

	return &Audio{
		SampleRate: int(format.SampleRate),
		BitDepth:   format.Precision * 8,
		Format:     formatName,
		Duration:   float64(streamer.Len()) / float64(format.SampleRate),
	}, nil
}
