package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	credentialPath string
	logLevel       string
	verbose        bool
	quiet          bool
)

func defaultCredentialPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "credentials.json"
	}
	return filepath.Join(home, ".config", "freeasr", "credentials.json")
}

var rootCmd = &cobra.Command{
	Use:   "freeasr",
	Short: "FreeASR — free speech-to-text with OpenAI-compatible API",
	Long: `FreeASR is a CLI tool and HTTP server for speech recognition.

It provides:
  • CLI transcription of audio files (wav, mp3, m4a, flac, etc.)
  • OpenAI-compatible HTTP API server (POST /v1/audio/transcriptions)
  • Multiple output formats: text, json, verbose_json, srt, vtt

Examples:
  freeasr transcribe audio.wav
  freeasr transcribe audio.mp3 --format json
  freeasr serve --port 8080
  freeasr doctor`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&credentialPath, "credential-path", defaultCredentialPath(), "path to credential cache file")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (error|warn|info|debug)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress non-error output")
}
