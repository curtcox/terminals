package admin

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/config"
)

func TestLogsJSONLEndpointReturnsMatchingEvents(t *testing.T) {
	logDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(logDir, "terminals.jsonl"), []byte(
		`{"ts":"2026-04-16T10:00:00Z","seq":1,"event":"scenario.activation.started","activation_id":"act-1"}`+"\n"+
			`{"ts":"2026-04-16T10:00:01Z","seq":2,"event":"device.registered","activation_id":""}`+"\n",
	), 0o644); err != nil {
		t.Fatalf("WriteFile(logs) error = %v", err)
	}
	h := testHandler(t, config.Config{MDNSName: "HomeServer", LogDir: logDir})
	req := httptest.NewRequest(http.MethodGet, "/admin/logs.jsonl?event=scenario.activation.started", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "scenario.activation.started") {
		t.Fatalf("body = %q", body)
	}
	if strings.Contains(body, "device.registered") {
		t.Fatalf("body should not include unfiltered event: %q", body)
	}
}

func TestLogsTraceEndpointRendersHTML(t *testing.T) {
	logDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(logDir, "terminals.jsonl"), []byte(
		`{"ts":"2026-04-16T10:00:00Z","seq":1,"trace_id":"t1","span_id":"s1","event":"root"}`+"\n"+
			`{"ts":"2026-04-16T10:00:01Z","seq":2,"trace_id":"t1","span_id":"s2","parent_span_id":"s1","event":"child"}`+"\n",
	), 0o644); err != nil {
		t.Fatalf("WriteFile(logs) error = %v", err)
	}
	h := testHandler(t, config.Config{MDNSName: "HomeServer", LogDir: logDir})
	req := httptest.NewRequest(http.MethodGet, "/admin/logs/trace/t1", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("content type = %q, want html", w.Header().Get("Content-Type"))
	}
	body := w.Body.String()
	if !strings.Contains(body, "Trace Timeline") || !strings.Contains(body, "child") {
		t.Fatalf("body = %q", body)
	}
}

func TestLogsActivationEndpointRendersHTML(t *testing.T) {
	logDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(logDir, "terminals.jsonl"), []byte(
		`{"ts":"2026-04-16T10:00:00Z","seq":1,"activation_id":"act-1","event":"scenario.activation.started"}`+"\n",
	), 0o644); err != nil {
		t.Fatalf("WriteFile(logs) error = %v", err)
	}
	h := testHandler(t, config.Config{MDNSName: "HomeServer", LogDir: logDir})
	req := httptest.NewRequest(http.MethodGet, "/admin/logs/activation/act-1", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Activation Timeline") || !strings.Contains(body, "scenario.activation.started") {
		t.Fatalf("body = %q", body)
	}
}
