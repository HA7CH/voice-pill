package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Credentials holds device registration data and ASR token.
type Credentials struct {
	DeviceID      string `json:"device_id"`
	InstallID     string `json:"install_id,omitempty"`
	CDID          string `json:"cdid,omitempty"`
	OpenUDID      string `json:"openudid,omitempty"`
	ClientUDID    string `json:"clientudid,omitempty"`
	Token         string `json:"token"`
	TokenUpdateMs int64  `json:"token_updated_at_ms,omitempty"`
}

// NeedsRefresh returns true if the token is missing or older than 12 hours.
func (c *Credentials) NeedsRefresh() bool {
	if c.Token == "" || c.TokenUpdateMs == 0 {
		return true
	}
	return time.Now().UnixMilli()-c.TokenUpdateMs >= TokenRefreshInterval
}

// CredentialStore manages credential persistence with 12h token TTL.
type CredentialStore struct {
	path   string
	mu     sync.Mutex
	creds  *Credentials
	client *http.Client
}

// NewCredentialStore creates a store backed by the given file path.
func NewCredentialStore(path string) *CredentialStore {
	return &CredentialStore{
		path:   path,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Ensure returns valid credentials, registering a device and fetching
// a token if necessary. Thread-safe.
func (cs *CredentialStore) Ensure(ctx context.Context) (*Credentials, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Try loading from memory first
	if cs.creds != nil && !cs.creds.NeedsRefresh() {
		return cs.creds, nil
	}

	// Try loading from file
	if cs.creds == nil {
		loaded, err := cs.loadFromFile()
		if err == nil && loaded != nil {
			cs.creds = loaded
			slog.Info("loaded credentials from file", "device_id", loaded.DeviceID)
		}
	}

	// Register device if we don't have a device_id
	if cs.creds == nil || cs.creds.DeviceID == "" {
		slog.Info("registering new device")
		creds, err := RegisterDevice(ctx, cs.client)
		if err != nil {
			return nil, fmt.Errorf("register device: %w", err)
		}
		cs.creds = creds
		slog.Info("device registered", "device_id", creds.DeviceID)
	}

	// Refresh token if needed
	if cs.creds.NeedsRefresh() {
		slog.Info("refreshing ASR token", "device_id", cs.creds.DeviceID)
		token, err := FetchASRToken(ctx, cs.client, cs.creds.DeviceID, cs.creds.CDID)
		if err != nil {
			// Fall back to cached token if available
			if cs.creds.Token != "" {
				slog.Warn("token refresh failed, using cached token", "error", err)
				return cs.creds, nil
			}
			return nil, fmt.Errorf("fetch ASR token: %w", err)
		}
		cs.creds.Token = token
		cs.creds.TokenUpdateMs = time.Now().UnixMilli()
		slog.Info("ASR token acquired")
	}

	// Save to file
	if cs.path != "" {
		if err := cs.saveToFile(); err != nil {
			slog.Warn("failed to save credentials", "error", err)
		}
	}

	return cs.creds, nil
}

func (cs *CredentialStore) loadFromFile() (*Credentials, error) {
	if cs.path == "" {
		return nil, fmt.Errorf("no credential path configured")
	}
	data, err := os.ReadFile(cs.path)
	if err != nil {
		return nil, err
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

func (cs *CredentialStore) saveToFile() error {
	if cs.path == "" || cs.creds == nil {
		return nil
	}
	dir := filepath.Dir(cs.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create credential dir: %w", err)
	}
	data, err := json.MarshalIndent(cs.creds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	return os.WriteFile(cs.path, data, 0o600)
}
