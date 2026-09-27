package protocol

import (
	"encoding/json"
	"fmt"

	pb "github.com/WEIFENG2333/FreeASR/internal/pb"
	"google.golang.org/protobuf/proto"
)

// SessionConfig is the JSON payload for StartSession.
type SessionConfig struct {
	AudioInfo             AudioInfo             `json:"audio_info"`
	EnablePunctuation     bool                  `json:"enable_punctuation"`
	EnableSpeechRejection bool                  `json:"enable_speech_rejection"`
	Extra                 SessionExtraConfig    `json:"extra"`
}

// AudioInfo describes the audio format.
type AudioInfo struct {
	Channel    int    `json:"channel"`
	Format     string `json:"format"`
	SampleRate int    `json:"sample_rate"`
}

// SessionExtraConfig contains additional session parameters.
type SessionExtraConfig struct {
	AppName            string `json:"app_name"`
	CellCompressRate   int    `json:"cell_compress_rate"`
	DID                string `json:"did"`
	EnableASRThreePass bool   `json:"enable_asr_threepass"`
	EnableASRTwoPass   bool   `json:"enable_asr_twopass"`
	InputMode          string `json:"input_mode"`
}

// BuildStartTask creates a serialized StartTask protobuf message.
func BuildStartTask(requestID, token string) ([]byte, error) {
	msg := &pb.AsrRequest{
		Token:       token,
		ServiceName: "ASR",
		MethodName:  "StartTask",
		RequestId:   requestID,
	}
	return proto.Marshal(msg)
}

// BuildStartSession creates a serialized StartSession protobuf message.
func BuildStartSession(requestID, token string, cfg SessionConfig) ([]byte, error) {
	payload, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal session config: %w", err)
	}
	msg := &pb.AsrRequest{
		Token:       token,
		ServiceName: "ASR",
		MethodName:  "StartSession",
		RequestId:   requestID,
		Payload:     string(payload),
	}
	return proto.Marshal(msg)
}

// BuildFinishSession creates a serialized FinishSession protobuf message.
func BuildFinishSession(requestID, token string) ([]byte, error) {
	msg := &pb.AsrRequest{
		Token:       token,
		ServiceName: "ASR",
		MethodName:  "FinishSession",
		RequestId:   requestID,
	}
	return proto.Marshal(msg)
}

// BuildAudioFrame creates a serialized audio frame protobuf message.
func BuildAudioFrame(requestID string, opusData []byte, state pb.FrameState, timestampMs int64) ([]byte, error) {
	metadata, _ := json.Marshal(map[string]any{
		"extra":        map[string]any{},
		"timestamp_ms": timestampMs,
	})
	msg := &pb.AsrRequest{
		ServiceName: "ASR",
		MethodName:  "TaskRequest",
		Payload:     string(metadata),
		AudioData:   opusData,
		RequestId:   requestID,
		FrameState:  state,
	}
	return proto.Marshal(msg)
}

// DefaultSessionConfig returns a session configuration with sensible defaults.
func DefaultSessionConfig(deviceID string) SessionConfig {
	return SessionConfig{
		AudioInfo: AudioInfo{
			Channel:    1,
			Format:     "speech_opus",
			SampleRate: 16000,
		},
		EnablePunctuation:     true,
		EnableSpeechRejection: false,
		Extra: SessionExtraConfig{
			AppName:            "com.android.chrome",
			CellCompressRate:   8,
			DID:                deviceID,
			EnableASRThreePass: true,
			EnableASRTwoPass:   true,
			InputMode:          "tool",
		},
	}
}
