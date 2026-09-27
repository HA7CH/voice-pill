package asr

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/WEIFENG2333/FreeASR/internal/protocol"
)

// Config holds all ASR client configuration.
type Config struct {
	// Credential management
	CredentialPath string

	// Audio parameters
	SampleRate      int
	Channels        int
	FrameDurationMs int

	// Session parameters
	EnablePunctuation bool
	EnableTwoPass     bool
	EnableThreePass   bool

	// Behavior
	Realtime bool // pace audio sending at real-time speed

	// Timeouts
	ConnectTimeout time.Duration
	RecvTimeout    time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		SampleRate:        16000,
		Channels:          1,
		FrameDurationMs:   20,
		EnablePunctuation: true,
		EnableTwoPass:     true,
		EnableThreePass:   true,
		ConnectTimeout:    10 * time.Second,
		RecvTimeout:       10 * time.Second,
	}
}

// Client is the high-level ASR client.
type Client struct {
	cfg        *Config
	credStore  *protocol.CredentialStore
}

// NewClient creates a new ASR client.
func NewClient(cfg *Config, credStore *protocol.CredentialStore) *Client {
	return &Client{
		cfg:       cfg,
		credStore: credStore,
	}
}

// Transcribe performs a full transcription of PCM audio data.
func (c *Client) Transcribe(ctx context.Context, pcmData []byte) (string, error) {
	agg := NewAggregator()

	events, err := c.TranscribeStream(ctx, pcmData)
	if err != nil {
		return "", err
	}

	for evt := range events {
		switch evt.Type {
		case protocol.EventInterim:
			agg.UpdateInterim(evt.Text)
		case protocol.EventDefinite:
			agg.UpdateDefinite(evt.Text)
		case protocol.EventFinal:
			agg.UpdateFinal(evt.Text)
		case protocol.EventError:
			return "", fmt.Errorf("ASR error: %s", evt.ErrorMsg)
		}
	}

	if err := ctx.Err(); err != nil {
		return "", err
	}
	return agg.BestText(), nil
}

// TranscribeStream starts a streaming transcription and returns an event channel.
func (c *Client) TranscribeStream(ctx context.Context, pcmData []byte) (<-chan protocol.Event, error) {
	// Ensure credentials
	creds, err := c.credStore.Ensure(ctx)
	if err != nil {
		return nil, fmt.Errorf("ensure credentials: %w", err)
	}

	slog.Info("starting ASR session",
		"pcm_bytes", len(pcmData),
		"duration_s", fmt.Sprintf("%.2f", float64(len(pcmData))/float64(c.cfg.SampleRate*2*c.cfg.Channels)),
	)

	session := NewSession(c.cfg, creds)
	return session.Run(ctx, pcmData)
}
