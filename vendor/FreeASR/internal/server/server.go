package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/WEIFENG2333/FreeASR/internal/asr"
	"github.com/WEIFENG2333/FreeASR/internal/protocol"
)

// Config holds server configuration.
type Config struct {
	Host           string
	Port           int
	AuthToken      string
	MaxConcurrency int
	ASRConfig      *asr.Config
	CredStore      *protocol.CredentialStore
}

// Server is the HTTP server for OpenAI-compatible API.
type Server struct {
	cfg    Config
	router chi.Router
	sem    chan struct{} // concurrency limiter
}

// New creates a new Server.
func New(cfg Config) *Server {
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = 4
	}

	s := &Server{
		cfg: cfg,
		sem: make(chan struct{}, cfg.MaxConcurrency),
	}

	r := chi.NewRouter()
	r.Use(corsMiddleware)
	r.Use(loggingMiddleware)
	if cfg.AuthToken != "" {
		r.Use(bearerAuthMiddleware(cfg.AuthToken))
	}

	r.Post("/v1/audio/transcriptions", s.handleTranscribe)
	r.Get("/v1/models", s.handleModels)
	r.Get("/health", s.handleHealth)

	s.router = r
	return s
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	return http.ListenAndServe(addr, s.router)
}

// ServeHTTP implements http.Handler for testing.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
