package scenario

import (
	"context"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestRuntimeTargetDevicesUsesPlacementZoneScope(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen mic"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Kitchen display"})
	engine := NewEngine()
	engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
	placement := &fakePlacement{
		findRefs: []DeviceRef{{DeviceID: "d2"}},
	}
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: ui.NewMemoryBroadcaster(),
		Placement: placement,
	})

	matched, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "terminal",
		Arguments: map[string]string{
			"zone": "kitchen",
		},
	})
	if err != nil {
		t.Fatalf("HandleTrigger(terminal+zone) error = %v", err)
	}
	if matched != "terminal" {
		t.Fatalf("matched scenario = %q, want terminal", matched)
	}

	if active, ok := engine.Active("d2"); !ok || active != "terminal" {
		t.Fatalf("active(d2) = (%q, %v), want (terminal, true)", active, ok)
	}
	if _, ok := engine.Active("d1"); ok {
		t.Fatalf("d1 should remain inactive when placement scopes to d2")
	}
	if placement.findCalls != 1 {
		t.Fatalf("placement.Find calls = %d, want 1", placement.findCalls)
	}
	if placement.lastQuery.Scope.Zone != "kitchen" {
		t.Fatalf("placement query zone = %q, want kitchen", placement.lastQuery.Scope.Zone)
	}
}

func TestRuntimeTargetDevicesUsesPlacementNearest(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Origin"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Nearest screen"})
	engine := NewEngine()
	engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
	placement := &fakePlacement{
		nearestRef: DeviceRef{DeviceID: "d3"},
	}
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: ui.NewMemoryBroadcaster(),
		Placement: placement,
	})

	_, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "terminal",
		Arguments: map[string]string{
			"nearest":            "true",
			"nearest_capability": "screen",
		},
	})
	if err != nil {
		t.Fatalf("HandleTrigger(terminal+nearest) error = %v", err)
	}

	if active, ok := engine.Active("d3"); !ok || active != "terminal" {
		t.Fatalf("active(d3) = (%q, %v), want (terminal, true)", active, ok)
	}
	if placement.nearestCalls != 1 {
		t.Fatalf("placement.NearestWith calls = %d, want 1", placement.nearestCalls)
	}
	if placement.lastNearestCap != "screen" {
		t.Fatalf("placement nearest capability = %q, want screen", placement.lastNearestCap)
	}
}

func TestRuntimeTargetDevicesTrimsExplicitDeviceID(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Device 1"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Device 2"})

	engine := NewEngine()
	engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	matched, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "terminal",
		Arguments: map[string]string{
			"device_id": " d2 ",
		},
	})
	if err != nil {
		t.Fatalf("HandleTrigger(terminal+device_id) error = %v", err)
	}
	if matched != "terminal" {
		t.Fatalf("matched scenario = %q, want terminal", matched)
	}

	if active, ok := engine.Active("d2"); !ok || active != "terminal" {
		t.Fatalf("active(d2) = (%q, %v), want (terminal, true)", active, ok)
	}
	if _, ok := engine.Active("d1"); ok {
		t.Fatalf("d1 should remain inactive when explicit device_id targets d2")
	}
}

func TestRuntimeHandleTriggerTargetsExplicitDeviceIDs(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: PhotoFrameScenario{},
		Priority: PriorityLow,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices: devices,
	})

	name, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "photo frame",
		Arguments: map[string]string{
			"device_ids": "d1,d3",
		},
	})
	if err != nil {
		t.Fatalf("HandleTrigger() error = %v", err)
	}
	if name != "photo_frame" {
		t.Fatalf("name = %q, want photo_frame", name)
	}

	if active, ok := engine.Active("d1"); !ok || active != "photo_frame" {
		t.Fatalf("d1 active scenario = %q (ok=%t), want photo_frame", active, ok)
	}
	if _, ok := engine.Active("d2"); ok {
		t.Fatalf("expected d2 to remain inactive")
	}
	if active, ok := engine.Active("d3"); !ok || active != "photo_frame" {
		t.Fatalf("d3 active scenario = %q (ok=%t), want photo_frame", active, ok)
	}
}

func TestRuntimeIntercomTargetsExplicitDeviceIDs(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
	router := iorouter.NewRouter()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &IntercomScenario{}, Priority: PriorityHigh})
	runtime := NewRuntime(engine, &Environment{
		Devices: devices,
		IO:      router,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "intercom",
		Arguments: map[string]string{
			"device_ids": "d1,d3",
		},
	}); err != nil {
		t.Fatalf("HandleTrigger(intercom targeted) error = %v", err)
	}

	if got := router.RouteCount(); got != 2 {
		t.Fatalf("route count = %d, want 2", got)
	}
	routes := router.Routes()
	for _, route := range routes {
		if route.SourceID == "d2" || route.TargetID == "d2" {
			t.Fatalf("unexpected route involving d2: %+v", route)
		}
	}
}

func TestRuntimePATargetsExplicitDeviceIDs(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
	router := iorouter.NewRouter()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &PASystemScenario{}, Priority: PriorityHigh})
	runtime := NewRuntime(engine, &Environment{
		Devices: devices,
		IO:      router,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "pa_system",
		Arguments: map[string]string{
			"device_ids": "d1,d3",
		},
	}); err != nil {
		t.Fatalf("HandleTrigger(pa targeted) error = %v", err)
	}

	if got := router.RouteCount(); got != 1 {
		t.Fatalf("route count = %d, want 1", got)
	}
	routes := router.Routes()
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1", len(routes))
	}
	if routes[0].SourceID != "d1" || routes[0].TargetID != "d3" || routes[0].StreamKind != "pa_audio" {
		t.Fatalf("route = %+v, want d1->d3 pa_audio", routes[0])
	}
}
