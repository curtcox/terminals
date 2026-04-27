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
	"github.com/curtcox/terminals/terminal_server/internal/world"
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

	h := NewHandler(control, runtime, nil, nil, nil, nil, devices, config.Config{MDNSName: "HomeServer"}, nil)

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

	h := NewHandler(control, runtime, nil, nil, nil, nil, devices, config.Config{MDNSName: "HomeServer"}, nil)

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
	worldModel := world.NewModel()
	worldModel.UpsertGeometry(context.Background(), world.DeviceGeometry{DeviceID: "d1", Zone: "kitchen"})

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
	return NewHandler(control, runtime, nil, nil, nil, nil, devices, cfg, worldModel)
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

	h := NewHandler(control, runtime, nil, nil, appRuntime, func() {}, devices, config.Config{MDNSName: "HomeServer"}, nil)

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
		"/admin/api/message/rooms",
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
		"/admin/api/ui/views",
		"/admin/api/ui/snapshot?device_id=d1",
		"/admin/api/recent",
		"/admin/api/store/get?namespace=ns&key=k",
		"/admin/api/store/ns",
		"/admin/api/store/ls?namespace=ns",
		"/admin/api/store/watch?namespace=ns&prefix=k",
		"/admin/api/bus",
		"/admin/api/bus?kind=event&name=alarm&limit=1",
		"/admin/api/bus/replay?from=bus-1&to=bus-9&kind=event",
		"/admin/api/handlers",
		"/admin/api/scenarios/inline",
		"/admin/api/sim/devices",
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
		allowConflict bool
	}{
		{path: "/admin/api/identity/ack", form: url.Values{"subject_ref": {"message:msg-1"}, "actor": {"device:kitchen-screen"}, "mode": {"dismissed"}}},
		{path: "/admin/api/session/create", form: url.Values{"kind": {"help"}, "target": {"room"}}},
		{path: "/admin/api/session/attach", form: url.Values{"session_id": {"missing"}, "device_ref": {"device:screen-1"}}, allowNotFound: true},
		{path: "/admin/api/session/detach", form: url.Values{"session_id": {"missing"}, "device_ref": {"device:screen-1"}}, allowNotFound: true},
		{path: "/admin/api/session/control/request", form: url.Values{"session_id": {"missing"}, "participant": {"alice"}, "control_type": {"keyboard"}}, allowNotFound: true},
		{path: "/admin/api/session/control/grant", form: url.Values{"session_id": {"missing"}, "participant": {"alice"}, "granted_by": {"moderator"}, "control_type": {"keyboard"}}, allowNotFound: true},
		{path: "/admin/api/session/control/revoke", form: url.Values{"session_id": {"missing"}, "participant": {"alice"}, "revoked_by": {"moderator"}}, allowNotFound: true},
		{path: "/admin/api/message/room", form: url.Values{"name": {"kitchen"}}},
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
		{path: "/admin/api/ui/views/upsert", form: url.Values{"view_id": {"kitchen-home"}, "root_id": {"root-main"}, "descriptor": {`{"type":"stack"}`}}},
		{path: "/admin/api/ui/push", form: url.Values{"device_id": {"d1"}, "descriptor": {`{"type":"stack"}`}, "root_id": {"root-main"}}},
		{path: "/admin/api/ui/patch", form: url.Values{"device_id": {"d1"}, "component_id": {"banner"}, "descriptor": {`{"type":"text"}`}}},
		{path: "/admin/api/ui/transition", form: url.Values{"device_id": {"d1"}, "component_id": {"banner"}, "transition": {"fade"}, "duration_ms": {"150"}}},
		{path: "/admin/api/ui/subscribe", form: url.Values{"device_id": {"d1"}, "to": {"cohort:family-screens"}}},
		{path: "/admin/api/store/put", form: url.Values{"namespace": {"ns"}, "key": {"k"}, "value": {"v"}}},
		{path: "/admin/api/store/bind", form: url.Values{"namespace": {"ns"}, "key": {"k"}, "to": {"device-1:chat"}}},
		{path: "/admin/api/store/del", form: url.Values{"namespace": {"ns"}, "key": {"k"}}},
		{path: "/admin/api/bus/emit", form: url.Values{"kind": {"event"}, "name": {"alarm"}, "payload": {"ring"}}},
		{path: "/admin/api/handlers/on", form: url.Values{"selector": {"scenario=chat"}, "action": {"submit"}, "run": {"store put notes key value"}}},
		{path: "/admin/api/handlers/off", form: url.Values{"handler_id": {"handler-1"}}},
		{path: "/admin/api/scenarios/inline/define", form: url.Values{"name": {"red_alert"}, "match_intent": {"red alert"}, "on_start": {"ui broadcast all_screens banner"}}},
		{path: "/admin/api/scenarios/inline/undefine", form: url.Values{"name": {"red_alert"}}},
		{path: "/admin/api/sim/devices/new", form: url.Values{"device_id": {"sim-kitchen"}, "caps": {"display,keyboard"}}},
		{path: "/admin/api/sim/input", form: url.Values{"device_id": {"sim-kitchen"}, "component_id": {"chat_box"}, "action": {"submit"}, "value": {"hello"}}},
		{path: "/admin/api/sim/expect", form: url.Values{"device_id": {"sim-kitchen"}, "kind": {"ui"}, "selector": {"chat"}}, allowConflict: true},
		{path: "/admin/api/sim/record", form: url.Values{"device_id": {"sim-kitchen"}, "duration": {"1s"}}, allowNotFound: true},
		{path: "/admin/api/sim/devices/rm", form: url.Values{"device_id": {"sim-kitchen"}}},
		{path: "/admin/api/scripts/run", form: url.Values{"path": {"missing.term"}}, allowNotFound: true},
	}
	for _, tc := range postCases {
		req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if tc.allowNotFound && w.Code == http.StatusNotFound {
			continue
		}
		if tc.allowConflict && w.Code == http.StatusConflict {
			continue
		}
		if w.Code != http.StatusOK {
			t.Fatalf("POST %s status = %d, want 200 body=%s", tc.path, w.Code, w.Body.String())
		}
	}

	roomShowReq := httptest.NewRequest(http.MethodGet, "/admin/api/message/room?room=kitchen", nil)
	roomShowW := httptest.NewRecorder()
	h.ServeHTTP(roomShowW, roomShowReq)
	if roomShowW.Code != http.StatusOK {
		t.Fatalf("GET /admin/api/message/room status = %d, want 200 body=%s", roomShowW.Code, roomShowW.Body.String())
	}
	if !strings.Contains(roomShowW.Body.String(), `"name":"kitchen"`) {
		t.Fatalf("message room show missing kitchen payload: %s", roomShowW.Body.String())
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

	getMessageReq := httptest.NewRequest(http.MethodGet, "/admin/api/message/get?message_id="+url.QueryEscape(messageID), nil)
	getMessageW := httptest.NewRecorder()
	h.ServeHTTP(getMessageW, getMessageReq)
	if getMessageW.Code != http.StatusOK {
		t.Fatalf("GET /admin/api/message/get status = %d, want 200 body=%s", getMessageW.Code, getMessageW.Body.String())
	}
	if !strings.Contains(getMessageW.Body.String(), messageID) {
		t.Fatalf("message get response missing message id %q: %s", messageID, getMessageW.Body.String())
	}

	threadReq := httptest.NewRequest(http.MethodPost, "/admin/api/message/thread", strings.NewReader(url.Values{
		"root_ref": {messageID},
		"text":     {"thread follow-up"},
	}.Encode()))
	threadReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	threadW := httptest.NewRecorder()
	h.ServeHTTP(threadW, threadReq)
	if threadW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/message/thread status = %d, want 200 body=%s", threadW.Code, threadW.Body.String())
	}
	if !strings.Contains(threadW.Body.String(), `"thread_root_ref":"`+messageID+`"`) {
		t.Fatalf("thread response missing root ref %q: %s", messageID, threadW.Body.String())
	}
	var threaded map[string]any
	if err := json.Unmarshal(threadW.Body.Bytes(), &threaded); err != nil {
		t.Fatalf("decode message thread response error = %v", err)
	}
	threadMessageMap, _ := threaded["message"].(map[string]any)
	threadMessageID, _ := threadMessageMap["id"].(string)
	if strings.TrimSpace(threadMessageID) == "" {
		t.Fatalf("missing threaded message id in response: %s", threadW.Body.String())
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

	ackThreadReq := httptest.NewRequest(http.MethodPost, "/admin/api/message/ack", strings.NewReader(url.Values{
		"identity_id": {"alice"},
		"message_id":  {threadMessageID},
	}.Encode()))
	ackThreadReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ackThreadW := httptest.NewRecorder()
	h.ServeHTTP(ackThreadW, ackThreadReq)
	if ackThreadW.Code != http.StatusOK {
		t.Fatalf("POST /admin/api/message/ack (thread) status = %d, want 200 body=%s", ackThreadW.Code, ackThreadW.Body.String())
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

func TestSimAndScriptsEndpoints(t *testing.T) {
	h := testHandler(t)

	newReq := httptest.NewRequest(http.MethodPost, "/admin/api/sim/devices/new", strings.NewReader(url.Values{
		"device_id": {"sim-lab"},
		"caps":      {"display,keyboard"},
	}.Encode()))
	newReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	newW := httptest.NewRecorder()
	h.ServeHTTP(newW, newReq)
	if newW.Code != http.StatusOK {
		t.Fatalf("sim device new status = %d, want 200 body=%s", newW.Code, newW.Body.String())
	}

	inputReq := httptest.NewRequest(http.MethodPost, "/admin/api/sim/input", strings.NewReader(url.Values{
		"device_id":    {"sim-lab"},
		"component_id": {"banner"},
		"action":       {"tap"},
		"value":        {"ack"},
	}.Encode()))
	inputReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	inputW := httptest.NewRecorder()
	h.ServeHTTP(inputW, inputReq)
	if inputW.Code != http.StatusOK {
		t.Fatalf("sim input status = %d, want 200 body=%s", inputW.Code, inputW.Body.String())
	}

	uiReq := httptest.NewRequest(http.MethodGet, "/admin/api/sim/ui?device_id=sim-lab", nil)
	uiW := httptest.NewRecorder()
	h.ServeHTTP(uiW, uiReq)
	if uiW.Code != http.StatusOK {
		t.Fatalf("sim ui status = %d, want 200 body=%s", uiW.Code, uiW.Body.String())
	}
	if !strings.Contains(uiW.Body.String(), `"device_id":"sim-lab"`) {
		t.Fatalf("sim ui body missing device id: %s", uiW.Body.String())
	}

	expectReq := httptest.NewRequest(http.MethodPost, "/admin/api/sim/expect", strings.NewReader(url.Values{
		"device_id": {"sim-lab"},
		"kind":      {"ui"},
		"selector":  {"banner"},
	}.Encode()))
	expectReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	expectW := httptest.NewRecorder()
	h.ServeHTTP(expectW, expectReq)
	if expectW.Code != http.StatusConflict {
		t.Fatalf("sim expect status = %d, want 409 body=%s", expectW.Code, expectW.Body.String())
	}

	recordReq := httptest.NewRequest(http.MethodPost, "/admin/api/sim/record", strings.NewReader(url.Values{
		"device_id": {"sim-lab"},
		"duration":  {"5s"},
	}.Encode()))
	recordReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recordW := httptest.NewRecorder()
	h.ServeHTTP(recordW, recordReq)
	if recordW.Code != http.StatusOK {
		t.Fatalf("sim record status = %d, want 200 body=%s", recordW.Code, recordW.Body.String())
	}
	if !strings.Contains(recordW.Body.String(), `"duration":"5s"`) {
		t.Fatalf("sim record body missing duration: %s", recordW.Body.String())
	}

	scriptPath := filepath.Join(t.TempDir(), "smoke.term")
	if err := os.WriteFile(scriptPath, []byte("# comment\n\nstore put notes k v\nui push d1 banner\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(script) error = %v", err)
	}
	dryRunReq := httptest.NewRequest(http.MethodPost, "/admin/api/scripts/dry-run", strings.NewReader(url.Values{
		"path": {scriptPath},
	}.Encode()))
	dryRunReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	dryRunW := httptest.NewRecorder()
	h.ServeHTTP(dryRunW, dryRunReq)
	if dryRunW.Code != http.StatusOK {
		t.Fatalf("scripts dry-run status = %d, want 200 body=%s", dryRunW.Code, dryRunW.Body.String())
	}
	if !strings.Contains(dryRunW.Body.String(), `"command_count":2`) {
		t.Fatalf("scripts dry-run body missing command count: %s", dryRunW.Body.String())
	}

	runReq := httptest.NewRequest(http.MethodPost, "/admin/api/scripts/run", strings.NewReader(url.Values{
		"path": {scriptPath},
	}.Encode()))
	runReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)
	if runW.Code != http.StatusOK {
		t.Fatalf("scripts run status = %d, want 200 body=%s", runW.Code, runW.Body.String())
	}
	if !strings.Contains(runW.Body.String(), `"executed_count":2`) {
		t.Fatalf("scripts run body missing executed count: %s", runW.Body.String())
	}
}

func TestScriptsRunCrossUsecaseSimulationFixture(t *testing.T) {
	h := testHandler(t)
	fixturePath := filepath.Join("..", "..", "testdata", "repl", "phase12-cross-usecase.term")

	dryRunReq := httptest.NewRequest(http.MethodPost, "/admin/api/scripts/dry-run", strings.NewReader(url.Values{
		"path": {fixturePath},
	}.Encode()))
	dryRunReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	dryRunW := httptest.NewRecorder()
	h.ServeHTTP(dryRunW, dryRunReq)
	if dryRunW.Code != http.StatusOK {
		t.Fatalf("fixture scripts dry-run status = %d, want 200 body=%s", dryRunW.Code, dryRunW.Body.String())
	}
	if !strings.Contains(dryRunW.Body.String(), `"command_count":21`) {
		t.Fatalf("fixture scripts dry-run body missing command count: %s", dryRunW.Body.String())
	}

	runReq := httptest.NewRequest(http.MethodPost, "/admin/api/scripts/run", strings.NewReader(url.Values{
		"path": {fixturePath},
	}.Encode()))
	runReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	runW := httptest.NewRecorder()
	h.ServeHTTP(runW, runReq)
	if runW.Code != http.StatusOK {
		t.Fatalf("fixture scripts run status = %d, want 200 body=%s", runW.Code, runW.Body.String())
	}
	body := runW.Body.String()
	if !strings.Contains(body, `"executed_count":21`) || !strings.Contains(body, `"failed_count":0`) {
		t.Fatalf("fixture scripts run body missing execution counters: %s", body)
	}

	storeReq := httptest.NewRequest(http.MethodGet, "/admin/api/store/get?namespace=phase12&key=status", nil)
	storeW := httptest.NewRecorder()
	h.ServeHTTP(storeW, storeReq)
	if storeW.Code != http.StatusOK {
		t.Fatalf("fixture store get status = %d, want 200 body=%s", storeW.Code, storeW.Body.String())
	}
	if !strings.Contains(storeW.Body.String(), `"namespace":"phase12"`) || !strings.Contains(storeW.Body.String(), `"value":"seeded"`) {
		t.Fatalf("fixture store get body missing seeded record: %s", storeW.Body.String())
	}

	messageReq := httptest.NewRequest(http.MethodGet, "/admin/api/message?room=phase12-room", nil)
	messageW := httptest.NewRecorder()
	h.ServeHTTP(messageW, messageReq)
	if messageW.Code != http.StatusOK {
		t.Fatalf("fixture message ls status = %d, want 200 body=%s", messageW.Code, messageW.Body.String())
	}
	if !strings.Contains(messageW.Body.String(), `"room":"phase12-room"`) || !strings.Contains(messageW.Body.String(), `"text":"fixture-layer2-mutating"`) {
		t.Fatalf("fixture message ls body missing layer2 message side effect: %s", messageW.Body.String())
	}

	boardReq := httptest.NewRequest(http.MethodGet, "/admin/api/board?board=phase12-board", nil)
	boardW := httptest.NewRecorder()
	h.ServeHTTP(boardW, boardReq)
	if boardW.Code != http.StatusOK {
		t.Fatalf("fixture board ls status = %d, want 200 body=%s", boardW.Code, boardW.Body.String())
	}
	if !strings.Contains(boardW.Body.String(), `"board":"phase12-board"`) || !strings.Contains(boardW.Body.String(), `"text":"fixture-board-mutating"`) {
		t.Fatalf("fixture board ls body missing layer2 board side effect: %s", boardW.Body.String())
	}

	artifactsReq := httptest.NewRequest(http.MethodGet, "/admin/api/artifact", nil)
	artifactsW := httptest.NewRecorder()
	h.ServeHTTP(artifactsW, artifactsReq)
	if artifactsW.Code != http.StatusOK {
		t.Fatalf("fixture artifact ls status = %d, want 200 body=%s", artifactsW.Code, artifactsW.Body.String())
	}
	var artifactsBody map[string]any
	if err := json.Unmarshal(artifactsW.Body.Bytes(), &artifactsBody); err != nil {
		t.Fatalf("decode fixture artifact ls body error = %v body=%s", err, artifactsW.Body.String())
	}
	artifactItems, _ := artifactsBody["artifacts"].([]any)
	artifactID := ""
	for _, item := range artifactItems {
		artifactMap, _ := item.(map[string]any)
		if artifactMap == nil {
			continue
		}
		if title, _ := artifactMap["title"].(string); title == "fixture-artifact-mutating" {
			artifactID, _ = artifactMap["id"].(string)
			break
		}
	}
	if strings.TrimSpace(artifactID) == "" {
		t.Fatalf("fixture artifact ls missing layer2 artifact side effect: %s", artifactsW.Body.String())
	}

	artifactReq := httptest.NewRequest(http.MethodGet, "/admin/api/artifact/history?artifact_id="+url.QueryEscape(artifactID), nil)
	artifactW := httptest.NewRecorder()
	h.ServeHTTP(artifactW, artifactReq)
	if artifactW.Code != http.StatusOK {
		t.Fatalf("fixture artifact history status = %d, want 200 body=%s", artifactW.Code, artifactW.Body.String())
	}
	if !strings.Contains(artifactW.Body.String(), `"artifact_id":"`+artifactID+`"`) || !strings.Contains(artifactW.Body.String(), `"action":"create"`) || !strings.Contains(artifactW.Body.String(), `"title":"fixture-artifact-mutating"`) {
		t.Fatalf("fixture artifact history missing layer2 artifact side effect: %s", artifactW.Body.String())
	}

	canvasReq := httptest.NewRequest(http.MethodGet, "/admin/api/canvas?canvas=phase12-canvas", nil)
	canvasW := httptest.NewRecorder()
	h.ServeHTTP(canvasW, canvasReq)
	if canvasW.Code != http.StatusOK {
		t.Fatalf("fixture canvas ls status = %d, want 200 body=%s", canvasW.Code, canvasW.Body.String())
	}
	if !strings.Contains(canvasW.Body.String(), `"canvas":"phase12-canvas"`) || !strings.Contains(canvasW.Body.String(), `"text":"fixture-canvas-mutating"`) {
		t.Fatalf("fixture canvas ls body missing layer2 canvas side effect: %s", canvasW.Body.String())
	}

	sessionsReq := httptest.NewRequest(http.MethodGet, "/admin/api/session", nil)
	sessionsW := httptest.NewRecorder()
	h.ServeHTTP(sessionsW, sessionsReq)
	if sessionsW.Code != http.StatusOK {
		t.Fatalf("fixture session ls status = %d, want 200 body=%s", sessionsW.Code, sessionsW.Body.String())
	}
	var sessionsBody map[string]any
	if err := json.Unmarshal(sessionsW.Body.Bytes(), &sessionsBody); err != nil {
		t.Fatalf("decode fixture session ls body error = %v body=%s", err, sessionsW.Body.String())
	}
	sessionItems, _ := sessionsBody["sessions"].([]any)
	sessionID := ""
	for _, item := range sessionItems {
		sessionMap, _ := item.(map[string]any)
		if sessionMap == nil {
			continue
		}
		if kind, _ := sessionMap["kind"].(string); kind != "lesson" {
			continue
		}
		if target, _ := sessionMap["target"].(string); target != "phase12-session" {
			continue
		}
		sessionID, _ = sessionMap["id"].(string)
		break
	}
	if strings.TrimSpace(sessionID) == "" {
		t.Fatalf("fixture session ls missing layer2 session side effect: %s", sessionsW.Body.String())
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/admin/api/session/members?session_id="+url.QueryEscape(sessionID), nil)
	sessionW := httptest.NewRecorder()
	h.ServeHTTP(sessionW, sessionReq)
	if sessionW.Code != http.StatusOK {
		t.Fatalf("fixture session members status = %d, want 200 body=%s", sessionW.Code, sessionW.Body.String())
	}
	if !strings.Contains(sessionW.Body.String(), `"session_id":"`+sessionID+`"`) || !strings.Contains(sessionW.Body.String(), `"identity_id":"fixture-session-member"`) {
		t.Fatalf("fixture session members body missing session side effect: %s", sessionW.Body.String())
	}

	simReq := httptest.NewRequest(http.MethodGet, "/admin/api/sim/ui?device_id=sim-fixture", nil)
	simW := httptest.NewRecorder()
	h.ServeHTTP(simW, simReq)
	if simW.Code != http.StatusNotFound {
		t.Fatalf("fixture sim ui status after cleanup = %d, want 404 body=%s", simW.Code, simW.Body.String())
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

	h := NewHandler(control, runtime, replSvc, nil, nil, nil, devices, config.Config{MDNSName: "HomeServer"}, nil)

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

	h := NewHandler(control, runtime, replSvc, aiSvc, nil, nil, devices, config.Config{MDNSName: "HomeServer"}, nil)

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

func TestStorePutTTLValidation(t *testing.T) {
	h := testHandler(t)

	badTTLReq := httptest.NewRequest(http.MethodPost, "/admin/api/store/put", strings.NewReader(url.Values{
		"namespace": {"ns"},
		"key":       {"k"},
		"value":     {"v"},
		"ttl":       {"invalid"},
	}.Encode()))
	badTTLReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	badTTLW := httptest.NewRecorder()
	h.ServeHTTP(badTTLW, badTTLReq)
	if badTTLW.Code != http.StatusBadRequest {
		t.Fatalf("invalid ttl status = %d, want 400 body=%s", badTTLW.Code, badTTLW.Body.String())
	}

	zeroTTLReq := httptest.NewRequest(http.MethodPost, "/admin/api/store/put", strings.NewReader(url.Values{
		"namespace": {"ns"},
		"key":       {"k"},
		"value":     {"v"},
		"ttl":       {"0s"},
	}.Encode()))
	zeroTTLReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	zeroTTLW := httptest.NewRecorder()
	h.ServeHTTP(zeroTTLW, zeroTTLReq)
	if zeroTTLW.Code != http.StatusBadRequest {
		t.Fatalf("zero ttl status = %d, want 400 body=%s", zeroTTLW.Code, zeroTTLW.Body.String())
	}

	goodTTLReq := httptest.NewRequest(http.MethodPost, "/admin/api/store/put", strings.NewReader(url.Values{
		"namespace": {"ns"},
		"key":       {"k"},
		"value":     {"v"},
		"ttl":       {"1m"},
	}.Encode()))
	goodTTLReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	goodTTLW := httptest.NewRecorder()
	h.ServeHTTP(goodTTLW, goodTTLReq)
	if goodTTLW.Code != http.StatusOK {
		t.Fatalf("valid ttl status = %d, want 200 body=%s", goodTTLW.Code, goodTTLW.Body.String())
	}
	if !strings.Contains(goodTTLW.Body.String(), "expires_at") {
		t.Fatalf("valid ttl response missing expires_at: %s", goodTTLW.Body.String())
	}
}

func TestStoreBindAndBusTailLimitValidation(t *testing.T) {
	h := testHandler(t)

	putReq := httptest.NewRequest(http.MethodPost, "/admin/api/store/put", strings.NewReader(url.Values{
		"namespace": {"ns"},
		"key":       {"k"},
		"value":     {"v"},
	}.Encode()))
	putReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	putW := httptest.NewRecorder()
	h.ServeHTTP(putW, putReq)
	if putW.Code != http.StatusOK {
		t.Fatalf("store put status = %d, want 200 body=%s", putW.Code, putW.Body.String())
	}

	badBindReq := httptest.NewRequest(http.MethodPost, "/admin/api/store/bind", strings.NewReader(url.Values{
		"namespace": {"ns"},
		"key":       {"k"},
		"to":        {"badbinding"},
	}.Encode()))
	badBindReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	badBindW := httptest.NewRecorder()
	h.ServeHTTP(badBindW, badBindReq)
	if badBindW.Code != http.StatusBadRequest {
		t.Fatalf("bad bind status = %d, want 400 body=%s", badBindW.Code, badBindW.Body.String())
	}

	goodBindReq := httptest.NewRequest(http.MethodPost, "/admin/api/store/bind", strings.NewReader(url.Values{
		"namespace": {"ns"},
		"key":       {"k"},
		"to":        {"device-1:chat"},
	}.Encode()))
	goodBindReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	goodBindW := httptest.NewRecorder()
	h.ServeHTTP(goodBindW, goodBindReq)
	if goodBindW.Code != http.StatusOK {
		t.Fatalf("good bind status = %d, want 200 body=%s", goodBindW.Code, goodBindW.Body.String())
	}

	badLimitReq := httptest.NewRequest(http.MethodGet, "/admin/api/bus?limit=zero", nil)
	badLimitW := httptest.NewRecorder()
	h.ServeHTTP(badLimitW, badLimitReq)
	if badLimitW.Code != http.StatusBadRequest {
		t.Fatalf("bad bus limit status = %d, want 400 body=%s", badLimitW.Code, badLimitW.Body.String())
	}
}

func TestCohortEndpointsCRUDAndMembers(t *testing.T) {
	h := testHandler(t)

	placementReq := httptest.NewRequest(http.MethodPost, "/admin/api/devices/placement", strings.NewReader(url.Values{
		"device_id": {"d1"},
		"zone":      {"kitchen"},
		"roles":     {"screen"},
	}.Encode()))
	placementReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	placementW := httptest.NewRecorder()
	h.ServeHTTP(placementW, placementReq)
	if placementW.Code != http.StatusOK {
		t.Fatalf("placement status = %d, want 200 body=%s", placementW.Code, placementW.Body.String())
	}

	upsertReq := httptest.NewRequest(http.MethodPost, "/admin/api/cohort/upsert", strings.NewReader(url.Values{
		"name":      {"Family-Screens"},
		"selectors": {"zone:kitchen,role:screen"},
	}.Encode()))
	upsertReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	upsertW := httptest.NewRecorder()
	h.ServeHTTP(upsertW, upsertReq)
	if upsertW.Code != http.StatusOK {
		t.Fatalf("cohort upsert status = %d, want 200 body=%s", upsertW.Code, upsertW.Body.String())
	}
	if !strings.Contains(upsertW.Body.String(), `"name":"family-screens"`) {
		t.Fatalf("cohort upsert body missing normalized name: %s", upsertW.Body.String())
	}
	if !strings.Contains(upsertW.Body.String(), `"members":["d1"]`) {
		t.Fatalf("cohort upsert body missing resolved members: %s", upsertW.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/api/cohort", nil)
	listW := httptest.NewRecorder()
	h.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("cohort list status = %d, want 200 body=%s", listW.Code, listW.Body.String())
	}
	if !strings.Contains(listW.Body.String(), `"family-screens"`) {
		t.Fatalf("cohort list body missing family-screens: %s", listW.Body.String())
	}

	showReq := httptest.NewRequest(http.MethodGet, "/admin/api/cohort?name=family-screens", nil)
	showW := httptest.NewRecorder()
	h.ServeHTTP(showW, showReq)
	if showW.Code != http.StatusOK {
		t.Fatalf("cohort show status = %d, want 200 body=%s", showW.Code, showW.Body.String())
	}
	if !strings.Contains(showW.Body.String(), `"members":["d1"]`) {
		t.Fatalf("cohort show body missing members: %s", showW.Body.String())
	}

	delReq := httptest.NewRequest(http.MethodPost, "/admin/api/cohort/del", strings.NewReader(url.Values{
		"name": {"family-screens"},
	}.Encode()))
	delReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	delW := httptest.NewRecorder()
	h.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusOK {
		t.Fatalf("cohort delete status = %d, want 200 body=%s", delW.Code, delW.Body.String())
	}
	if !strings.Contains(delW.Body.String(), `"deleted":true`) {
		t.Fatalf("cohort delete body missing deleted=true: %s", delW.Body.String())
	}
}

func TestUIViewEndpointsCRUD(t *testing.T) {
	h := testHandler(t)

	upsertReq := httptest.NewRequest(http.MethodPost, "/admin/api/ui/views/upsert", strings.NewReader(url.Values{
		"view_id":    {"Kitchen-Home"},
		"root_id":    {"root-main"},
		"descriptor": {`{"type":"stack","children":[]}`},
	}.Encode()))
	upsertReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	upsertW := httptest.NewRecorder()
	h.ServeHTTP(upsertW, upsertReq)
	if upsertW.Code != http.StatusOK {
		t.Fatalf("ui view upsert status = %d, want 200 body=%s", upsertW.Code, upsertW.Body.String())
	}
	if !strings.Contains(upsertW.Body.String(), `"view_id":"kitchen-home"`) {
		t.Fatalf("ui view upsert body missing normalized view_id: %s", upsertW.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/api/ui/views", nil)
	listW := httptest.NewRecorder()
	h.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("ui views list status = %d, want 200 body=%s", listW.Code, listW.Body.String())
	}
	if !strings.Contains(listW.Body.String(), `"kitchen-home"`) {
		t.Fatalf("ui views list missing kitchen-home: %s", listW.Body.String())
	}

	showReq := httptest.NewRequest(http.MethodGet, "/admin/api/ui/views?view_id=kitchen-home", nil)
	showW := httptest.NewRecorder()
	h.ServeHTTP(showW, showReq)
	if showW.Code != http.StatusOK {
		t.Fatalf("ui views show status = %d, want 200 body=%s", showW.Code, showW.Body.String())
	}
	if !strings.Contains(showW.Body.String(), `"root_id":"root-main"`) {
		t.Fatalf("ui views show missing root_id: %s", showW.Body.String())
	}

	delReq := httptest.NewRequest(http.MethodPost, "/admin/api/ui/views/del", strings.NewReader(url.Values{
		"view_id": {"kitchen-home"},
	}.Encode()))
	delReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	delW := httptest.NewRecorder()
	h.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusOK {
		t.Fatalf("ui views delete status = %d, want 200 body=%s", delW.Code, delW.Body.String())
	}
	if !strings.Contains(delW.Body.String(), `"deleted":true`) {
		t.Fatalf("ui views delete body missing deleted=true: %s", delW.Body.String())
	}

	placementReq := httptest.NewRequest(http.MethodPost, "/admin/api/devices/placement", strings.NewReader(url.Values{
		"device_id": {"d1"},
		"zone":      {"kitchen"},
		"roles":     {"screen"},
	}.Encode()))
	placementReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	placementW := httptest.NewRecorder()
	h.ServeHTTP(placementW, placementReq)
	if placementW.Code != http.StatusOK {
		t.Fatalf("placement status = %d, want 200 body=%s", placementW.Code, placementW.Body.String())
	}

	cohortReq := httptest.NewRequest(http.MethodPost, "/admin/api/cohort/upsert", strings.NewReader(url.Values{
		"name":      {"family-screens"},
		"selectors": {"zone:kitchen,role:screen"},
	}.Encode()))
	cohortReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	cohortW := httptest.NewRecorder()
	h.ServeHTTP(cohortW, cohortReq)
	if cohortW.Code != http.StatusOK {
		t.Fatalf("cohort upsert status = %d, want 200 body=%s", cohortW.Code, cohortW.Body.String())
	}

	pushReq := httptest.NewRequest(http.MethodPost, "/admin/api/ui/push", strings.NewReader(url.Values{
		"device_id":  {"d1"},
		"root_id":    {"root-main"},
		"descriptor": {`{"type":"stack"}`},
	}.Encode()))
	pushReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	pushW := httptest.NewRecorder()
	h.ServeHTTP(pushW, pushReq)
	if pushW.Code != http.StatusOK {
		t.Fatalf("ui push status = %d, want 200 body=%s", pushW.Code, pushW.Body.String())
	}
	if !strings.Contains(pushW.Body.String(), `"device_id":"d1"`) {
		t.Fatalf("ui push body missing device id: %s", pushW.Body.String())
	}

	patchReq := httptest.NewRequest(http.MethodPost, "/admin/api/ui/patch", strings.NewReader(url.Values{
		"device_id":    {"d1"},
		"component_id": {"banner"},
		"descriptor":   {`{"type":"text"}`},
	}.Encode()))
	patchReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	patchW := httptest.NewRecorder()
	h.ServeHTTP(patchW, patchReq)
	if patchW.Code != http.StatusOK {
		t.Fatalf("ui patch status = %d, want 200 body=%s", patchW.Code, patchW.Body.String())
	}
	if !strings.Contains(patchW.Body.String(), `"last_patch_component_id":"banner"`) {
		t.Fatalf("ui patch body missing patch component id: %s", patchW.Body.String())
	}

	transitionReq := httptest.NewRequest(http.MethodPost, "/admin/api/ui/transition", strings.NewReader(url.Values{
		"device_id":    {"d1"},
		"component_id": {"banner"},
		"transition":   {"fade"},
		"duration_ms":  {"150"},
	}.Encode()))
	transitionReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	transitionW := httptest.NewRecorder()
	h.ServeHTTP(transitionW, transitionReq)
	if transitionW.Code != http.StatusOK {
		t.Fatalf("ui transition status = %d, want 200 body=%s", transitionW.Code, transitionW.Body.String())
	}
	if !strings.Contains(transitionW.Body.String(), `"last_transition":"fade"`) {
		t.Fatalf("ui transition body missing transition name: %s", transitionW.Body.String())
	}

	broadcastReq := httptest.NewRequest(http.MethodPost, "/admin/api/ui/broadcast", strings.NewReader(url.Values{
		"cohort":     {"family-screens"},
		"descriptor": {`{"type":"banner"}`},
		"patch_id":   {"alert-banner"},
	}.Encode()))
	broadcastReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	broadcastW := httptest.NewRecorder()
	h.ServeHTTP(broadcastW, broadcastReq)
	if broadcastW.Code != http.StatusOK {
		t.Fatalf("ui broadcast status = %d, want 200 body=%s", broadcastW.Code, broadcastW.Body.String())
	}
	if !strings.Contains(broadcastW.Body.String(), `"members":["d1"]`) {
		t.Fatalf("ui broadcast body missing resolved members: %s", broadcastW.Body.String())
	}

	subscribeReq := httptest.NewRequest(http.MethodPost, "/admin/api/ui/subscribe", strings.NewReader(url.Values{
		"device_id": {"d1"},
		"to":        {"cohort:family-screens"},
	}.Encode()))
	subscribeReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	subscribeW := httptest.NewRecorder()
	h.ServeHTTP(subscribeW, subscribeReq)
	if subscribeW.Code != http.StatusOK {
		t.Fatalf("ui subscribe status = %d, want 200 body=%s", subscribeW.Code, subscribeW.Body.String())
	}
	if !strings.Contains(subscribeW.Body.String(), `"subscriptions":["cohort:family-screens"]`) {
		t.Fatalf("ui subscribe body missing subscription target: %s", subscribeW.Body.String())
	}

	snapshotReq := httptest.NewRequest(http.MethodGet, "/admin/api/ui/snapshot?device_id=d1", nil)
	snapshotW := httptest.NewRecorder()
	h.ServeHTTP(snapshotW, snapshotReq)
	if snapshotW.Code != http.StatusOK {
		t.Fatalf("ui snapshot status = %d, want 200 body=%s", snapshotW.Code, snapshotW.Body.String())
	}
	if !strings.Contains(snapshotW.Body.String(), `"device_id":"d1"`) {
		t.Fatalf("ui snapshot body missing device id: %s", snapshotW.Body.String())
	}
}

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
