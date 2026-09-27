package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/WEIFENG2333/FreeASR/internal/protocol"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Register device and fetch ASR token",
	Long: `Manually trigger device registration and token acquisition.
Credentials are saved to the credential file for reuse.

Examples:
  freeasr auth
  freeasr auth --credential-path ./my-credentials.json`,
	RunE: runAuth,
}

func init() {
	rootCmd.AddCommand(authCmd)
}

func runAuth(cmd *cobra.Command, args []string) error {
	setupLogging()

	credStore := protocol.NewCredentialStore(credentialPath)
	creds, err := credStore.Ensure(context.Background())
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	data, _ := json.MarshalIndent(creds, "", "  ")
	fmt.Println(string(data))
	fmt.Printf("\nCredentials saved to: %s\n", credentialPath)
	return nil
}
