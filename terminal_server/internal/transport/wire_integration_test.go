package transport

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestWireSessionProtocolViolationRecoverable(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			WireClientMessage{Heartbeat: &WireHeartbeatRequest{DeviceID: "device-1"}},
			WireClientMessage{Register: &WireRegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"}},
			WireClientMessage{Heartbeat: &WireHeartbeatRequest{DeviceID: "device-1"}},
		},
	}

	if err := RunProtoSession(handler, control, stream, WireProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}
	if len(stream.sent) < 3 {
		t.Fatalf("len(sent) = %d, want at least 3", len(stream.sent))
	}

	first, ok := stream.sent[0].(WireServerMessage)
	if !ok {
		t.Fatalf("first response type = %T, want WireServerMessage", stream.sent[0])
	}
	if first.Error == nil || first.Error.Code != WireControlErrorCodeProtocolViolation {
		t.Fatalf("ErrorCode = %+v, want %d", first.Error, WireControlErrorCodeProtocolViolation)
	}
	if !strings.Contains(first.Error.Message, "register required") {
		t.Fatalf("Error = %q, expected register-required text", first.Error.Message)
	}
}

func TestWireSessionCommandValidationErrorCode(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			WireClientMessage{Register: &WireRegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"}},
			WireClientMessage{Command: &WireCommandRequest{
				RequestID: "bad-manual",
				DeviceID:  "device-1",
				Kind:      WireCommandKindManual,
				Intent:    "   ",
			}},
		},
	}

	if err := RunProtoSession(handler, control, stream, WireProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}
	if len(stream.sent) < 3 {
		t.Fatalf("len(sent) = %d, want at least 3", len(stream.sent))
	}
	last, ok := stream.sent[len(stream.sent)-1].(WireServerMessage)
	if !ok {
		t.Fatalf("last response type = %T, want WireServerMessage", stream.sent[len(stream.sent)-1])
	}
	if last.Error == nil || last.Error.Code != WireControlErrorCodeMissingCommandIntent {
		t.Fatalf("ErrorCode = %+v, want %d", last.Error, WireControlErrorCodeMissingCommandIntent)
	}
}

func TestWireSessionSystemDataDeterministicOrder(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			WireClientMessage{Register: &WireRegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"}},
			WireClientMessage{Command: &WireCommandRequest{
				RequestID: "sys-help",
				Kind:      WireCommandKindSystem,
				Intent:    SystemIntentHelp,
			}},
		},
	}

	if err := RunProtoSession(handler, control, stream, WireProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}
	helpMsg, ok := stream.sent[len(stream.sent)-1].(WireServerMessage)
	if !ok {
		t.Fatalf("help response type = %T, want WireServerMessage", stream.sent[len(stream.sent)-1])
	}
	if helpMsg.CommandResult == nil || len(helpMsg.CommandResult.Data) < 2 {
		t.Fatalf("expected help command result data entries, got %+v", helpMsg.CommandResult)
	}
	for i := 1; i < len(helpMsg.CommandResult.Data); i++ {
		if helpMsg.CommandResult.Data[i-1].Key > helpMsg.CommandResult.Data[i].Key {
			t.Fatalf("data entries not sorted: %+v", helpMsg.CommandResult.Data)
		}
	}
}

func TestWireSessionIntercomEmitsRouteStream(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "device-2", DeviceName: "Hall"})
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			WireClientMessage{Register: &WireRegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"}},
			WireClientMessage{Command: &WireCommandRequest{
				RequestID: "intercom-start",
				DeviceID:  "device-1",
				Kind:      WireCommandKindManual,
				Intent:    "intercom",
			}},
		},
	}

	if err := RunProtoSession(handler, control, stream, WireProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawRoute bool
	for _, sent := range stream.sent {
		msg, ok := sent.(WireServerMessage)
		if !ok || msg.RouteStream == nil {
			continue
		}
		if msg.RouteStream.SourceDeviceID == "device-1" &&
			msg.RouteStream.TargetDeviceID == "device-2" &&
			msg.RouteStream.Kind == "audio" {
			sawRoute = true
		}
	}
	if !sawRoute {
		t.Fatalf("expected route_stream payload for intercom start")
	}
}

func TestWireSessionIntercomStopEmitsStopStream(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "device-2", DeviceName: "Hall"})
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			WireClientMessage{Register: &WireRegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"}},
			WireClientMessage{Command: &WireCommandRequest{
				RequestID: "intercom-start",
				DeviceID:  "device-1",
				Kind:      WireCommandKindManual,
				Intent:    "intercom",
			}},
			WireClientMessage{Command: &WireCommandRequest{
				RequestID: "intercom-stop",
				DeviceID:  "device-1",
				Action:    WireCommandActionStop,
				Kind:      WireCommandKindManual,
				Intent:    "intercom",
			}},
		},
	}

	if err := RunProtoSession(handler, control, stream, WireProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawStop bool
	for _, sent := range stream.sent {
		msg, ok := sent.(WireServerMessage)
		if !ok || msg.StopStream == nil {
			continue
		}
		if msg.StopStream.StreamID == "route:device-1|device-2|audio" {
			sawStop = true
		}
	}
	if !sawStop {
		t.Fatalf("expected stop_stream payload for intercom stop")
	}
}

func TestWireSessionPASystemRelaysReceiverOverlayAndTransitions(t *testing.T) {
	globalSessionRelayRegistry = newSessionRelayRegistry()
	t.Cleanup(func() {
		globalSessionRelayRegistry = newSessionRelayRegistry()
	})

	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	stream1 := &asyncFakeProtoStream{
		ctx:    context.Background(),
		recvCh: make(chan ProtoClientEnvelope, 8),
		sentCh: make(chan ProtoServerEnvelope, 16),
	}
	stream2 := &asyncFakeProtoStream{
		ctx:    context.Background(),
		recvCh: make(chan ProtoClientEnvelope, 8),
		sentCh: make(chan ProtoServerEnvelope, 16),
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var runErr1 error
	var runErr2 error
	go func() {
		defer wg.Done()
		runErr1 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream1, WireProtoAdapter{})
	}()
	go func() {
		defer wg.Done()
		runErr2 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream2, WireProtoAdapter{})
	}()

	waitFor := func(label string, ch <-chan ProtoServerEnvelope, pred func(WireServerMessage) bool) WireServerMessage {
		seen := make([]string, 0, 8)
		deadline := time.After(2 * time.Second)
		for {
			select {
			case env := <-ch:
				msg, ok := env.(WireServerMessage)
				if !ok {
					t.Fatalf("unexpected envelope type %T", env)
				}
				switch {
				case msg.CommandResult != nil:
					seen = append(seen, "command_result:"+msg.CommandResult.ScenarioStart+"/"+msg.CommandResult.ScenarioStop)
				case msg.UpdateUI != nil:
					seen = append(seen, "update_ui:"+msg.UpdateUI.ComponentID)
				case msg.TransitionUI != nil:
					seen = append(seen, "transition_ui:"+msg.TransitionUI.Transition)
				case msg.StartStream != nil:
					seen = append(seen, "start_stream:"+msg.StartStream.StreamID)
				case msg.StopStream != nil:
					seen = append(seen, "stop_stream:"+msg.StopStream.StreamID)
				case msg.RouteStream != nil:
					seen = append(seen, "route_stream:"+msg.RouteStream.StreamID)
				default:
					seen = append(seen, "other")
				}
				if pred(msg) {
					return msg
				}
			case <-deadline:
				t.Fatalf("timed out waiting for %s (seen=%v)", label, seen)
			}
		}
	}

	stream1.recvCh <- WireClientMessage{Register: &WireRegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"}}
	stream2.recvCh <- WireClientMessage{Register: &WireRegisterRequest{DeviceID: "device-2", DeviceName: "Hall"}}

	for i := 0; i < 2; i++ {
		<-stream1.sentCh
		<-stream2.sentCh
	}

	stream1.recvCh <- WireClientMessage{Command: &WireCommandRequest{
		RequestID: "pa-start",
		DeviceID:  "device-1",
		Kind:      WireCommandKindManual,
		Intent:    "pa_system",
	}}

	startDone := false
	sourceEnterDone := false
	waitFor("pa source start payloads", stream1.sentCh, func(msg WireServerMessage) bool {
		if msg.CommandResult != nil && msg.CommandResult.ScenarioStart == "pa_system" {
			startDone = true
		}
		if msg.TransitionUI != nil && msg.TransitionUI.Transition == "pa_source_enter" {
			sourceEnterDone = true
		}
		return startDone && sourceEnterDone
	})

	receiverOverlayDone := false
	receiverEnterDone := false
	waitFor("pa receiver start payloads", stream2.sentCh, func(msg WireServerMessage) bool {
		if msg.UpdateUI != nil && msg.UpdateUI.ComponentID == ui.GlobalOverlayComponentID {
			if got := DecodeDataEntries(msg.UpdateUI.Node.Props)["id"]; got != ui.GlobalOverlayComponentID {
				t.Fatalf("receiver overlay id prop = %q, want %q", got, ui.GlobalOverlayComponentID)
			}
			receiverOverlayDone = true
		}
		if msg.TransitionUI != nil && msg.TransitionUI.Transition == "pa_receive_enter" {
			receiverEnterDone = true
		}
		return receiverOverlayDone && receiverEnterDone
	})

	stream1.recvCh <- WireClientMessage{Command: &WireCommandRequest{
		RequestID: "pa-stop",
		DeviceID:  "device-1",
		Action:    WireCommandActionStop,
		Kind:      WireCommandKindManual,
		Intent:    "pa_system",
	}}

	stopDone := false
	sourceExitDone := false
	waitFor("pa source stop payloads", stream1.sentCh, func(msg WireServerMessage) bool {
		if msg.CommandResult != nil && msg.CommandResult.ScenarioStop == "pa_system" {
			stopDone = true
		}
		if msg.TransitionUI != nil && msg.TransitionUI.Transition == "pa_source_exit" {
			sourceExitDone = true
		}
		return stopDone && sourceExitDone
	})

	receiverClearDone := false
	receiverExitDone := false
	waitFor("pa receiver stop payloads", stream2.sentCh, func(msg WireServerMessage) bool {
		if msg.UpdateUI != nil && msg.UpdateUI.ComponentID == ui.GlobalOverlayComponentID {
			if DecodeDataEntries(msg.UpdateUI.Node.Props)["id"] == ui.GlobalOverlayComponentID &&
				len(msg.UpdateUI.Node.Children) == 0 {
				receiverClearDone = true
			}
		}
		if msg.TransitionUI != nil && msg.TransitionUI.Transition == "pa_receive_exit" {
			receiverExitDone = true
		}
		return receiverClearDone && receiverExitDone
	})

	close(stream1.recvCh)
	close(stream2.recvCh)
	wg.Wait()
	if runErr1 != nil {
		t.Fatalf("session1 RunProtoSession() error = %v", runErr1)
	}
	if runErr2 != nil {
		t.Fatalf("session2 RunProtoSession() error = %v", runErr2)
	}
}

func TestWireSessionRedAlertRelaysBroadcastNotification(t *testing.T) {
	globalSessionRelayRegistry = newSessionRelayRegistry()
	t.Cleanup(func() {
		globalSessionRelayRegistry = newSessionRelayRegistry()
	})

	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	stream1 := &asyncFakeProtoStream{
		ctx:    context.Background(),
		recvCh: make(chan ProtoClientEnvelope, 8),
		sentCh: make(chan ProtoServerEnvelope, 16),
	}
	stream2 := &asyncFakeProtoStream{
		ctx:    context.Background(),
		recvCh: make(chan ProtoClientEnvelope, 8),
		sentCh: make(chan ProtoServerEnvelope, 16),
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var runErr1 error
	var runErr2 error
	go func() {
		defer wg.Done()
		runErr1 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream1, WireProtoAdapter{})
	}()
	go func() {
		defer wg.Done()
		runErr2 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream2, WireProtoAdapter{})
	}()

	waitFor := func(label string, ch <-chan ProtoServerEnvelope, pred func(WireServerMessage) bool) {
		deadline := time.After(2 * time.Second)
		for {
			select {
			case env := <-ch:
				msg, ok := env.(WireServerMessage)
				if !ok {
					t.Fatalf("unexpected envelope type %T", env)
				}
				if pred(msg) {
					return
				}
			case <-deadline:
				t.Fatalf("timed out waiting for %s", label)
			}
		}
	}

	stream1.recvCh <- WireClientMessage{Register: &WireRegisterRequest{DeviceID: "d1", DeviceName: "Kitchen"}}
	stream2.recvCh <- WireClientMessage{Register: &WireRegisterRequest{DeviceID: "d2", DeviceName: "Hall"}}
	for i := 0; i < 2; i++ {
		<-stream1.sentCh
		<-stream2.sentCh
	}

	stream1.recvCh <- WireClientMessage{Command: &WireCommandRequest{
		RequestID: "cmd-red-alert",
		DeviceID:  "d1",
		Kind:      WireCommandKindVoice,
		Text:      "red alert",
	}}

	waitFor("source red_alert command result", stream1.sentCh, func(msg WireServerMessage) bool {
		return msg.CommandResult != nil && msg.CommandResult.ScenarioStart == "red_alert"
	})
	waitFor("peer RED ALERT notification relay", stream2.sentCh, func(msg WireServerMessage) bool {
		return msg.CommandResult != nil && msg.CommandResult.Notification == "RED ALERT"
	})

	close(stream1.recvCh)
	close(stream2.recvCh)
	wg.Wait()
	if runErr1 != nil {
		t.Fatalf("session1 RunProtoSession() error = %v", runErr1)
	}
	if runErr2 != nil {
		t.Fatalf("session2 RunProtoSession() error = %v", runErr2)
	}
}
