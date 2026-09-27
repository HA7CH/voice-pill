package protocol

import (
	"testing"

	pb "github.com/WEIFENG2333/FreeASR/internal/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestBuildStartTask(t *testing.T) {
	data, err := BuildStartTask("req-123", "token-abc")
	require.NoError(t, err)

	msg := &pb.AsrRequest{}
	require.NoError(t, proto.Unmarshal(data, msg))

	assert.Equal(t, "token-abc", msg.Token)
	assert.Equal(t, "ASR", msg.ServiceName)
	assert.Equal(t, "StartTask", msg.MethodName)
	assert.Equal(t, "req-123", msg.RequestId)
}

func TestBuildStartSession(t *testing.T) {
	cfg := DefaultSessionConfig("dev-123")
	data, err := BuildStartSession("req-123", "token-abc", cfg)
	require.NoError(t, err)

	msg := &pb.AsrRequest{}
	require.NoError(t, proto.Unmarshal(data, msg))

	assert.Equal(t, "StartSession", msg.MethodName)
	assert.Contains(t, msg.Payload, "speech_opus")
	assert.Contains(t, msg.Payload, "dev-123")
}

func TestBuildFinishSession(t *testing.T) {
	data, err := BuildFinishSession("req-123", "token-abc")
	require.NoError(t, err)

	msg := &pb.AsrRequest{}
	require.NoError(t, proto.Unmarshal(data, msg))

	assert.Equal(t, "FinishSession", msg.MethodName)
}

func TestBuildAudioFrame(t *testing.T) {
	opusData := []byte{0x01, 0x02, 0x03}
	data, err := BuildAudioFrame("req-123", opusData, pb.FrameState_FRAME_STATE_FIRST, 1000)
	require.NoError(t, err)

	msg := &pb.AsrRequest{}
	require.NoError(t, proto.Unmarshal(data, msg))

	assert.Equal(t, "TaskRequest", msg.MethodName)
	assert.Equal(t, opusData, msg.AudioData)
	assert.Equal(t, pb.FrameState_FRAME_STATE_FIRST, msg.FrameState)
	assert.Contains(t, msg.Payload, "timestamp_ms")
}

func TestDefaultSessionConfig(t *testing.T) {
	cfg := DefaultSessionConfig("test-device")
	assert.Equal(t, 1, cfg.AudioInfo.Channel)
	assert.Equal(t, "speech_opus", cfg.AudioInfo.Format)
	assert.Equal(t, 16000, cfg.AudioInfo.SampleRate)
	assert.True(t, cfg.EnablePunctuation)
	assert.False(t, cfg.EnableSpeechRejection)
	assert.Equal(t, "test-device", cfg.Extra.DID)
	assert.True(t, cfg.Extra.EnableASRThreePass)
	assert.True(t, cfg.Extra.EnableASRTwoPass)
}
