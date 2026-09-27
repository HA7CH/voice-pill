package audio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFrameConfig_Defaults(t *testing.T) {
	cfg := DefaultFrameConfig()
	assert.Equal(t, 320, cfg.SamplesPerFrame())
	assert.Equal(t, 640, cfg.BytesPerFrame())
}

func TestFramePCM_ExactFrames(t *testing.T) {
	cfg := DefaultFrameConfig()
	pcm := make([]byte, 640*3) // exactly 3 frames
	frames := FramePCM(pcm, cfg)
	assert.Len(t, frames, 3)
	for _, f := range frames {
		assert.Len(t, f, 640)
	}
}

func TestFramePCM_PartialLastFrame(t *testing.T) {
	cfg := DefaultFrameConfig()
	pcm := make([]byte, 640*2+100) // 2 full frames + 100 bytes
	frames := FramePCM(pcm, cfg)
	assert.Len(t, frames, 3)
	assert.Len(t, frames[2], 640) // padded to full frame

	// Verify padding is zeros
	for i := 100; i < 640; i++ {
		assert.Equal(t, byte(0), frames[2][i])
	}
}

func TestFramePCM_Empty(t *testing.T) {
	cfg := DefaultFrameConfig()
	frames := FramePCM(nil, cfg)
	assert.Nil(t, frames)

	frames = FramePCM([]byte{}, cfg)
	assert.Nil(t, frames)
}

func TestFramePCM_SingleByte(t *testing.T) {
	cfg := DefaultFrameConfig()
	pcm := []byte{0x42}
	frames := FramePCM(pcm, cfg)
	assert.Len(t, frames, 1)
	assert.Len(t, frames[0], 640)
	assert.Equal(t, byte(0x42), frames[0][0])
}

func TestPCMDuration(t *testing.T) {
	// 16000 samples * 2 bytes = 32000 bytes = 1 second
	assert.InDelta(t, 1.0, PCMDuration(32000, 16000, 1), 0.001)
	assert.InDelta(t, 8.75, PCMDuration(280000, 16000, 1), 0.001)
	assert.InDelta(t, 0.0, PCMDuration(0, 16000, 1), 0.001)
}
