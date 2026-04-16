package query

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadAllOrdersArchivesThenActive(t *testing.T) {
	dir := t.TempDir()
	writeJSONL(t, filepath.Join(dir, "terminals.jsonl.2"), map[string]any{"seq": 1.0, "ts": "2026-04-16T10:00:00Z", "event": "a"})
	writeJSONL(t, filepath.Join(dir, "terminals.jsonl.1"), map[string]any{"seq": 2.0, "ts": "2026-04-16T10:00:01Z", "event": "b"})
	writeJSONL(t, filepath.Join(dir, "terminals.jsonl"), map[string]any{"seq": 3.0, "ts": "2026-04-16T10:00:02Z", "event": "c"})

	recs, err := ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(recs) != 3 {
		t.Fatalf("len = %d", len(recs))
	}
	if recs[0]["event"] != "a" || recs[1]["event"] != "b" || recs[2]["event"] != "c" {
		t.Fatalf("unexpected order: %+v", recs)
	}
}

func TestSearchFilters(t *testing.T) {
	dir := t.TempDir()
	writeJSONL(t, filepath.Join(dir, "terminals.jsonl"),
		map[string]any{"seq": 1.0, "ts": "2026-04-16T10:00:00Z", "event": "scenario.activation.started", "activation_id": "act-1", "level": "info"},
		map[string]any{"seq": 2.0, "ts": "2026-04-16T10:00:01Z", "event": "scenario.activation.failed", "activation_id": "act-1", "level": "error"},
		map[string]any{"seq": 3.0, "ts": "2026-04-16T10:00:02Z", "event": "device.registered", "activation_id": "", "level": "info"},
	)
	results, err := Search(dir, []string{"activation_id=act-1", "level>=warn"}, time.Date(2026, 4, 16, 10, 0, 5, 0, time.UTC))
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len = %d", len(results))
	}
	if results[0]["event"] != "scenario.activation.failed" {
		t.Fatalf("event = %v", results[0]["event"])
	}
}

func writeJSONL(t *testing.T, path string, rows ...map[string]any) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatalf("OpenFile(%s) error = %v", path, err)
	}
	defer func() { _ = f.Close() }()
	for _, row := range rows {
		b, _ := json.Marshal(row)
		if _, err := f.Write(append(b, '\n')); err != nil {
			t.Fatalf("write error = %v", err)
		}
	}
}
