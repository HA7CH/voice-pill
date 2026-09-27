package protocol

import (
	"encoding/json"
	"fmt"

	pb "github.com/WEIFENG2333/FreeASR/internal/pb"
	"google.golang.org/protobuf/proto"
)

// EventType classifies ASR response events.
type EventType int

const (
	EventTaskStarted EventType = iota
	EventSessionStarted
	EventSessionFinished
	EventVADStart
	EventInterim
	EventDefinite
	EventFinal
	EventHeartbeat
	EventError
	EventUnknown
)

// String returns the event type name.
func (e EventType) String() string {
	switch e {
	case EventTaskStarted:
		return "task.started"
	case EventSessionStarted:
		return "session.started"
	case EventSessionFinished:
		return "session.finished"
	case EventVADStart:
		return "vad.start"
	case EventInterim:
		return "transcript.delta"
	case EventDefinite:
		return "transcript.definite"
	case EventFinal:
		return "transcript.segment"
	case EventHeartbeat:
		return "heartbeat"
	case EventError:
		return "error"
	default:
		return "unknown"
	}
}

// Event represents a parsed ASR response event.
type Event struct {
	Type         EventType
	Text         string
	IsInterim    bool
	VADFinished  bool
	PacketNumber int
	ErrorMsg     string
	RawJSON      map[string]any
}

// ParseResponse deserializes a protobuf response into an Event.
func ParseResponse(data []byte) (*Event, error) {
	msg := &pb.AsrResponse{}
	if err := proto.Unmarshal(data, msg); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	switch msg.MessageType {
	case "TaskStarted":
		return &Event{Type: EventTaskStarted}, nil
	case "SessionStarted":
		return &Event{Type: EventSessionStarted}, nil
	case "SessionFinished":
		return &Event{Type: EventSessionFinished}, nil
	case "TaskFailed", "SessionFailed":
		return &Event{
			Type:     EventError,
			ErrorMsg: msg.StatusMessage,
		}, nil
	}

	// Parse result_json
	if msg.ResultJson == "" {
		return &Event{Type: EventUnknown}, nil
	}

	var jsonData map[string]any
	if err := json.Unmarshal([]byte(msg.ResultJson), &jsonData); err != nil {
		return &Event{Type: EventUnknown}, nil
	}

	results, _ := jsonData["results"].([]any)
	extra, _ := jsonData["extra"].(map[string]any)

	// Heartbeat: no results
	if results == nil {
		pn := -1
		if extra != nil {
			if v, ok := extra["packet_number"].(float64); ok {
				pn = int(v)
			}
		}
		return &Event{
			Type:         EventHeartbeat,
			PacketNumber: pn,
			RawJSON:      jsonData,
		}, nil
	}

	// VAD start
	if extra != nil {
		if vs, ok := extra["vad_start"].(bool); ok && vs {
			return &Event{Type: EventVADStart, RawJSON: jsonData}, nil
		}
	}

	// Parse recognition results
	var text string
	isInterim := true
	vadFinished := false
	nonstreamResult := false

	for _, r := range results {
		rm, ok := r.(map[string]any)
		if !ok {
			continue
		}
		if t, ok := rm["text"].(string); ok && t != "" {
			text = t
		}
		if v, ok := rm["is_interim"].(bool); ok && !v {
			isInterim = false
		}
		if v, ok := rm["is_vad_finished"].(bool); ok && v {
			vadFinished = true
		}
		if re, ok := rm["extra"].(map[string]any); ok {
			if v, ok := re["nonstream_result"].(bool); ok && v {
				nonstreamResult = true
			}
		}
	}

	// Final result
	if nonstreamResult || (!isInterim && vadFinished) {
		return &Event{
			Type:        EventFinal,
			Text:        text,
			VADFinished: vadFinished,
			RawJSON:     jsonData,
		}, nil
	}

	// Definite (second-pass confirmed, non-interim)
	if !isInterim {
		return &Event{
			Type:    EventDefinite,
			Text:    text,
			RawJSON: jsonData,
		}, nil
	}

	// Interim
	return &Event{
		Type:      EventInterim,
		Text:      text,
		IsInterim: true,
		RawJSON:   jsonData,
	}, nil
}
