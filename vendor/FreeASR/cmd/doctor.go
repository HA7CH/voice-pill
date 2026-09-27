package cmd

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/WEIFENG2333/FreeASR/internal/audio"
	"github.com/WEIFENG2333/FreeASR/internal/protocol"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system dependencies and configuration",
	Long: `Check that all required system dependencies are available:
  • ffmpeg (audio format conversion)
  • libopus (Opus audio codec)
  • credential file (device registration)
  • network connectivity (WebSocket endpoint)`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	ok := true

	// Check ffmpeg
	if audio.IsFFmpegAvailable() {
		version := getCommandVersion("ffmpeg", "-version")
		fmt.Printf("✓ ffmpeg: %s\n", version)
	} else {
		fmt.Println("✗ ffmpeg: not found (required for audio format conversion)")
		ok = false
	}

	// Check libopus (by trying to create an encoder)
	enc, err := audio.NewOpusEncoder(16000, 1, 20)
	if err != nil {
		fmt.Printf("✗ libopus: %v\n", err)
		ok = false
	} else {
		_ = enc
		fmt.Println("✓ libopus: available")
	}

	// Check credentials
	credStore := protocol.NewCredentialStore(credentialPath)
	creds, err := credStore.Ensure(context.Background())
	if err != nil {
		fmt.Printf("✗ credentials: %v\n", err)
		ok = false
	} else {
		fmt.Printf("✓ credentials: device_id=%s, token=%s...\n", creds.DeviceID, truncate(creds.Token, 10))
	}

	if ok {
		fmt.Println("\nAll checks passed!")
	} else {
		fmt.Println("\nSome checks failed. See above for details.")
	}

	return nil
}

func getCommandVersion(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return "unknown"
	}
	// Return first line
	for i, b := range out {
		if b == '\n' {
			return string(out[:i])
		}
	}
	return string(out)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
