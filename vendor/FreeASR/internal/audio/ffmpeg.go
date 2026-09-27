package audio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// ConvertToMonoPCM16k uses ffmpeg to convert any audio format to 16kHz mono 16-bit PCM.
func ConvertToMonoPCM16k(ctx context.Context, input io.Reader) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", "pipe:0", // read from stdin
		"-f", "s16le", // 16-bit signed little-endian
		"-acodec", "pcm_s16le",
		"-ar", "16000", // 16kHz
		"-ac", "1", // mono
		"pipe:1", // write to stdout
	)
	cmd.Stdin = input

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if len(errMsg) > 200 {
			errMsg = errMsg[len(errMsg)-200:]
		}
		return nil, fmt.Errorf("ffmpeg conversion failed: %w (%s)", err, errMsg)
	}

	return stdout.Bytes(), nil
}

// ConvertFileToMonoPCM16k converts a file on disk to 16kHz mono PCM.
func ConvertFileToMonoPCM16k(ctx context.Context, path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if pcm, handled, err := NativeWAVPCM(data); handled || err != nil {
		return pcm, err
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", path,
		"-f", "s16le",
		"-acodec", "pcm_s16le",
		"-ar", "16000",
		"-ac", "1",
		"pipe:1",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if len(errMsg) > 200 {
			errMsg = errMsg[len(errMsg)-200:]
		}
		return nil, fmt.Errorf("ffmpeg conversion failed: %w (%s)", err, errMsg)
	}

	return stdout.Bytes(), nil
}

// IsFFmpegAvailable checks if ffmpeg is in PATH.
func IsFFmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

// IsWAVFile checks if data starts with a RIFF/WAVE header.
func IsWAVFile(data []byte) bool {
	if len(data) < 12 {
		return false
	}
	return string(data[:4]) == "RIFF" && string(data[8:12]) == "WAVE"
}

// StripWAVHeader returns PCM data after the WAV header.
// For 16kHz mono 16-bit WAV, the data chunk starts after the header.
// If it's not a valid WAV or has different format, returns the original data.
func StripWAVHeader(data []byte) []byte {
	if !IsWAVFile(data) {
		return data
	}
	// Find the "data" chunk
	for i := 12; i+8 < len(data); {
		chunkID := string(data[i : i+4])
		chunkSize := int(data[i+4]) | int(data[i+5])<<8 | int(data[i+6])<<16 | int(data[i+7])<<24
		if chunkID == "data" {
			start := i + 8
			end := start + chunkSize
			if end > len(data) {
				end = len(data)
			}
			return data[start:end]
		}
		i += 8 + chunkSize
		if chunkSize%2 != 0 {
			i++ // padding byte
		}
	}
	return data
}
