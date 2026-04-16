package eventlog

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmitIncludesRunIDSeqAndContextSpan(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(Config{Dir: dir, MaxBytes: 1 << 20, MaxArchives: 2, ServerID: "HomeServer", ServerVersion: "1"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	SetDefault(logger)

	ctx, _ := WithSpan(context.Background(), "test:emit")
	ctx = WithAttrs(ctx, slog.String("activation_id", "act-1"))
	Emit(ctx, "scenario.activation.started", slog.LevelInfo, "scenario activation started", slog.String("scenario", "bootstrap"))
	if err := logger.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, defaultFileName))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	line := strings.TrimSpace(string(raw))
	if line == "" {
		t.Fatalf("expected one log line")
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got["event"] != "scenario.activation.started" {
		t.Fatalf("event = %v", got["event"])
	}
	if got["run_id"] == "" {
		t.Fatalf("run_id missing")
	}
	if got["seq"] == nil {
		t.Fatalf("seq missing")
	}
	if got["trace_id"] == "" {
		t.Fatalf("trace_id missing")
	}
	if got["span_id"] == "" {
		t.Fatalf("span_id missing")
	}
	if got["activation_id"] != "act-1" {
		t.Fatalf("activation_id = %v", got["activation_id"])
	}
	if got["server_id"] != "HomeServer" {
		t.Fatalf("server_id = %v", got["server_id"])
	}
}

func TestStdLogAdapterWritesLegacyEvent(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(Config{Dir: dir})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := logger.StdLogAdapter("legacy").Write([]byte("hello legacy\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := logger.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, defaultFileName))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(raw), "legacy.log") {
		t.Fatalf("expected legacy.log event, got %q", string(raw))
	}
}
