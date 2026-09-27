package server

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/WEIFENG2333/FreeASR/internal/asr"
	"github.com/WEIFENG2333/FreeASR/internal/audio"
	"github.com/WEIFENG2333/FreeASR/internal/format"
)

const maxUploadSize = 32 << 20 // 32 MB

func (s *Server) handleTranscribe(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "Failed to parse multipart form: "+err.Error(), "invalid_request_error", "invalid_form")
		return
	}

	// Extract audio file
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Missing 'file' field in multipart form", "invalid_request_error", "missing_file")
		return
	}
	defer file.Close()

	slog.Info("transcription request", "filename", header.Filename, "size", header.Size)

	// Read response_format (default: json)
	responseFormat := r.FormValue("response_format")
	if responseFormat == "" {
		responseFormat = "json"
	}

	// Validate format
	switch responseFormat {
	case "text", "json", "verbose_json", "srt", "vtt":
		// ok
	default:
		writeError(w, http.StatusBadRequest,
			fmt.Sprintf("Invalid response_format: %s. Supported: text, json, verbose_json, srt, vtt", responseFormat),
			"invalid_request_error", "invalid_format")
		return
	}

	// Read audio data
	audioData, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read audio file", "server_error", "read_error")
		return
	}

	// Convert to PCM using ffmpeg
	pcmData, err := audio.ConvertToMonoPCM16k(r.Context(), io.NopCloser(
		io.NewSectionReader(readerAt(audioData), 0, int64(len(audioData))),
	))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to convert audio: "+err.Error(), "invalid_request_error", "audio_conversion_error")
		return
	}

	if len(pcmData) == 0 {
		writeError(w, http.StatusBadRequest, "No audio data after conversion", "invalid_request_error", "empty_audio")
		return
	}

	duration := audio.PCMDuration(len(pcmData), 16000, 1)

	// Acquire semaphore for concurrency control
	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	default:
		writeError(w, http.StatusServiceUnavailable, "Server is at maximum concurrency", "server_error", "too_many_requests")
		return
	}

	// Create ASR client and transcribe
	cfg := asr.DefaultConfig()
	cfg.CredentialPath = s.cfg.ASRConfig.CredentialPath
	client := asr.NewClient(cfg, s.cfg.CredStore)

	text, err := client.Transcribe(r.Context(), pcmData)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Transcription failed: "+err.Error(), "server_error", "transcription_error")
		return
	}

	// Build result
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

	// Format and respond
	output, contentType, err := format.Format(result, responseFormat)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "server_error", "format_error")
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, output)
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, defaultModelList())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
