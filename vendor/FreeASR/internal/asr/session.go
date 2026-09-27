package asr

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/WEIFENG2333/FreeASR/internal/audio"
	pb "github.com/WEIFENG2333/FreeASR/internal/pb"
	"github.com/WEIFENG2333/FreeASR/internal/protocol"
)

// Session manages a single WebSocket ASR session lifecycle.
type Session struct {
	cfg       *Config
	creds     *protocol.Credentials
	requestID string
	encoder   *audio.OpusEncoder

	ws   *websocket.Conn
	wsMu sync.Mutex // serializes WebSocket writes
}

// NewSession creates a new ASR session.
func NewSession(cfg *Config, creds *protocol.Credentials) *Session {
	return &Session{
		cfg:       cfg,
		creds:     creds,
		requestID: uuid.New().String(),
	}
}

// Run connects to the ASR service, performs handshake, then streams audio
// from pcmData while receiving events. Returns a channel of events.
func (s *Session) Run(ctx context.Context, pcmData []byte) (<-chan protocol.Event, error) {
	// Create Opus encoder
	enc, err := audio.NewOpusEncoder(s.cfg.SampleRate, s.cfg.Channels, s.cfg.FrameDurationMs)
	if err != nil {
		return nil, fmt.Errorf("create opus encoder: %w", err)
	}
	s.encoder = enc

	// Connect WebSocket
	wsURL := fmt.Sprintf("%s?aid=%d&device_id=%s", protocol.WebSocketURL, protocol.AID, s.creds.DeviceID)
	slog.Info("connecting to ASR", "url", protocol.WebSocketURL, "device_id", s.creds.DeviceID)

	header := http.Header{}
	header.Set("User-Agent", protocol.UserAgent)
	header.Set("proto-version", "v2")
	header.Set("x-custom-keepalive", "true")

	dialer := websocket.Dialer{
		HandshakeTimeout: s.cfg.ConnectTimeout,
	}
	conn, _, err := dialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return nil, fmt.Errorf("websocket connect: %w", err)
	}
	s.ws = conn

	// Handshake: StartTask + StartSession
	if err := s.handshake(ctx); err != nil {
		s.ws.Close()
		return nil, fmt.Errorf("handshake: %w", err)
	}

	slog.Info("ASR session ready", "request_id", s.requestID)

	// Launch concurrent send/recv loops
	events := make(chan protocol.Event, 64)

	go func() {
		defer close(events)

		done := make(chan struct{})

		// Receive loop
		go func() {
			defer func() { close(done) }()
			s.recvLoop(ctx, events)
		}()

		// Send loop
		s.sendLoop(ctx, pcmData)

		// Wait for receive loop to finish
		select {
		case <-done:
		case <-ctx.Done():
		}

		s.ws.Close()
	}()

	return events, nil
}

func (s *Session) handshake(ctx context.Context) error {
	token := s.creds.Token

	// StartTask
	startTask, err := protocol.BuildStartTask(s.requestID, token)
	if err != nil {
		return fmt.Errorf("build StartTask: %w", err)
	}
	if err := s.writeMessage(startTask); err != nil {
		return fmt.Errorf("send StartTask: %w", err)
	}

	// Wait for TaskStarted
	evt, err := s.readOneEvent()
	if err != nil {
		return fmt.Errorf("read TaskStarted: %w", err)
	}
	if evt.Type == protocol.EventError {
		return fmt.Errorf("StartTask failed: %s", evt.ErrorMsg)
	}
	if evt.Type != protocol.EventTaskStarted {
		return fmt.Errorf("expected TaskStarted, got %s", evt.Type)
	}
	slog.Debug("TaskStarted received")

	// StartSession
	sessionCfg := s.buildSessionConfig()
	startSession, err := protocol.BuildStartSession(s.requestID, token, sessionCfg)
	if err != nil {
		return fmt.Errorf("build StartSession: %w", err)
	}
	if err := s.writeMessage(startSession); err != nil {
		return fmt.Errorf("send StartSession: %w", err)
	}

	// Wait for SessionStarted
	evt, err = s.readOneEvent()
	if err != nil {
		return fmt.Errorf("read SessionStarted: %w", err)
	}
	if evt.Type == protocol.EventError {
		return fmt.Errorf("StartSession failed: %s", evt.ErrorMsg)
	}
	if evt.Type != protocol.EventSessionStarted {
		return fmt.Errorf("expected SessionStarted, got %s", evt.Type)
	}
	slog.Debug("SessionStarted received")

	return nil
}

func (s *Session) sendLoop(ctx context.Context, pcmData []byte) {
	frameCfg := audio.FrameConfig{
		SampleRate:      s.cfg.SampleRate,
		Channels:        s.cfg.Channels,
		FrameDurationMs: s.cfg.FrameDurationMs,
	}

	frames := audio.FramePCM(pcmData, frameCfg)
	timestampMs := time.Now().UnixMilli()
	frameInterval := time.Duration(s.cfg.FrameDurationMs) * time.Millisecond

	for i, frame := range frames {
		if ctx.Err() != nil {
			return
		}

		// Encode to Opus
		opusData, err := s.encoder.EncodeFrame(frame)
		if err != nil {
			slog.Error("opus encode failed", "error", err)
			return
		}

		// Determine frame state
		var state pb.FrameState
		switch {
		case i == 0:
			state = pb.FrameState_FRAME_STATE_FIRST
		case i == len(frames)-1:
			state = pb.FrameState_FRAME_STATE_LAST
		default:
			state = pb.FrameState_FRAME_STATE_MIDDLE
		}

		// Build and send audio frame
		msg, err := protocol.BuildAudioFrame(s.requestID, opusData, state, timestampMs+int64(i)*int64(s.cfg.FrameDurationMs))
		if err != nil {
			slog.Error("build audio frame failed", "error", err)
			return
		}
		if err := s.writeMessage(msg); err != nil {
			slog.Error("send audio frame failed", "error", err)
			return
		}

		// Pace sending in realtime mode
		if s.cfg.Realtime {
			select {
			case <-time.After(frameInterval):
			case <-ctx.Done():
				return
			}
		}
	}

	// Send FinishSession
	finish, err := protocol.BuildFinishSession(s.requestID, s.creds.Token)
	if err != nil {
		slog.Error("build FinishSession failed", "error", err)
		return
	}
	if err := s.writeMessage(finish); err != nil {
		slog.Error("send FinishSession failed", "error", err)
		return
	}
	slog.Debug("FinishSession sent")
}

func (s *Session) recvLoop(ctx context.Context, events chan<- protocol.Event) {
	for {
		if ctx.Err() != nil {
			return
		}

		_, data, err := s.ws.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}
			if ctx.Err() != nil {
				return
			}
			slog.Error("websocket read error", "error", err)
			events <- protocol.Event{
				Type:     protocol.EventError,
				ErrorMsg: fmt.Sprintf("websocket read: %v", err),
			}
			return
		}

		evt, err := protocol.ParseResponse(data)
		if err != nil {
			slog.Warn("parse response failed", "error", err)
			continue
		}

		// Skip heartbeats from the event channel
		if evt.Type == protocol.EventHeartbeat {
			continue
		}

		events <- *evt

		// Stop on terminal events
		if evt.Type == protocol.EventSessionFinished || evt.Type == protocol.EventError {
			return
		}
	}
}

func (s *Session) writeMessage(data []byte) error {
	s.wsMu.Lock()
	defer s.wsMu.Unlock()
	return s.ws.WriteMessage(websocket.BinaryMessage, data)
}

func (s *Session) readOneEvent() (*protocol.Event, error) {
	_, data, err := s.ws.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("read message: %w", err)
	}
	return protocol.ParseResponse(data)
}

func (s *Session) buildSessionConfig() protocol.SessionConfig {
	cfg := protocol.DefaultSessionConfig(s.creds.DeviceID)
	cfg.EnablePunctuation = s.cfg.EnablePunctuation
	cfg.EnableSpeechRejection = false
	cfg.Extra.EnableASRThreePass = s.cfg.EnableThreePass
	cfg.Extra.EnableASRTwoPass = s.cfg.EnableTwoPass
	return cfg
}
