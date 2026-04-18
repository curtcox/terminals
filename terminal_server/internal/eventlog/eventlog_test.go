package eventlog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEmitWritesHeaderAndContentLines(t *testing.T) {
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
	lines := splitJSONLines(string(raw))
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines (header + content), got %d: %q", len(lines), raw)
	}

	header := decodeLine(t, lines[0])
	if header["event"] != "log.run_started" {
		t.Fatalf("header event = %v", header["event"])
	}
	if header["run_id"] == "" || header["run_id"] == nil {
		t.Fatalf("header run_id missing")
	}
	if header["server_id"] != "HomeServer" {
		t.Fatalf("header server_id = %v", header["server_id"])
	}
	if header["server_version"] != "1" {
		t.Fatalf("header server_version = %v", header["server_version"])
	}
	if header["pid"] == nil {
		t.Fatalf("header pid missing")
	}

	got := decodeLine(t, lines[1])
	if got["event"] != "scenario.activation.started" {
		t.Fatalf("event = %v", got["event"])
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
	// Per-run invariants are emitted only in the header, not on every line.
	if _, ok := got["run_id"]; ok {
		t.Fatalf("content line should not carry run_id, got %v", got["run_id"])
	}
	if _, ok := got["server_id"]; ok {
		t.Fatalf("content line should not carry server_id, got %v", got["server_id"])
	}
	if _, ok := got["server_version"]; ok {
		t.Fatalf("content line should not carry server_version, got %v", got["server_version"])
	}
	if _, ok := got["pid"]; ok {
		t.Fatalf("content line should not carry pid, got %v", got["pid"])
	}
	// Component must appear exactly once.
	if got["component"] != "main" {
		t.Fatalf("component = %v, want main", got["component"])
	}
}

func TestComponentAttrIsNotDuplicated(t *testing.T) {
	dir := t.TempDir()
	logger, err := New(Config{Dir: dir})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Simulate the path that previously produced duplicate "component" keys:
	// base logger has component=main via the handler, Component("transport.grpc")
	// attaches a new component via .With, and the call-site also passes
	// "component" as a record-level attr. All three paths should collapse into
	// exactly one key in the output.
	scoped := logger.Component("transport.grpc")
	scoped.Info("listener ready", "event", "transport.grpc.listener_ready", "component", "transport.grpc.override")
	if err := logger.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, defaultFileName))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	lines := splitJSONLines(string(raw))
	if len(lines) < 2 {
		t.Fatalf("expected header + content, got %d lines: %q", len(lines), raw)
	}
	content := lines[len(lines)-1]
	if count := strings.Count(content, `"component":`); count != 1 {
		t.Fatalf("component key appeared %d times in %q", count, content)
	}
	got := decodeLine(t, content)
	if got["component"] != "transport.grpc.override" {
		t.Fatalf("component = %v, want record-level override", got["component"])
	}
}

func splitJSONLines(raw string) []string {
	out := []string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func decodeLine(t *testing.T, line string) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("Unmarshal(%q) error = %v", line, err)
	}
	return got
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

func TestWriteFailureEmitterWritesStructuredEvent(t *testing.T) {
	var stderr bytes.Buffer
	emitter := makeWriteFailureEmitter(writeFailureEmitterConfig{
		stderr:        &stderr,
		runID:         "run-123",
		pid:           42,
		serverID:      "HomeServer",
		serverVersion: "1",
	})
	emitter(WriteFailure{
		At:  time.Date(2026, 4, 16, 14, 3, 22, 184000000, time.UTC),
		Err: errors.New("disk full"),
	})

	line := strings.TrimSpace(stderr.String())
	if line == "" {
		t.Fatalf("expected structured failure event")
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got["event"] != "housekeeping.log.write_failed" {
		t.Fatalf("event = %v", got["event"])
	}
	if got["component"] != "housekeeping" {
		t.Fatalf("component = %v", got["component"])
	}
	if got["run_id"] != "run-123" {
		t.Fatalf("run_id = %v", got["run_id"])
	}
	if got["server_id"] != "HomeServer" {
		t.Fatalf("server_id = %v", got["server_id"])
	}
	errField, ok := got["error"].(map[string]any)
	if !ok {
		t.Fatalf("error field missing or invalid: %T", got["error"])
	}
	if errField["message"] != "disk full" {
		t.Fatalf("error.message = %v", errField["message"])
	}
}
