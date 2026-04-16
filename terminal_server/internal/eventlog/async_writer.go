package eventlog

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type WriteFailure struct {
	At  time.Time
	Err error
}

// AsyncWriter provides non-blocking buffered writes around a sink.
// When the queue is full, it drops the oldest entry and enqueues the latest.
type AsyncWriter struct {
	sink     io.Writer
	queue    chan []byte
	wg       sync.WaitGroup
	closed   atomic.Bool
	dropped  atomic.Uint64
	reported atomic.Uint64
	inFlight atomic.Int64

	errMu           sync.Mutex
	lastErrReported time.Time
	now             func() time.Time
	errInterval     time.Duration
	stderr          io.Writer
	onWriteFailure  func(WriteFailure)
}

func NewAsyncWriter(sink io.Writer, capacity int) *AsyncWriter {
	if capacity <= 0 {
		capacity = 4096
	}
	w := &AsyncWriter{
		sink:        sink,
		queue:       make(chan []byte, capacity),
		now:         time.Now,
		errInterval: time.Minute,
		stderr:      os.Stderr,
	}
	w.wg.Add(1)
	go w.run()
	return w
}

func (w *AsyncWriter) SetWriteFailureCallback(callback func(WriteFailure)) {
	w.errMu.Lock()
	defer w.errMu.Unlock()
	w.onWriteFailure = callback
}

func (w *AsyncWriter) Write(p []byte) (int, error) {
	if w.closed.Load() {
		return 0, os.ErrClosed
	}
	copyBuf := make([]byte, len(p))
	copy(copyBuf, p)
	select {
	case w.queue <- copyBuf:
		w.inFlight.Add(1)
		return len(p), nil
	default:
	}
	select {
	case <-w.queue:
		w.dropped.Add(1)
	default:
	}
	select {
	case w.queue <- copyBuf:
		w.inFlight.Add(1)
		return len(p), nil
	default:
		w.dropped.Add(1)
		return len(p), nil
	}
}

func (w *AsyncWriter) run() {
	defer w.wg.Done()
	for payload := range w.queue {
		if len(payload) == 0 {
			continue
		}
		if _, err := w.sink.Write(payload); err != nil {
			w.reportWriteError(err)
		} else {
			w.markHealthy()
		}
		w.inFlight.Add(-1)
	}
}

func (w *AsyncWriter) reportWriteError(err error) {
	w.errMu.Lock()
	now := w.now().UTC()
	if !w.lastErrReported.IsZero() && now.Sub(w.lastErrReported) < w.errInterval {
		w.errMu.Unlock()
		return
	}
	w.lastErrReported = now
	stderr := w.stderr
	callback := w.onWriteFailure
	w.errMu.Unlock()
	_, _ = fmt.Fprintf(stderr, "eventlog sink write failed: %v\n", err)
	if callback != nil {
		callback(WriteFailure{At: now, Err: err})
	}
}

func (w *AsyncWriter) markHealthy() {
	w.errMu.Lock()
	w.lastErrReported = time.Time{}
	w.errMu.Unlock()
}

func (w *AsyncWriter) DroppedSinceLast() uint64 {
	total := w.dropped.Load()
	last := w.reported.Swap(total)
	if total <= last {
		return 0
	}
	return total - last
}

func (w *AsyncWriter) Flush() error {
	deadline := time.After(3 * time.Second)
	for {
		if len(w.queue) == 0 && w.inFlight.Load() == 0 {
			break
		}
		select {
		case <-deadline:
			return nil
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
	if syncer, ok := w.sink.(interface{ Sync() error }); ok {
		return syncer.Sync()
	}
	return nil
}

func (w *AsyncWriter) Close() error {
	if !w.closed.CompareAndSwap(false, true) {
		return nil
	}
	close(w.queue)
	w.wg.Wait()
	if closer, ok := w.sink.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
