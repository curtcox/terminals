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

func TestHandleMessageTerminalSessionLifecycle(t *testing.T) {
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

	startOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-terminal-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})
	if err != nil {
		t.Fatalf("start command error = %v", err)
	}
	if len(startOut) != 3 {
		t.Fatalf("start len(out) = %d, want 3", len(startOut))
	}
	activeSessions := handler.terminals.List()
	if len(activeSessions) != 1 {
		t.Fatalf("terminal session count after start = %d, want 1", len(activeSessions))
	}
	firstSessionID := activeSessions[0].ID

	// Starting terminal again should reuse existing session for the device.
	_, err = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-terminal-start-again",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})
	if err != nil {
		t.Fatalf("second start command error = %v", err)
	}
	activeSessions = handler.terminals.List()
	if len(activeSessions) != 1 {
		t.Fatalf("terminal session count after second start = %d, want 1", len(activeSessions))
	}
	if activeSessions[0].ID != firstSessionID {
		t.Fatalf("session id changed on second start: got %q want %q", activeSessions[0].ID, firstSessionID)
	}

	stopOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-terminal-stop",
			DeviceID:  "device-1",
			Action:    "stop",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})
	if err != nil {
		t.Fatalf("stop command error = %v", err)
	}
	if len(stopOut) != 2 {
		t.Fatalf("stop len(out) = %d, want 2", len(stopOut))
	}
	if stopOut[0].ScenarioStop != "terminal" {
		t.Fatalf("ScenarioStop = %q, want terminal", stopOut[0].ScenarioStop)
	}
	if stopOut[1].TransitionUI == nil {
		t.Fatalf("expected stop response to include TransitionUI")
	}
	if stopOut[1].TransitionUI.Transition != "terminal_exit" {
		t.Fatalf("stop transition = %q, want terminal_exit", stopOut[1].TransitionUI.Transition)
	}
	if len(handler.terminals.List()) != 0 {
		t.Fatalf("terminal sessions should be empty after stop")
	}

	_, err = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-terminal-restart",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})
	if err != nil {
		t.Fatalf("restart command error = %v", err)
	}
	activeSessions = handler.terminals.List()
	if len(activeSessions) != 1 {
		t.Fatalf("terminal session count after restart = %d, want 1", len(activeSessions))
	}
	if activeSessions[0].ID == firstSessionID {
		t.Fatalf("expected new session id after stop/restart, still %q", firstSessionID)
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
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	if out[0].ScenarioStop != "photo_frame" {
		t.Fatalf("ScenarioStop = %q, want photo_frame", out[0].ScenarioStop)
	}
	if out[0].CommandAck != "cmd-3-stop" {
		t.Fatalf("CommandAck = %q, want cmd-3-stop", out[0].CommandAck)
	}
	if out[1].TransitionUI == nil || out[1].TransitionUI.Transition != "photo_frame_exit" {
		t.Fatalf("expected photo_frame_exit transition, got %+v", out[1].TransitionUI)
	}
}

func TestHandleMessageCommandStopRedAlertRestoresPhotoFrameUI(t *testing.T) {
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
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-photo-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	if err != nil {
		t.Fatalf("photo frame start error = %v", err)
	}
	_, err = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-alert-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "red alert",
		},
	})
	if err != nil {
		t.Fatalf("red alert start error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-alert-stop",
			DeviceID:  "device-1",
			Action:    "stop",
			Kind:      "manual",
			Intent:    "red alert",
		},
	})
	if err != nil {
		t.Fatalf("red alert stop error = %v", err)
	}
	if len(out) < 3 {
		t.Fatalf("expected resumed photo frame UI after red alert stop, got %+v", out)
	}
	if out[0].ScenarioStop != "red_alert" {
		t.Fatalf("ScenarioStop = %q, want red_alert", out[0].ScenarioStop)
	}
	foundPhotoUI := false
	foundPhotoTransition := false
	for _, msg := range out {
		if msg.SetUI != nil && msg.SetUI.Props["id"] == "photo_frame_root" {
			foundPhotoUI = true
		}
		if msg.TransitionUI != nil && msg.TransitionUI.Transition == "photo_frame_enter" {
			foundPhotoTransition = true
		}
	}
	if !foundPhotoUI {
		t.Fatalf("expected resumed photo frame SetUI after red alert stop: %+v", out)
	}
	if !foundPhotoTransition {
		t.Fatalf("expected photo_frame_enter transition on resume: %+v", out)
	}
}

func TestHandleMessageCommandStopRedAlertRestoresPhotoFrameUIForTargetedDevices(t *testing.T) {
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
		Register: &RegisterRequest{DeviceID: "device-2", DeviceName: "Hall Tablet"},
	})

	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-photo-start-targeted",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
			Arguments: map[string]string{
				"device_ids": "device-1,device-2",
			},
		},
	})
	if err != nil {
		t.Fatalf("photo frame targeted start error = %v", err)
	}
	_, err = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-alert-start-targeted",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "red alert",
			Arguments: map[string]string{
				"device_ids": "device-1,device-2",
			},
		},
	})
	if err != nil {
		t.Fatalf("red alert targeted start error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-alert-stop-targeted",
			DeviceID:  "device-1",
			Action:    "stop",
			Kind:      "manual",
			Intent:    "red alert",
			Arguments: map[string]string{
				"device_ids": "device-1,device-2",
			},
		},
	})
	if err != nil {
		t.Fatalf("red alert targeted stop error = %v", err)
	}

	foundLocalPhotoUI := false
	foundLocalPhotoTransition := false
	foundPeerPhotoUI := false
	foundPeerPhotoTransition := false
	for _, msg := range out {
		if msg.SetUI != nil && msg.SetUI.Props["id"] == "photo_frame_root" {
			if msg.RelayToDeviceID == "" {
				foundLocalPhotoUI = true
			}
			if msg.RelayToDeviceID == "device-2" {
				foundPeerPhotoUI = true
			}
		}
		if msg.TransitionUI != nil && msg.TransitionUI.Transition == "photo_frame_enter" {
			if msg.RelayToDeviceID == "" {
				foundLocalPhotoTransition = true
			}
			if msg.RelayToDeviceID == "device-2" {
				foundPeerPhotoTransition = true
			}
		}
	}
	if !foundLocalPhotoUI || !foundLocalPhotoTransition {
		t.Fatalf("expected local resumed photo frame UI+transition; out=%+v", out)
	}
	if !foundPeerPhotoUI || !foundPeerPhotoTransition {
		t.Fatalf("expected relayed resumed photo frame UI+transition for device-2; out=%+v", out)
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
