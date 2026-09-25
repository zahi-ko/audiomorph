# audiomorph

[![CI](https://github.com/zahi-ko/audiomorph/actions/workflows/ci.yml/badge.svg)](https://github.com/zahi-ko/audiomorph/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/zahi-ko/audiomorph/branch/main/graph/badge.svg)](https://codecov.io/gh/zahi-ko/audiomorph)
[![Release](https://img.shields.io/github/v/release/zahi-ko/audiomorph)](https://github.com/zahi-ko/audiomorph/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/zahi-ko/audiomorph.svg)](https://pkg.go.dev/github.com/zahi-ko/audiomorph)

A Go library and CLI tool for decoding and encoding audio files across multiple formats. audiomorph provides a unified interface for reading audio data from WAV, AIFF, MP3, OGG, and FLAC files, and encoding to WAV, AIFF, MP3, OGG, and FLAC formats.

## How It Works

audiomorph decodes audio files into a common in-memory representation (`Audio` struct) containing mono PCM data (multi-channel sources are downmixed to mono), then encodes that data to your desired output format. The input format is detected automatically by content sniffing, so files do not need a correct extension. The library handles format-specific quirks and provides a consistent API regardless of the underlying codec.

### Dependencies

This project is grateful for and depends on these excellent Go libraries:

- [faiface/beep](https://github.com/faiface/beep) - Audio decoding for MP3 and OGG Vorbis
- [go-audio/wav](https://github.com/go-audio/wav) - WAV file encoding/decoding
- [go-audio/aiff](https://github.com/go-audio/aiff) - AIFF file encoding/decoding
- [mewkiz/flac](https://github.com/mewkiz/flac) - FLAC file decoding
- [schollz/goflac](https://github.com/schollz/goflac) - FLAC file encoding
- [braheezy/shine-mp3](https://github.com/braheezy/shine-mp3) - MP3 file encoding

## API

The library exposes three primary functions:

```go
// Decode audio from file (WAV, AIFF, MP3, OGG, FLAC — format auto-detected)
audio, err := audiomorph.DecodeFile("input.mp3")

// Decode audio from any io.ReadSeeker (bytes.Reader, bufio.Reader, network stream...)
audio, err := audiomorph.Decode(bytes.NewReader(data))

// Encode audio to file (supports WAV, AIFF, MP3, OGG, FLAC)
err = audiomorph.EncodeFile(audio, "output.wav")
```

The `Audio` struct provides access to all audio properties:

```go
type Audio struct {
    SampleRate  int      // Sample rate in Hz
    BitDepth    int      // Bit depth (bits per sample)
    Format      string   // Detected source format: "wav", "aiff", "mp3", "ogg", "flac"
    Data        []int    // Mono PCM data: Data[sample] (multi-channel sources are downmixed to mono)
    Duration    float64  // Duration in seconds
}
```

### Format Detection

`Decode` and `DecodeFile` identify the input format by sniffing its leading bytes (magic numbers), not by file extension. You can also detect a format on its own:

```go
f, _ := os.Open("mystery_audio")
defer f.Close()

format, err := audiomorph.DetectFormat(f) // "wav", "aiff", "mp3", "ogg", or "flac"
```

## Usage

### Installation

```bash
go install github.com/zahi-ko/audiomorph/cmd/audiomorph@latest
```

### Command Line

Display audio file statistics:

```bash
audiomorph input.mp3
```

Transform audio between formats:

```bash
audiomorph input.mp3 output.wav
audiomorph input.flac output.mp3
audiomorph input.wav output.aiff
```

Convert sample rate during transformation:

```bash
# Upsample to 48kHz using linear interpolation (default)
audiomorph input.wav output.wav --sample-rate 48000

# Downsample to 22.05kHz using Lanczos3 interpolation
audiomorph input.mp3 output.mp3 --sample-rate 22050 --interpolation lanczos3
```

Available interpolation methods: `linear` (default), `cubic`, `hermite`, `lanczos2`, `lanczos3`, `bspline3`, `bspline5`, `monotonic`

Convert bit depth during transformation:

```bash
# Convert to 24-bit audio
audiomorph input.wav output.wav --bit-depth 24

# Convert to 8-bit audio
audiomorph input.flac output.wav --bit-depth 8

# Combine bit depth and sample rate conversion
audiomorph input.mp3 output.wav --bit-depth 24 --sample-rate 48000
```

Supported bit depths: `8`, `16`, `24`, `32`

### Library Usage

```go
import "github.com/zahi-ko/audiomorph"

// Decode audio file (format is detected automatically)
audio, err := audiomorph.DecodeFile("input.mp3")
if err != nil {
    log.Fatal(err)
}

// Process audio data...
fmt.Printf("Duration: %.2f seconds\n", audio.Duration)
fmt.Printf("Sample rate: %d Hz\n", audio.SampleRate)
fmt.Printf("Source format: %s\n", audio.Format)

// Encode to different format
err = audiomorph.EncodeFile(audio, "output.wav")
if err != nil {
    log.Fatal(err)
}

// Encode with sample rate conversion
err = audiomorph.EncodeFile(audio, "output.wav",
    audiomorph.OptionSampleRate(48000),
    audiomorph.OptionInterpolationMethod("lanczos3"))
if err != nil {
    log.Fatal(err)
}

// Encode with bit depth conversion
err = audiomorph.EncodeFile(audio, "output.wav",
    audiomorph.OptionBitDepth(24))
if err != nil {
    log.Fatal(err)
}

// Encode with both sample rate and bit depth conversion
err = audiomorph.EncodeFile(audio, "output.wav",
    audiomorph.OptionSampleRate(48000),
    audiomorph.OptionBitDepth(24))
if err != nil {
    log.Fatal(err)
}
```

## License

MIT
