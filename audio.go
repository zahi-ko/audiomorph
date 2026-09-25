package audiomorph

// Audio represents decoded audio data. Multi-channel sources are downmixed
// to mono during decoding, so Data always holds single-channel samples.
type Audio struct {
	SampleRate          int
	BitDepth            int
	Format              string  // Detected source format: "wav", "aiff", "mp3", "ogg", "flac"
	Data                []float32 // Mono PCM data: Data[sample], normalized to [-1.0, 1.0]
	Duration            float64 // in seconds
	targetSampleRate    int
	targetBitDepth      int
	interpolationMethod string
}

// Option is the type all options need to adhere to
type Option func(a *Audio)
