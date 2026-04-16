package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/config"
)

func TestBugIntakeAndListAndDetail(t *testing.T) {
	logDir := t.TempDir()
	h := testHandler(t, config.Config{MDNSName: "HomeServer", LogDir: logDir})

	postReq := httptest.NewRequest(http.MethodPost, "/bug/intake", strings.NewReader(url.Values{
		"reporter_device_id": {"d1"},
		"subject_device_id":  {"d1"},
		"source":             {"admin"},
		"tags":               {"unresponsive,ui_glitch"},
		"description":        {"screen stuck"},
	}.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postW := httptest.NewRecorder()
	h.ServeHTTP(postW, postReq)
	if postW.Code != http.StatusSeeOther {
		t.Fatalf("POST /bug/intake status = %d, want 303 body=%s", postW.Code, postW.Body.String())
	}
	location := postW.Header().Get("Location")
	if !strings.HasPrefix(location, "/admin/bugs/") {
		t.Fatalf("redirect location = %q, want /admin/bugs/<id>", location)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/api/bugs", nil)
	listW := httptest.NewRecorder()
	h.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("GET /admin/api/bugs status = %d, want 200 body=%s", listW.Code, listW.Body.String())
	}
	payload := map[string][]map[string]any{}
	if err := json.Unmarshal(listW.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode bug list: %v", err)
	}
	if len(payload["bugs"]) != 1 {
		t.Fatalf("bugs len = %d, want 1", len(payload["bugs"]))
	}
	if confirmed, ok := payload["bugs"][0]["confirmed"].(bool); !ok || confirmed {
		t.Fatalf("confirmed = %v, want false", payload["bugs"][0]["confirmed"])
	}

	detailReq := httptest.NewRequest(http.MethodGet, location, nil)
	detailW := httptest.NewRecorder()
	h.ServeHTTP(detailW, detailReq)
	if detailW.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200 body=%s", location, detailW.Code, detailW.Body.String())
	}
	if !strings.Contains(detailW.Body.String(), "screen stuck") {
		t.Fatalf("detail should include submitted description")
	}
}

func TestBugListFilterByTag(t *testing.T) {
	logDir := t.TempDir()
	h := testHandler(t, config.Config{MDNSName: "HomeServer", LogDir: logDir})

	post := func(tags string) {
		req := httptest.NewRequest(http.MethodPost, "/bug/intake", strings.NewReader(url.Values{
			"reporter_device_id": {"d1"},
			"subject_device_id":  {"d1"},
			"source":             {"admin"},
			"tags":               {tags},
			"description":        {"screen stuck"},
		}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusSeeOther {
			t.Fatalf("POST /bug/intake status = %d, want 303 body=%s", w.Code, w.Body.String())
		}
	}

	post("ui_glitch")
	post("lost_connection")

	listReq := httptest.NewRequest(http.MethodGet, "/admin/api/bugs?tag=lost_connection", nil)
	listW := httptest.NewRecorder()
	h.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("GET /admin/api/bugs?tag=lost_connection status = %d", listW.Code)
	}
	payload := map[string][]map[string]any{}
	if err := json.Unmarshal(listW.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode bug list: %v", err)
	}
	if len(payload["bugs"]) != 1 {
		t.Fatalf("filtered bugs len = %d, want 1", len(payload["bugs"]))
	}
}
