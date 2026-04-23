package eventlog

import (
	"bytes"
	"errors"
	"strings"
	"sync/atomic"
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

func TestAsyncWriterWriteFailureIsThrottledAndResetsAfterRecovery(t *testing.T) {
	var (
		stderr   bytes.Buffer
		failSink failingWriter
		callback atomic.Int64
	)
	failSink.fail.Store(true)

	w := NewAsyncWriter(&failSink, 8)
	w.SetStderr(&stderr)
	w.SetWriteFailureCallback(func(WriteFailure) {
		callback.Add(1)
	})

	if _, err := w.Write([]byte("first\n")); err != nil {
		t.Fatalf("Write(first) error = %v", err)
	}
	if _, err := w.Write([]byte("second\n")); err != nil {
		t.Fatalf("Write(second) error = %v", err)
	}
	// Flush waits for inFlight==0, which is decremented after reportWriteError
	// finishes writing to stderr — so reading stderr.String() after Flush is
	// race-free.
	_ = w.Flush()

	got := stderr.String()
	if strings.Count(got, "eventlog sink write failed: sink failed") != 1 {
		t.Fatalf("expected exactly one throttled stderr failure line, got %q", got)
	}
	if callback.Load() != 1 {
		t.Fatalf("callback count = %d, want 1", callback.Load())
	}

	failSink.fail.Store(false)
	if _, err := w.Write([]byte("healthy\n")); err != nil {
		t.Fatalf("Write(healthy) error = %v", err)
	}
	_ = w.Flush()

	failSink.fail.Store(true)
	if _, err := w.Write([]byte("failed-again\n")); err != nil {
		t.Fatalf("Write(failed-again) error = %v", err)
	}
	_ = w.Flush()

	got = stderr.String()
	if strings.Count(got, "eventlog sink write failed: sink failed") != 2 {
		t.Fatalf("expected second failure after recovery, got %q", got)
	}
	if callback.Load() != 2 {
		t.Fatalf("callback count = %d, want 2", callback.Load())
	}

	_ = w.Close()
}

type failingWriter struct {
	fail atomic.Bool
}

func (w *failingWriter) Write(p []byte) (int, error) {
	if !w.fail.Load() {
		return len(p), nil
	}
	return 0, errors.New("sink failed")
}
