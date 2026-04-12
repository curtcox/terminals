package scenario

import (
	"context"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestRuntimeHandleTrigger(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: AlertScenario{},
		Priority: PriorityCritical,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
	})

	name, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:   TriggerVoice,
		Intent: "red alert",
	})
	if err != nil {
		t.Fatalf("HandleTrigger() error = %v", err)
	}
	if name != "red_alert" {
		t.Fatalf("scenario name = %q, want red_alert", name)
	}

	events := broadcaster.Events()
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].Message != "RED ALERT" {
		t.Fatalf("event message = %q, want RED ALERT", events[0].Message)
	}
}

func TestRuntimeNoMatch(t *testing.T) {
	runtime := NewRuntime(NewEngine(), &Environment{})
	if _, err := runtime.HandleTrigger(context.Background(), Trigger{Intent: "unknown"}); err != ErrNoMatchingScenario {
		t.Fatalf("HandleTrigger() error = %v, want %v", err, ErrNoMatchingScenario)
	}
}

func TestRuntimeHandleVoiceTimer(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()
	scheduler := storage.NewMemoryScheduler()

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: &TimerReminderScenario{},
		Priority: PriorityNormal,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Scheduler: scheduler,
		Broadcast: broadcaster,
	})

	_, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"set a timer for 5 minutes",
		time.Date(2026, 4, 11, 21, 30, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("HandleVoiceText() error = %v", err)
	}

	scheduled := scheduler.List()
	if len(scheduled) != 1 {
		t.Fatalf("len(scheduled) = %d, want 1", len(scheduled))
	}
	events := broadcaster.Events()
	if len(events) != 1 || events[0].Message != "Timer set" {
		t.Fatalf("unexpected broadcast events: %+v", events)
	}
}

func TestRuntimeStopTrigger(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	engine := NewEngine()
	engine.Register(Registration{
		Scenario: PhotoFrameScenario{},
		Priority: PriorityLow,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices: devices,
	})

	name, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:   TriggerManual,
		Intent: "photo frame",
	})
	if err != nil {
		t.Fatalf("HandleTrigger() error = %v", err)
	}
	if name != "photo_frame" {
		t.Fatalf("name = %q, want photo_frame", name)
	}

	stopped, err := runtime.StopTrigger(context.Background(), Trigger{
		Kind:   TriggerManual,
		Intent: "photo frame",
	})
	if err != nil {
		t.Fatalf("StopTrigger() error = %v", err)
	}
	if stopped != "photo_frame" {
		t.Fatalf("stopped = %q, want photo_frame", stopped)
	}
	if _, ok := engine.Active("d1"); ok {
		t.Fatalf("expected no active scenario after stop")
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
		Devices: devices,
		IO:      routes,
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
}

func TestRuntimeProcessDueTimers(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	scheduler := storage.NewMemoryScheduler()
	broadcaster := ui.NewMemoryBroadcaster()
	runtime := NewRuntime(NewEngine(), &Environment{
		Devices:   devices,
		Scheduler: scheduler,
		Broadcast: broadcaster,
	})

	now := time.Date(2026, 4, 12, 9, 0, 0, 0, time.UTC)
	_ = scheduler.Schedule(context.Background(), "timer:d1:100", now.UnixMilli()-1000)
	_ = scheduler.Schedule(context.Background(), "timer:d1:200", now.UnixMilli()+60_000)

	processed, err := runtime.ProcessDueTimers(context.Background(), now)
	if err != nil {
		t.Fatalf("ProcessDueTimers() error = %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
	events := broadcaster.Events()
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if events[0].Message != "Timer complete" {
		t.Fatalf("message = %q, want Timer complete", events[0].Message)
	}
	if len(scheduler.Due(now.UnixMilli())) != 0 {
		t.Fatalf("expected due timers to be removed")
	}

	// Not-yet-due timer should still exist.
	laterDue := scheduler.Due(now.Add(2 * time.Minute).UnixMilli())
	if len(laterDue) != 1 || laterDue[0] != "timer:d1:200" {
		t.Fatalf("later due = %+v, want timer:d1:200", laterDue)
	}
}
