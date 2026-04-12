package scenario

import (
	"context"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
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
