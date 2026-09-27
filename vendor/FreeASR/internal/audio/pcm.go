package audio

// FrameConfig holds PCM framing parameters.
type FrameConfig struct {
	SampleRate      int // e.g., 16000
	Channels        int // e.g., 1
	FrameDurationMs int // e.g., 20
}

// DefaultFrameConfig returns the standard 16kHz mono 20ms configuration.
func DefaultFrameConfig() FrameConfig {
	return FrameConfig{
		SampleRate:      16000,
		Channels:        1,
		FrameDurationMs: 20,
	}
}

// SamplesPerFrame returns the number of samples in one frame.
func (c FrameConfig) SamplesPerFrame() int {
	return c.SampleRate * c.FrameDurationMs / 1000
}

// BytesPerFrame returns the number of bytes in one frame (16-bit PCM).
func (c FrameConfig) BytesPerFrame() int {
	return c.SamplesPerFrame() * 2 * c.Channels
}

// FramePCM splits PCM data into fixed-size frames, padding the last frame with silence.
func FramePCM(pcm []byte, cfg FrameConfig) [][]byte {
	frameSize := cfg.BytesPerFrame()
	if frameSize <= 0 || len(pcm) == 0 {
		return nil
	}

	numFrames := (len(pcm) + frameSize - 1) / frameSize
	frames := make([][]byte, 0, numFrames)

	for i := 0; i < len(pcm); i += frameSize {
		end := i + frameSize
		if end <= len(pcm) {
			frame := make([]byte, frameSize)
			copy(frame, pcm[i:end])
			frames = append(frames, frame)
		} else {
			// Last frame: pad with zeros (silence)
			frame := make([]byte, frameSize)
			copy(frame, pcm[i:])
			frames = append(frames, frame)
		}
	}

	return frames
}

// PCMDuration returns the duration in seconds for PCM data at the given sample rate.
func PCMDuration(pcmBytes int, sampleRate, channels int) float64 {
	bytesPerSample := 2 * channels // 16-bit
	if bytesPerSample == 0 || sampleRate == 0 {
		return 0
	}
	return float64(pcmBytes) / float64(bytesPerSample) / float64(sampleRate)
}
