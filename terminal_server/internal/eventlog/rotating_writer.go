package eventlog

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// RotatingWriter appends to one file and rotates size-based archives.
type RotatingWriter struct {
	mu          sync.Mutex
	activePath  string
	maxBytes    int64
	maxArchives int
	file        *os.File
	size        int64
}

// NewRotatingWriter creates a size-based rotating file writer.
func NewRotatingWriter(activePath string, maxBytes int64, maxArchives int) (*RotatingWriter, error) {
	activePath = strings.TrimSpace(activePath)
	if activePath == "" {
		return nil, fmt.Errorf("active path is required")
	}
	if maxBytes <= 0 {
		maxBytes = 100 * 1024 * 1024
	}
	if maxArchives < 0 {
		maxArchives = 0
	}
	if err := os.MkdirAll(filepath.Dir(activePath), 0o755); err != nil {
		return nil, fmt.Errorf("create log parent dir: %w", err)
	}
	f, err := os.OpenFile(activePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open active log file: %w", err)
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("stat active log file: %w", err)
	}
	return &RotatingWriter{
		activePath:  activePath,
		maxBytes:    maxBytes,
		maxArchives: maxArchives,
		file:        f,
		size:        st.Size(),
	}, nil
}

func (w *RotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return 0, os.ErrClosed
	}
	if w.maxBytes > 0 && w.size > 0 && w.size+int64(len(p)) > w.maxBytes {
		if err := w.rotateLocked(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

// Sync flushes the active file descriptor to disk.
func (w *RotatingWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	return w.file.Sync()
}

// Close closes the active file descriptor.
func (w *RotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *RotatingWriter) rotateLocked() error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("close active log file: %w", err)
		}
		w.file = nil
	}

	if w.maxArchives > 0 {
		oldest := w.activePath + "." + strconv.Itoa(w.maxArchives)
		_ = os.Remove(oldest)
		for i := w.maxArchives - 1; i >= 1; i-- {
			from := w.activePath + "." + strconv.Itoa(i)
			to := w.activePath + "." + strconv.Itoa(i+1)
			if _, err := os.Stat(from); err == nil {
				_ = os.Rename(from, to)
			}
		}
		if _, err := os.Stat(w.activePath); err == nil {
			if err := os.Rename(w.activePath, w.activePath+".1"); err != nil {
				return fmt.Errorf("rotate active log file: %w", err)
			}
		}
	} else {
		_ = os.Remove(w.activePath)
	}

	f, err := os.OpenFile(w.activePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("reopen active log file: %w", err)
	}
	w.file = f
	w.size = 0
	return nil
}
