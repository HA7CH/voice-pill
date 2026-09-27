package audio

import (
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func wavFixture(rate uint32) []byte {
	b := make([]byte, 48)
	copy(b, "RIFF")
	binary.LittleEndian.PutUint32(b[4:], 40)
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1)
	binary.LittleEndian.PutUint16(b[22:], 1)
	binary.LittleEndian.PutUint32(b[24:], rate)
	binary.LittleEndian.PutUint32(b[28:], rate*2)
	binary.LittleEndian.PutUint16(b[32:], 2)
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], 4)
	copy(b[44:], []byte{1, 2, 3, 4})
	return b
}
func TestNativeWAVWithoutFFmpeg(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p := filepath.Join(t.TempDir(), "recording.wav")
	if err := os.WriteFile(p, wavFixture(16000), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := ConvertFileToMonoPCM16k(context.Background(), p)
	if err != nil || !bytes.Equal(got, []byte{1, 2, 3, 4}) {
		t.Fatalf("%v %v", got, err)
	}
}
func TestNativeWAVRejectsWrongRateAndTruncation(t *testing.T) {
	if _, ok, err := NativeWAVPCM(wavFixture(48000)); ok || err != nil {
		t.Fatal("wrong rate must fall back")
	}
	b := wavFixture(16000)
	if _, _, err := NativeWAVPCM(b[:45]); err == nil {
		t.Fatal("truncated WAV accepted")
	}
	binary.LittleEndian.PutUint32(b[40:], 0xffffffff)
	if _, _, err := NativeWAVPCM(b); err == nil {
		t.Fatal("invalid chunk accepted")
	}
}
