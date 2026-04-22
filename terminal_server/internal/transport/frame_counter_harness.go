package transport

import "encoding/binary"

const harnessFrameCounterPrefixBytes = 8

// decodeHarnessFrameCounterPayload parses the test-harness-only frame prefix.
// The first 8 bytes are a big-endian monotonic counter, followed by payload.
func decodeHarnessFrameCounterPayload(frame []byte) (uint64, []byte, bool) {
	if len(frame) < harnessFrameCounterPrefixBytes {
		return 0, nil, false
	}
	counter := binary.BigEndian.Uint64(frame[:harnessFrameCounterPrefixBytes])
	payload := frame[harnessFrameCounterPrefixBytes:]
	return counter, payload, true
}

type harnessFrameCounterRecorder struct {
	countersSeen []uint64
}

func newHarnessFrameCounterRecorder() *harnessFrameCounterRecorder {
	return &harnessFrameCounterRecorder{}
}

func (r *harnessFrameCounterRecorder) record(frame []byte) ([]byte, bool) {
	counter, payload, ok := decodeHarnessFrameCounterPayload(frame)
	if !ok {
		return nil, false
	}
	r.countersSeen = append(r.countersSeen, counter)
	return payload, true
}

func (r *harnessFrameCounterRecorder) counters() []uint64 {
	out := make([]uint64, len(r.countersSeen))
	copy(out, r.countersSeen)
	return out
}
