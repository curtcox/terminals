package transport

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestGeneratedSessionProtocolViolationRecoverable(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Heartbeat{
					Heartbeat: &controlv1.Heartbeat{DeviceId: "device-1"},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Heartbeat{
					Heartbeat: &controlv1.Heartbeat{DeviceId: "device-1"},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}
	if len(stream.sent) < 3 {
		t.Fatalf("len(sent) = %d, want at least 3", len(stream.sent))
	}

	first, ok := stream.sent[0].(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("first response type = %T, want *controlv1.ConnectResponse", stream.sent[0])
	}
	if first.GetError() == nil || first.GetError().GetCode() != controlv1.ControlErrorCode_CONTROL_ERROR_CODE_PROTOCOL_VIOLATION {
		t.Fatalf("error code = %+v, want protocol violation", first.GetError())
	}
	if !strings.Contains(first.GetError().GetMessage(), "register required") {
		t.Fatalf("error message = %q, expected register-required text", first.GetError().GetMessage())
	}
}

func TestGeneratedSessionCommandValidationErrorCode(t *testing.T) {
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
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "bad-manual",
						DeviceId:  "device-1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
						Intent:    "   ",
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}
	if len(stream.sent) < 3 {
		t.Fatalf("len(sent) = %d, want at least 3", len(stream.sent))
	}
	last, ok := stream.sent[len(stream.sent)-1].(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("last response type = %T, want *controlv1.ConnectResponse", stream.sent[len(stream.sent)-1])
	}
	if last.GetError() == nil || last.GetError().GetCode() != controlv1.ControlErrorCode_CONTROL_ERROR_CODE_MISSING_COMMAND_INTENT {
		t.Fatalf("error code = %+v, want missing_command_intent", last.GetError())
	}
}

func TestGeneratedSessionSystemDataPayload(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "sys-help",
						Kind:      controlv1.CommandKind_COMMAND_KIND_SYSTEM,
						Intent:    SystemIntentHelp,
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}
	last, ok := stream.sent[len(stream.sent)-1].(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("help response type = %T, want *controlv1.ConnectResponse", stream.sent[len(stream.sent)-1])
	}
	if last.GetCommandResult() == nil {
		t.Fatalf("expected command result payload")
	}
	data := last.GetCommandResult().GetData()
	if data["system_intents"] == "" {
		t.Fatalf("system_intents entry missing from system help payload: %+v", data)
	}
	if data["command_kinds"] == "" || data["command_actions"] == "" {
		t.Fatalf("command metadata missing from system help payload: %+v", data)
	}
}

func TestGeneratedSessionTerminalTransitions(t *testing.T) {
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
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "terminal-start",
						DeviceId:  "device-1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
						Intent:    "terminal",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "terminal-stop",
						DeviceId:  "device-1",
						Action:    controlv1.CommandAction_COMMAND_ACTION_STOP,
						Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
						Intent:    "terminal",
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawEnter bool
	var sawExit bool
	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok || resp.GetTransitionUi() == nil {
			continue
		}
		switch resp.GetTransitionUi().GetTransition() {
		case "terminal_enter":
			sawEnter = true
		case "terminal_exit":
			sawExit = true
		}
	}
	if !sawEnter {
		t.Fatalf("expected terminal_enter transition payload")
	}
	if !sawExit {
		t.Fatalf("expected terminal_exit transition payload")
	}
}

func TestGeneratedSessionIntercomEmitsRouteStream(t *testing.T) {
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
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "intercom-start",
						DeviceId:  "device-1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
						Intent:    "intercom",
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawRoute bool
	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok || resp.GetRouteStream() == nil {
			continue
		}
		if resp.GetRouteStream().GetSourceDeviceId() == "device-1" &&
			resp.GetRouteStream().GetTargetDeviceId() == "device-2" &&
			resp.GetRouteStream().GetKind() == "audio" {
			sawRoute = true
		}
	}
	if !sawRoute {
		t.Fatalf("expected route_stream payload for intercom start")
	}
}

func TestGeneratedSessionIntercomStopEmitsStopStream(t *testing.T) {
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
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "intercom-start",
						DeviceId:  "device-1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
						Intent:    "intercom",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "intercom-stop",
						DeviceId:  "device-1",
						Action:    controlv1.CommandAction_COMMAND_ACTION_STOP,
						Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
						Intent:    "intercom",
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawStop bool
	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok || resp.GetStopStream() == nil {
			continue
		}
		if resp.GetStopStream().GetStreamId() == "route:device-1|device-2|audio" {
			sawStop = true
		}
	}
	if !sawStop {
		t.Fatalf("expected stop_stream payload for intercom stop")
	}
}

func TestGeneratedSessionIntercomFanOutRelaysMediaToPeerSession(t *testing.T) {
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
		sentCh: make(chan ProtoServerEnvelope, 24),
	}
	stream2 := &asyncFakeProtoStream{
		ctx:    context.Background(),
		recvCh: make(chan ProtoClientEnvelope, 8),
		sentCh: make(chan ProtoServerEnvelope, 24),
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var runErr1 error
	var runErr2 error
	go func() {
		defer wg.Done()
		runErr1 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream1, GeneratedProtoAdapter{})
	}()
	go func() {
		defer wg.Done()
		runErr2 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream2, GeneratedProtoAdapter{})
	}()

	waitFor := func(label string, ch <-chan ProtoServerEnvelope, pred func(*controlv1.ConnectResponse) bool) {
		deadline := time.After(2 * time.Second)
		for {
			select {
			case env := <-ch:
				resp, ok := env.(*controlv1.ConnectResponse)
				if !ok {
					t.Fatalf("unexpected envelope type %T", env)
				}
				if pred(resp) {
					return
				}
			case <-deadline:
				t.Fatalf("timed out waiting for %s", label)
			}
		}
	}

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "d1",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
				},
			},
		},
	}
	stream2.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "d2",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Hall"},
				},
			},
		},
	}
	for i := 0; i < 2; i++ {
		<-stream1.sentCh
		<-stream2.sentCh
	}

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "intercom-start",
				DeviceId:  "d1",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "intercom",
			},
		},
	}

	waitFor("source intercom scenario start", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
		return resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "intercom"
	})

	expected := map[string]bool{
		"route:d1|d2|audio": false,
		"route:d2|d1|audio": false,
	}
	startSeen := map[string]bool{}
	startMetadataSeen := map[string]bool{}
	routeSeen := map[string]bool{}
	waitFor("peer intercom start+route fan-out", stream2.sentCh, func(resp *controlv1.ConnectResponse) bool {
		if start := resp.GetStartStream(); start != nil {
			if _, ok := expected[start.GetStreamId()]; ok {
				startSeen[start.GetStreamId()] = true
				if start.GetMetadata()["origin"] == "route_delta" {
					startMetadataSeen[start.GetStreamId()] = true
				}
			}
		}
		if route := resp.GetRouteStream(); route != nil {
			if _, ok := expected[route.GetStreamId()]; ok {
				routeSeen[route.GetStreamId()] = true
			}
		}
		for streamID := range expected {
			if !startSeen[streamID] || !startMetadataSeen[streamID] || !routeSeen[streamID] {
				return false
			}
		}
		return true
	})

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "intercom-stop",
				DeviceId:  "d1",
				Action:    controlv1.CommandAction_COMMAND_ACTION_STOP,
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "intercom",
			},
		},
	}

	waitFor("source intercom scenario stop", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
		return resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStop() == "intercom"
	})

	stopSeen := map[string]bool{}
	waitFor("peer intercom stop fan-out", stream2.sentCh, func(resp *controlv1.ConnectResponse) bool {
		stop := resp.GetStopStream()
		if stop == nil {
			return false
		}
		if _, ok := expected[stop.GetStreamId()]; ok {
			stopSeen[stop.GetStreamId()] = true
		}
		for streamID := range expected {
			if !stopSeen[streamID] {
				return false
			}
		}
		return true
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

func TestGeneratedSessionPASystemRelaysReceiverOverlayAndTransitions(t *testing.T) {
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
		runErr1 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream1, GeneratedProtoAdapter{})
	}()
	go func() {
		defer wg.Done()
		runErr2 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream2, GeneratedProtoAdapter{})
	}()

	waitFor := func(label string, ch <-chan ProtoServerEnvelope, pred func(*controlv1.ConnectResponse) bool) *controlv1.ConnectResponse {
		seen := make([]string, 0, 8)
		deadline := time.After(2 * time.Second)
		for {
			select {
			case env := <-ch:
				resp, ok := env.(*controlv1.ConnectResponse)
				if !ok {
					t.Fatalf("unexpected envelope type %T", env)
				}
				switch {
				case resp.GetCommandResult() != nil:
					seen = append(seen, "command_result:"+resp.GetCommandResult().GetScenarioStart()+"/"+resp.GetCommandResult().GetScenarioStop())
				case resp.GetUpdateUi() != nil:
					seen = append(seen, "update_ui:"+resp.GetUpdateUi().GetComponentId())
				case resp.GetTransitionUi() != nil:
					seen = append(seen, "transition_ui:"+resp.GetTransitionUi().GetTransition())
				case resp.GetStartStream() != nil:
					seen = append(seen, "start_stream:"+resp.GetStartStream().GetStreamId())
				case resp.GetStopStream() != nil:
					seen = append(seen, "stop_stream:"+resp.GetStopStream().GetStreamId())
				case resp.GetRouteStream() != nil:
					seen = append(seen, "route_stream:"+resp.GetRouteStream().GetStreamId())
				default:
					seen = append(seen, "other")
				}
				if pred(resp) {
					return resp
				}
			case <-deadline:
				t.Fatalf("timed out waiting for %s (seen=%v)", label, seen)
			}
		}
	}

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "device-1",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
				},
			},
		},
	}
	stream2.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "device-2",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Hall"},
				},
			},
		},
	}

	for i := 0; i < 2; i++ {
		<-stream1.sentCh
		<-stream2.sentCh
	}

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "pa-start",
				DeviceId:  "device-1",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "pa_system",
			},
		},
	}

	startDone := false
	sourceEnterDone := false
	waitFor("pa source start payloads", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
		if resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "pa_system" {
			startDone = true
		}
		if resp.GetTransitionUi() != nil && resp.GetTransitionUi().GetTransition() == "pa_source_enter" {
			sourceEnterDone = true
		}
		return startDone && sourceEnterDone
	})

	receiverOverlayDone := false
	receiverEnterDone := false
	waitFor("pa receiver start payloads", stream2.sentCh, func(resp *controlv1.ConnectResponse) bool {
		if resp.GetUpdateUi() != nil && resp.GetUpdateUi().GetComponentId() == ui.GlobalOverlayComponentID {
			if got := resp.GetUpdateUi().GetNode().GetProps()["id"]; got != ui.GlobalOverlayComponentID {
				t.Fatalf("receiver overlay id prop = %q, want %q", got, ui.GlobalOverlayComponentID)
			}
			receiverOverlayDone = true
		}
		if resp.GetTransitionUi() != nil && resp.GetTransitionUi().GetTransition() == "pa_receive_enter" {
			receiverEnterDone = true
		}
		return receiverOverlayDone && receiverEnterDone
	})

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "pa-stop",
				DeviceId:  "device-1",
				Action:    controlv1.CommandAction_COMMAND_ACTION_STOP,
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "pa_system",
			},
		},
	}

	stopDone := false
	sourceExitDone := false
	waitFor("pa source stop payloads", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
		if resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStop() == "pa_system" {
			stopDone = true
		}
		if resp.GetTransitionUi() != nil && resp.GetTransitionUi().GetTransition() == "pa_source_exit" {
			sourceExitDone = true
		}
		return stopDone && sourceExitDone
	})

	receiverClearDone := false
	receiverExitDone := false
	waitFor("pa receiver stop payloads", stream2.sentCh, func(resp *controlv1.ConnectResponse) bool {
		if resp.GetUpdateUi() != nil && resp.GetUpdateUi().GetComponentId() == ui.GlobalOverlayComponentID {
			node := resp.GetUpdateUi().GetNode()
			if node.GetProps()["id"] == ui.GlobalOverlayComponentID &&
				len(node.GetChildren()) == 0 {
				receiverClearDone = true
			}
		}
		if resp.GetTransitionUi() != nil && resp.GetTransitionUi().GetTransition() == "pa_receive_exit" {
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

func TestGeneratedSessionRedAlertRelaysBroadcastNotification(t *testing.T) {
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
		runErr1 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream1, GeneratedProtoAdapter{})
	}()
	go func() {
		defer wg.Done()
		runErr2 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream2, GeneratedProtoAdapter{})
	}()

	waitFor := func(label string, ch <-chan ProtoServerEnvelope, pred func(*controlv1.ConnectResponse) bool) {
		deadline := time.After(2 * time.Second)
		for {
			select {
			case env := <-ch:
				resp, ok := env.(*controlv1.ConnectResponse)
				if !ok {
					t.Fatalf("unexpected envelope type %T", env)
				}
				if pred(resp) {
					return
				}
			case <-deadline:
				t.Fatalf("timed out waiting for %s", label)
			}
		}
	}

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "d1",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
				},
			},
		},
	}
	stream2.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "d2",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Hall"},
				},
			},
		},
	}
	for i := 0; i < 2; i++ {
		<-stream1.sentCh
		<-stream2.sentCh
	}

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "cmd-red-alert",
				DeviceId:  "d1",
				Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
				Text:      "red alert",
			},
		},
	}

	waitFor("source red_alert command result", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
		return resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "red_alert"
	})
	waitFor("peer RED ALERT notification relay", stream2.sentCh, func(resp *controlv1.ConnectResponse) bool {
		return resp.GetCommandResult() != nil && resp.GetCommandResult().GetNotification() == "RED ALERT"
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

func TestGeneratedSessionPASystemVoiceStopAliasesRelayCleanup(t *testing.T) {
	for _, spoken := range []string{"end pa", "stop pa"} {
		t.Run(spoken, func(t *testing.T) {
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
				sentCh: make(chan ProtoServerEnvelope, 24),
			}
			stream2 := &asyncFakeProtoStream{
				ctx:    context.Background(),
				recvCh: make(chan ProtoClientEnvelope, 8),
				sentCh: make(chan ProtoServerEnvelope, 24),
			}

			var wg sync.WaitGroup
			wg.Add(2)
			var runErr1 error
			var runErr2 error
			go func() {
				defer wg.Done()
				runErr1 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream1, GeneratedProtoAdapter{})
			}()
			go func() {
				defer wg.Done()
				runErr2 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream2, GeneratedProtoAdapter{})
			}()

			waitFor := func(label string, ch <-chan ProtoServerEnvelope, pred func(*controlv1.ConnectResponse) bool) {
				deadline := time.After(2 * time.Second)
				for {
					select {
					case env := <-ch:
						resp, ok := env.(*controlv1.ConnectResponse)
						if !ok {
							t.Fatalf("unexpected envelope type %T", env)
						}
						if pred(resp) {
							return
						}
					case <-deadline:
						t.Fatalf("timed out waiting for %s", label)
					}
				}
			}

			stream1.recvCh <- &controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "d1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			}
			stream2.recvCh <- &controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "d2",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Hall"},
						},
					},
				},
			}
			for i := 0; i < 2; i++ {
				<-stream1.sentCh
				<-stream2.sentCh
			}

			stream1.recvCh <- &controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "pa-start",
						DeviceId:  "d1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
						Intent:    "pa_system",
					},
				},
			}

			waitFor("source pa start", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
				return resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "pa_system"
			})
			waitFor("peer pa start route", stream2.sentCh, func(resp *controlv1.ConnectResponse) bool {
				return resp.GetStartStream() != nil && resp.GetStartStream().GetStreamId() == "route:d1|d2|pa_audio"
			})

			stream1.recvCh <- &controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "pa-stop-voice",
						DeviceId:  "d1",
						Action:    controlv1.CommandAction_COMMAND_ACTION_STOP,
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      spoken,
					},
				},
			}

			waitFor("source pa stop via voice alias", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
				return resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStop() == "pa_system"
			})
			waitFor("source pa source_exit transition", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
				return resp.GetTransitionUi() != nil && resp.GetTransitionUi().GetTransition() == "pa_source_exit"
			})

			peerStopSeen := false
			peerOverlayClearSeen := false
			peerReceiveExitSeen := false
			waitFor("peer pa stop cleanup relays", stream2.sentCh, func(resp *controlv1.ConnectResponse) bool {
				if stop := resp.GetStopStream(); stop != nil && stop.GetStreamId() == "route:d1|d2|pa_audio" {
					peerStopSeen = true
				}
				if update := resp.GetUpdateUi(); update != nil && update.GetComponentId() == ui.GlobalOverlayComponentID {
					node := update.GetNode()
					if node.GetProps()["id"] == ui.GlobalOverlayComponentID && len(node.GetChildren()) == 0 {
						peerOverlayClearSeen = true
					}
				}
				if transition := resp.GetTransitionUi(); transition != nil && transition.GetTransition() == "pa_receive_exit" {
					peerReceiveExitSeen = true
				}
				return peerStopSeen && peerOverlayClearSeen && peerReceiveExitSeen
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
		})
	}
}

func TestGeneratedSessionWebRTCSignalRelayAcrossSessions(t *testing.T) {
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
		sentCh: make(chan ProtoServerEnvelope, 24),
	}
	stream2 := &asyncFakeProtoStream{
		ctx:    context.Background(),
		recvCh: make(chan ProtoClientEnvelope, 8),
		sentCh: make(chan ProtoServerEnvelope, 24),
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var runErr1 error
	var runErr2 error
	go func() {
		defer wg.Done()
		runErr1 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream1, GeneratedProtoAdapter{})
	}()
	go func() {
		defer wg.Done()
		runErr2 = RunProtoSession(NewStreamHandlerWithRuntime(control, runtime), control, stream2, GeneratedProtoAdapter{})
	}()

	waitFor := func(label string, ch <-chan ProtoServerEnvelope, pred func(*controlv1.ConnectResponse) bool) {
		deadline := time.After(2 * time.Second)
		for {
			select {
			case env := <-ch:
				resp, ok := env.(*controlv1.ConnectResponse)
				if !ok {
					t.Fatalf("unexpected envelope type %T", env)
				}
				if pred(resp) {
					return
				}
			case <-deadline:
				t.Fatalf("timed out waiting for %s", label)
			}
		}
	}

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "d1",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
				},
			},
		},
	}
	stream2.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "d2",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Hall"},
				},
			},
		},
	}
	for i := 0; i < 2; i++ {
		<-stream1.sentCh
		<-stream2.sentCh
	}

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "intercom-start",
				DeviceId:  "d1",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "intercom",
			},
		},
	}
	waitFor("source intercom start", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
		return resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "intercom"
	})
	waitFor("peer intercom route", stream2.sentCh, func(resp *controlv1.ConnectResponse) bool {
		return resp.GetRouteStream() != nil && resp.GetRouteStream().GetStreamId() == "route:d1|d2|audio"
	})

	stream1.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_WebrtcSignal{
			WebrtcSignal: &controlv1.WebRTCSignal{
				StreamId:   "route:d1|d2|audio",
				SignalType: "offer",
				Payload:    "{\"sdp\":\"v=0-offer\"}",
			},
		},
	}
	waitFor("relayed offer to peer", stream2.sentCh, func(resp *controlv1.ConnectResponse) bool {
		signal := resp.GetWebrtcSignal()
		return signal != nil &&
			signal.GetStreamId() == "route:d1|d2|audio" &&
			signal.GetSignalType() == "offer" &&
			signal.GetPayload() == "{\"sdp\":\"v=0-offer\"}"
	})

	stream2.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_WebrtcSignal{
			WebrtcSignal: &controlv1.WebRTCSignal{
				StreamId:   "route:d1|d2|audio",
				SignalType: "answer",
				Payload:    "{\"sdp\":\"v=0-answer\"}",
			},
		},
	}
	waitFor("relayed answer to source", stream1.sentCh, func(resp *controlv1.ConnectResponse) bool {
		signal := resp.GetWebrtcSignal()
		return signal != nil &&
			signal.GetStreamId() == "route:d1|d2|audio" &&
			signal.GetSignalType() == "answer" &&
			signal.GetPayload() == "{\"sdp\":\"v=0-answer\"}"
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

func TestGeneratedSessionVoiceStandDownStopsRedAlert(t *testing.T) {
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
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "red-alert-start",
						DeviceId:  "device-1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "red alert",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "red-alert-stop-stand-down",
						DeviceId:  "device-1",
						Action:    controlv1.CommandAction_COMMAND_ACTION_STOP,
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "stand down",
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawStart bool
	var sawStop bool
	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok || resp.GetCommandResult() == nil {
			continue
		}
		switch {
		case resp.GetCommandResult().GetScenarioStart() == "red_alert":
			sawStart = true
		case resp.GetCommandResult().GetScenarioStop() == "red_alert":
			sawStop = true
		}
	}
	if !sawStart {
		t.Fatalf("expected scenario_start=red_alert command result")
	}
	if !sawStop {
		t.Fatalf("expected scenario_stop=red_alert command result via stand down")
	}
}

func TestGeneratedSessionVoiceStopRedAlertStopsRedAlert(t *testing.T) {
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
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "red-alert-start",
						DeviceId:  "device-1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "red alert",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "red-alert-stop-stop-red-alert",
						DeviceId:  "device-1",
						Action:    controlv1.CommandAction_COMMAND_ACTION_STOP,
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "stop red alert",
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawStart bool
	var sawStop bool
	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok || resp.GetCommandResult() == nil {
			continue
		}
		switch {
		case resp.GetCommandResult().GetScenarioStart() == "red_alert":
			sawStart = true
		case resp.GetCommandResult().GetScenarioStop() == "red_alert":
			sawStop = true
		}
	}
	if !sawStart {
		t.Fatalf("expected scenario_start=red_alert command result")
	}
	if !sawStop {
		t.Fatalf("expected scenario_stop=red_alert command result via stop red alert")
	}
}

func TestGeneratedSessionVoicePAModeStartsPASystem(t *testing.T) {
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
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "pa-mode-start",
						DeviceId:  "device-1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "pa mode",
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawStart bool
	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok || resp.GetCommandResult() == nil {
			continue
		}
		if resp.GetCommandResult().GetScenarioStart() == "pa_system" {
			sawStart = true
			break
		}
	}
	if !sawStart {
		t.Fatalf("expected scenario_start=pa_system command result via pa mode")
	}
}
