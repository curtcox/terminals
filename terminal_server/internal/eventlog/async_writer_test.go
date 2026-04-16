package eventlog

import (
	"bytes"
	"testing"
	"time"
)

func TestAsyncWriterDropsOldestWhenQueueFull(t *testing.T) {
	var sink bytes.Buffer
	w := NewAsyncWriter(&sink, 1)
	defer func() { _ = w.Close() }()

	// Rapid writes should force queue pressure; at least one drop is expected.
	for i := 0; i < 1000; i++ {
		_, _ = w.Write([]byte("line\n"))
	}
	_ = w.Flush()
	time.Sleep(10 * time.Millisecond)

	if dropped := w.DroppedSinceLast(); dropped == 0 {
		t.Fatalf("expected dropped > 0")
	}
	if got := w.DroppedSinceLast(); got != 0 {
		t.Fatalf("second dropped read = %d, want 0", got)
	}
}
