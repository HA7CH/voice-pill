package audio

import (
	"encoding/binary"
	"fmt"
)

// NativeWAVPCM accepts only the PCM format recorded by Voice Pill. Other
// formats fall back to ffmpeg; malformed WAV files return an error.
func NativeWAVPCM(data []byte) ([]byte, bool, error) {
	if !IsWAVFile(data) {
		return nil, false, nil
	}
	limit := uint64(binary.LittleEndian.Uint32(data[4:8])) + 8
	if limit > uint64(len(data)) || limit < 12 {
		return nil, true, fmt.Errorf("truncated WAV")
	}
	var pcm []byte
	compatible := false
	for offset := uint64(12); offset < limit; {
		if offset+8 > limit {
			return nil, true, fmt.Errorf("truncated WAV chunk")
		}
		size := uint64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		start, end := offset+8, offset+8+size
		if end > limit {
			return nil, true, fmt.Errorf("truncated WAV payload")
		}
		switch string(data[offset : offset+4]) {
		case "fmt ":
			if size < 16 {
				return nil, true, fmt.Errorf("invalid WAV format")
			}
			f := data[start:end]
			compatible = binary.LittleEndian.Uint16(f[0:2]) == 1 && binary.LittleEndian.Uint16(f[2:4]) == 1 && binary.LittleEndian.Uint32(f[4:8]) == 16000 && binary.LittleEndian.Uint16(f[12:14]) == 2 && binary.LittleEndian.Uint16(f[14:16]) == 16
		case "data":
			pcm = data[start:end]
		}
		offset = end + size%2
	}
	if !compatible {
		return nil, false, nil
	}
	if pcm == nil || len(pcm)%2 != 0 {
		return nil, true, fmt.Errorf("invalid WAV PCM data")
	}
	return pcm, true, nil
}
