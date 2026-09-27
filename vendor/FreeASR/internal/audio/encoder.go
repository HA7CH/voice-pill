package audio

import (
	"fmt"

	"gopkg.in/hraban/opus.v2"
)

// OpusEncoder wraps the libopus encoder for streaming ASR.
type OpusEncoder struct {
	enc       *opus.Encoder
	frameSize int // samples per frame
}

// NewOpusEncoder creates an Opus encoder with the given parameters.
func NewOpusEncoder(sampleRate, channels, frameDurationMs int) (*OpusEncoder, error) {
	enc, err := opus.NewEncoder(sampleRate, channels, opus.AppAudio)
	if err != nil {
		return nil, fmt.Errorf("create opus encoder: %w", err)
	}
	frameSize := sampleRate * frameDurationMs / 1000
	return &OpusEncoder{
		enc:       enc,
		frameSize: frameSize,
	}, nil
}

// EncodeFrame encodes a single PCM frame (16-bit LE) into Opus.
// The input must be exactly BytesPerFrame bytes.
func (e *OpusEncoder) EncodeFrame(pcm []byte) ([]byte, error) {
	pcm16 := bytesToInt16(pcm)
	output := make([]byte, 4000) // max opus frame
	n, err := e.enc.Encode(pcm16, output)
	if err != nil {
		return nil, fmt.Errorf("opus encode: %w", err)
	}
	return output[:n], nil
}

// EncodePCM splits PCM data into frames and encodes each to Opus.
func (e *OpusEncoder) EncodePCM(pcm []byte, cfg FrameConfig) ([][]byte, error) {
	frames := FramePCM(pcm, cfg)
	opusFrames := make([][]byte, 0, len(frames))
	for _, frame := range frames {
		encoded, err := e.EncodeFrame(frame)
		if err != nil {
			return nil, err
		}
		opusFrames = append(opusFrames, encoded)
	}
	return opusFrames, nil
}

func bytesToInt16(b []byte) []int16 {
	n := len(b) / 2
	result := make([]int16, n)
	for i := 0; i < n; i++ {
		result[i] = int16(b[2*i]) | int16(b[2*i+1])<<8
	}
	return result
}
