package asr

import (
	"context"
	"github.com/WEIFENG2333/FreeASR/internal/audio"
	pb "github.com/WEIFENG2333/FreeASR/internal/pb"
	"github.com/WEIFENG2333/FreeASR/internal/protocol"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLiveSendsBeforeEOFAndFinalizes(t *testing.T) {
	received := make(chan *pb.AsrRequest, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrade := websocket.Upgrader{}
		conn, err := upgrade.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			msg := &pb.AsrRequest{}
			if proto.Unmarshal(data, msg) != nil {
				return
			}
			received <- msg
		}
	}))
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	defer conn.Close()
	cfg := DefaultConfig()
	s := NewSession(cfg, &protocol.Credentials{Token: "fixture"})
	s.ws = conn
	s.encoder, err = audio.NewOpusEncoder(16000, 1, 20)
	require.NoError(t, err)
	input, writer := io.Pipe()
	defer input.Close()
	defer writer.Close()
	done := make(chan error, 1)
	go func() { done <- s.sendLive(context.Background(), input) }()
	_, err = writer.Write(make([]byte, 640))
	require.NoError(t, err)
	select {
	case frame := <-received:
		require.Equal(t, pb.FrameState_FRAME_STATE_FIRST, frame.FrameState)
		require.NotEmpty(t, frame.AudioData)
	case <-time.After(time.Second):
		t.Fatal("buffered audio until EOF instead of streaming")
	}
	// A partial last frame must be padded, followed by FinishSession.
	_, err = writer.Write(make([]byte, 100))
	require.NoError(t, err)
	writer.Close()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("did not finish")
	}
	select {
	case frame := <-received:
		require.Equal(t, pb.FrameState_FRAME_STATE_LAST, frame.FrameState)
	case <-time.After(time.Second):
		t.Fatal("missing terminal frame")
	}
	select {
	case frame := <-received:
		require.Equal(t, "FinishSession", frame.MethodName)
	case <-time.After(time.Second):
		t.Fatal("missing finish")
	}
}
