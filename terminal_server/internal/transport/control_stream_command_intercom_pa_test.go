package transport

import (
	"context"
	"strings"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

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
	if len(out) != 3 {
		t.Fatalf("len(out) = %d, want 3", len(out))
	}
	if out[0].ScenarioStart != "photo_frame" {
		t.Fatalf("ScenarioStart = %q, want photo_frame", out[0].ScenarioStart)
	}
	if out[0].CommandAck != "cmd-2" {
		t.Fatalf("CommandAck = %q, want cmd-2", out[0].CommandAck)
	}
	if out[1].SetUI == nil || out[1].SetUI.Props["id"] != "photo_frame_root" {
		t.Fatalf("expected photo frame SetUI, got %+v", out[1].SetUI)
	}
	if out[2].TransitionUI == nil || out[2].TransitionUI.Transition != "photo_frame_enter" {
		t.Fatalf("expected photo_frame_enter transition, got %+v", out[2].TransitionUI)
	}
}

func TestHandleMessageCommandManualWithDeviceIDsRelaysPhotoFrameUI(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "device-2", DeviceName: "Hall"})
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

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-photo-targeted",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
			Arguments: map[string]string{
				"device_ids": "device-1,device-2",
			},
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(targeted photo frame) error = %v", err)
	}

	var localSetUI bool
	var relayedSetUI bool
	var relayedTransition bool
	for _, msg := range out {
		if msg.SetUI != nil && msg.RelayToDeviceID == "" && msg.SetUI.Props["id"] == "photo_frame_root" {
			localSetUI = true
		}
		if msg.SetUI != nil && msg.RelayToDeviceID == "device-2" && msg.SetUI.Props["id"] == "photo_frame_root" {
			relayedSetUI = true
		}
		if msg.TransitionUI != nil && msg.RelayToDeviceID == "device-2" && msg.TransitionUI.Transition == "photo_frame_enter" {
			relayedTransition = true
		}
	}
	if !localSetUI {
		t.Fatalf("expected local photo frame SetUI in command responses: %+v", out)
	}
	if !relayedSetUI {
		t.Fatalf("expected relayed photo frame SetUI for device-2: %+v", out)
	}
	if !relayedTransition {
		t.Fatalf("expected relayed photo_frame_enter transition for device-2: %+v", out)
	}

	if active, ok := runtime.Engine.Active("device-1"); !ok || active != "photo_frame" {
		t.Fatalf("device-1 active = %q (ok=%t), want photo_frame", active, ok)
	}
	if active, ok := runtime.Engine.Active("device-2"); !ok || active != "photo_frame" {
		t.Fatalf("device-2 active = %q (ok=%t), want photo_frame", active, ok)
	}
}

func TestHandleMessageCommandIntercomEmitsRouteStreams(t *testing.T) {
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

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-intercom-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "intercom",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(command intercom) error = %v", err)
	}
	if len(out) != 9 {
		t.Fatalf("len(out) = %d, want 9", len(out))
	}
	if out[0].ScenarioStart != "intercom" {
		t.Fatalf("ScenarioStart = %q, want intercom", out[0].ScenarioStart)
	}
	startSeen := map[string]map[string]bool{}
	routeSeen := map[string]map[string]bool{}
	for _, msg := range out[1:] {
		if msg.StartStream != nil {
			if msg.StartStream.Kind != "audio" {
				t.Fatalf("start stream kind = %q, want audio", msg.StartStream.Kind)
			}
			relay := msg.RelayToDeviceID
			if relay == "" {
				relay = "local"
			}
			streamID := msg.StartStream.StreamID
			if startSeen[streamID] == nil {
				startSeen[streamID] = map[string]bool{}
			}
			startSeen[streamID][relay] = true
		}
		if msg.RouteStream != nil {
			relay := msg.RelayToDeviceID
			if relay == "" {
				relay = "local"
			}
			streamID := msg.RouteStream.StreamID
			if routeSeen[streamID] == nil {
				routeSeen[streamID] = map[string]bool{}
			}
			routeSeen[streamID][relay] = true
		}
	}
	expectedStreams := []string{
		"route:device-1|device-2|audio",
		"route:device-2|device-1|audio",
	}
	for _, streamID := range expectedStreams {
		if !startSeen[streamID]["local"] || !startSeen[streamID]["device-2"] {
			t.Fatalf("missing local/relayed start_stream delivery for %s: %+v", streamID, startSeen[streamID])
		}
		if !routeSeen[streamID]["local"] || !routeSeen[streamID]["device-2"] {
			t.Fatalf("missing local/relayed route_stream delivery for %s: %+v", streamID, routeSeen[streamID])
		}
	}

	// Starting intercom again should not emit duplicate route stream messages.
	out, err = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-intercom-2",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "intercom",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(command intercom duplicate) error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out duplicate) = %d, want 1", len(out))
	}
	if out[0].ScenarioStart != "intercom" {
		t.Fatalf("duplicate ScenarioStart = %q, want intercom", out[0].ScenarioStart)
	}
}

func TestHandleMessageCommandIntercomStopEmitsStopStream(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	router := io.NewRouter()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        router,
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
		Register: &RegisterRequest{DeviceID: "device-2", DeviceName: "Hall Display"},
	})

	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-intercom-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "intercom",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(command intercom start) error = %v", err)
	}
	if router.RouteCount() != 2 {
		t.Fatalf("route count after start = %d, want 2", router.RouteCount())
	}

	stopOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-intercom-stop",
			DeviceID:  "device-1",
			Action:    CommandActionStop,
			Kind:      "manual",
			Intent:    "intercom",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(command intercom stop) error = %v", err)
	}
	if len(stopOut) != 5 {
		t.Fatalf("len(stopOut) = %d, want 5", len(stopOut))
	}
	if stopOut[0].ScenarioStop != "intercom" {
		t.Fatalf("ScenarioStop = %q, want intercom", stopOut[0].ScenarioStop)
	}
	stopSeen := map[string]map[string]bool{}
	for _, msg := range stopOut[1:] {
		if msg.StopStream == nil {
			t.Fatalf("expected stop_stream response after intercom stop, got %+v", msg)
		}
		relay := msg.RelayToDeviceID
		if relay == "" {
			relay = "local"
		}
		streamID := msg.StopStream.StreamID
		if stopSeen[streamID] == nil {
			stopSeen[streamID] = map[string]bool{}
		}
		stopSeen[streamID][relay] = true
	}
	expectedStopStreams := []string{
		"route:device-1|device-2|audio",
		"route:device-2|device-1|audio",
	}
	for _, streamID := range expectedStopStreams {
		if !stopSeen[streamID]["local"] || !stopSeen[streamID]["device-2"] {
			t.Fatalf("missing local/relayed stop_stream delivery for %s: %+v", streamID, stopSeen[streamID])
		}
	}
	if router.RouteCount() != 0 {
		t.Fatalf("route count after stop = %d, want 0", router.RouteCount())
	}
}

func TestHandleMessageCommandPASystemRelaysReceiverNotifications(t *testing.T) {
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
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-3", DeviceName: "Office Display"},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-pa-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "pa_system",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(command pa_system) error = %v", err)
	}

	seenScenarioStart := false
	seenRelay := map[string]bool{}
	seenOverlayRelay := map[string]bool{}
	seenReceiveEnter := map[string]bool{}
	seenSourceEnter := false
	for _, msg := range out {
		if msg.ScenarioStart == "pa_system" {
			seenScenarioStart = true
		}
		if msg.Notification == "PA from device-1" {
			seenRelay[msg.RelayToDeviceID] = true
		}
		if msg.UpdateUI != nil &&
			(msg.UpdateUI.ComponentID == ui.GlobalOverlayComponentID || strings.HasSuffix(msg.UpdateUI.ComponentID, "/"+ui.GlobalOverlayComponentID)) &&
			msg.UpdateUI.Node.Type == "overlay" &&
			(msg.UpdateUI.Node.Props["id"] == ui.GlobalOverlayComponentID || strings.HasSuffix(msg.UpdateUI.Node.Props["id"], "/"+ui.GlobalOverlayComponentID)) {
			seenOverlayRelay[msg.RelayToDeviceID] = true
		}
		if msg.TransitionUI != nil {
			if msg.TransitionUI.Transition == "pa_source_enter" && msg.RelayToDeviceID == "" {
				seenSourceEnter = true
			}
			if msg.TransitionUI.Transition == "pa_receive_enter" {
				seenReceiveEnter[msg.RelayToDeviceID] = true
			}
		}
	}
	if !seenScenarioStart {
		t.Fatalf("expected pa_system scenario start in command output")
	}
	if !seenRelay["device-2"] || !seenRelay["device-3"] {
		t.Fatalf("expected PA receiver notifications relayed to device-2 and device-3, got %+v", seenRelay)
	}
	if !seenOverlayRelay["device-2"] || !seenOverlayRelay["device-3"] {
		t.Fatalf("expected PA overlay updates relayed to device-2 and device-3, got %+v", seenOverlayRelay)
	}
	if !seenSourceEnter {
		t.Fatalf("expected local pa_source_enter transition")
	}
	if !seenReceiveEnter["device-2"] || !seenReceiveEnter["device-3"] {
		t.Fatalf("expected pa_receive_enter transitions relayed to device-2 and device-3, got %+v", seenReceiveEnter)
	}
}

func TestHandleMessageCommandPASystemStopClearsReceiverOverlays(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	broadcaster := ui.NewMemoryBroadcaster()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	router := io.NewRouter()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        router,
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
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-3", DeviceName: "Office Display"},
	})

	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-pa-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "pa_system",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(command pa_system start) error = %v", err)
	}
	if router.RouteCount() != 2 {
		t.Fatalf("route count after PA start = %d, want 2", router.RouteCount())
	}

	stopOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-pa-stop",
			DeviceID:  "device-1",
			Action:    CommandActionStop,
			Kind:      "manual",
			Intent:    "pa_system",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(command pa_system stop) error = %v", err)
	}
	if router.RouteCount() != 0 {
		t.Fatalf("route count after PA stop = %d, want 0", router.RouteCount())
	}

	clearsByTarget := map[string]bool{}
	seenSourceExit := false
	seenReceiveExit := map[string]bool{}
	for _, msg := range stopOut {
		if msg.TransitionUI != nil {
			if msg.TransitionUI.Transition == "pa_source_exit" && msg.RelayToDeviceID == "" {
				seenSourceExit = true
			}
			if msg.TransitionUI.Transition == "pa_receive_exit" {
				seenReceiveExit[msg.RelayToDeviceID] = true
			}
		}
		if msg.UpdateUI == nil {
			continue
		}
		if msg.UpdateUI.ComponentID != ui.GlobalOverlayComponentID {
			continue
		}
		if msg.UpdateUI.Node.Type != "overlay" || msg.UpdateUI.Node.Props["id"] != ui.GlobalOverlayComponentID {
			t.Fatalf("unexpected overlay clear patch payload: %+v", msg.UpdateUI.Node)
		}
		if len(msg.UpdateUI.Node.Children) != 0 {
			t.Fatalf("expected empty overlay clear patch, got children=%d", len(msg.UpdateUI.Node.Children))
		}
		clearsByTarget[msg.RelayToDeviceID] = true
	}
	if !clearsByTarget["device-2"] || !clearsByTarget["device-3"] {
		t.Fatalf("expected PA overlay clear relays to device-2 and device-3, got %+v", clearsByTarget)
	}
	if !seenSourceExit {
		t.Fatalf("expected local pa_source_exit transition")
	}
	if !seenReceiveExit["device-2"] || !seenReceiveExit["device-3"] {
		t.Fatalf("expected pa_receive_exit transitions relayed to device-2 and device-3, got %+v", seenReceiveExit)
	}
}
