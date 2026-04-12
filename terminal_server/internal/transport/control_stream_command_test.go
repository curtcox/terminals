package transport

import (
	"context"
	"testing"
	"time"

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
			RequestID: "cmd-2",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
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
	if out[0].CommandAck != "cmd-2" {
		t.Fatalf("CommandAck = %q, want cmd-2", out[0].CommandAck)
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
			RequestID: "cmd-3-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	if err != nil {
		t.Fatalf("start command error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-3-stop",
			DeviceID:  "device-1",
			Action:    "stop",
			Kind:      "manual",
			Intent:    "photo frame",
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
	if out[0].CommandAck != "cmd-3-stop" {
		t.Fatalf("CommandAck = %q, want cmd-3-stop", out[0].CommandAck)
	}
}

func TestHandleMessageCommandDedupesRequestID(t *testing.T) {
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

	first, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "dup-1",
			DeviceID:  "device-1",
			Kind:      "voice",
			Text:      "red alert",
		},
	})
	if err != nil {
		t.Fatalf("first HandleMessage() error = %v", err)
	}
	second, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "dup-1",
			DeviceID:  "device-1",
			Kind:      "voice",
			Text:      "red alert",
		},
	})
	if err != nil {
		t.Fatalf("second HandleMessage() error = %v", err)
	}
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("unexpected response lengths")
	}
	if first[0].CommandAck != "dup-1" || second[0].CommandAck != "dup-1" {
		t.Fatalf("unexpected command ack values")
	}

	events := broadcaster.Events()
	if len(events) != 1 {
		t.Fatalf("expected exactly one broadcast event after duplicate request id, got %d", len(events))
	}
}

func TestHandleMessageCommandRejectsInvalidAction(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "bad-1",
			DeviceID:  "device-1",
			Action:    "pause",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	if err != ErrInvalidCommandAction {
		t.Fatalf("error = %v, want %v", err, ErrInvalidCommandAction)
	}
}

func TestHandleMessageSystemListDevices(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook", Platform: "linux"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-2", DeviceName: "Hall Tablet", Platform: "android"},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-1",
			Kind:      "system",
			Intent:    "list_devices",
		},
	})
	if err != nil {
		t.Fatalf("system list devices error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].CommandAck != "sys-1" {
		t.Fatalf("CommandAck = %q, want sys-1", out[0].CommandAck)
	}
	if len(out[0].Data) != 2 {
		t.Fatalf("len(Data) = %d, want 2", len(out[0].Data))
	}
}

func TestHandleMessageSystemActiveScenarios(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-2",
			Kind:      "system",
			Intent:    "active_scenarios",
		},
	})
	if err != nil {
		t.Fatalf("system active_scenarios error = %v", err)
	}
	if out[0].Data["device-1"] != "photo_frame" {
		t.Fatalf("active scenario = %q, want photo_frame", out[0].Data["device-1"])
	}
}

func TestHandleMessageSystemServerStatus(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	control.started = control.now().Add(-2 * time.Hour)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-status-1",
			Kind:      "system",
			Intent:    "server_status",
		},
	})
	if err != nil {
		t.Fatalf("system server_status error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].Data["server_id"] != "srv-1" {
		t.Fatalf("server_id = %q, want srv-1", out[0].Data["server_id"])
	}
	if out[0].Data["devices_total"] != "1" {
		t.Fatalf("devices_total = %q, want 1", out[0].Data["devices_total"])
	}
	if out[0].CommandAck != "sys-status-1" {
		t.Fatalf("CommandAck = %q, want sys-status-1", out[0].CommandAck)
	}
}

func TestHandleMessageSystemRuntimeStatus(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	routes := io.NewRouter()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
		IO:      routes,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-start-rs",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	_ = routes.Connect("device-1", "device-2", "audio")

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-runtime-1",
			Kind:      "system",
			Intent:    "runtime_status",
		},
	})
	if err != nil {
		t.Fatalf("system runtime_status error = %v", err)
	}
	if out[0].Data["active_scenarios"] != "1" {
		t.Fatalf("active_scenarios = %q, want 1", out[0].Data["active_scenarios"])
	}
	if out[0].Data["active_routes"] != "1" {
		t.Fatalf("active_routes = %q, want 1", out[0].Data["active_routes"])
	}
}

func TestHandleMessageSystemWithoutRuntime(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-no-runtime",
			Kind:      "system",
			Intent:    "server_status",
		},
	})
	if err != nil {
		t.Fatalf("expected server_status to work without runtime, err=%v", err)
	}
	if len(out) != 1 || out[0].Data["server_id"] != "srv-1" {
		t.Fatalf("unexpected server_status response: %+v", out)
	}
}

func TestHandleMessageDedupeEviction(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{Devices: devices})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	handler.seenLimit = 1

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "r1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "r2",
			DeviceID:  "device-1",
			Action:    "stop",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})

	handler.mu.Lock()
	_, hasR1 := handler.seen["r1"]
	_, hasR2 := handler.seen["r2"]
	handler.mu.Unlock()
	if hasR1 || !hasR2 {
		t.Fatalf("expected r1 evicted and r2 retained, got hasR1=%v hasR2=%v", hasR1, hasR2)
	}
}

func TestHandleMessageRejectsInvalidCommandKind(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{Devices: devices})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "bad-kind",
			DeviceID:  "device-1",
			Kind:      "remote",
			Intent:    "photo frame",
		},
	})
	if err != ErrInvalidCommandKind {
		t.Fatalf("error = %v, want %v", err, ErrInvalidCommandKind)
	}
}
