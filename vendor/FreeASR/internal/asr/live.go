package asr

import (
	"context"
	"fmt"
	"github.com/WEIFENG2333/FreeASR/internal/audio"
	pb "github.com/WEIFENG2333/FreeASR/internal/pb"
	"github.com/WEIFENG2333/FreeASR/internal/protocol"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"time"
)

// Live consumes microphone PCM16LE, mono 16 kHz, until EOF.
func (c *Client) Live(ctx context.Context, input io.Reader, update func(string)) (string, error) {
	creds, err := c.credStore.Ensure(ctx)
	if err != nil {
		return "", err
	}
	s := NewSession(c.cfg, creds)
	s.encoder, err = audio.NewOpusEncoder(c.cfg.SampleRate, c.cfg.Channels, c.cfg.FrameDurationMs)
	if err != nil {
		return "", err
	}
	dialer := websocket.Dialer{HandshakeTimeout: c.cfg.ConnectTimeout}
	s.ws, _, err = dialer.DialContext(ctx, fmt.Sprintf("%s?aid=%d&device_id=%s", protocol.WebSocketURL, protocol.AID, creds.DeviceID), http.Header{"User-Agent": {protocol.UserAgent}, "Proto-Version": {"v2"}})
	if err != nil {
		return "", err
	}
	defer s.ws.Close()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() { <-ctx.Done(); s.ws.Close() }()
	s.ws.SetReadDeadline(time.Now().Add(15 * time.Second))
	if err = s.handshake(ctx); err != nil {
		return "", err
	}
	s.ws.SetReadDeadline(time.Time{})
	events := make(chan protocol.Event, 64)
	go func() { defer close(events); s.recvLoop(ctx, events) }()
	sendDone := make(chan error, 1)
	go func() {
		err := s.sendLive(ctx, input)
		sendDone <- err
		if err != nil {
			s.ws.Close()
		}
	}()
	agg := NewAggregator()
	finished := false
	var receiveErr error
	for evt := range events {
		switch evt.Type {
		case protocol.EventInterim:
			agg.UpdateInterim(evt.Text)
			update(agg.LivePreview())
		case protocol.EventDefinite:
			agg.UpdateDefinite(evt.Text)
			agg.UpdateInterim(evt.Text)
			update(agg.LivePreview())
		case protocol.EventFinal:
			agg.UpdateFinal(evt.Text)
			update(agg.LivePreview())
		case protocol.EventError:
			receiveErr = fmt.Errorf("ASR: %s", evt.ErrorMsg)
		case protocol.EventSessionFinished:
			finished = true
		}
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if receiveErr != nil {
		return "", receiveErr
	}
	if !finished {
		return "", fmt.Errorf("ASR disconnected before final result")
	}
	select {
	case err = <-sendDone:
		if err != nil {
			return "", err
		}
	case <-ctx.Done():
		return "", ctx.Err()
    default:
        // The service ended while microphone input is still open.
        // Report failure promptly so the app keeps recording for a full retry.
        return "", fmt.Errorf("ASR session ended before audio input finished")
	}
	return agg.BestText(), nil
}

func (s *Session) sendLive(ctx context.Context, input io.Reader) error {
	size := s.cfg.SampleRate * s.cfg.Channels * s.cfg.FrameDurationMs / 1000 * 2
	index := int64(0)
	stamp := time.Now().UnixMilli()
	for {
		frame := make([]byte, size)
		_, err := io.ReadFull(input, frame)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		state := pb.FrameState_FRAME_STATE_MIDDLE
		if index == 0 {
			state = pb.FrameState_FRAME_STATE_FIRST
		}
		// EOF gets a padded terminal frame, including exact frame boundaries.
		if err != nil && index > 0 {
			state = pb.FrameState_FRAME_STATE_LAST
		}
		encoded, e := s.encoder.EncodeFrame(frame)
		if e != nil {
			return e
		}
		msg, e := protocol.BuildAudioFrame(s.requestID, encoded, state, stamp+index*int64(s.cfg.FrameDurationMs))
		if e != nil {
			return e
		}
		s.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if e = s.writeMessage(msg); e != nil {
			return e
		}
		index++
		if err != nil {
			break
		}
	}
	finish, err := protocol.BuildFinishSession(s.requestID, s.creds.Token)
	if err != nil {
		return err
	}
	s.ws.SetReadDeadline(time.Now().Add(20 * time.Second))
	return s.writeMessage(finish)
}
