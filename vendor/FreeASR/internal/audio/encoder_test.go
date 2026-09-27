package audio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpusEncoder_Create(t *testing.T) {
	enc, err := NewOpusEncoder(16000, 1, 20)
	require.NoError(t, err)
	assert.NotNil(t, enc)
}

func TestOpusEncoder_EncodeSilence(t *testing.T) {
	enc, err := NewOpusEncoder(16000, 1, 20)
	require.NoError(t, err)

	// 20ms of silence at 16kHz mono = 640 bytes
	silence := make([]byte, 640)
	opus, err := enc.EncodeFrame(silence)
	require.NoError(t, err)
	assert.NotEmpty(t, opus)
	// Silence should compress well
	assert.Less(t, len(opus), 640)
}

func TestOpusEncoder_EncodeMultipleFrames(t *testing.T) {
	enc, err := NewOpusEncoder(16000, 1, 20)
	require.NoError(t, err)

	// 3 frames of silence
	pcm := make([]byte, 640*3)
	cfg := DefaultFrameConfig()
	opusFrames, err := enc.EncodePCM(pcm, cfg)
	require.NoError(t, err)
	assert.Len(t, opusFrames, 3)

	for _, frame := range opusFrames {
		assert.NotEmpty(t, frame)
	}
}
