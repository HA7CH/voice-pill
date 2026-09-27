package server

import (
	"encoding/json"
	"net/http"
	"time"
)

// ErrorResponse follows OpenAI's error format.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error metadata.
type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// ModelList is the response for GET /v1/models.
type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// Model represents a single model entry.
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

func defaultModelList() ModelList {
	return ModelList{
		Object: "list",
		Data: []Model{
			{
				ID:      "freeasr-1",
				Object:  "model",
				Created: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
				OwnedBy: "freeasr",
			},
		},
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message, errType, code string) {
	writeJSON(w, status, ErrorResponse{
		Error: ErrorDetail{
			Message: message,
			Type:    errType,
			Code:    code,
		},
	})
}
