package protocol

import (
	"testing"

	pb "github.com/WEIFENG2333/FreeASR/internal/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func buildResponseBytes(t *testing.T, msg *pb.AsrResponse) []byte {
	t.Helper()
	data, err := proto.Marshal(msg)
	require.NoError(t, err)
	return data
}

func TestParseResponse_TaskStarted(t *testing.T) {
	data := buildResponseBytes(t, &pb.AsrResponse{MessageType: "TaskStarted"})
	evt, err := ParseResponse(data)
	require.NoError(t, err)
	assert.Equal(t, EventTaskStarted, evt.Type)
}

func TestParseResponse_SessionStarted(t *testing.T) {
	data := buildResponseBytes(t, &pb.AsrResponse{MessageType: "SessionStarted"})
	evt, err := ParseResponse(data)
	require.NoError(t, err)
	assert.Equal(t, EventSessionStarted, evt.Type)
}

func TestParseResponse_SessionFinished(t *testing.T) {
	data := buildResponseBytes(t, &pb.AsrResponse{MessageType: "SessionFinished"})
	evt, err := ParseResponse(data)
	require.NoError(t, err)
	assert.Equal(t, EventSessionFinished, evt.Type)
}

func TestParseResponse_TaskFailed(t *testing.T) {
	data := buildResponseBytes(t, &pb.AsrResponse{
		MessageType:   "TaskFailed",
		StatusMessage: "InternalError",
	})
	evt, err := ParseResponse(data)
	require.NoError(t, err)
	assert.Equal(t, EventError, evt.Type)
	assert.Equal(t, "InternalError", evt.ErrorMsg)
}

func TestParseResponse_Heartbeat(t *testing.T) {
	data := buildResponseBytes(t, &pb.AsrResponse{
		ResultJson: `{"extra":{"packet_number":42}}`,
	})
	evt, err := ParseResponse(data)
	require.NoError(t, err)
	assert.Equal(t, EventHeartbeat, evt.Type)
	assert.Equal(t, 42, evt.PacketNumber)
}

func TestParseResponse_VADStart(t *testing.T) {
	data := buildResponseBytes(t, &pb.AsrResponse{
		ResultJson: `{"results":[],"extra":{"vad_start":true}}`,
	})
	evt, err := ParseResponse(data)
	require.NoError(t, err)
	assert.Equal(t, EventVADStart, evt.Type)
}

func TestParseResponse_InterimResult(t *testing.T) {
	data := buildResponseBytes(t, &pb.AsrResponse{
		ResultJson: `{"results":[{"text":"你好","is_interim":true}],"extra":{}}`,
	})
	evt, err := ParseResponse(data)
	require.NoError(t, err)
	assert.Equal(t, EventInterim, evt.Type)
	assert.Equal(t, "你好", evt.Text)
	assert.True(t, evt.IsInterim)
}

func TestParseResponse_DefiniteResult(t *testing.T) {
	data := buildResponseBytes(t, &pb.AsrResponse{
		ResultJson: `{"results":[{"text":"你好世界","is_interim":false}],"extra":{}}`,
	})
	evt, err := ParseResponse(data)
	require.NoError(t, err)
	assert.Equal(t, EventDefinite, evt.Type)
	assert.Equal(t, "你好世界", evt.Text)
}

func TestParseResponse_FinalResult(t *testing.T) {
	data := buildResponseBytes(t, &pb.AsrResponse{
		ResultJson: `{"results":[{"text":"你好世界。","is_interim":false,"is_vad_finished":true}],"extra":{}}`,
	})
	evt, err := ParseResponse(data)
	require.NoError(t, err)
	assert.Equal(t, EventFinal, evt.Type)
	assert.Equal(t, "你好世界。", evt.Text)
	assert.True(t, evt.VADFinished)
}

func TestParseResponse_NonstreamFinal(t *testing.T) {
	data := buildResponseBytes(t, &pb.AsrResponse{
		ResultJson: `{"results":[{"text":"最终结果","is_interim":false,"extra":{"nonstream_result":true}}],"extra":{}}`,
	})
	evt, err := ParseResponse(data)
	require.NoError(t, err)
	assert.Equal(t, EventFinal, evt.Type)
	assert.Equal(t, "最终结果", evt.Text)
}

func TestParseResponse_EmptyResultJSON(t *testing.T) {
	data := buildResponseBytes(t, &pb.AsrResponse{ResultJson: ""})
	evt, err := ParseResponse(data)
	require.NoError(t, err)
	assert.Equal(t, EventUnknown, evt.Type)
}
