package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/WEIFENG2333/FreeASR/internal/asr"
	"github.com/WEIFENG2333/FreeASR/internal/protocol"
	"github.com/WEIFENG2333/FreeASR/internal/server"
)

var (
	serverHost       string
	serverPort       int
	serverAuthToken  string
	maxConcurrency   int
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start OpenAI-compatible HTTP server",
	Long: `Start an HTTP server that exposes an OpenAI-compatible audio transcription API.

Endpoints:
  POST /v1/audio/transcriptions  — transcribe audio file (multipart/form-data)
  GET  /v1/models               — list available models
  GET  /health                   — health check

Examples:
  freeasr serve
  freeasr serve --port 9090
  freeasr serve --port 8080 --auth-token my-secret

  # Then use with OpenAI SDK:
  curl -X POST http://localhost:8080/v1/audio/transcriptions \
    -F file=@audio.wav -F model=freeasr-1 -F response_format=json`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().StringVar(&serverHost, "host", "127.0.0.1", "listen host")
	serveCmd.Flags().IntVar(&serverPort, "port", 8080, "listen port")
	serveCmd.Flags().StringVar(&serverAuthToken, "auth-token", "", "optional Bearer auth token")
	serveCmd.Flags().IntVar(&maxConcurrency, "max-concurrency", 4, "max concurrent transcriptions")

	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	setupLogging()

	asrCfg := asr.DefaultConfig()
	asrCfg.CredentialPath = credentialPath

	credStore := protocol.NewCredentialStore(credentialPath)

	srv := server.New(server.Config{
		Host:           serverHost,
		Port:           serverPort,
		AuthToken:      serverAuthToken,
		MaxConcurrency: maxConcurrency,
		ASRConfig:      asrCfg,
		CredStore:      credStore,
	})

	addr := fmt.Sprintf("%s:%d", serverHost, serverPort)
	slog.Info("starting FreeASR server",
		"addr", addr,
		"transcriptions", fmt.Sprintf("http://%s/v1/audio/transcriptions", addr),
		"models", fmt.Sprintf("http://%s/v1/models", addr),
		"health", fmt.Sprintf("http://%s/health", addr),
	)

	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		return err
	}
	return nil
}
