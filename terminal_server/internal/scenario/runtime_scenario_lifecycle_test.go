package scenario

import (
	"context"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestRuntimeStartScenarioTargetsSpecificDevices(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
	})

	started, err := runtime.StartScenario(context.Background(), "terminal", []string{"d2"})
	if err != nil {
		t.Fatalf("StartScenario() error = %v", err)
	}
	if started != "terminal" {
		t.Fatalf("started = %q, want terminal", started)
	}
	if active, ok := engine.Active("d2"); !ok || active != "terminal" {
		t.Fatalf("active(d2) = (%q, %v), want (terminal, true)", active, ok)
	}
	if _, ok := engine.Active("d1"); ok {
		t.Fatalf("d1 should remain inactive when d2 is explicitly targeted")
	}
}

func TestRuntimeStopScenarioStopsTargetedDevice(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
	})

	if _, err := runtime.StartScenario(context.Background(), "terminal", []string{"d1"}); err != nil {
		t.Fatalf("StartScenario() error = %v", err)
	}
	stopped, err := runtime.StopScenario(context.Background(), "terminal", []string{"d1"})
	if err != nil {
		t.Fatalf("StopScenario() error = %v", err)
	}
	if stopped != "terminal" {
		t.Fatalf("stopped = %q, want terminal", stopped)
	}
	if _, ok := engine.Active("d1"); ok {
		t.Fatalf("d1 should be inactive after StopScenario")
	}
}

func TestRuntimeStatusData(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	routes := iorouter.NewRouter()
	engine := NewEngine()
	engine.Register(Registration{
		Scenario: PhotoFrameScenario{},
		Priority: PriorityLow,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		IO:        routes,
		Scheduler: storage.NewMemoryScheduler(),
	})

	_, _ = runtime.HandleTrigger(context.Background(), Trigger{
		Kind:   TriggerManual,
		Intent: "photo frame",
	})
	_ = routes.Connect("d1", "d2", "audio")

	status := runtime.StatusData()
	if status["active_scenarios"] != "1" {
		t.Fatalf("active_scenarios = %q, want 1", status["active_scenarios"])
	}
	if status["active_routes"] != "1" {
		t.Fatalf("active_routes = %q, want 1", status["active_routes"])
	}
	if status["registered_scenarios"] != "1" {
		t.Fatalf("registered_scenarios = %q, want 1", status["registered_scenarios"])
	}
	if status["pending_timers"] != "0" {
		t.Fatalf("pending_timers = %q, want 0", status["pending_timers"])
	}
}

func TestRuntimeRecoverActivations(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	store := storage.NewMemoryStore()
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: &TerminalScenario{},
		Priority: PriorityNormal,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Storage:   store,
		Broadcast: broadcaster,
	})

	if _, err := runtime.StartScenario(context.Background(), "terminal", []string{"d1"}); err != nil {
		t.Fatalf("StartScenario() error = %v", err)
	}

	recoveredEngine := NewEngine()
	recoveredEngine.Register(Registration{
		Scenario: &TerminalScenario{},
		Priority: PriorityNormal,
	})
	recovered := NewRuntime(recoveredEngine, &Environment{
		Devices:   devices,
		Storage:   store,
		Broadcast: broadcaster,
	})
	if err := recovered.RecoverActivations(context.Background()); err != nil {
		t.Fatalf("RecoverActivations() error = %v", err)
	}
	if got, ok := recovered.Engine.Active("d1"); !ok || got != "terminal" {
		t.Fatalf("Active(d1) = (%q,%v), want (terminal,true)", got, ok)
	}
}
