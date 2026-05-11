package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHandlersEndpointsCRUDAndValidation(t *testing.T) {
	h := testHandler(t)

	runReq := httptest.NewRequest(http.MethodPost, "/admin/api/handlers/on", strings.NewReader(url.Values{
		"selector": {"scenario=chat"},
		"action":   {"submit"},
		"run":      {"store put notes key value"},
	}.Encode()))
	runReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)
	if runW.Code != http.StatusOK {
		t.Fatalf("handlers on(run) status = %d, want 200 body=%s", runW.Code, runW.Body.String())
	}
	if !strings.Contains(runW.Body.String(), `"run_command":"store put notes key value"`) {
		t.Fatalf("handlers on(run) body missing run_command: %s", runW.Body.String())
	}
	var runBody map[string]any
	if err := json.Unmarshal(runW.Body.Bytes(), &runBody); err != nil {
		t.Fatalf("decode handlers on(run) response: %v", err)
	}
	handlerMap, _ := runBody["handler"].(map[string]any)
	handlerID, _ := handlerMap["id"].(string)
	if strings.TrimSpace(handlerID) == "" {
		t.Fatalf("handlers on(run) response missing handler id: %s", runW.Body.String())
	}

	emitReq := httptest.NewRequest(http.MethodPost, "/admin/api/handlers/on", strings.NewReader(url.Values{
		"selector":     {"scenario=chat"},
		"action":       {"submit"},
		"emit_kind":    {"intent"},
		"emit_name":    {"alert_ack"},
		"emit_payload": {"device=d1"},
	}.Encode()))
	emitReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	emitW := httptest.NewRecorder()
	h.ServeHTTP(emitW, emitReq)
	if emitW.Code != http.StatusOK {
		t.Fatalf("handlers on(emit) status = %d, want 200 body=%s", emitW.Code, emitW.Body.String())
	}
	if !strings.Contains(emitW.Body.String(), `"emit_name":"alert_ack"`) {
		t.Fatalf("handlers on(emit) body missing emit_name: %s", emitW.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/api/handlers", nil)
	listW := httptest.NewRecorder()
	h.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("handlers list status = %d, want 200 body=%s", listW.Code, listW.Body.String())
	}
	if !strings.Contains(listW.Body.String(), `"handlers":[`) {
		t.Fatalf("handlers list body missing handlers array: %s", listW.Body.String())
	}

	offReq := httptest.NewRequest(http.MethodPost, "/admin/api/handlers/off", strings.NewReader(url.Values{
		"handler_id": {handlerID},
	}.Encode()))
	offReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	offW := httptest.NewRecorder()
	h.ServeHTTP(offW, offReq)
	if offW.Code != http.StatusOK {
		t.Fatalf("handlers off status = %d, want 200 body=%s", offW.Code, offW.Body.String())
	}
	if !strings.Contains(offW.Body.String(), `"deleted":true`) {
		t.Fatalf("handlers off body missing deleted=true: %s", offW.Body.String())
	}

	badReq := httptest.NewRequest(http.MethodPost, "/admin/api/handlers/on", strings.NewReader(url.Values{
		"selector":  {"scenario=chat"},
		"action":    {"submit"},
		"run":       {"store put notes key value"},
		"emit_name": {"alert_ack"},
	}.Encode()))
	badReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	badW := httptest.NewRecorder()
	h.ServeHTTP(badW, badReq)
	if badW.Code != http.StatusBadRequest {
		t.Fatalf("handlers on mixed target status = %d, want 400 body=%s", badW.Code, badW.Body.String())
	}
}
