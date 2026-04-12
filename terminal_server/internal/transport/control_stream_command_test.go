package transport

import (
	"context"
	"strings"
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

func TestHandleMessageCommandManualTerminal(t *testing.T) {
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
			RequestID: "cmd-terminal-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(command manual terminal) error = %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("len(out) = %d, want 3", len(out))
	}
	if out[0].ScenarioStart != "terminal" {
		t.Fatalf("ScenarioStart = %q, want terminal", out[0].ScenarioStart)
	}
	if out[0].CommandAck != "cmd-terminal-1" {
		t.Fatalf("CommandAck = %q, want cmd-terminal-1", out[0].CommandAck)
	}
	if out[1].SetUI == nil {
		t.Fatalf("expected second response to include SetUI")
	}
	if out[1].SetUI.Type != "stack" {
		t.Fatalf("terminal SetUI root type = %q, want stack", out[1].SetUI.Type)
	}
	if out[2].TransitionUI == nil {
		t.Fatalf("expected third response to include TransitionUI")
	}
	if out[2].TransitionUI.Transition != "terminal_enter" {
		t.Fatalf("transition = %q, want terminal_enter", out[2].TransitionUI.Transition)
	}
	events := broadcaster.Events()
	if len(events) == 0 {
		t.Fatalf("expected terminal broadcast event")
	}
}

func TestHandleMessageInputTerminal(t *testing.T) {
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
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-terminal-input-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "terminal_input",
			Action:      "submit",
			Value:       "echo terminal-input-test",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input terminal) error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].UpdateUI == nil {
		t.Fatalf("expected UpdateUI response for terminal input")
	}
	if out[0].UpdateUI.ComponentID != "terminal_output" {
		t.Fatalf("update component_id = %q, want terminal_output", out[0].UpdateUI.ComponentID)
	}
	if !strings.Contains(out[0].UpdateUI.Node.Props["value"], "terminal-input-test") {
		t.Fatalf("terminal output patch did not include command marker: %+v", out[0].UpdateUI.Node.Props)
	}
}

func TestHandleMessageHeartbeatFlushesTerminalOutput(t *testing.T) {
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
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-terminal-heartbeat-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})

	initialOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "terminal_input",
			Action:      "submit",
			Value:       "sleep 1; printf '\\x68\\x62\\x2d\\x64\\x65\\x6c\\x61\\x79\\x65\\x64\\x2d\\x34\\x32\\n'",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input delayed command) error = %v", err)
	}
	if len(initialOut) != 1 || initialOut[0].UpdateUI == nil {
		t.Fatalf("expected immediate terminal update response, got %+v", initialOut)
	}
	initialValue := initialOut[0].UpdateUI.Node.Props["value"]
	if strings.Contains(initialValue, "hb-delayed-42") {
		t.Fatalf("unexpected delayed marker in initial response: %+v", initialOut[0].UpdateUI.Node.Props)
	}

	time.Sleep(1200 * time.Millisecond)

	heartbeatOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Heartbeat: &HeartbeatRequest{DeviceID: "device-1"},
	})
	if err != nil {
		t.Fatalf("HandleMessage(heartbeat) error = %v", err)
	}
	if len(heartbeatOut) != 1 || heartbeatOut[0].UpdateUI == nil {
		t.Fatalf("expected UpdateUI from heartbeat flush, got %+v", heartbeatOut)
	}
	if heartbeatOut[0].UpdateUI.ComponentID != "terminal_output" {
		t.Fatalf("update component_id = %q, want terminal_output", heartbeatOut[0].UpdateUI.ComponentID)
	}
	heartbeatValue := heartbeatOut[0].UpdateUI.Node.Props["value"]
	if !strings.Contains(heartbeatValue, "hb-delayed-42") {
		t.Fatalf("heartbeat output did not include delayed command result: %+v", heartbeatOut[0].UpdateUI.Node.Props)
	}
	if len(heartbeatValue) <= len(initialValue) {
		t.Fatalf("heartbeat output should extend terminal text (initial=%d heartbeat=%d)", len(initialValue), len(heartbeatValue))
	}
}

func TestHandleMessageInputTerminalChangeUsesDraftOnSubmit(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-terminal-input-start-change",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})

	changeOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "terminal_input",
			Action:      "change",
			Value:       "echo from-change-draft",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(change) error = %v", err)
	}
	if len(changeOut) != 0 {
		t.Fatalf("len(changeOut) = %d, want 0", len(changeOut))
	}

	submitOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "terminal_input",
			Action:      "submit",
			Value:       "",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(submit) error = %v", err)
	}
	if len(submitOut) != 1 {
		t.Fatalf("len(submitOut) = %d, want 1", len(submitOut))
	}
	if submitOut[0].UpdateUI == nil {
		t.Fatalf("expected UpdateUI response for terminal submit")
	}
	if !strings.Contains(submitOut[0].UpdateUI.Node.Props["value"], "from-change-draft") {
		t.Fatalf("terminal output patch did not include draft command marker: %+v", submitOut[0].UpdateUI.Node.Props)
	}
}

func TestHandleMessageInputTerminalInteractiveActions(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-terminal-input-start-interactive",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})

	firstOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "music_toggle",
			Action:      "toggle",
			Value:       "true",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(toggle) error = %v", err)
	}
	if len(firstOut) != 1 || firstOut[0].UpdateUI == nil {
		t.Fatalf("expected one UpdateUI response for toggle, got %+v", firstOut)
	}
	if !strings.Contains(firstOut[0].UpdateUI.Node.Props["value"], "[ui_action] music_toggle toggle = true") {
		t.Fatalf("missing toggle action in output: %+v", firstOut[0].UpdateUI.Node.Props)
	}

	secondOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "camera_source",
			Action:      "select",
			Value:       "front-door",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(select) error = %v", err)
	}
	if len(secondOut) != 1 || secondOut[0].UpdateUI == nil {
		t.Fatalf("expected one UpdateUI response for select, got %+v", secondOut)
	}
	output := secondOut[0].UpdateUI.Node.Props["value"]
	if !strings.Contains(output, "[ui_action] music_toggle toggle = true") {
		t.Fatalf("expected accumulated output to include prior toggle action: %+v", secondOut[0].UpdateUI.Node.Props)
	}
	if !strings.Contains(output, "[ui_action] camera_source select = front-door") {
		t.Fatalf("missing select action in output: %+v", secondOut[0].UpdateUI.Node.Props)
	}
}

func TestHandleMessageInputRoutesStartActionWhenScenarioIsActive(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-ui-route-photo-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "open_terminal_button",
			Action:      "start:terminal",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input start:terminal) error = %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("len(out) = %d, want 3", len(out))
	}
	if out[0].ScenarioStart != "terminal" {
		t.Fatalf("ScenarioStart = %q, want terminal", out[0].ScenarioStart)
	}
	if out[1].SetUI == nil {
		t.Fatalf("expected SetUI response after starting terminal")
	}
	if out[2].TransitionUI == nil || out[2].TransitionUI.Transition != "terminal_enter" {
		t.Fatalf("expected terminal_enter transition, got %+v", out[2].TransitionUI)
	}
	active, ok := runtime.Engine.Active("device-1")
	if !ok || active != "terminal" {
		t.Fatalf("active scenario = %q (ok=%t), want terminal", active, ok)
	}
}

func TestHandleMessageInputRoutesStopActiveAction(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-ui-route-terminal-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "stop_gesture",
			Action:      "stop_active",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input stop_active) error = %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	if out[0].ScenarioStop != "terminal" {
		t.Fatalf("ScenarioStop = %q, want terminal", out[0].ScenarioStop)
	}
	if out[1].TransitionUI == nil || out[1].TransitionUI.Transition != "terminal_exit" {
		t.Fatalf("expected terminal_exit transition, got %+v", out[1].TransitionUI)
	}
	if _, ok := runtime.Engine.Active("device-1"); ok {
		t.Fatalf("expected no active scenario after stop_active")
	}
}

func TestHandleMessageInputActionIgnoredWithoutActiveScenario(t *testing.T) {
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

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "open_terminal_button",
			Action:      "start:terminal",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input action without active scenario) error = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("len(out) = %d, want 0", len(out))
	}
}

func TestHandleMessageInputTapIgnoredWithActiveScenario(t *testing.T) {
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
			RequestID: "cmd-ui-route-photo-start-tap",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "generic_tap",
			Action:      "tap",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input tap) error = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("len(out) = %d, want 0", len(out))
	}
	active, ok := runtime.Engine.Active("device-1")
	if !ok || active != "photo_frame" {
		t.Fatalf("active scenario = %q (ok=%t), want photo_frame", active, ok)
	}
}

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
	if out[0].Data["registered_scenarios"] == "" {
		t.Fatalf("expected registered_scenarios in runtime_status")
	}
	if out[0].Data["pending_timers"] == "" {
		t.Fatalf("expected pending_timers in runtime_status")
	}
}

func TestHandleMessageSystemScenarioRegistry(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
		IO:      io.NewRouter(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-registry-1",
			Kind:      "system",
			Intent:    "scenario_registry",
		},
	})
	if err != nil {
		t.Fatalf("system scenario_registry error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].Data["red_alert"] == "" {
		t.Fatalf("expected red_alert in registry data")
	}
}

func TestHandleMessageSystemRunDueTimers(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	scheduler := storage.NewMemoryScheduler()
	broadcaster := ui.NewMemoryBroadcaster()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Scheduler: scheduler,
		Broadcast: broadcaster,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_ = scheduler.Schedule(context.Background(), "timer:device-1:1", control.now().Add(-1*time.Minute).UnixMilli())

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-run-due-1",
			Kind:      "system",
			Intent:    "run_due_timers",
		},
	})
	if err != nil {
		t.Fatalf("system run_due_timers error = %v", err)
	}
	if out[0].Data["processed"] != "1" {
		t.Fatalf("processed = %q, want 1", out[0].Data["processed"])
	}
	events := broadcaster.Events()
	if len(events) != 1 || events[0].Message != "Timer complete" {
		t.Fatalf("unexpected broadcast events: %+v", events)
	}
}

func TestHandleMessageSystemTransportMetrics(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
		IO:      io.NewRouter(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Heartbeat: &HeartbeatRequest{DeviceID: "device-1"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "metrics-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-metrics-1",
			Kind:      "system",
			Intent:    "transport_metrics",
		},
	})
	if err != nil {
		t.Fatalf("system transport_metrics error = %v", err)
	}
	if out[0].Data["register_received"] != "1" {
		t.Fatalf("register_received = %q, want 1", out[0].Data["register_received"])
	}
	if out[0].Data["heartbeat_received"] != "1" {
		t.Fatalf("heartbeat_received = %q, want 1", out[0].Data["heartbeat_received"])
	}
	if out[0].Data["command_received"] != "2" {
		t.Fatalf("command_received = %q, want 2", out[0].Data["command_received"])
	}
	if out[0].Data["dedupe_hits"] != "0" {
		t.Fatalf("dedupe_hits = %q, want 0", out[0].Data["dedupe_hits"])
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

func TestHandleMessageTransportMetricsIncludesDedupeHits(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
		IO:      io.NewRouter(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "dup-metrics-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "dup-metrics-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-metrics-dedupe",
			Kind:      "system",
			Intent:    "transport_metrics",
		},
	})
	if err != nil {
		t.Fatalf("transport_metrics query error = %v", err)
	}
	if out[0].Data["dedupe_hits"] != "1" {
		t.Fatalf("dedupe_hits = %q, want 1", out[0].Data["dedupe_hits"])
	}
}

func TestHandleMessageSystemHelp(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-help-1",
			Kind:      "system",
			Intent:    "system_help",
		},
	})
	if err != nil {
		t.Fatalf("system_help error = %v", err)
	}
	if out[0].Data["system_intents"] == "" || out[0].Data["command_kinds"] == "" {
		t.Fatalf("missing expected system_help fields: %+v", out[0].Data)
	}
	if out[0].Data["system_intents"] == "" || !contains(out[0].Data["system_intents"], "pending_timers") {
		t.Fatalf("system_help missing pending_timers intent: %+v", out[0].Data)
	}
	if !contains(out[0].Data["system_intents"], "recent_commands") {
		t.Fatalf("system_help missing recent_commands intent: %+v", out[0].Data)
	}
}

func TestHandleMessageSystemDeviceStatus(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
			DeviceType: "laptop",
			Platform:   "linux",
			Capabilities: map[string]string{
				"screen.width": "1920",
			},
		},
	})
	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-device-1",
			Kind:      "system",
			Intent:    "device_status device-1",
		},
	})
	if err != nil {
		t.Fatalf("device_status error = %v", err)
	}
	if out[0].Data["device_id"] != "device-1" {
		t.Fatalf("device_id = %q, want device-1", out[0].Data["device_id"])
	}
	if out[0].Data["cap.screen.width"] != "1920" {
		t.Fatalf("cap.screen.width = %q, want 1920", out[0].Data["cap.screen.width"])
	}
}

func TestHandleMessageSystemPendingTimers(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	scheduler := storage.NewMemoryScheduler()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Scheduler: scheduler,
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_ = scheduler.Schedule(context.Background(), "timer:device-1:100", control.now().Add(5*time.Minute).UnixMilli())

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-pending-1",
			Kind:      "system",
			Intent:    "pending_timers",
		},
	})
	if err != nil {
		t.Fatalf("pending_timers error = %v", err)
	}
	if out[0].Data["timer:device-1:100"] != "scheduled" {
		t.Fatalf("pending timer missing from response: %+v", out[0].Data)
	}
}

func TestHandleMessageSystemRecentCommands(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "audit-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "audit-2",
			DeviceID:  "device-1",
			Action:    "stop",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-recent-1",
			Kind:      "system",
			Intent:    "recent_commands",
		},
	})
	if err != nil {
		t.Fatalf("recent_commands error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if len(out[0].Data) < 2 {
		t.Fatalf("expected at least 2 recent command events, got %d", len(out[0].Data))
	}
	foundAudit1 := false
	for _, v := range out[0].Data {
		if strings.Contains(v, "audit-1") {
			foundAudit1 = true
			break
		}
	}
	if !foundAudit1 {
		t.Fatalf("expected recent_commands payload to include audit-1 event")
	}
}

func TestHandleMessageSystemReconcileLivenessDefault(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	base := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	stale := base.Add(-10 * time.Minute)
	control.now = func() time.Time { return stale }
	handler := NewStreamHandler(control)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Heartbeat: &HeartbeatRequest{DeviceID: "device-1"},
	})
	control.now = func() time.Time { return base }

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-reconcile-1",
			Kind:      "system",
			Intent:    "reconcile_liveness",
		},
	})
	if err != nil {
		t.Fatalf("reconcile_liveness default error = %v", err)
	}
	if out[0].Data["updated"] != "1" {
		t.Fatalf("updated = %q, want 1", out[0].Data["updated"])
	}
	if out[0].Data["timeout_seconds"] != "120" {
		t.Fatalf("timeout_seconds = %q, want 120", out[0].Data["timeout_seconds"])
	}
}

func TestHandleMessageSystemReconcileLivenessCustom(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	base := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	stale := base.Add(-45 * time.Second)
	control.now = func() time.Time { return stale }
	handler := NewStreamHandler(control)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Heartbeat: &HeartbeatRequest{DeviceID: "device-1"},
	})
	control.now = func() time.Time { return base }

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-reconcile-2",
			Kind:      "system",
			Intent:    "reconcile_liveness 30",
		},
	})
	if err != nil {
		t.Fatalf("reconcile_liveness custom error = %v", err)
	}
	if out[0].Data["updated"] != "1" {
		t.Fatalf("updated = %q, want 1", out[0].Data["updated"])
	}
	if out[0].Data["timeout_seconds"] != "30" {
		t.Fatalf("timeout_seconds = %q, want 30", out[0].Data["timeout_seconds"])
	}
}

func TestHandleMessageSystemReconcileLivenessInvalidSeconds(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-reconcile-bad",
			Kind:      "system",
			Intent:    "reconcile_liveness nope",
		},
	})
	if err == nil {
		t.Fatalf("expected error for invalid reconcile_liveness seconds")
	}
}

func TestHandleMessageRecentCommandsEviction(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
		IO:      io.NewRouter(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	handler.recentLimit = 2

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "evict-1", DeviceID: "device-1", Kind: "manual", Intent: "photo frame"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "evict-2", DeviceID: "device-1", Action: "stop", Kind: "manual", Intent: "photo frame"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "evict-3", Kind: "system", Intent: "server_status"},
	})

	if len(handler.recent) != 2 {
		t.Fatalf("len(recent) = %d, want 2", len(handler.recent))
	}
	if handler.recent[0].RequestID != "evict-2" || handler.recent[1].RequestID != "evict-3" {
		t.Fatalf("unexpected recent eviction order: %+v", handler.recent)
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

func TestHandleMessageRejectsMissingManualIntent(t *testing.T) {
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
			RequestID: "bad-intent",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "   ",
		},
	})
	if err != ErrMissingCommandIntent {
		t.Fatalf("error = %v, want %v", err, ErrMissingCommandIntent)
	}
}

func TestHandleMessageRejectsMissingVoiceText(t *testing.T) {
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
			RequestID: "bad-text",
			DeviceID:  "device-1",
			Kind:      "voice",
			Text:      "",
		},
	})
	if err != ErrMissingCommandText {
		t.Fatalf("error = %v, want %v", err, ErrMissingCommandText)
	}
}

func TestHandleMessageRejectsMissingCommandDeviceID(t *testing.T) {
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
			RequestID: "bad-device",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	if err != ErrMissingCommandDeviceID {
		t.Fatalf("error = %v, want %v", err, ErrMissingCommandDeviceID)
	}
}

func contains(s, needle string) bool {
	return strings.Contains(s, needle)
}
