package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/WEIFENG2333/FreeASR/internal/asr"
	"github.com/WEIFENG2333/FreeASR/internal/audio"
	"github.com/WEIFENG2333/FreeASR/internal/format"
	"github.com/WEIFENG2333/FreeASR/internal/protocol"
)

var (
	outputFormat string
	outputFile   string
	noPunct      bool
	noThreePass  bool
	realtime     bool
)

var transcribeCmd = &cobra.Command{
	Use:   "transcribe <file|->",
	Short: "Transcribe an audio file",
	Long: `Transcribe an audio file to text.

Supports wav, mp3, m4a, flac, ogg, and video files (via ffmpeg).
Use '-' to read from stdin.

Examples:
  freeasr transcribe recording.wav
  freeasr transcribe podcast.mp3 --format json
  freeasr transcribe video.mp4 --format srt -o subtitles.srt
  cat audio.wav | freeasr transcribe -
  ffmpeg -i video.mp4 -f wav - | freeasr transcribe - --format vtt`,
	Args: cobra.ExactArgs(1),
	RunE: runTranscribe,
}

func init() {
	transcribeCmd.Flags().StringVarP(&outputFormat, "format", "f", "", "output format: text|json|verbose_json|srt|vtt (default: auto)")
	transcribeCmd.Flags().StringVarP(&outputFile, "output", "o", "", "write output to file")
	transcribeCmd.Flags().BoolVar(&noPunct, "no-punctuation", false, "disable punctuation")
	transcribeCmd.Flags().BoolVar(&noThreePass, "disable-three-pass", false, "disable third-pass recognition")
	transcribeCmd.Flags().BoolVar(&realtime, "realtime", false, "send audio at real-time speed")

	rootCmd.AddCommand(transcribeCmd)
}

func runTranscribe(cmd *cobra.Command, args []string) error {
	setupLogging()

	inputPath := args[0]

	// Read audio data
	var pcmData []byte
	var err error

	if inputPath == "-" {
		// Read from stdin
		slog.Info("reading audio from stdin")
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		pcmData, err = convertToPCM(cmd.Context(), data)
		if err != nil {
			return err
		}
	} else {
		// Read from file
		slog.Info("reading audio file", "path", inputPath)
		pcmData, err = audio.ConvertFileToMonoPCM16k(cmd.Context(), inputPath)
		if err != nil {
			return fmt.Errorf("convert audio: %w", err)
		}
	}

	duration := audio.PCMDuration(len(pcmData), 16000, 1)
	slog.Info("audio loaded", "pcm_bytes", len(pcmData), "duration_s", fmt.Sprintf("%.2f", duration))

	if len(pcmData) == 0 {
		return fmt.Errorf("no audio data")
	}

	// Auto-detect output format
	if outputFormat == "" {
		outputFormat = "text"
	}

	// Create ASR client
	cfg := asr.DefaultConfig()
	cfg.CredentialPath = credentialPath
	cfg.EnablePunctuation = !noPunct
	cfg.EnableThreePass = !noThreePass
	cfg.Realtime = realtime

	credStore := protocol.NewCredentialStore(credentialPath)
	client := asr.NewClient(cfg, credStore)

	// Transcribe
	ctx, cancel := context.WithTimeout(cmd.Context(), 90*time.Second)
	defer cancel()
	text, err := client.Transcribe(ctx, pcmData)
	if err != nil {
		return fmt.Errorf("transcribe: %w", err)
	}

	// Format output
	result := &format.Result{
		Text:     text,
		Language: "zh",
		Duration: duration,
		Segments: []format.Segment{{
			ID:    0,
			Start: 0,
			End:   duration,
			Text:  text,
		}},
	}

	output, _, err := format.Format(result, outputFormat)
	if err != nil {
		return err
	}

	// Write output
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output+"\n"), 0o644); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}
		slog.Info("output written", "path", outputFile)
	} else {
		fmt.Println(output)
	}

	return nil
}

func convertToPCM(ctx context.Context, data []byte) ([]byte, error) {
	if pcm, handled, err := audio.NativeWAVPCM(data); handled || err != nil {
		return pcm, err
	}

	// Use ffmpeg for all other formats
	if !audio.IsFFmpegAvailable() {
		return nil, fmt.Errorf("ffmpeg not found; install ffmpeg for audio format conversion")
	}

	return audio.ConvertToMonoPCM16k(ctx, io.NopCloser(
		io.NewSectionReader(readerAt(data), 0, int64(len(data))),
	))
}

type readerAt []byte

func (r readerAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(r)) {
		return 0, io.EOF
	}
	n := copy(p, r[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func setupLogging() {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	if quiet {
		level = slog.LevelError
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}
