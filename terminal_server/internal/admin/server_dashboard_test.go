package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDashboardRenders(t *testing.T) {
	h := testHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Terminals Admin Dashboard") {
		t.Fatalf("dashboard body missing title")
	}
}

func TestStatusEndpointIncludesServerRuntimeAndConfig(t *testing.T) {
	h := testHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/admin/api/status", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if _, ok := payload["server"]; !ok {
		t.Fatalf("status payload missing server")
	}
	if _, ok := payload["runtime"]; !ok {
		t.Fatalf("status payload missing runtime")
	}
	if _, ok := payload["config"]; !ok {
		t.Fatalf("status payload missing config")
	}
}

func TestActivationsEndpointIncludesInspectionData(t *testing.T) {
	h := testHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/admin/api/activations", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode activations: %v", err)
	}
	if _, ok := payload["active_by_device"]; !ok {
		t.Fatalf("activations payload missing active_by_device")
	}
	if _, ok := payload["suspended_by_device"]; !ok {
		t.Fatalf("activations payload missing suspended_by_device")
	}
	if _, ok := payload["event_tail"]; !ok {
		t.Fatalf("activations payload missing event_tail")
	}
}
