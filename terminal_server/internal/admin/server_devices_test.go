package admin

import (
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

	h := NewHandler(control, runtime, nil, nil, nil, nil, devices, config.Config{MDNSName: "HomeServer"}, nil, nil)

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

	h := NewHandler(control, runtime, nil, nil, nil, nil, devices, config.Config{MDNSName: "HomeServer"}, nil, nil)

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
