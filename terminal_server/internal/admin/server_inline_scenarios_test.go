package admin

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestInlineScenarioEndpointsCRUDAndValidation(t *testing.T) {
	h := testHandler(t)

	defineReq := httptest.NewRequest(http.MethodPost, "/admin/api/scenarios/inline/define", strings.NewReader(url.Values{
		"name":             {"red_alert"},
		"match_intent":     {"red alert", "all hands"},
		"match_event":      {"alarm.triggered"},
		"priority":         {"high"},
		"on_start":         {"ui broadcast all_screens banner"},
		"on_event_kind":    {"alarm.triggered"},
		"on_event_command": {"bus emit event alarm_ack"},
	}.Encode()))
	defineReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	defineW := httptest.NewRecorder()
	h.ServeHTTP(defineW, defineReq)
	if defineW.Code != http.StatusOK {
		t.Fatalf("inline scenario define status = %d, want 200 body=%s", defineW.Code, defineW.Body.String())
	}
	if !strings.Contains(defineW.Body.String(), `"name":"red_alert"`) {
		t.Fatalf("inline scenario define body missing scenario name: %s", defineW.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/api/scenarios/inline", nil)
	listW := httptest.NewRecorder()
	h.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("inline scenarios list status = %d, want 200 body=%s", listW.Code, listW.Body.String())
	}
	if !strings.Contains(listW.Body.String(), `"red_alert"`) {
		t.Fatalf("inline scenarios list body missing red_alert: %s", listW.Body.String())
	}

	showReq := httptest.NewRequest(http.MethodGet, "/admin/api/scenarios/inline?name=red_alert", nil)
	showW := httptest.NewRecorder()
	h.ServeHTTP(showW, showReq)
	if showW.Code != http.StatusOK {
		t.Fatalf("inline scenario show status = %d, want 200 body=%s", showW.Code, showW.Body.String())
	}
	if !strings.Contains(showW.Body.String(), `"priority":"high"`) {
		t.Fatalf("inline scenario show body missing priority: %s", showW.Body.String())
	}

	undefineReq := httptest.NewRequest(http.MethodPost, "/admin/api/scenarios/inline/undefine", strings.NewReader(url.Values{
		"name": {"red_alert"},
	}.Encode()))
	undefineReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	undefineW := httptest.NewRecorder()
	h.ServeHTTP(undefineW, undefineReq)
	if undefineW.Code != http.StatusOK {
		t.Fatalf("inline scenario undefine status = %d, want 200 body=%s", undefineW.Code, undefineW.Body.String())
	}
	if !strings.Contains(undefineW.Body.String(), `"deleted":true`) {
		t.Fatalf("inline scenario undefine body missing deleted=true: %s", undefineW.Body.String())
	}

	badReq := httptest.NewRequest(http.MethodPost, "/admin/api/scenarios/inline/define", strings.NewReader(url.Values{
		"name":             {"broken"},
		"on_event_kind":    {"alarm.triggered", "alarm.cleared"},
		"on_event_command": {"bus emit event alarm_ack"},
	}.Encode()))
	badReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	badW := httptest.NewRecorder()
	h.ServeHTTP(badW, badReq)
	if badW.Code != http.StatusBadRequest {
		t.Fatalf("inline scenario define mismatched on_event status = %d, want 400 body=%s", badW.Code, badW.Body.String())
	}
}
