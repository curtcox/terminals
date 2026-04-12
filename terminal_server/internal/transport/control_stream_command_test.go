package transport

import (
	"context"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestHandleMessageCommandVoice(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	broadcaster := ui.NewMemoryBroadcaster()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		AI:        nil,
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: broadcaster,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			DeviceID: "device-1",
			Kind:     "voice",
			Text:     "red alert",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(command voice) error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].ScenarioStart != "red_alert" {
		t.Fatalf("ScenarioStart = %q, want red_alert", out[0].ScenarioStart)
	}

	events := broadcaster.Events()
	if len(events) == 0 {
		t.Fatalf("expected broadcast event")
	}
}

func TestHandleMessageCommandManual(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	broadcaster := ui.NewMemoryBroadcaster()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: broadcaster,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			DeviceID: "device-1",
			Kind:     "manual",
			Intent:   "photo frame",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(command manual) error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].ScenarioStart != "photo_frame" {
		t.Fatalf("ScenarioStart = %q, want photo_frame", out[0].ScenarioStart)
	}
}

func TestHandleMessageCommandStop(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	broadcaster := ui.NewMemoryBroadcaster()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: broadcaster,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	})
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			DeviceID: "device-1",
			Kind:     "manual",
			Intent:   "photo frame",
		},
	})
	if err != nil {
		t.Fatalf("start command error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			DeviceID: "device-1",
			Action:   "stop",
			Kind:     "manual",
			Intent:   "photo frame",
		},
	})
	if err != nil {
		t.Fatalf("stop command error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].ScenarioStop != "photo_frame" {
		t.Fatalf("ScenarioStop = %q, want photo_frame", out[0].ScenarioStop)
	}
}
