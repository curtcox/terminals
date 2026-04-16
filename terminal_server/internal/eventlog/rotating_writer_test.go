package eventlog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRotatingWriterRotatesAndCapsArchives(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "terminals.jsonl")
	w, err := NewRotatingWriter(path, 32, 2)
	if err != nil {
		t.Fatalf("NewRotatingWriter() error = %v", err)
	}
	defer func() { _ = w.Close() }()

	for i := 0; i < 8; i++ {
		if _, err := w.Write([]byte("0123456789\n")); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("active file missing: %v", err)
	}
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("archive .1 missing: %v", err)
	}
	if _, err := os.Stat(path + ".2"); err != nil {
		t.Fatalf("archive .2 missing: %v", err)
	}
	if _, err := os.Stat(path + ".3"); !os.IsNotExist(err) {
		t.Fatalf("archive .3 should not exist")
	}
}
