package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/appruntime"
	"github.com/curtcox/terminals/terminal_server/internal/config"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/replai"
	"github.com/curtcox/terminals/terminal_server/internal/replsession"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/terminal"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
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

func TestStartAndStopScenarioEndpoints(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "kitchen-1", DeviceName: "Kitchen"})
	control := transport.NewControlService("HomeServer", devices)
	engine := scenario.NewEngine()
	engine.Register(scenario.Registration{Scenario: &scenario.TerminalScenario{}, Priority: scenario.PriorityNormal})
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	h := NewHandler(control, runtime, nil, nil, nil, nil, devices, config.Config{MDNSName: "HomeServer"})

	startReq := httptest.NewRequest(http.MethodPost, "/admin/api/scenarios/start", strings.NewReader(url.Values{
		"scenario":   {"terminal"},
		"device_ids": {"kitchen-1"},
	}.Encode()))
	startReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	startW := httptest.NewRecorder()
	h.ServeHTTP(startW, startReq)
	if startW.Code != http.StatusOK {
		t.Fatalf("start status = %d, want 200, body=%s", startW.Code, startW.Body.String())
	}

	if active, ok := engine.Active("kitchen-1"); !ok || active != "terminal" {
		t.Fatalf("active(kitchen-1) = (%q, %v), want (terminal, true)", active, ok)
	}

	stopReq := httptest.NewRequest(http.MethodPost, "/admin/api/scenarios/stop", strings.NewReader(url.Values{
		"scenario":   {"terminal"},
		"device_ids": {"kitchen-1"},
	}.Encode()))
	stopReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	stopW := httptest.NewRecorder()
	h.ServeHTTP(stopW, stopReq)
	if stopW.Code != http.StatusOK {
		t.Fatalf("stop status = %d, want 200, body=%s", stopW.Code, stopW.Body.String())
	}
	if _, ok := engine.Active("kitchen-1"); ok {
		t.Fatalf("kitchen-1 should be inactive after stop")
	}
}

func TestUpdateDevicePlacementEndpoint(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "kitchen-1", DeviceName: "Kitchen"})
	control := transport.NewControlService("HomeServer", devices)
	engine := scenario.NewEngine()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	h := NewHandler(control, runtime, nil, nil, nil, nil, devices, config.Config{MDNSName: "HomeServer"})

	req := httptest.NewRequest(http.MethodPost, "/admin/api/devices/placement", strings.NewReader(url.Values{
		"device_id": {"kitchen-1"},
		"zone":      {"kitchen"},
		"roles":     {"kitchen_display,screen"},
		"mobility":  {"fixed"},
		"affinity":  {"home"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", w.Code, w.Body.String())
	}

	found, ok := devices.Get("kitchen-1")
	if !ok {
		t.Fatalf("kitchen-1 should exist")
	}
	if found.Placement.Zone != "kitchen" {
		t.Fatalf("zone = %q, want kitchen", found.Placement.Zone)
	}
	if len(found.Placement.Roles) != 2 || found.Placement.Roles[0] != "kitchen_display" || found.Placement.Roles[1] != "screen" {
		t.Fatalf("roles = %+v, want [kitchen_display screen]", found.Placement.Roles)
	}
}

func testHandler(t *testing.T, cfgOverride ...config.Config) http.Handler {
	t.Helper()
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	control := transport.NewControlService("HomeServer", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	cfg := config.Config{
		GRPCHost:      "0.0.0.0",
		GRPCPort:      50051,
		MDNSService:   "_terminals._tcp.local.",
		MDNSName:      "HomeServer",
		Version:       "1",
		AdminHTTPHost: "127.0.0.1",
		AdminHTTPPort: 50053,
		LogDir:        filepath.Join(t.TempDir(), "logs"),
	}
	if len(cfgOverride) > 0 {
		override := cfgOverride[0]
		if strings.TrimSpace(override.MDNSName) != "" {
			cfg.MDNSName = override.MDNSName
		}
		if strings.TrimSpace(override.LogDir) != "" {
			cfg.LogDir = override.LogDir
		}
	}
	return NewHandler(control, runtime, nil, nil, nil, nil, devices, cfg)
}

func TestAppsEndpointsListReloadAndRollback(t *testing.T) {
	appRoot := createTestAppPackage(t, "sound_watch", "1.0.0")
	appRuntime := appruntime.NewRuntime()
	if _, err := appRuntime.LoadPackage(context.Background(), appRoot); err != nil {
		t.Fatalf("LoadPackage(v1) error = %v", err)
	}

	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "kitchen-1", DeviceName: "Kitchen"})
	control := transport.NewControlService("HomeServer", devices)
	engine := scenario.NewEngine()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	h := NewHandler(control, runtime, nil, nil, appRuntime, func() {}, devices, config.Config{MDNSName: "HomeServer"})

	listReq := httptest.NewRequest(http.MethodGet, "/admin/api/apps", nil)
	listW := httptest.NewRecorder()
	h.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("apps list status = %d, want 200 body=%s", listW.Code, listW.Body.String())
	}
	var listed map[string][]map[string]any
	if err := json.Unmarshal(listW.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode apps list: %v", err)
	}
	if len(listed["apps"]) != 1 || listed["apps"][0]["name"] != "sound_watch" {
		t.Fatalf("apps list = %+v, want one sound_watch app", listed["apps"])
	}

	time.Sleep(5 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(appRoot, "manifest.toml"), []byte(
		"name = \"sound_watch\"\nversion = \"1.1.0\"\nlanguage = \"tal/1\"\nexports = [\"watch\"]\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v2) error = %v", err)
	}
	reloadReq := httptest.NewRequest(http.MethodPost, "/admin/api/apps/reload", strings.NewReader(url.Values{
		"app": {"sound_watch"},
	}.Encode()))
	reloadReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reloadW := httptest.NewRecorder()
	h.ServeHTTP(reloadW, reloadReq)
	if reloadW.Code != http.StatusOK {
		t.Fatalf("reload status = %d, want 200 body=%s", reloadW.Code, reloadW.Body.String())
	}
	var reloaded map[string]any
	if err := json.Unmarshal(reloadW.Body.Bytes(), &reloaded); err != nil {
		t.Fatalf("decode reload: %v", err)
	}
	if reloaded["version"] != "1.1.0" {
		t.Fatalf("reloaded version = %v, want 1.1.0", reloaded["version"])
	}

	rollbackReq := httptest.NewRequest(http.MethodPost, "/admin/api/apps/rollback", strings.NewReader(url.Values{
		"app": {"sound_watch"},
	}.Encode()))
	rollbackReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rollbackW := httptest.NewRecorder()
	h.ServeHTTP(rollbackW, rollbackReq)
	if rollbackW.Code != http.StatusOK {
		t.Fatalf("rollback status = %d, want 200 body=%s", rollbackW.Code, rollbackW.Body.String())
	}
	var rolledBack map[string]any
	if err := json.Unmarshal(rollbackW.Body.Bytes(), &rolledBack); err != nil {
		t.Fatalf("decode rollback: %v", err)
	}
	if rolledBack["version"] != "1.0.0" {
		t.Fatalf("rolled back version = %v, want 1.0.0", rolledBack["version"])
	}
}

func TestCapabilityClosureEndpoints(t *testing.T) {
	h := testHandler(t)

	getCases := []string{
		"/admin/api/identity",
		"/admin/api/identity/show?identity=system",
		"/admin/api/identity/groups",
		"/admin/api/identity/prefs?identity=system",
		"/admin/api/identity/resolve?audience=group:family",
		"/admin/api/identity/ack?subject_ref=message:msg-1",
		"/admin/api/session",
		"/admin/api/message",
		"/admin/api/message/unread?identity_id=alice",
		"/admin/api/board",
		"/admin/api/artifact",
		"/admin/api/canvas",
		"/admin/api/search?q=hello",
		"/admin/api/search/timeline?scope=message",
		"/admin/api/search/related?subject=hello",
		"/admin/api/search/recent?scope=memory",
		"/admin/api/memory?q=hello",
		"/admin/api/memory/stream?scope=kitchen",
		"/admin/api/placement",
		"/admin/api/recent",
		"/admin/api/store/get?namespace=ns&key=k",
		"/admin/api/store/ls?namespace=ns",
		"/admin/api/bus",
	}
	for _, path := range getCases {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200 body=%s", path, w.Code, w.Body.String())
		}
	}

	postCases := []struct {
		path          string
		form          url.Values
		allowNotFound bool
	}{
		{path: "/admin/api/identity/ack", form: url.Values{"subject_ref": {"message:msg-1"}, "actor": {"device:kitchen-screen"}, "mode": {"dismissed"}}},
		{path: "/admin/api/session/create", form: url.Values{"kind": {"help"}, "target": {"room"}}},
		{path: "/admin/api/session/attach", form: url.Values{"session_id": {"missing"}, "device_ref": {"device:screen-1"}}, allowNotFound: true},
		{path: "/admin/api/session/detach", form: url.Values{"session_id": {"missing"}, "device_ref": {"device:screen-1"}}, allowNotFound: true},
		{path: "/admin/api/session/control/request", form: url.Values{"session_id": {"missing"}, "participant": {"alice"}, "control_type": {"keyboard"}}, allowNotFound: true},
		{path: "/admin/api/session/control/grant", form: url.Values{"session_id": {"missing"}, "participant": {"alice"}, "granted_by": {"moderator"}, "control_type": {"keyboard"}}, allowNotFound: true},
		{path: "/admin/api/session/control/revoke", form: url.Values{"session_id": {"missing"}, "participant": {"alice"}, "revoked_by": {"moderator"}}, allowNotFound: true},
		{path: "/admin/api/message/post", form: url.Values{"room": {"room-1"}, "text": {"hello"}}},
		{path: "/admin/api/message/dm", form: url.Values{"target_ref": {"mom"}, "text": {"hello"}}},
		{path: "/admin/api/board/post", form: url.Values{"board": {"family"}, "text": {"note"}}},
		{path: "/admin/api/board/pin", form: url.Values{"board": {"family"}, "text": {"note"}}},
		{path: "/admin/api/artifact/create", form: url.Values{"kind": {"lesson"}, "title": {"math"}}},
		{path: "/admin/api/artifact/replace", form: url.Values{"artifact_id": {"missing"}, "title": {"ignored"}}, allowNotFound: true},
		{path: "/admin/api/artifact/template/save", form: url.Values{"name": {"base"}, "source_artifact_id": {"missing"}}, allowNotFound: true},
		{path: "/admin/api/artifact/template/apply", form: url.Values{"name": {"base"}, "target_artifact_id": {"missing"}}, allowNotFound: true},
		{path: "/admin/api/canvas/annotate", form: url.Values{"canvas": {"c1"}, "text": {"draw"}}},
		{path: "/admin/api/memory/remember", form: url.Values{"scope": {"kitchen"}, "text": {"milk"}}},
		{path: "/admin/api/store/put", form: url.Values{"namespace": {"ns"}, "key": {"k"}, "value": {"v"}}},
		{path: "/admin/api/bus/emit", form: url.Values{"kind": {"event"}, "name": {"alarm"}, "payload": {"ring"}}},
	}
	for _, tc := range postCases {
		req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if tc.allowNotFound && w.Code == http.StatusNotFound {
			continue
		}
		if w.Code != http.StatusOK {
			t.Fatalf("POST %s status = %d, want 200 body=%s", tc.path, w.Code, w.Body.String())
		}
	}

	createReq := httptest.NewRequest(http.MethodPost, "/admin/api/session/create", strings.NewReader(url.Values{
		"kind":   {"lesson"},
		"target": {"math-room"},
	}.Encode()))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createW := httptest.NewRecorder()
	h.ServeHTTP(createW, createReq)
	if createW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/session/create status = %d, want 200 body=%s", createW.Code, createW.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode session create response error = %v", err)
	}
	sessionMap, _ := created["session"].(map[string]any)
	sessionID, _ := sessionMap["id"].(string)
	if strings.TrimSpace(sessionID) == "" {
		t.Fatalf("missing created session id in response: %s", createW.Body.String())
	}

	for _, path := range []string{"/admin/api/session/join", "/admin/api/session/leave"} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(url.Values{
			"session_id":  {sessionID},
			"participant": {"alice"},
		}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("POST %s status = %d, want 200 body=%s", path, w.Code, w.Body.String())
		}
	}

	attachReq := httptest.NewRequest(http.MethodPost, "/admin/api/session/attach", strings.NewReader(url.Values{
		"session_id": {sessionID},
		"device_ref": {"device:kitchen-display"},
	}.Encode()))
	attachReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	attachW := httptest.NewRecorder()
	h.ServeHTTP(attachW, attachReq)
	if attachW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/session/attach status = %d, want 200 body=%s", attachW.Code, attachW.Body.String())
	}
	if !strings.Contains(attachW.Body.String(), "device:kitchen-display") {
		t.Fatalf("attach response missing device ref: %s", attachW.Body.String())
	}

	for _, path := range []struct {
		url  string
		form url.Values
	}{
		{url: "/admin/api/session/control/request", form: url.Values{"session_id": {sessionID}, "participant": {"alice"}, "control_type": {"keyboard"}}},
		{url: "/admin/api/session/control/grant", form: url.Values{"session_id": {sessionID}, "participant": {"alice"}, "granted_by": {"moderator"}, "control_type": {"keyboard"}}},
		{url: "/admin/api/session/control/revoke", form: url.Values{"session_id": {sessionID}, "participant": {"alice"}, "revoked_by": {"moderator"}}},
		{url: "/admin/api/session/detach", form: url.Values{"session_id": {sessionID}, "device_ref": {"device:kitchen-display"}}},
	} {
		req := httptest.NewRequest(http.MethodPost, path.url, strings.NewReader(path.form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("POST %s status = %d, want 200 body=%s", path.url, w.Code, w.Body.String())
		}
	}

	for _, path := range []string{
		"/admin/api/session/show?session_id=" + url.QueryEscape(sessionID),
		"/admin/api/session/members?session_id=" + url.QueryEscape(sessionID),
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200 body=%s", path, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), sessionID) {
			t.Fatalf("GET %s missing session id %q in body=%s", path, sessionID, w.Body.String())
		}
	}

	postMessageReq := httptest.NewRequest(http.MethodPost, "/admin/api/message/post", strings.NewReader(url.Values{
		"room": {"room-1"},
		"text": {"bring milk"},
	}.Encode()))
	postMessageReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postMessageW := httptest.NewRecorder()
	h.ServeHTTP(postMessageW, postMessageReq)
	if postMessageW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/message/post status = %d, want 200 body=%s", postMessageW.Code, postMessageW.Body.String())
	}
	var posted map[string]any
	if err := json.Unmarshal(postMessageW.Body.Bytes(), &posted); err != nil {
		t.Fatalf("decode message post response error = %v", err)
	}
	messageMap, _ := posted["message"].(map[string]any)
	messageID, _ := messageMap["id"].(string)
	if strings.TrimSpace(messageID) == "" {
		t.Fatalf("missing message id in response: %s", postMessageW.Body.String())
	}

	directReq := httptest.NewRequest(http.MethodPost, "/admin/api/message/dm", strings.NewReader(url.Values{
		"target_ref": {"mom"},
		"text":       {"come downstairs"},
	}.Encode()))
	directReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	directW := httptest.NewRecorder()
	h.ServeHTTP(directW, directReq)
	if directW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/message/dm status = %d, want 200 body=%s", directW.Code, directW.Body.String())
	}
	if !strings.Contains(directW.Body.String(), "person:mom") {
		t.Fatalf("direct message response missing normalized target ref: %s", directW.Body.String())
	}

	boardPostReq := httptest.NewRequest(http.MethodPost, "/admin/api/board/post", strings.NewReader(url.Values{
		"board": {"family"},
		"text":  {"Need milk"},
	}.Encode()))
	boardPostReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	boardPostW := httptest.NewRecorder()
	h.ServeHTTP(boardPostW, boardPostReq)
	if boardPostW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/board/post status = %d, want 200 body=%s", boardPostW.Code, boardPostW.Body.String())
	}
	if strings.Contains(boardPostW.Body.String(), `"pinned":true`) {
		t.Fatalf("board post response should not be pinned: %s", boardPostW.Body.String())
	}

	ackReq := httptest.NewRequest(http.MethodPost, "/admin/api/message/ack", strings.NewReader(url.Values{
		"identity_id": {"alice"},
		"message_id":  {messageID},
	}.Encode()))
	ackReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ackW := httptest.NewRecorder()
	h.ServeHTTP(ackW, ackReq)
	if ackW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/message/ack status = %d, want 200 body=%s", ackW.Code, ackW.Body.String())
	}

	unreadReq := httptest.NewRequest(http.MethodGet, "/admin/api/message/unread?identity_id=alice&room=room-1", nil)
	unreadW := httptest.NewRecorder()
	h.ServeHTTP(unreadW, unreadReq)
	if unreadW.Code != http.StatusOK {
		t.Fatalf("GET /admin/api/message/unread status = %d, want 200 body=%s", unreadW.Code, unreadW.Body.String())
	}
	if strings.Contains(unreadW.Body.String(), messageID) {
		t.Fatalf("expected message %q to be acknowledged, unread body=%s", messageID, unreadW.Body.String())
	}

	createArtifactReq := httptest.NewRequest(http.MethodPost, "/admin/api/artifact/create", strings.NewReader(url.Values{
		"kind":  {"lesson"},
		"title": {"math basics"},
	}.Encode()))
	createArtifactReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createArtifactW := httptest.NewRecorder()
	h.ServeHTTP(createArtifactW, createArtifactReq)
	if createArtifactW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/artifact/create status = %d, want 200 body=%s", createArtifactW.Code, createArtifactW.Body.String())
	}
	var createdArtifact map[string]any
	if err := json.Unmarshal(createArtifactW.Body.Bytes(), &createdArtifact); err != nil {
		t.Fatalf("decode artifact create response error = %v", err)
	}
	artifactMap, _ := createdArtifact["artifact"].(map[string]any)
	artifactID, _ := artifactMap["id"].(string)
	if strings.TrimSpace(artifactID) == "" {
		t.Fatalf("missing artifact id in response: %s", createArtifactW.Body.String())
	}

	patchArtifactReq := httptest.NewRequest(http.MethodPost, "/admin/api/artifact/patch", strings.NewReader(url.Values{
		"artifact_id": {artifactID},
		"title":       {"math mastery"},
	}.Encode()))
	patchArtifactReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	patchArtifactW := httptest.NewRecorder()
	h.ServeHTTP(patchArtifactW, patchArtifactReq)
	if patchArtifactW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/artifact/patch status = %d, want 200 body=%s", patchArtifactW.Code, patchArtifactW.Body.String())
	}
	if !strings.Contains(patchArtifactW.Body.String(), "math mastery") {
		t.Fatalf("patched artifact missing updated title: %s", patchArtifactW.Body.String())
	}

	replaceArtifactReq := httptest.NewRequest(http.MethodPost, "/admin/api/artifact/replace", strings.NewReader(url.Values{
		"artifact_id": {artifactID},
		"title":       {"math replacement"},
	}.Encode()))
	replaceArtifactReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	replaceArtifactW := httptest.NewRecorder()
	h.ServeHTTP(replaceArtifactW, replaceArtifactReq)
	if replaceArtifactW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/artifact/replace status = %d, want 200 body=%s", replaceArtifactW.Code, replaceArtifactW.Body.String())
	}
	if !strings.Contains(replaceArtifactW.Body.String(), "math replacement") {
		t.Fatalf("replaced artifact missing updated title: %s", replaceArtifactW.Body.String())
	}

	saveTemplateReq := httptest.NewRequest(http.MethodPost, "/admin/api/artifact/template/save", strings.NewReader(url.Values{
		"name":               {"lesson-base"},
		"source_artifact_id": {artifactID},
	}.Encode()))
	saveTemplateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	saveTemplateW := httptest.NewRecorder()
	h.ServeHTTP(saveTemplateW, saveTemplateReq)
	if saveTemplateW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/artifact/template/save status = %d, want 200 body=%s", saveTemplateW.Code, saveTemplateW.Body.String())
	}
	if !strings.Contains(saveTemplateW.Body.String(), "lesson-base") {
		t.Fatalf("template save response missing template name: %s", saveTemplateW.Body.String())
	}

	applyTemplateReq := httptest.NewRequest(http.MethodPost, "/admin/api/artifact/template/apply", strings.NewReader(url.Values{
		"name":               {"lesson-base"},
		"target_artifact_id": {artifactID},
	}.Encode()))
	applyTemplateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	applyTemplateW := httptest.NewRecorder()
	h.ServeHTTP(applyTemplateW, applyTemplateReq)
	if applyTemplateW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/artifact/template/apply status = %d, want 200 body=%s", applyTemplateW.Code, applyTemplateW.Body.String())
	}
	if !strings.Contains(applyTemplateW.Body.String(), "lesson-base") && !strings.Contains(applyTemplateW.Body.String(), "math replacement") {
		t.Fatalf("template apply response missing expected payload: %s", applyTemplateW.Body.String())
	}

	getArtifactReq := httptest.NewRequest(http.MethodGet, "/admin/api/artifact/get?artifact_id="+url.QueryEscape(artifactID), nil)
	getArtifactW := httptest.NewRecorder()
	h.ServeHTTP(getArtifactW, getArtifactReq)
	if getArtifactW.Code != http.StatusOK {
		t.Fatalf("GET /admin/api/artifact/get status = %d, want 200 body=%s", getArtifactW.Code, getArtifactW.Body.String())
	}
	if !strings.Contains(getArtifactW.Body.String(), `"version":4`) {
		t.Fatalf("artifact get body missing updated version: %s", getArtifactW.Body.String())
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/admin/api/artifact/history?artifact_id="+url.QueryEscape(artifactID), nil)
	historyW := httptest.NewRecorder()
	h.ServeHTTP(historyW, historyReq)
	if historyW.Code != http.StatusOK {
		t.Fatalf("GET /admin/api/artifact/history status = %d, want 200 body=%s", historyW.Code, historyW.Body.String())
	}
	if !strings.Contains(historyW.Body.String(), `"action":"create"`) || !strings.Contains(historyW.Body.String(), `"action":"patch"`) || !strings.Contains(historyW.Body.String(), `"action":"replace"`) {
		t.Fatalf("artifact history missing create/patch/replace entries: %s", historyW.Body.String())
	}
}

func TestIdentityResolveEndpointRequiresGET(t *testing.T) {
	h := testHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/admin/api/identity/resolve", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405 body=%s", w.Code, w.Body.String())
	}
}

func TestReplSessionGetAndDeleteEndpoints(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	control := transport.NewControlService("HomeServer", devices)
	engine := scenario.NewEngine()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	replSvc := replsession.NewService(terminal.NewManager())
	created, err := replSvc.CreateSession(context.Background(), replsession.CreateSessionRequest{
		DeviceID: "d1",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	defer func() {
		_, _ = replSvc.TerminateSession(context.Background(), replsession.TerminateSessionRequest{
			SessionID: created.Session.ID,
		})
	}()

	h := NewHandler(control, runtime, replSvc, nil, nil, nil, devices, config.Config{MDNSName: "HomeServer"})

	getReq := httptest.NewRequest(http.MethodGet, "/admin/api/repl/sessions/"+created.Session.ID, nil)
	getW := httptest.NewRecorder()
	h.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("GET repl session status = %d, want 200 body=%s", getW.Code, getW.Body.String())
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/admin/api/repl/sessions/"+created.Session.ID, nil)
	delW := httptest.NewRecorder()
	h.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusOK {
		t.Fatalf("DELETE repl session status = %d, want 200 body=%s", delW.Code, delW.Body.String())
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/admin/api/repl/sessions/"+created.Session.ID, nil)
	missingW := httptest.NewRecorder()
	h.ServeHTTP(missingW, missingReq)
	if missingW.Code != http.StatusNotFound {
		t.Fatalf("GET after delete status = %d, want 404 body=%s", missingW.Code, missingW.Body.String())
	}
}

func TestReplAIEndpoints(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	control := transport.NewControlService("HomeServer", devices)
	engine := scenario.NewEngine()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	replSvc := replsession.NewService(terminal.NewManager())
	created, err := replSvc.CreateSession(context.Background(), replsession.CreateSessionRequest{
		DeviceID: "d1",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	defer func() {
		_, _ = replSvc.TerminateSession(context.Background(), replsession.TerminateSessionRequest{
			SessionID: created.Session.ID,
		})
	}()
	aiSvc := replai.NewService(replSvc, replai.Config{
		DefaultProvider: "ollama",
		DefaultModel:    "llama3.1",
		Providers: []replai.ProviderConfig{
			{Name: "openrouter", Models: []string{"anthropic/claude-sonnet-4-6"}},
			{Name: "ollama", Models: []string{"llama3.1"}},
		},
	})

	h := NewHandler(control, runtime, replSvc, aiSvc, nil, nil, devices, config.Config{MDNSName: "HomeServer"})

	providersReq := httptest.NewRequest(http.MethodGet, "/admin/api/repl/ai/providers", nil)
	providersW := httptest.NewRecorder()
	h.ServeHTTP(providersW, providersReq)
	if providersW.Code != http.StatusOK {
		t.Fatalf("providers status = %d, want 200 body=%s", providersW.Code, providersW.Body.String())
	}
	if !strings.Contains(providersW.Body.String(), "openrouter") || !strings.Contains(providersW.Body.String(), "ollama") {
		t.Fatalf("providers body = %s", providersW.Body.String())
	}

	modelsReq := httptest.NewRequest(http.MethodGet, "/admin/api/repl/ai/models?provider=ollama", nil)
	modelsW := httptest.NewRecorder()
	h.ServeHTTP(modelsW, modelsReq)
	if modelsW.Code != http.StatusOK {
		t.Fatalf("models status = %d, want 200 body=%s", modelsW.Code, modelsW.Body.String())
	}
	if !strings.Contains(modelsW.Body.String(), "llama3.1") {
		t.Fatalf("models body = %s", modelsW.Body.String())
	}

	getSelectionReq := httptest.NewRequest(http.MethodGet, "/admin/api/repl/ai/selection?session_id="+created.Session.ID, nil)
	getSelectionW := httptest.NewRecorder()
	h.ServeHTTP(getSelectionW, getSelectionReq)
	if getSelectionW.Code != http.StatusOK {
		t.Fatalf("selection GET status = %d, want 200 body=%s", getSelectionW.Code, getSelectionW.Body.String())
	}
	if !strings.Contains(getSelectionW.Body.String(), "\"provider\":\"ollama\"") {
		t.Fatalf("selection GET body = %s", getSelectionW.Body.String())
	}

	setSelectionReq := httptest.NewRequest(http.MethodPost, "/admin/api/repl/ai/selection", strings.NewReader(url.Values{
		"session_id": {created.Session.ID},
		"provider":   {"openrouter"},
		"model":      {"anthropic/claude-sonnet-4-6"},
	}.Encode()))
	setSelectionReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	setSelectionW := httptest.NewRecorder()
	h.ServeHTTP(setSelectionW, setSelectionReq)
	if setSelectionW.Code != http.StatusOK {
		t.Fatalf("selection POST status = %d, want 200 body=%s", setSelectionW.Code, setSelectionW.Body.String())
	}
	if !strings.Contains(setSelectionW.Body.String(), "\"provider\":\"openrouter\"") {
		t.Fatalf("selection POST body = %s", setSelectionW.Body.String())
	}
}

func createTestAppPackage(t *testing.T, name, version string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll(app) error = %v", err)
	}
	manifest := "name = \"" + name + "\"\nversion = \"" + version + "\"\nlanguage = \"tal/1\"\nexports = [\"watch\"]\n"
	if err := os.WriteFile(filepath.Join(root, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main.tal) error = %v", err)
	}
	return root
}
