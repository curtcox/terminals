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
	if out[0].UpdateUI.ComponentID != "terminal_output" && !strings.HasSuffix(out[0].UpdateUI.ComponentID, "/terminal_output") {
		t.Fatalf("update component_id = %q, want terminal_output (scoped or legacy)", out[0].UpdateUI.ComponentID)
	}
	if !strings.Contains(out[0].UpdateUI.Node.Props["value"], "terminal-input-test") {
		t.Fatalf("terminal output patch did not include command marker: %+v", out[0].UpdateUI.Node.Props)
	}
}

func TestHandleMessageInputTerminalKeyText(t *testing.T) {
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
			RequestID: "cmd-terminal-key-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID: "device-1",
			KeyText:  "echo key-forwarded-test\n",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input key text) error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].UpdateUI == nil {
		t.Fatalf("expected UpdateUI response for key text input")
	}
	if !strings.Contains(out[0].UpdateUI.Node.Props["value"], "key-forwarded-test") {
		t.Fatalf("key text output missing marker: %+v", out[0].UpdateUI.Node.Props)
	}
}

func TestHandleMessageInputTerminalWithShortReadWindow(t *testing.T) {
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
	handler.terminalReadDeadline = 40 * time.Millisecond
	handler.terminalReadInterval = 5 * time.Millisecond

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-terminal-short-window-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})

	start := time.Now()
	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "terminal_input",
			Action:      "submit",
			Value:       "echo short-read-window-test",
		},
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("HandleMessage(input terminal short read window) error = %v", err)
	}
	if len(out) != 1 || out[0].UpdateUI == nil {
		t.Fatalf("expected single UpdateUI response, got %+v", out)
	}
	if !strings.Contains(out[0].UpdateUI.Node.Props["value"], "short-read-window-test") {
		t.Fatalf("terminal output missing marker: %+v", out[0].UpdateUI.Node.Props)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("terminal input handling took %v, want <= 500ms", elapsed)
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

	var heartbeatOut []ServerMessage
	var heartbeatValue string
	for i := 0; i < 10; i++ {
		heartbeatOut, err = handler.HandleMessage(context.Background(), ClientMessage{
			Heartbeat: &HeartbeatRequest{DeviceID: "device-1"},
		})
		if err != nil {
			t.Fatalf("HandleMessage(heartbeat) error = %v", err)
		}
		if len(heartbeatOut) == 1 && heartbeatOut[0].UpdateUI != nil {
			heartbeatValue = heartbeatOut[0].UpdateUI.Node.Props["value"]
			if strings.Contains(heartbeatValue, "hb-delayed-42") {
				break
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	if len(heartbeatOut) != 1 || heartbeatOut[0].UpdateUI == nil {
		t.Fatalf("expected UpdateUI from heartbeat flush, got %+v", heartbeatOut)
	}
	if heartbeatOut[0].UpdateUI.ComponentID != "terminal_output" && !strings.HasSuffix(heartbeatOut[0].UpdateUI.ComponentID, "/terminal_output") {
		t.Fatalf("update component_id = %q, want terminal_output (scoped or legacy)", heartbeatOut[0].UpdateUI.ComponentID)
	}
	if heartbeatValue == "" {
		heartbeatValue = heartbeatOut[0].UpdateUI.Node.Props["value"]
	}
	if !strings.Contains(heartbeatValue, "hb-delayed-42") {
		t.Fatalf("heartbeat output did not include delayed command result: %+v", heartbeatOut[0].UpdateUI.Node.Props)
	}
	if len(heartbeatValue) <= len(initialValue) {
		t.Fatalf("heartbeat output should extend terminal text (initial=%d heartbeat=%d)", len(initialValue), len(heartbeatValue))
	}
}

func TestHandleMessageHeartbeatCoalescesTerminalOutputUpdates(t *testing.T) {
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
	handler.terminalUIInterval = 800 * time.Millisecond

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-terminal-heartbeat-coalesce-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})

	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "terminal_input",
			Action:      "submit",
			Value:       "sleep 0.3; printf '\\x63\\x6f\\x61\\x6c\\x65\\x73\\x63\\x65\\x2d\\x68\\x62\\n'",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input delayed command) error = %v", err)
	}

	time.Sleep(450 * time.Millisecond)

	firstHeartbeat, err := handler.HandleMessage(context.Background(), ClientMessage{
		Heartbeat: &HeartbeatRequest{DeviceID: "device-1"},
	})
	if err != nil {
		t.Fatalf("HandleMessage(first heartbeat) error = %v", err)
	}
	if len(firstHeartbeat) != 0 {
		t.Fatalf("expected first heartbeat output to be coalesced, got %+v", firstHeartbeat)
	}

	time.Sleep(450 * time.Millisecond)

	secondHeartbeat, err := handler.HandleMessage(context.Background(), ClientMessage{
		Heartbeat: &HeartbeatRequest{DeviceID: "device-1"},
	})
	if err != nil {
		t.Fatalf("HandleMessage(second heartbeat) error = %v", err)
	}
	if len(secondHeartbeat) != 1 || secondHeartbeat[0].UpdateUI == nil {
		t.Fatalf("expected coalesced UpdateUI on second heartbeat, got %+v", secondHeartbeat)
	}
	if !strings.Contains(secondHeartbeat[0].UpdateUI.Node.Props["value"], "coalesce-hb") {
		t.Fatalf("coalesced heartbeat output missing marker: %+v", secondHeartbeat[0].UpdateUI.Node.Props)
	}
}

func TestHandleMessageHeartbeatRotatesPhotoFrameAfterInterval(t *testing.T) {
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
	now := time.Date(2026, 4, 14, 18, 0, 0, 0, time.UTC)
	control.now = func() time.Time { return now }
	handler := NewStreamHandlerWithRuntime(control, runtime)
	handler.photoFrameInterval = 5 * time.Second

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	startOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-photo-rotate-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	if err != nil {
		t.Fatalf("photo frame start error = %v", err)
	}
	if len(startOut) < 2 || startOut[1].SetUI == nil {
		t.Fatalf("expected initial photo frame SetUI, got %+v", startOut)
	}
	firstURL := findNodePropValue(startOut[1].SetUI, "photo_frame_image", "url")
	if firstURL == "" {
		t.Fatalf("expected initial photo url in SetUI descriptor")
	}

	now = now.Add(3 * time.Second)
	firstHeartbeat, err := handler.HandleMessage(context.Background(), ClientMessage{
		Heartbeat: &HeartbeatRequest{DeviceID: "device-1"},
	})
	if err != nil {
		t.Fatalf("first heartbeat error = %v", err)
	}
	if len(firstHeartbeat) != 0 {
		t.Fatalf("expected no rotation before interval, got %+v", firstHeartbeat)
	}

	now = now.Add(3 * time.Second)
	secondHeartbeat, err := handler.HandleMessage(context.Background(), ClientMessage{
		Heartbeat: &HeartbeatRequest{DeviceID: "device-1"},
	})
	if err != nil {
		t.Fatalf("second heartbeat error = %v", err)
	}
	if len(secondHeartbeat) != 1 || secondHeartbeat[0].SetUI == nil {
		t.Fatalf("expected photo frame SetUI after interval, got %+v", secondHeartbeat)
	}
	secondURL := findNodePropValue(secondHeartbeat[0].SetUI, "photo_frame_image", "url")
	if secondURL == "" {
		t.Fatalf("expected rotated photo url in heartbeat SetUI descriptor")
	}
	if secondURL == firstURL {
		t.Fatalf("expected heartbeat rotation to advance photo url; url stayed %q", firstURL)
	}
}

func TestHandleMessageManualRefreshBypassesHeartbeatCoalescing(t *testing.T) {
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
	handler.terminalUIInterval = 10 * time.Second

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-terminal-refresh-force-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})

	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "terminal_input",
			Action:      "submit",
			Value:       "sleep 0.3; printf '\\x66\\x6f\\x72\\x63\\x65\\x2d\\x66\\x6c\\x75\\x73\\x68\\n'",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input delayed command) error = %v", err)
	}

	time.Sleep(450 * time.Millisecond)

	hbOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Heartbeat: &HeartbeatRequest{DeviceID: "device-1"},
	})
	if err != nil {
		t.Fatalf("HandleMessage(heartbeat) error = %v", err)
	}
	if len(hbOut) != 0 {
		t.Fatalf("expected heartbeat update to be throttled, got %+v", hbOut)
	}

	refreshOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "manual-refresh-bypass-coalesce",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    SystemIntentTerminalRefresh,
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(manual terminal_refresh) error = %v", err)
	}
	if len(refreshOut) != 2 || refreshOut[1].UpdateUI == nil {
		t.Fatalf("expected command ack + forced UpdateUI, got %+v", refreshOut)
	}
	if !strings.Contains(refreshOut[1].UpdateUI.Node.Props["value"], "force-flush") {
		t.Fatalf("manual refresh should force pending output flush: %+v", refreshOut[1].UpdateUI.Node.Props)
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
