package audiomorph

// Audio represents decoded audio data
type Audio struct {
	NumChannels         int
	SampleRate          int
	BitDepth            int
	Format              string  // Detected source format: "wav", "aiff", "mp3", "ogg", "flac"
	Data                []int   // Interleaved PCM data: Data[sample*NumChannels + channel]
	Duration            float64 // in seconds
	useChannels         []int
	targetSampleRate    int
	targetBitDepth      int
	interpolationMethod string
}

// Option is the type all options need to adhere to
type Option func(a *Audio)
