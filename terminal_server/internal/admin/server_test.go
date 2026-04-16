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
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
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

	h := NewHandler(control, runtime, nil, nil, devices, config.Config{MDNSName: "HomeServer"})

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

	h := NewHandler(control, runtime, nil, nil, devices, config.Config{MDNSName: "HomeServer"})

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
	return NewHandler(control, runtime, nil, nil, devices, cfg)
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

	h := NewHandler(control, runtime, appRuntime, func() {}, devices, config.Config{MDNSName: "HomeServer"})

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
