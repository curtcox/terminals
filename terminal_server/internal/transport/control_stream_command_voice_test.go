package transport

import (
	"context"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/recording"
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
			RequestID: "cmd-1",
			DeviceID:  "device-1",
			Kind:      "voice",
			Text:      "red alert",
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
	if out[0].CommandAck != "cmd-1" {
		t.Fatalf("CommandAck = %q, want cmd-1", out[0].CommandAck)
	}

	events := broadcaster.Events()
	if len(events) == 0 {
		t.Fatalf("expected broadcast event")
	}
}

func TestHandleMessageCommandVoiceTimerRelaysCountdownUI(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	uiHost := ui.NewMemoryHost()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		Scheduler: storage.NewMemoryScheduler(),
		UI:        uiHost,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Tablet",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-timer",
			DeviceID:  "device-1",
			Kind:      "voice",
			Text:      "set a timer for 1 minutes pasta",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(timer command) error = %v", err)
	}
	var foundSetUI bool
	for _, msg := range out {
		if msg.SetUI != nil && findNodePropValue(msg.SetUI, "remaining", "value") == "01:00" {
			foundSetUI = true
		}
	}
	if !foundSetUI {
		t.Fatalf("expected countdown SetUI in responses: %+v", out)
	}
}

func TestHandleMessageIntercomStartStopUpdatesRecordingManager(t *testing.T) {
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
	recorder := recording.NewMemoryManager()
	handler.SetRecordingManager(recorder)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-2", DeviceName: "Hall"},
	})

	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-intercom-start-recording",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "intercom",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(intercom start) error = %v", err)
	}
	active := recorder.Active()
	if len(active) != 2 {
		t.Fatalf("len(recorder.Active()) after start = %d, want 2", len(active))
	}

	_, err = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-intercom-stop-recording",
			DeviceID:  "device-1",
			Action:    CommandActionStop,
			Kind:      "manual",
			Intent:    "intercom",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(intercom stop) error = %v", err)
	}
	if len(recorder.Active()) != 0 {
		t.Fatalf("len(recorder.Active()) after stop = %d, want 0", len(recorder.Active()))
	}
}

func TestHandleMessageCommandVoiceStopPAAliases(t *testing.T) {
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
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-2", DeviceName: "Hall Display"},
	})

	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-pa-start",
			DeviceID:  "device-1",
			Kind:      "voice",
			Text:      "pa mode",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(command voice pa start) error = %v", err)
	}

	stopOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-pa-stop",
			DeviceID:  "device-1",
			Action:    CommandActionStop,
			Kind:      "voice",
			Text:      "end pa",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(command voice pa stop) error = %v", err)
	}

	sawStop := false
	for _, msg := range stopOut {
		if msg.ScenarioStop == "pa_system" {
			sawStop = true
			break
		}
	}
	if !sawStop {
		t.Fatalf("expected pa_system scenario stop in voice stop output: %+v", stopOut)
	}
}
