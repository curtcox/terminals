package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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

	h := NewHandler(control, runtime, devices, config.Config{MDNSName: "HomeServer"})

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

	h := NewHandler(control, runtime, devices, config.Config{MDNSName: "HomeServer"})

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

func testHandler(t *testing.T) http.Handler {
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

	return NewHandler(control, runtime, devices, config.Config{
		GRPCHost:      "0.0.0.0",
		GRPCPort:      50051,
		MDNSService:   "_terminals._tcp.local.",
		MDNSName:      "HomeServer",
		Version:       "1",
		AdminHTTPHost: "127.0.0.1",
		AdminHTTPPort: 50053,
	})
}
