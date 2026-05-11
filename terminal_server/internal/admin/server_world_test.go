package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestWorldCalibrationEndpointReturnsGeometry(t *testing.T) {
	h := testHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/admin/api/world/calibration?device_id=d1", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	payload := map[string]any{}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode calibration payload: %v", err)
	}
	devices, ok := payload["devices"].([]any)
	if !ok {
		t.Fatalf("devices = %T, want []any", payload["devices"])
	}
	if len(devices) != 1 {
		t.Fatalf("len(devices) = %d, want 1", len(devices))
	}
}

func TestWorldVerifyEndpointUpdatesVerificationState(t *testing.T) {
	h := testHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/admin/api/world/verify", strings.NewReader(url.Values{
		"device_id": {"d1"},
		"method":    {"marker"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	payload := map[string]any{}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode verify payload: %v", err)
	}
	device, ok := payload["device"].(map[string]any)
	if !ok {
		t.Fatalf("device = %T, want object", payload["device"])
	}
	geometry, ok := device["geometry"].(map[string]any)
	if !ok {
		t.Fatalf("geometry = %T, want object", device["geometry"])
	}
	if geometry["VerificationState"] != "marker" {
		t.Fatalf("VerificationState = %v, want marker", geometry["VerificationState"])
	}
	history, ok := device["history"].([]any)
	if !ok {
		t.Fatalf("history = %T, want []any", device["history"])
	}
	if len(history) != 1 {
		t.Fatalf("len(history) = %d, want 1", len(history))
	}
}
