package transport

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestDecodeHarnessFrameCounterPayload(t *testing.T) {
	raw := make([]byte, 8, 11)
	binary.BigEndian.PutUint64(raw, 42)
	raw = append(raw, []byte("pcm")...)

	counter, payload, ok := decodeHarnessFrameCounterPayload(raw)
	if !ok {
		t.Fatalf("decodeHarnessFrameCounterPayload ok = false, want true")
	}
	if counter != 42 {
		t.Fatalf("counter = %d, want 42", counter)
	}
	if !bytes.Equal(payload, []byte("pcm")) {
		t.Fatalf("payload = %q, want %q", payload, []byte("pcm"))
	}
}

func TestDecodeHarnessFrameCounterPayloadRejectsShortFrame(t *testing.T) {
	counter, payload, ok := decodeHarnessFrameCounterPayload([]byte{1, 2, 3, 4})
	if ok {
		t.Fatalf("decodeHarnessFrameCounterPayload ok = true, want false")
	}
	if counter != 0 {
		t.Fatalf("counter = %d, want 0 when decode fails", counter)
	}
	if payload != nil {
		t.Fatalf("payload = %v, want nil when decode fails", payload)
	}
}

func TestHarnessFrameCounterRecorderRecordsCounterAndStripsPrefix(t *testing.T) {
	recorder := newHarnessFrameCounterRecorder()
	frame := make([]byte, 8, 19)
	binary.BigEndian.PutUint64(frame, 7)
	frame = append(frame, []byte("audio-frame")...)

	payload, ok := recorder.record(frame)
	if !ok {
		t.Fatalf("record ok = false, want true")
	}
	if !bytes.Equal(payload, []byte("audio-frame")) {
		t.Fatalf("payload = %q, want %q", payload, []byte("audio-frame"))
	}
	if got := recorder.counters(); len(got) != 1 || got[0] != 7 {
		t.Fatalf("counters = %v, want [7]", got)
	}
}
