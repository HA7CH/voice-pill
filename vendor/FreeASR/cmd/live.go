package cmd

import (
	"context"
	"encoding/json"
	"github.com/WEIFENG2333/FreeASR/internal/asr"
	"github.com/WEIFENG2333/FreeASR/internal/protocol"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	command := &cobra.Command{Use: "live", Short: "Stream PCM16LE mono 16kHz from stdin; emit JSON lines", RunE: func(cmd *cobra.Command, args []string) error {
		setupLogging()
		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()
		cfg := asr.DefaultConfig()
		cfg.EnablePunctuation = !noPunct
		client := asr.NewClient(cfg, protocol.NewCredentialStore(credentialPath))
		output := json.NewEncoder(os.Stdout)
		last := ""
		text, err := client.Live(ctx, os.Stdin, func(text string) {
			if text != last {
				output.Encode(map[string]string{"type": "partial", "text": text})
				last = text
			}
		})
		if err != nil {
			return err
		}
		return output.Encode(map[string]string{"type": "final", "text": text})
	}}
	command.Flags().BoolVar(&noPunct, "no-punctuation", false, "disable automatic punctuation")
	rootCmd.AddCommand(command)
}
