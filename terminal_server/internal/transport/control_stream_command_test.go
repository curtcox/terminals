package transport

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/recording"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func waitForRefreshMarker(
	t *testing.T,
	handler *StreamHandler,
	request ClientMessage,
	marker string,
	attempts int,
	delay time.Duration,
) []ServerMessage {
	t.Helper()
	for i := 0; i < attempts; i++ {
		out, err := handler.HandleMessage(context.Background(), request)
		if err != nil {
			t.Fatalf("refresh request error = %v", err)
		}
		if len(out) >= 2 && out[len(out)-1].UpdateUI != nil &&
			strings.Contains(out[len(out)-1].UpdateUI.Node.Props["value"], marker) {
			return out
		}
		time.Sleep(delay)
	}
	t.Fatalf("timed out waiting for refresh marker %q", marker)
	return nil
}

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

func TestHandleMessageWebRTCSignalProducesRelayToPeer(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		SessionDeviceID: "device-1",
		WebRTCSignal: &WebRTCSignalRequest{
			StreamID:   "route:device-1|device-2|audio",
			SignalType: "offer",
			Payload:    "{\"sdp\":\"v=0-offer\"}",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(webrtc_signal) error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].RelayToDeviceID != "device-2" {
		t.Fatalf("RelayToDeviceID = %q, want device-2", out[0].RelayToDeviceID)
	}
	if out[0].WebRTCSignal == nil {
		t.Fatalf("expected webrtc signal payload")
	}
	if out[0].WebRTCSignal.StreamID != "route:device-1|device-2|audio" {
		t.Fatalf("stream_id = %q, want route stream id", out[0].WebRTCSignal.StreamID)
	}
	if out[0].WebRTCSignal.SignalType != "offer" {
		t.Fatalf("signal_type = %q, want offer", out[0].WebRTCSignal.SignalType)
	}
	if out[0].WebRTCSignal.Payload != "{\"sdp\":\"v=0-offer\"}" {
		t.Fatalf("payload = %q, want offer payload", out[0].WebRTCSignal.Payload)
	}

	replyOut, replyErr := handler.HandleMessage(context.Background(), ClientMessage{
		SessionDeviceID: "device-2",
		WebRTCSignal: &WebRTCSignalRequest{
			StreamID:   "route:device-1|device-2|audio",
			SignalType: "answer",
			Payload:    "{\"sdp\":\"v=0-answer\"}",
		},
	})
	if replyErr != nil {
		t.Fatalf("HandleMessage(webrtc answer) error = %v", replyErr)
	}
	if len(replyOut) != 1 {
		t.Fatalf("len(replyOut) = %d, want 1", len(replyOut))
	}
	if replyOut[0].RelayToDeviceID != "device-1" {
		t.Fatalf("reply RelayToDeviceID = %q, want device-1", replyOut[0].RelayToDeviceID)
	}
}

type fakeWebRTCSignalEngine struct {
	responses   []WebRTCSignalEngineResponse
	err         error
	lastRequest WebRTCSignalEngineRequest
	removeCalls []string
}

func (f *fakeWebRTCSignalEngine) HandleSignal(_ context.Context, req WebRTCSignalEngineRequest) ([]WebRTCSignalEngineResponse, error) {
	f.lastRequest = req
	if f.err != nil {
		return nil, f.err
	}
	out := append([]WebRTCSignalEngineResponse(nil), f.responses...)
	return out, nil
}

func (f *fakeWebRTCSignalEngine) RemoveStream(streamID string) {
	f.removeCalls = append(f.removeCalls, streamID)
}

func TestHandleMessageWebRTCSignalUsesServerManagedEngine(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)
	engine := &fakeWebRTCSignalEngine{
		responses: []WebRTCSignalEngineResponse{
			{
				TargetDeviceID: "device-1",
				Signal: WebRTCSignalResponse{
					StreamID:   "route:device-1|device-2|audio",
					SignalType: "answer",
					Payload:    "{\"sdp\":\"v=0-answer\"}",
				},
			},
		},
	}
	handler.SetWebRTCSignalEngine(engine)
	handler.registerMediaStream(StartStreamResponse{
		StreamID:       "route:device-1|device-2|audio",
		Kind:           "audio",
		SourceDeviceID: "device-1",
		TargetDeviceID: "device-2",
		Metadata: map[string]string{
			"webrtc_mode": "server_managed",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		SessionDeviceID: "device-1",
		WebRTCSignal: &WebRTCSignalRequest{
			StreamID:   "route:device-1|device-2|audio",
			SignalType: "offer",
			Payload:    "{\"sdp\":\"v=0-offer\"}",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(server-managed webrtc signal) error = %v", err)
	}
	if engine.lastRequest.DeviceID != "device-1" {
		t.Fatalf("engine request device = %q, want device-1", engine.lastRequest.DeviceID)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].RelayToDeviceID != "" {
		t.Fatalf("RelayToDeviceID = %q, want empty (back to session device)", out[0].RelayToDeviceID)
	}
	if out[0].WebRTCSignal == nil || out[0].WebRTCSignal.SignalType != "answer" {
		t.Fatalf("expected answer signal from engine, got %+v", out[0].WebRTCSignal)
	}
}

func TestHandleMessageWebRTCSignalServerManagedFallsBackToRelayOnEngineError(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)
	engine := &fakeWebRTCSignalEngine{err: ErrInvalidClientMessage}
	handler.SetWebRTCSignalEngine(engine)
	handler.registerMediaStream(StartStreamResponse{
		StreamID:       "route:device-1|device-2|audio",
		Kind:           "audio",
		SourceDeviceID: "device-1",
		TargetDeviceID: "device-2",
		Metadata: map[string]string{
			"webrtc_mode": "server_managed",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		SessionDeviceID: "device-1",
		WebRTCSignal: &WebRTCSignalRequest{
			StreamID:   "route:device-1|device-2|audio",
			SignalType: "offer",
			Payload:    "{\"sdp\":\"v=0-offer\"}",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(server-managed fallback) error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].RelayToDeviceID != "device-2" {
		t.Fatalf("RelayToDeviceID = %q, want device-2 fallback relay", out[0].RelayToDeviceID)
	}
	if out[0].WebRTCSignal == nil || out[0].WebRTCSignal.SignalType != "offer" {
		t.Fatalf("expected fallback offer relay, got %+v", out[0].WebRTCSignal)
	}
}

func TestUnregisterMediaStreamRemovesServerManagedWebRTCStream(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)
	engine := &fakeWebRTCSignalEngine{}
	handler.SetWebRTCSignalEngine(engine)

	streamID := "route:device-1|device-2|audio"
	handler.registerMediaStream(StartStreamResponse{
		StreamID:       streamID,
		Kind:           "audio",
		SourceDeviceID: "device-1",
		TargetDeviceID: "device-2",
		Metadata: map[string]string{
			"webrtc_mode": "server_managed",
		},
	})
	handler.unregisterMediaStream(streamID)

	if len(engine.removeCalls) != 1 || engine.removeCalls[0] != streamID {
		t.Fatalf("removeCalls = %+v, want [%s]", engine.removeCalls, streamID)
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

func TestHandleMessageInputCornerOpenTogglesMenuOverlayAndClaim(t *testing.T) {
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

	openOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "act:device-1/__affordance.corner__",
			Action:      "open",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input corner open) error = %v", err)
	}
	if len(openOut) != 1 || openOut[0].UpdateUI == nil {
		t.Fatalf("expected one UpdateUI response for menu open, got %+v", openOut)
	}
	if openOut[0].UpdateUI.ComponentID != ui.GlobalOverlayComponentID {
		t.Fatalf("UpdateUI.ComponentID = %q, want %q", openOut[0].UpdateUI.ComponentID, ui.GlobalOverlayComponentID)
	}
	if openOut[0].UpdateUI.Node.Type != "overlay" {
		t.Fatalf("UpdateUI node type = %q, want overlay", openOut[0].UpdateUI.Node.Type)
	}
	if findNodeByID(&openOut[0].UpdateUI.Node, "act:menu-overlay:device-1/menu.privacy_toggle") == nil {
		t.Fatalf("expected privacy toggle button in menu overlay descriptor")
	}
	if findNodeByID(&openOut[0].UpdateUI.Node, "act:menu-overlay:device-1/menu.bug_report") == nil {
		t.Fatalf("expected bug report button in menu overlay descriptor")
	}

	claims := router.Claims().Snapshot("device-1")
	if !hasClaim(claims, "menu-overlay:device-1", "screen.overlay") {
		t.Fatalf("expected screen.overlay claim for menu overlay activation, got %+v", claims)
	}

	closeOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "act:device-1/__affordance.corner__",
			Action:      "corner.open",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input second corner open) error = %v", err)
	}
	if len(closeOut) != 1 || closeOut[0].UpdateUI == nil {
		t.Fatalf("expected one UpdateUI response for menu close, got %+v", closeOut)
	}
	if closeOut[0].UpdateUI.Node.Type != "overlay" {
		t.Fatalf("close UpdateUI node type = %q, want overlay", closeOut[0].UpdateUI.Node.Type)
	}
	if len(closeOut[0].UpdateUI.Node.Children) != 0 {
		t.Fatalf("expected menu overlay clear patch on close, got children=%d", len(closeOut[0].UpdateUI.Node.Children))
	}

	claims = router.Claims().Snapshot("device-1")
	if hasClaim(claims, "menu-overlay:device-1", "screen.overlay") {
		t.Fatalf("expected menu overlay claim released on close, got %+v", claims)
	}
}

func TestHandleMessageInputMenuCloseActionReleasesOverlayClaim(t *testing.T) {
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
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "act:device-1/__affordance.corner__",
			Action:      "open",
		},
	})

	closeOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "act:menu-overlay:device-1/menu.close",
			Action:      "close",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input menu close) error = %v", err)
	}
	if len(closeOut) != 1 || closeOut[0].UpdateUI == nil {
		t.Fatalf("expected one UpdateUI response for menu close action, got %+v", closeOut)
	}
	if len(closeOut[0].UpdateUI.Node.Children) != 0 {
		t.Fatalf("expected menu overlay clear patch, got children=%d", len(closeOut[0].UpdateUI.Node.Children))
	}

	claims := router.Claims().Snapshot("device-1")
	if hasClaim(claims, "menu-overlay:device-1", "screen.overlay") {
		t.Fatalf("expected menu overlay claim released by close action, got %+v", claims)
	}
}

type stubIdentityService struct {
	actorsByDevice map[string]Actor
}

func (s stubIdentityService) ResolveActor(deviceID string) Actor {
	if actor, ok := s.actorsByDevice[deviceID]; ok {
		return actor
	}
	return Actor{Kind: "device", ID: strings.TrimSpace(deviceID)}
}

type fixtureMenuPolicy struct{}

func (fixtureMenuPolicy) VisibleApps(actor Actor, apps []string) []string {
	if strings.EqualFold(strings.TrimSpace(actor.Kind), "anonymous") {
		out := make([]string, 0, len(apps))
		for _, app := range apps {
			if app == "photo_frame" {
				out = append(out, app)
			}
		}
		return out
	}
	return append([]string(nil), apps...)
}

type countingAudioPublisher struct {
	mu     sync.Mutex
	count  int
	device string
}

func (p *countingAudioPublisher) Publish(deviceID string, _ []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.count++
	p.device = deviceID
}

func (p *countingAudioPublisher) Snapshot() (int, string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.count, p.device
}

func TestHandleMessageInputMenuOverlayCompositionVariesByResolvedActor(t *testing.T) {
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
	handler.SetIdentityService(stubIdentityService{
		actorsByDevice: map[string]Actor{
			"device-anon":   {Kind: "anonymous", ID: "kiosk"},
			"device-person": {Kind: "person", ID: "alice"},
		},
	})
	handler.SetMenuAppPolicy(fixtureMenuPolicy{})

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-anon", DeviceName: "Kiosk"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-person", DeviceName: "Kitchen Tablet"},
	})

	anonOpenOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-anon",
			ComponentID: "act:device-anon/__affordance.corner__",
			Action:      "open",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input corner open anonymous) error = %v", err)
	}
	personOpenOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-person",
			ComponentID: "act:device-person/__affordance.corner__",
			Action:      "open",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input corner open person) error = %v", err)
	}

	if len(anonOpenOut) != 1 || anonOpenOut[0].UpdateUI == nil {
		t.Fatalf("expected one UpdateUI for anonymous menu open, got %+v", anonOpenOut)
	}
	if len(personOpenOut) != 1 || personOpenOut[0].UpdateUI == nil {
		t.Fatalf("expected one UpdateUI for person menu open, got %+v", personOpenOut)
	}

	anonApps := menuAppNamesFromDescriptor(&anonOpenOut[0].UpdateUI.Node)
	personApps := menuAppNamesFromDescriptor(&personOpenOut[0].UpdateUI.Node)

	if len(anonApps) == 0 {
		t.Fatalf("expected anonymous actor to see at least one app")
	}
	if _, ok := anonApps["terminal"]; ok {
		t.Fatalf("anonymous menu should hide terminal app, got %+v", anonApps)
	}
	if _, ok := personApps["terminal"]; !ok {
		t.Fatalf("person menu should include terminal app, got %+v", personApps)
	}
	if len(personApps) <= len(anonApps) {
		t.Fatalf("expected actor-variant menu app counts, anonymous=%d person=%d", len(anonApps), len(personApps))
	}
}

func TestHandleMessageInputMenuOverlayDefaultMixedPolicyBlocksMainPointerButKeepsAudioLive(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	pub := &countingAudioPublisher{}
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
	handler.SetDeviceAudioPublisher(pub)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "cmd-terminal", DeviceID: "device-1", Kind: "manual", Intent: "terminal"},
	}); err != nil {
		t.Fatalf("terminal start error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:device-1/__affordance.corner__", Action: "open"},
	}); err != nil {
		t.Fatalf("menu open error = %v", err)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		VoiceAudio: &VoiceAudioRequest{DeviceID: "device-1", Audio: []byte("live-audio"), SampleRate: 16000, IsFinal: false},
	}); err != nil {
		t.Fatalf("voice audio error = %v", err)
	}
	if count, deviceID := pub.Snapshot(); count != 1 || deviceID != "device-1" {
		t.Fatalf("audio publish snapshot = (%d,%q), want (1,device-1)", count, deviceID)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "stop_gesture", Action: "stop_active"},
	})
	if err != nil {
		t.Fatalf("stop_active input error = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no routed main-layer response while overlay is open under MIXED policy, got %+v", out)
	}
	if active, ok := runtime.Engine.Active("device-1"); !ok || active != "terminal" {
		t.Fatalf("active scenario after stop_active = (%q, %v), want terminal,true", active, ok)
	}

	out, err = handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:menu-overlay:device-1/menu.close", Action: "close"},
	})
	if err != nil {
		t.Fatalf("menu close error = %v", err)
	}
	if len(out) == 0 || out[0].UpdateUI == nil {
		t.Fatalf("expected overlay clear patch on close, got %+v", out)
	}

	out, err = handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "stop_gesture", Action: "stop_active"},
	})
	if err != nil {
		t.Fatalf("stop_active post-close error = %v", err)
	}
	if len(out) == 0 || out[0].ScenarioStop != "terminal" {
		t.Fatalf("expected stop_active routed after menu close, got %+v", out)
	}
}

func TestCapabilityDeltaWhileMenuOverlayOpenPreservesMainAndOverlayActivations(t *testing.T) {
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

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"screen.width":       "1920",
				"screen.height":      "1080",
				"screen.orientation": "landscape",
			},
		},
	}); err != nil {
		t.Fatalf("capability snapshot error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "cmd-terminal", DeviceID: "device-1", Kind: "manual", Intent: "terminal"},
	}); err != nil {
		t.Fatalf("terminal start error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:device-1/__affordance.corner__", Action: "open"},
	}); err != nil {
		t.Fatalf("menu open error = %v", err)
	}

	if active, ok := runtime.Engine.Active("device-1"); !ok || active != "terminal" {
		t.Fatalf("active scenario before orientation delta = (%q, %v), want terminal,true", active, ok)
	}
	if !hasClaim(router.Claims().Snapshot("device-1"), "menu-overlay:device-1", "screen.overlay") {
		t.Fatalf("expected overlay claim before orientation delta")
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Reason:     "display_geometry_change",
			Capabilities: map[string]string{
				"screen.width":       "1080",
				"screen.height":      "1920",
				"screen.orientation": "portrait",
			},
		},
	})
	if err != nil {
		t.Fatalf("capability delta error = %v", err)
	}
	if len(out) == 0 || out[0].CapabilityAck == nil {
		t.Fatalf("expected capability ack response, got %+v", out)
	}
	for _, msg := range out {
		if msg.UpdateUI != nil && len(msg.UpdateUI.Node.Children) == 0 {
			t.Fatalf("unexpected overlay clear patch on orientation delta: %+v", msg.UpdateUI)
		}
	}

	if active, ok := runtime.Engine.Active("device-1"); !ok || active != "terminal" {
		t.Fatalf("active scenario after orientation delta = (%q, %v), want terminal,true", active, ok)
	}
	if !hasClaim(router.Claims().Snapshot("device-1"), "menu-overlay:device-1", "screen.overlay") {
		t.Fatalf("expected overlay claim to remain active after orientation delta")
	}

	closeOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:menu-overlay:device-1/menu.close", Action: "close"},
	})
	if err != nil {
		t.Fatalf("menu close after orientation delta error = %v", err)
	}
	if len(closeOut) == 0 || closeOut[0].UpdateUI == nil {
		t.Fatalf("expected overlay close patch after orientation delta, got %+v", closeOut)
	}
	if len(closeOut[0].UpdateUI.Node.Children) != 0 {
		t.Fatalf("expected overlay close patch to clear children, got %+v", closeOut[0].UpdateUI.Node.Children)
	}
}

func TestLifecycleCapabilityDeltaWhileMenuOverlayOpenPreservesMainAndOverlayActivations(t *testing.T) {
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

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilitySnap: &CapabilitySnapshotRequest{
			DeviceID:   "device-1",
			Generation: 1,
			Capabilities: map[string]string{
				"screen.width":          "1920",
				"screen.height":         "1080",
				"screen.orientation":    "landscape",
				"edge.operators":        "monitor.lifecycle.foreground",
				"monitor.runtime_state": "foreground",
			},
		},
	}); err != nil {
		t.Fatalf("capability snapshot error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "cmd-terminal", DeviceID: "device-1", Kind: "manual", Intent: "terminal"},
	}); err != nil {
		t.Fatalf("terminal start error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:device-1/__affordance.corner__", Action: "open"},
	}); err != nil {
		t.Fatalf("menu open error = %v", err)
	}

	if active, ok := runtime.Engine.Active("device-1"); !ok || active != "terminal" {
		t.Fatalf("active scenario before lifecycle delta = (%q, %v), want terminal,true", active, ok)
	}
	if !hasClaim(router.Claims().Snapshot("device-1"), "menu-overlay:device-1", "screen.overlay") {
		t.Fatalf("expected overlay claim before lifecycle delta")
	}

	backgroundOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 2,
			Reason:     "app_lifecycle_change",
			Capabilities: map[string]string{
				"screen.width":          "1920",
				"screen.height":         "1080",
				"screen.orientation":    "landscape",
				"edge.operators":        "monitor.lifecycle.background",
				"monitor.runtime_state": "background",
			},
		},
	})
	if err != nil {
		t.Fatalf("background lifecycle capability delta error = %v", err)
	}
	if len(backgroundOut) == 0 || backgroundOut[0].CapabilityAck == nil {
		t.Fatalf("expected capability ack for background lifecycle delta, got %+v", backgroundOut)
	}
	for _, msg := range backgroundOut {
		if msg.UpdateUI != nil && len(msg.UpdateUI.Node.Children) == 0 {
			t.Fatalf("unexpected overlay clear patch on background lifecycle delta: %+v", msg.UpdateUI)
		}
	}
	if active, ok := runtime.Engine.Active("device-1"); !ok || active != "terminal" {
		t.Fatalf("active scenario after background lifecycle delta = (%q, %v), want terminal,true", active, ok)
	}
	if !hasClaim(router.Claims().Snapshot("device-1"), "menu-overlay:device-1", "screen.overlay") {
		t.Fatalf("expected overlay claim to remain active after background lifecycle delta")
	}

	foregroundOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		CapabilityDelta: &CapabilityDeltaRequest{
			DeviceID:   "device-1",
			Generation: 3,
			Reason:     "app_lifecycle_change",
			Capabilities: map[string]string{
				"screen.width":          "1920",
				"screen.height":         "1080",
				"screen.orientation":    "landscape",
				"edge.operators":        "monitor.lifecycle.foreground",
				"monitor.runtime_state": "foreground",
			},
		},
	})
	if err != nil {
		t.Fatalf("foreground lifecycle capability delta error = %v", err)
	}
	if len(foregroundOut) == 0 || foregroundOut[0].CapabilityAck == nil {
		t.Fatalf("expected capability ack for foreground lifecycle delta, got %+v", foregroundOut)
	}
	for _, msg := range foregroundOut {
		if msg.UpdateUI != nil && len(msg.UpdateUI.Node.Children) == 0 {
			t.Fatalf("unexpected overlay clear patch on foreground lifecycle delta: %+v", msg.UpdateUI)
		}
	}
	if active, ok := runtime.Engine.Active("device-1"); !ok || active != "terminal" {
		t.Fatalf("active scenario after foreground lifecycle delta = (%q, %v), want terminal,true", active, ok)
	}
	if !hasClaim(router.Claims().Snapshot("device-1"), "menu-overlay:device-1", "screen.overlay") {
		t.Fatalf("expected overlay claim to remain active after foreground lifecycle delta")
	}

	closeOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:menu-overlay:device-1/menu.close", Action: "close"},
	})
	if err != nil {
		t.Fatalf("menu close after lifecycle deltas error = %v", err)
	}
	if len(closeOut) == 0 || closeOut[0].UpdateUI == nil {
		t.Fatalf("expected overlay close patch after lifecycle deltas, got %+v", closeOut)
	}
	if len(closeOut[0].UpdateUI.Node.Children) != 0 {
		t.Fatalf("expected overlay close patch to clear children, got %+v", closeOut[0].UpdateUI.Node.Children)
	}
}

func TestHandleMessageInputMenuOverlayLivePolicyKeepsMainPointerActive(t *testing.T) {
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
	handler.SetMenuOverlayInputPolicyForTesting("LIVE", nil)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "cmd-terminal", DeviceID: "device-1", Kind: "manual", Intent: "terminal"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:device-1/__affordance.corner__", Action: "open"},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "stop_gesture", Action: "stop_active"},
	})
	if err != nil {
		t.Fatalf("stop_active input error = %v", err)
	}
	if len(out) == 0 || out[0].ScenarioStop != "terminal" {
		t.Fatalf("expected stop_active routed with LIVE policy while overlay open, got %+v", out)
	}
}

func TestHandleMessageInputMenuOverlayPausedPolicyTearsDownAndRestoresRoutes(t *testing.T) {
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
	handler.SetMenuOverlayInputPolicyForTesting("PAUSED", map[string]bool{
		"audio": false,
	})

	if err := router.Connect("device-1", "device-2", "audio"); err != nil {
		t.Fatalf("connect audio route error = %v", err)
	}
	if got := len(router.RoutesForDevice("device-1")); got != 1 {
		t.Fatalf("routes before menu open = %d, want 1", got)
	}

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "cmd-terminal", DeviceID: "device-1", Kind: "manual", Intent: "terminal"},
	})
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:device-1/__affordance.corner__", Action: "open"},
	}); err != nil {
		t.Fatalf("menu open error = %v", err)
	}
	if got := len(router.RoutesForDevice("device-1")); got != 0 {
		t.Fatalf("routes after menu open with PAUSED policy = %d, want 0", got)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "stop_gesture", Action: "stop_active"},
	})
	if err != nil {
		t.Fatalf("stop_active while paused error = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected main pointer action blocked while PAUSED overlay open, got %+v", out)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{DeviceID: "device-1", ComponentID: "act:menu-overlay:device-1/menu.close", Action: "close"},
	}); err != nil {
		t.Fatalf("menu close error = %v", err)
	}
	if got := len(router.RoutesForDevice("device-1")); got != 1 {
		t.Fatalf("routes after menu close with PAUSED policy = %d, want 1", got)
	}
}

func menuAppNamesFromDescriptor(node *ui.Descriptor) map[string]struct{} {
	out := map[string]struct{}{}
	if node == nil {
		return out
	}
	if id := node.Props["id"]; strings.Contains(id, "/menu.app.") {
		parts := strings.SplitN(id, "/menu.app.", 2)
		if len(parts) == 2 {
			out[parts[1]] = struct{}{}
		}
	}
	for i := range node.Children {
		for name := range menuAppNamesFromDescriptor(&node.Children[i]) {
			out[name] = struct{}{}
		}
	}
	return out
}

func TestHandleInputActionMapTurnoverDropsPriorMainActivationScopedIDs(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	initialUI := ui.New("stack", map[string]string{
		"id": "act:main-a/root",
	}, ui.New("button", map[string]string{
		"id":     "act:main-a/__affordance.corner__",
		"label":  "Menu",
		"action": "corner.open",
	}))
	if _, err := handler.prepareOutboundUI("device-1", ServerMessage{SetUI: &initialUI}); err != nil {
		t.Fatalf("prepareOutboundUI(initial) error = %v", err)
	}

	swappedUI := ui.New("stack", map[string]string{
		"id": "act:main-b/root",
	}, ui.New("button", map[string]string{
		"id":     "act:main-b/__affordance.corner__",
		"label":  "Menu",
		"action": "corner.open",
	}))
	if _, err := handler.prepareOutboundUI("device-1", ServerMessage{SetUI: &swappedUI}); err != nil {
		t.Fatalf("prepareOutboundUI(swapped) error = %v", err)
	}

	openNew, err := handler.handleInput(context.Background(), &InputRequest{
		DeviceID:    "device-1",
		ComponentID: "act:main-b/__affordance.corner__",
		Action:      "open",
	})
	if err != nil {
		t.Fatalf("handleInput(new activation) error = %v", err)
	}
	if len(openNew) != 1 || openNew[0].UpdateUI == nil {
		t.Fatalf("expected overlay update for new activation component id, got %+v", openNew)
	}

	snapshot := handler.metrics.Snapshot()
	if snapshot[`ui_action_unknown_component_total{reason="unknown_activation"}`] != "0" {
		t.Fatalf("unknown_activation counter after new action = %q, want 0", snapshot[`ui_action_unknown_component_total{reason="unknown_activation"}`])
	}

	oldOut, err := handler.handleInput(context.Background(), &InputRequest{
		DeviceID:    "device-1",
		ComponentID: "act:main-a/__affordance.corner__",
		Action:      "open",
	})
	if err != nil {
		t.Fatalf("handleInput(old activation) error = %v", err)
	}
	if len(oldOut) != 0 {
		t.Fatalf("old activation action should be dropped, got %+v", oldOut)
	}

	snapshot = handler.metrics.Snapshot()
	if snapshot[`ui_action_unknown_component_total{reason="unknown_activation"}`] != "1" {
		t.Fatalf("unknown_activation counter after stale action = %q, want 1", snapshot[`ui_action_unknown_component_total{reason="unknown_activation"}`])
	}
}

func hasClaim(claims []io.Claim, activationID, resource string) bool {
	for _, claim := range claims {
		if claim.ActivationID == activationID && claim.Resource == resource {
			return true
		}
	}
	return false
}

func TestHandleMessageInputRoutesMultiWindowEndActionAndRestoresPriorTerminal(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "device-2", DeviceName: "Hall"})
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
			RequestID: "cmd-terminal-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-multi-window-start",
			DeviceID:  "device-1",
			Kind:      "voice",
			Text:      "all cameras",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "multi_window_end",
			Action:      "multi_window_end",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input multi_window_end) error = %v", err)
	}

	if len(out) < 3 {
		t.Fatalf("len(out) = %d, want at least 3", len(out))
	}
	if out[0].ScenarioStop != "multi_window" {
		t.Fatalf("ScenarioStop = %q, want multi_window", out[0].ScenarioStop)
	}

	sawTerminalRoot := false
	sawTerminalEnter := false
	for _, msg := range out {
		if set := msg.SetUI; set != nil && (set.Props["id"] == "terminal_root" || strings.HasSuffix(set.Props["id"], "/terminal_root")) {
			sawTerminalRoot = true
		}
		if transition := msg.TransitionUI; transition != nil && transition.Transition == "terminal_enter" {
			sawTerminalEnter = true
		}
	}
	if !sawTerminalRoot {
		t.Fatalf("expected restored terminal SetUI after multi_window_end")
	}
	if !sawTerminalEnter {
		t.Fatalf("expected terminal_enter transition after multi_window_end")
	}
}

func TestHandleMessageInputRoutesInternalVideoCallEndAction(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "device-2", DeviceName: "Hall"})
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
			RequestID: "cmd-video-call-start",
			DeviceID:  "device-1",
			Kind:      "voice",
			Text:      "video call device-2",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "internal_video_call_hangup",
			Action:      "internal_video_call_end",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(input internal_video_call_end) error = %v", err)
	}

	if len(out) < 2 {
		t.Fatalf("len(out) = %d, want at least 2", len(out))
	}
	if out[0].ScenarioStop != "internal_video_call" {
		t.Fatalf("ScenarioStop = %q, want internal_video_call", out[0].ScenarioStop)
	}
	if out[1].TransitionUI == nil || out[1].TransitionUI.Transition != "internal_video_call_exit" {
		t.Fatalf("expected internal_video_call_exit transition, got %+v", out[1].TransitionUI)
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
	if out[0].Data["media_streams_active"] != "0" {
		t.Fatalf("media_streams_active = %q, want 0", out[0].Data["media_streams_active"])
	}
	if out[0].Data["media_streams_ready"] != "0" {
		t.Fatalf("media_streams_ready = %q, want 0", out[0].Data["media_streams_ready"])
	}
	if out[0].Data["media_streams_pending"] != "0" {
		t.Fatalf("media_streams_pending = %q, want 0", out[0].Data["media_streams_pending"])
	}
	if out[0].Data["sensor_devices_reporting"] != "0" {
		t.Fatalf("sensor_devices_reporting = %q, want 0", out[0].Data["sensor_devices_reporting"])
	}
	if out[0].Data["sensor_latest_unix_ms"] != "0" {
		t.Fatalf("sensor_latest_unix_ms = %q, want 0", out[0].Data["sensor_latest_unix_ms"])
	}
	if out[0].Data["sensor_device_ids"] != "" {
		t.Fatalf("sensor_device_ids = %q, want empty", out[0].Data["sensor_device_ids"])
	}
	if out[0].Data["sensor_summaries"] != "" {
		t.Fatalf("sensor_summaries = %q, want empty", out[0].Data["sensor_summaries"])
	}
	if out[0].Data["recording_active_streams"] != "0" {
		t.Fatalf("recording_active_streams = %q, want 0", out[0].Data["recording_active_streams"])
	}
	if out[0].Data["recording_stream_ids"] != "" {
		t.Fatalf("recording_stream_ids = %q, want empty", out[0].Data["recording_stream_ids"])
	}
}

func TestHandleMessageSystemRuntimeStatusTracksMediaStreamLifecycle(t *testing.T) {
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
	handler.SetRecordingManager(recording.NewMemoryManager())

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-2", DeviceName: "Hall Display"},
	})

	startOut, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "intercom-start-runtime-status",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "intercom",
		},
	})
	if err != nil {
		t.Fatalf("intercom start error = %v", err)
	}
	streamIDs := map[string]struct{}{}
	for _, msg := range startOut {
		if msg.StartStream != nil {
			streamIDs[msg.StartStream.StreamID] = struct{}{}
		}
	}
	if len(streamIDs) == 0 {
		t.Fatalf("expected start_stream message in start output")
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-runtime-pre-ready",
			Kind:      "system",
			Intent:    "runtime_status",
		},
	})
	if err != nil {
		t.Fatalf("runtime_status pre-ready error = %v", err)
	}
	if out[0].Data["media_streams_active"] != "2" {
		t.Fatalf("media_streams_active pre-ready = %q, want 2", out[0].Data["media_streams_active"])
	}
	if out[0].Data["media_streams_ready"] != "0" {
		t.Fatalf("media_streams_ready pre-ready = %q, want 0", out[0].Data["media_streams_ready"])
	}
	if out[0].Data["media_streams_pending"] != "2" {
		t.Fatalf("media_streams_pending pre-ready = %q, want 2", out[0].Data["media_streams_pending"])
	}
	if out[0].Data["recording_active_streams"] != "2" {
		t.Fatalf("recording_active_streams pre-ready = %q, want 2", out[0].Data["recording_active_streams"])
	}

	for streamID := range streamIDs {
		_, err = handler.HandleMessage(context.Background(), ClientMessage{
			StreamReady: &StreamReadyRequest{StreamID: streamID},
		})
		if err != nil {
			t.Fatalf("stream_ready error = %v", err)
		}
	}

	out, err = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-runtime-ready",
			Kind:      "system",
			Intent:    "runtime_status",
		},
	})
	if err != nil {
		t.Fatalf("runtime_status ready error = %v", err)
	}
	if out[0].Data["media_streams_active"] != "2" {
		t.Fatalf("media_streams_active ready = %q, want 2", out[0].Data["media_streams_active"])
	}
	if out[0].Data["media_streams_ready"] != "2" {
		t.Fatalf("media_streams_ready = %q, want 2", out[0].Data["media_streams_ready"])
	}
	if out[0].Data["media_streams_pending"] != "0" {
		t.Fatalf("media_streams_pending ready = %q, want 0", out[0].Data["media_streams_pending"])
	}
	if out[0].Data["recording_active_streams"] != "2" {
		t.Fatalf("recording_active_streams ready = %q, want 2", out[0].Data["recording_active_streams"])
	}
	if !strings.Contains(out[0].Data["media_streams"], "ready=true") {
		t.Fatalf("media_streams details should contain ready=true, got %q", out[0].Data["media_streams"])
	}

	_, err = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "intercom-stop-runtime-status",
			DeviceID:  "device-1",
			Action:    CommandActionStop,
			Kind:      "manual",
			Intent:    "intercom",
		},
	})
	if err != nil {
		t.Fatalf("intercom stop error = %v", err)
	}

	out, err = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-runtime-post-stop",
			Kind:      "system",
			Intent:    "runtime_status",
		},
	})
	if err != nil {
		t.Fatalf("runtime_status post-stop error = %v", err)
	}
	if out[0].Data["media_streams_active"] != "0" {
		t.Fatalf("media_streams_active post-stop = %q, want 0", out[0].Data["media_streams_active"])
	}
	if out[0].Data["media_streams_ready"] != "0" {
		t.Fatalf("media_streams_ready post-stop = %q, want 0", out[0].Data["media_streams_ready"])
	}
	if out[0].Data["media_streams_pending"] != "0" {
		t.Fatalf("media_streams_pending post-stop = %q, want 0", out[0].Data["media_streams_pending"])
	}
	if out[0].Data["media_streams"] != "" {
		t.Fatalf("media_streams post-stop = %q, want empty", out[0].Data["media_streams"])
	}
	if out[0].Data["recording_active_streams"] != "0" {
		t.Fatalf("recording_active_streams post-stop = %q, want 0", out[0].Data["recording_active_streams"])
	}
	if out[0].Data["recording_stream_ids"] != "" {
		t.Fatalf("recording_stream_ids post-stop = %q, want empty", out[0].Data["recording_stream_ids"])
	}
}

func TestHandleMessageSystemRecordingEvents(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
		IO:      io.NewRouter(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	recorder, err := recording.NewDiskManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewDiskManager() error = %v", err)
	}
	handler.SetRecordingManager(recorder)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-2", DeviceName: "Hall Display"},
	})

	_, err = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "recording-events-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "intercom",
		},
	})
	if err != nil {
		t.Fatalf("intercom start error = %v", err)
	}
	_, err = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "recording-events-stop",
			DeviceID:  "device-1",
			Action:    CommandActionStop,
			Kind:      "manual",
			Intent:    "intercom",
		},
	})
	if err != nil {
		t.Fatalf("intercom stop error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-recording-events",
			Kind:      "system",
			Intent:    SystemIntentRecordingEvents,
		},
	})
	if err != nil {
		t.Fatalf("recording_events query error = %v", err)
	}
	if out[0].Notification != "System query: recording_events" {
		t.Fatalf("notification = %q, want recording_events", out[0].Notification)
	}
	if len(out[0].Data) == 0 {
		t.Fatalf("expected recording event rows")
	}
	foundStart := false
	foundStop := false
	for _, row := range out[0].Data {
		if strings.Contains(row, "|start|") {
			foundStart = true
		}
		if strings.Contains(row, "|stop|") {
			foundStop = true
		}
	}
	if !foundStart {
		t.Fatalf("recording event rows missing start action: %+v", out[0].Data)
	}
	if !foundStop {
		t.Fatalf("recording event rows missing stop action: %+v", out[0].Data)
	}
}

func TestHandleMessageSystemListPlaybackArtifacts(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	recorder, err := recording.NewDiskManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewDiskManager() error = %v", err)
	}
	handler.SetRecordingManager(recorder)
	if err := recorder.Start(context.Background(), recording.Stream{
		StreamID:       "route:device-1|device-2|audio",
		Kind:           "audio",
		SourceDeviceID: "device-1",
		TargetDeviceID: "device-2",
	}); err != nil {
		t.Fatalf("recorder.Start() error = %v", err)
	}
	if err := recorder.WriteDeviceAudio("device-1", []byte{0x01, 0x02}); err != nil {
		t.Fatalf("recorder.WriteDeviceAudio() error = %v", err)
	}
	if err := recorder.Stop(context.Background(), "route:device-1|device-2|audio"); err != nil {
		t.Fatalf("recorder.Stop() error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-playback-artifacts",
			Kind:      "system",
			Intent:    SystemIntentListPlaybackFiles,
		},
	})
	if err != nil {
		t.Fatalf("list_playback_artifacts query error = %v", err)
	}
	if out[0].Notification != "System query: list_playback_artifacts" {
		t.Fatalf("notification = %q, want list_playback_artifacts", out[0].Notification)
	}
	if len(out[0].Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(out[0].Data))
	}
	row := out[0].Data["000"]
	if !strings.Contains(row, "route:device-1|device-2|audio") {
		t.Fatalf("row = %q, want stream id", row)
	}
	if !strings.Contains(row, "|audio|device-1|device-2|") {
		t.Fatalf("row = %q, want kind/source/target columns", row)
	}
}

func TestHandleMessageManualPlaybackMetadata(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	recorder, err := recording.NewDiskManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewDiskManager() error = %v", err)
	}
	handler.SetRecordingManager(recorder)
	if err := recorder.Start(context.Background(), recording.Stream{
		StreamID:       "route:device-a|device-b|audio",
		Kind:           "audio",
		SourceDeviceID: "device-a",
		TargetDeviceID: "device-b",
	}); err != nil {
		t.Fatalf("recorder.Start() error = %v", err)
	}
	if err := recorder.WriteDeviceAudio("device-a", []byte{0xAA, 0xBB, 0xCC}); err != nil {
		t.Fatalf("recorder.WriteDeviceAudio() error = %v", err)
	}
	if err := recorder.Stop(context.Background(), "route:device-a|device-b|audio"); err != nil {
		t.Fatalf("recorder.Stop() error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "manual-playback-metadata",
			DeviceID:  "device-a",
			Kind:      "manual",
			Intent:    ManualIntentPlaybackMetadata,
			Arguments: map[string]string{
				"artifact_id":      "route:device-a|device-b|audio",
				"target_device_id": "hall-display",
			},
		},
	})
	if err != nil {
		t.Fatalf("manual playback_metadata error = %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2 (command result + play audio)", len(out))
	}
	if out[0].Notification != "Playback metadata ready" {
		t.Fatalf("notification = %q, want playback metadata ready", out[0].Notification)
	}
	if out[0].Data["artifact_id"] != "route:device-a|device-b|audio" {
		t.Fatalf("artifact_id = %q, want route:device-a|device-b|audio", out[0].Data["artifact_id"])
	}
	if out[0].Data["target_device_id"] != "hall-display" {
		t.Fatalf("target_device_id = %q, want hall-display", out[0].Data["target_device_id"])
	}
	if out[0].Data["size_bytes"] != "3" {
		t.Fatalf("size_bytes = %q, want 3", out[0].Data["size_bytes"])
	}
	if out[0].CommandAck != "manual-playback-metadata" {
		t.Fatalf("CommandAck = %q, want manual-playback-metadata", out[0].CommandAck)
	}
	if out[1].PlayAudio == nil {
		t.Fatalf("expected PlayAudio response")
	}
	if out[1].PlayAudio.DeviceID != "hall-display" {
		t.Fatalf("PlayAudio.DeviceID = %q, want hall-display", out[1].PlayAudio.DeviceID)
	}
	if out[1].PlayAudio.Format != "pcm16" {
		t.Fatalf("PlayAudio.Format = %q, want pcm16", out[1].PlayAudio.Format)
	}
	if out[1].RelayToDeviceID != "hall-display" {
		t.Fatalf("RelayToDeviceID = %q, want hall-display", out[1].RelayToDeviceID)
	}
	if string(out[1].PlayAudio.Audio) != string([]byte{0xAA, 0xBB, 0xCC}) {
		t.Fatalf("PlayAudio.Audio = %v, want %v", out[1].PlayAudio.Audio, []byte{0xAA, 0xBB, 0xCC})
	}
}

func TestHandleMessageManualPlaybackMetadataDefaultsTargetToCaller(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	recorder, err := recording.NewDiskManager(t.TempDir())
	if err != nil {
		t.Fatalf("NewDiskManager() error = %v", err)
	}
	handler.SetRecordingManager(recorder)
	if err := recorder.Start(context.Background(), recording.Stream{
		StreamID:       "route:device-z|device-y|audio",
		Kind:           "audio",
		SourceDeviceID: "device-z",
		TargetDeviceID: "device-y",
	}); err != nil {
		t.Fatalf("recorder.Start() error = %v", err)
	}
	if err := recorder.WriteDeviceAudio("device-z", []byte{0x01, 0x02, 0x03, 0x04}); err != nil {
		t.Fatalf("recorder.WriteDeviceAudio() error = %v", err)
	}
	if err := recorder.Stop(context.Background(), "route:device-z|device-y|audio"); err != nil {
		t.Fatalf("recorder.Stop() error = %v", err)
	}

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "manual-playback-default-target",
			DeviceID:  "device-z",
			Kind:      "manual",
			Intent:    ManualIntentPlaybackMetadata,
			Arguments: map[string]string{
				"artifact_id": "route:device-z|device-y|audio",
			},
		},
	})
	if err != nil {
		t.Fatalf("manual playback_metadata default target error = %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	if out[1].PlayAudio == nil {
		t.Fatalf("expected PlayAudio response")
	}
	if out[1].PlayAudio.DeviceID != "device-z" {
		t.Fatalf("PlayAudio.DeviceID = %q, want device-z", out[1].PlayAudio.DeviceID)
	}
	if out[1].RelayToDeviceID != "" {
		t.Fatalf("RelayToDeviceID = %q, want empty for local playback", out[1].RelayToDeviceID)
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
	if len(events) != 1 || events[0].Message != "Timer done!" {
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
	if !contains(out[0].Data["system_intents"], SystemIntentListPlaybackFiles) {
		t.Fatalf("system_help missing list_playback_artifacts intent: %+v", out[0].Data)
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
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Sensor: &SensorDataRequest{
			DeviceID: "device-1",
			UnixMS:   1713000000000,
			Values: map[string]float64{
				"accelerometer.x": 0.25,
				"accelerometer.y": -0.75,
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
	if out[0].Data["sensor.unix_ms"] != "1713000000000" {
		t.Fatalf("sensor.unix_ms = %q, want 1713000000000", out[0].Data["sensor.unix_ms"])
	}
	if out[0].Data["sensor.accelerometer.x"] != "0.25" {
		t.Fatalf("sensor.accelerometer.x = %q, want 0.25", out[0].Data["sensor.accelerometer.x"])
	}
	if out[0].Data["sensor.accelerometer.y"] != "-0.75" {
		t.Fatalf("sensor.accelerometer.y = %q, want -0.75", out[0].Data["sensor.accelerometer.y"])
	}
}

func TestHandleMessageSystemRuntimeStatusIncludesSensorSummary(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-2", DeviceName: "Hall Display"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Sensor: &SensorDataRequest{
			DeviceID: "device-1",
			UnixMS:   1713000000123,
			Values: map[string]float64{
				"temperature.c": 22.4,
			},
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Sensor: &SensorDataRequest{
			DeviceID: "device-2",
			UnixMS:   1713000000456,
			Values: map[string]float64{
				"temperature.c": 23.1,
				"humidity.pct":  45.5,
			},
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-runtime-sensor-summary",
			Kind:      "system",
			Intent:    "runtime_status",
		},
	})
	if err != nil {
		t.Fatalf("runtime_status error = %v", err)
	}
	if out[0].Data["sensor_devices_reporting"] != "2" {
		t.Fatalf("sensor_devices_reporting = %q, want 2", out[0].Data["sensor_devices_reporting"])
	}
	if out[0].Data["sensor_latest_unix_ms"] != "1713000000456" {
		t.Fatalf("sensor_latest_unix_ms = %q, want 1713000000456", out[0].Data["sensor_latest_unix_ms"])
	}
	if out[0].Data["sensor_device_ids"] != "device-1,device-2" {
		t.Fatalf("sensor_device_ids = %q, want device-1,device-2", out[0].Data["sensor_device_ids"])
	}
	if !strings.Contains(out[0].Data["sensor_summaries"], "device-1|unix_ms=1713000000123|keys=temperature.c") {
		t.Fatalf("sensor_summaries missing device-1 detail: %q", out[0].Data["sensor_summaries"])
	}
	if !strings.Contains(out[0].Data["sensor_summaries"], "device-2|unix_ms=1713000000456|keys=humidity.pct,temperature.c") {
		t.Fatalf("sensor_summaries missing device-2 detail: %q", out[0].Data["sensor_summaries"])
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
	_ = scheduler.ScheduleRecord(context.Background(), storage.ScheduleRecord{
		Key:      "structured-timer-1",
		Kind:     "timer",
		Subject:  "pasta",
		DeviceID: "device-1",
		UnixMS:   control.now().Add(6 * time.Minute).UnixMilli(),
		Payload:  map[string]string{"duration_seconds": "360"},
	})

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
	if out[0].Data["timer:device-1:100"] != "kind=timer" {
		t.Fatalf("pending timer missing from response: %+v", out[0].Data)
	}
	if out[0].Data["structured-timer-1"] != "kind=timer|device=device-1|subject=pasta|duration_seconds=360" {
		t.Fatalf("structured pending timer missing metadata: %+v", out[0].Data)
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

func TestHandleMessageManualTerminalRefresh(t *testing.T) {
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
			RequestID: "manual-refresh-start-terminal",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "terminal_input",
			Action:      "submit",
			Value:       "sleep 1; printf '\\x72\\x65\\x66\\x72\\x65\\x73\\x68\\x2d\\x6d\\x61\\x6e\\x75\\x61\\x6c\\n'",
		},
	})
	time.Sleep(1200 * time.Millisecond)

	out := waitForRefreshMarker(t, handler, ClientMessage{
		Command: &CommandRequest{
			RequestID: "manual-refresh-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    SystemIntentTerminalRefresh,
		},
	}, "refresh-manual", 10, 200*time.Millisecond)
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	if out[0].CommandAck != "manual-refresh-1" {
		t.Fatalf("CommandAck = %q, want manual-refresh-1", out[0].CommandAck)
	}
	if out[1].UpdateUI == nil {
		t.Fatalf("expected UpdateUI response after manual terminal_refresh")
	}
	if !strings.Contains(out[1].UpdateUI.Node.Props["value"], "refresh-manual") {
		t.Fatalf("manual terminal_refresh missing delayed output: %+v", out[1].UpdateUI.Node.Props)
	}
}

func TestHandleMessageSystemTerminalRefresh(t *testing.T) {
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
			RequestID: "system-refresh-start-terminal",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "terminal_input",
			Action:      "submit",
			Value:       "sleep 1; printf '\\x72\\x65\\x66\\x72\\x65\\x73\\x68\\x2d\\x73\\x79\\x73\\x74\\x65\\x6d\\n'",
		},
	})
	time.Sleep(1200 * time.Millisecond)

	out := waitForRefreshMarker(t, handler, ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-refresh-1",
			Kind:      "system",
			Intent:    SystemIntentTerminalRefresh + " device-1",
		},
	}, "refresh-system", 10, 200*time.Millisecond)
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	if out[0].Data["device_id"] != "device-1" {
		t.Fatalf("device_id = %q, want device-1", out[0].Data["device_id"])
	}
	if out[1].UpdateUI == nil {
		t.Fatalf("expected UpdateUI response after system terminal_refresh")
	}
	if !strings.Contains(out[1].UpdateUI.Node.Props["value"], "refresh-system") {
		t.Fatalf("system terminal_refresh missing delayed output: %+v", out[1].UpdateUI.Node.Props)
	}
}

func TestHandleMessageInputTerminalRefreshAction(t *testing.T) {
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
			RequestID: "ui-refresh-start-terminal",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "terminal",
		},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "terminal_input",
			Action:      "submit",
			Value:       "sleep 1; printf '\\x72\\x65\\x66\\x72\\x65\\x73\\x68\\x2d\\x75\\x69\\n'",
		},
	})
	time.Sleep(1200 * time.Millisecond)

	out := waitForRefreshMarker(t, handler, ClientMessage{
		Input: &InputRequest{
			DeviceID:    "device-1",
			ComponentID: "terminal_refresh_button",
			Action:      SystemIntentTerminalRefresh,
		},
	}, "refresh-ui", 10, 200*time.Millisecond)
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	if out[0].Notification == "" {
		t.Fatalf("expected notification response before ui update")
	}
	if out[1].UpdateUI == nil {
		t.Fatalf("expected UpdateUI response after terminal_refresh UIAction")
	}
	if !strings.Contains(out[1].UpdateUI.Node.Props["value"], "refresh-ui") {
		t.Fatalf("terminal_refresh UIAction missing delayed output: %+v", out[1].UpdateUI.Node.Props)
	}
}

func TestHandleMessageSystemTerminalRefreshRequiresDeviceID(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "sys-refresh-missing-device",
			Kind:      "system",
			Intent:    SystemIntentTerminalRefresh,
		},
	})
	if err != ErrMissingCommandDeviceID {
		t.Fatalf("error = %v, want %v", err, ErrMissingCommandDeviceID)
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

func TestHandleMessageManualBluetoothScanUsesPassthroughScenario(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	bridge := &testRuntimePassthroughBridge{}
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:     devices,
		Broadcast:   ui.NewMemoryBroadcaster(),
		Passthrough: bridge,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "ble-scan-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    ManualIntentBluetoothScan,
			Arguments: map[string]string{
				"window_ms": "5000",
			},
		},
	})
	if err != nil {
		t.Fatalf("manual bluetooth_scan error = %v", err)
	}
	if len(out) == 0 || out[0].ScenarioStart != "bluetooth_passthrough" {
		t.Fatalf("unexpected response: %+v", out)
	}
	if len(bridge.bluetooth) != 1 {
		t.Fatalf("len(bluetooth commands) = %d, want 1", len(bridge.bluetooth))
	}
	if bridge.bluetooth[0].Action != "scan" {
		t.Fatalf("bluetooth action = %q, want scan", bridge.bluetooth[0].Action)
	}
	if bridge.bluetooth[0].Parameters["window_ms"] != "5000" {
		t.Fatalf("window_ms = %q, want 5000", bridge.bluetooth[0].Parameters["window_ms"])
	}
}

func TestHandleMessageManualUSBClaimUsesPassthroughScenario(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	bridge := &testRuntimePassthroughBridge{}
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:     devices,
		Broadcast:   ui.NewMemoryBroadcaster(),
		Passthrough: bridge,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "usb-claim-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    ManualIntentUSBClaim,
			Arguments: map[string]string{
				"vendor_id":  "1a2b",
				"product_id": "3c4d",
			},
		},
	})
	if err != nil {
		t.Fatalf("manual usb_claim error = %v", err)
	}
	if len(out) == 0 || out[0].ScenarioStart != "usb_passthrough" {
		t.Fatalf("unexpected response: %+v", out)
	}
	if len(bridge.usb) != 1 {
		t.Fatalf("len(usb commands) = %d, want 1", len(bridge.usb))
	}
	if bridge.usb[0].Action != "claim" {
		t.Fatalf("usb action = %q, want claim", bridge.usb[0].Action)
	}
	if bridge.usb[0].VendorID != "1a2b" || bridge.usb[0].ProductID != "3c4d" {
		t.Fatalf("unexpected usb cmd: %+v", bridge.usb[0])
	}
}

type testRuntimePassthroughBridge struct {
	bluetooth []scenario.BluetoothCommand
	usb       []scenario.USBCommand
}

func (t *testRuntimePassthroughBridge) DispatchBluetoothCommand(_ context.Context, cmd scenario.BluetoothCommand) error {
	t.bluetooth = append(t.bluetooth, cmd)
	return nil
}

func (t *testRuntimePassthroughBridge) DispatchUSBCommand(_ context.Context, cmd scenario.USBCommand) error {
	t.usb = append(t.usb, cmd)
	return nil
}

func contains(s, needle string) bool {
	return strings.Contains(s, needle)
}

func findNodePropValue(node *ui.Descriptor, nodeID, prop string) string {
	if node == nil {
		return ""
	}
	if node.Props["id"] == nodeID {
		return node.Props[prop]
	}
	for i := range node.Children {
		child := &node.Children[i]
		if got := findNodePropValue(child, nodeID, prop); got != "" {
			return got
		}
	}
	return ""
}
