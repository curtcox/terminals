package transport

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	iov1 "github.com/curtcox/terminals/terminal_server/gen/go/io/v1"
	uiv1 "github.com/curtcox/terminals/terminal_server/gen/go/ui/v1"
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

func TestGeneratedSessionPrivacyToggleCapabilityLossStopsAudioAndVideoRoutes(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "device-2", DeviceName: "Hall"})
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	router := io.NewRouter()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        router,
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	if _, err := router.Claims().Request(context.Background(), []io.Claim{{
		ActivationID: "activation-mic",
		DeviceID:     "device-1",
		Resource:     "mic.capture",
		Mode:         io.ClaimExclusive,
		Priority:     1,
	}, {
		ActivationID: "activation-camera",
		DeviceID:     "device-1",
		Resource:     "camera.capture",
		Mode:         io.ClaimExclusive,
		Priority:     1,
	}}); err != nil {
		t.Fatalf("Claims().Request() error = %v", err)
	}
	if err := router.Connect("device-1", "device-2", "audio"); err != nil {
		t.Fatalf("Connect(audio) error = %v", err)
	}
	if err := router.Connect("device-1", "device-2", "video"); err != nil {
		t.Fatalf("Connect(video) error = %v", err)
	}

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
				Payload: &controlv1.ConnectRequest_CapabilitySnapshot{
					CapabilitySnapshot: &controlv1.CapabilitySnapshot{
						DeviceId:   "device-1",
						Generation: 2,
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId:   "device-1",
							Microphone: &capabilitiesv1.AudioInputCapability{},
							Camera:     &capabilitiesv1.CameraCapability{},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_CapabilityDelta{
					CapabilityDelta: &controlv1.CapabilityDelta{
						DeviceId:   "device-1",
						Generation: 3,
						Reason:     "privacy.toggle",
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
						},
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	if len(router.Claims().Snapshot("device-1")) != 0 {
		t.Fatalf("expected mic/camera claims to be released after privacy.toggle capability withdrawal")
	}

	stopStreamIDs := map[string]struct{}{}
	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok || resp.GetStopStream() == nil {
			continue
		}
		stopStreamIDs[resp.GetStopStream().GetStreamId()] = struct{}{}
	}

	expectedStopStreamIDs := map[string]struct{}{
		"route:device-1|device-2|audio": {},
		"route:device-1|device-2|video": {},
	}
	if len(stopStreamIDs) != len(expectedStopStreamIDs) {
		t.Fatalf("stop_stream count = %d, want %d (ids=%v)", len(stopStreamIDs), len(expectedStopStreamIDs), stopStreamIDs)
	}
	for streamID := range expectedStopStreamIDs {
		if _, ok := stopStreamIDs[streamID]; !ok {
			t.Fatalf("missing stop_stream for %q (ids=%v)", streamID, stopStreamIDs)
		}
	}
}

func TestGeneratedSessionPrivacyToggleExitReaddsCapabilitiesAndResumesClaims(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "device-2", DeviceName: "Hall"})
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	router := io.NewRouter()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        router,
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	if _, err := router.Claims().Request(context.Background(), []io.Claim{{
		ActivationID: "activation-mic",
		DeviceID:     "device-1",
		Resource:     "mic.capture",
		Mode:         io.ClaimExclusive,
		Priority:     1,
	}, {
		ActivationID: "activation-camera",
		DeviceID:     "device-1",
		Resource:     "camera.capture",
		Mode:         io.ClaimExclusive,
		Priority:     1,
	}}); err != nil {
		t.Fatalf("Claims().Request() error = %v", err)
	}

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
				Payload: &controlv1.ConnectRequest_CapabilitySnapshot{
					CapabilitySnapshot: &controlv1.CapabilitySnapshot{
						DeviceId:   "device-1",
						Generation: 1,
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId:   "device-1",
							Microphone: &capabilitiesv1.AudioInputCapability{},
							Camera:     &capabilitiesv1.CameraCapability{},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_CapabilityDelta{
					CapabilityDelta: &controlv1.CapabilityDelta{
						DeviceId:   "device-1",
						Generation: 2,
						Reason:     "privacy.toggle",
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_CapabilityDelta{
					CapabilityDelta: &controlv1.CapabilityDelta{
						DeviceId:   "device-1",
						Generation: 3,
						Reason:     "privacy.toggle",
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId:   "device-1",
							Microphone: &capabilitiesv1.AudioInputCapability{},
							Camera:     &capabilitiesv1.CameraCapability{},
						},
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	registered, ok := devices.Get("device-1")
	if !ok {
		t.Fatalf("device-1 should be registered")
	}
	if registered.Generation != 3 {
		t.Fatalf("generation after privacy exit = %d, want 3", registered.Generation)
	}

	claims := router.Claims().Snapshot("device-1")
	if len(claims) != 2 {
		t.Fatalf("active claim count after privacy exit = %d, want 2", len(claims))
	}
	claimByResource := map[string]string{}
	for _, claim := range claims {
		claimByResource[claim.Resource] = claim.ActivationID
	}
	if claimByResource["mic.capture"] != "activation-mic" {
		t.Fatalf("mic.capture claim owner = %q, want activation-mic", claimByResource["mic.capture"])
	}
	if claimByResource["camera.capture"] != "activation-camera" {
		t.Fatalf("camera.capture claim owner = %q, want activation-camera", claimByResource["camera.capture"])
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
		if resp.GetUpdateUi() != nil &&
			(resp.GetUpdateUi().GetComponentId() == ui.GlobalOverlayComponentID ||
				strings.HasSuffix(resp.GetUpdateUi().GetComponentId(), "/"+ui.GlobalOverlayComponentID)) {
			if got := resp.GetUpdateUi().GetNode().GetProps()["id"]; got != ui.GlobalOverlayComponentID &&
				!strings.HasSuffix(got, "/"+ui.GlobalOverlayComponentID) {
				t.Fatalf("receiver overlay id prop = %q, want scoped or legacy %q", got, ui.GlobalOverlayComponentID)
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
		if resp.GetUpdateUi() != nil &&
			(resp.GetUpdateUi().GetComponentId() == ui.GlobalOverlayComponentID ||
				strings.HasSuffix(resp.GetUpdateUi().GetComponentId(), "/"+ui.GlobalOverlayComponentID)) {
			node := resp.GetUpdateUi().GetNode()
			if (node.GetProps()["id"] == ui.GlobalOverlayComponentID ||
				strings.HasSuffix(node.GetProps()["id"], "/"+ui.GlobalOverlayComponentID)) &&
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
				if update := resp.GetUpdateUi(); update != nil &&
					(update.GetComponentId() == ui.GlobalOverlayComponentID ||
						strings.HasSuffix(update.GetComponentId(), "/"+ui.GlobalOverlayComponentID)) {
					node := update.GetNode()
					if (node.GetProps()["id"] == ui.GlobalOverlayComponentID ||
						strings.HasSuffix(node.GetProps()["id"], "/"+ui.GlobalOverlayComponentID)) &&
						len(node.GetChildren()) == 0 {
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

func TestGeneratedSessionVoiceShowAllCamerasStartsMultiWindow(t *testing.T) {
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
						RequestId: "show-all-cameras-start",
						DeviceId:  "device-1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "show all cameras",
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
		if resp.GetCommandResult().GetScenarioStart() == "multi_window" {
			sawStart = true
			break
		}
	}
	if !sawStart {
		t.Fatalf("expected scenario_start=multi_window command result via show all cameras")
	}
}

func TestGeneratedSessionVoiceAllCamerasStartsMultiWindow(t *testing.T) {
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
						RequestId: "all-cameras-start",
						DeviceId:  "device-1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "all cameras",
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
		if resp.GetCommandResult().GetScenarioStart() == "multi_window" {
			sawStart = true
			break
		}
	}
	if !sawStart {
		t.Fatalf("expected scenario_start=multi_window command result via all cameras")
	}
}

func TestGeneratedSessionMultiWindowAudioMixAndFocusSelection(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
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
							DeviceId: "d1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "all-cameras-start",
						DeviceId:  "d1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "all cameras",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "all-cameras-stop",
						DeviceId:  "d1",
						Action:    controlv1.CommandAction_COMMAND_ACTION_STOP,
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "all cameras",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "all-cameras-focus-start",
						DeviceId:  "d1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "all cameras focus d2",
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	sawMixD2 := false
	sawMixD3 := false
	sawMixStopD2 := false
	sawMixStopD3 := false
	focusStartIdx := -1
	for idx, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if start := resp.GetStartStream(); start != nil {
			switch start.GetStreamId() {
			case "route:d2|d1|audio_mix":
				sawMixD2 = true
			case "route:d3|d1|audio_mix":
				sawMixD3 = true
			}
		}
		if stop := resp.GetStopStream(); stop != nil {
			switch stop.GetStreamId() {
			case "route:d2|d1|audio_mix":
				sawMixStopD2 = true
			case "route:d3|d1|audio_mix":
				sawMixStopD3 = true
			}
		}
		if result := resp.GetCommandResult(); result != nil &&
			result.GetRequestId() == "all-cameras-focus-start" &&
			result.GetScenarioStart() == "multi_window" {
			focusStartIdx = idx
		}
	}

	if !sawMixD2 || !sawMixD3 {
		t.Fatalf("expected initial audio_mix start routes for d2 and d3")
	}
	if !sawMixStopD2 || !sawMixStopD3 {
		t.Fatalf("expected multi_window stop to emit stop_stream for both audio_mix routes")
	}
	if focusStartIdx == -1 {
		t.Fatalf("expected focused multi_window command_result")
	}

	sawFocusedAudio := false
	sawFocusedAudioMix := false
	for _, sent := range stream.sent[focusStartIdx+1:] {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		start := resp.GetStartStream()
		if start == nil {
			continue
		}
		if start.GetStreamId() == "route:d2|d1|audio" {
			sawFocusedAudio = true
		}
		if start.GetStreamId() == "route:d2|d1|audio_mix" || start.GetStreamId() == "route:d3|d1|audio_mix" {
			sawFocusedAudioMix = true
		}
	}
	if !sawFocusedAudio {
		t.Fatalf("expected focused audio start route route:d2|d1|audio")
	}
	if sawFocusedAudioMix {
		t.Fatalf("did not expect audio_mix start routes after focused restart")
	}
}

func TestGeneratedSessionMultiWindowSetUIAndFocusActionRouting(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
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
							DeviceId: "d1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "all-cameras-start",
						DeviceId:  "d1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "all cameras",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Input{
					Input: &iov1.InputEvent{
						DeviceId: "d1",
						Payload: &iov1.InputEvent_UiAction{
							UiAction: &iov1.UIAction{
								ComponentId: "multi_window_focus_d2",
								Action:      "multi_window_focus:d2",
							},
						},
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawGridColumns bool
	var sawFocusAction bool
	var sawFocusedLabel bool
	var sawEndAction bool
	walkNode := func(_ *uiv1.Node, _ func(*uiv1.Node)) {}
	walkNode = func(node *uiv1.Node, fn func(*uiv1.Node)) {
		if node == nil {
			return
		}
		fn(node)
		for _, child := range node.GetChildren() {
			walkNode(child, fn)
		}
	}

	scenarioStartCount := 0
	mixStopCount := 0
	focusAudioStartCount := 0
	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if result := resp.GetCommandResult(); result != nil && result.GetScenarioStart() == "multi_window" {
			scenarioStartCount++
		}
		if stop := resp.GetStopStream(); stop != nil {
			if stop.GetStreamId() == "route:d2|d1|audio_mix" || stop.GetStreamId() == "route:d3|d1|audio_mix" {
				mixStopCount++
			}
		}
		if start := resp.GetStartStream(); start != nil && start.GetStreamId() == "route:d2|d1|audio" {
			focusAudioStartCount++
		}
		if set := resp.GetSetUi(); set != nil {
			walkNode(set.GetRoot(), func(node *uiv1.Node) {
				propID := node.GetProps()["id"]
				if (propID == "multi_window_grid" || strings.HasSuffix(propID, "/multi_window_grid")) && node.GetGrid() != nil && node.GetGrid().GetColumns() == 2 {
					sawGridColumns = true
				}
				if button := node.GetButton(); button != nil {
					if button.GetAction() == "multi_window_end" {
						sawEndAction = true
					}
					if button.GetAction() == "multi_window_focus:d2" {
						sawFocusAction = true
					}
					if button.GetLabel() == "Hearing d2" {
						sawFocusedLabel = true
					}
				}
			})
		}
	}

	if !sawGridColumns {
		t.Fatalf("expected multi_window grid columns to be set to 2")
	}
	if !sawFocusAction {
		t.Fatalf("expected multi_window focus button action for d2")
	}
	if !sawEndAction {
		t.Fatalf("expected multi_window end button action")
	}
	if scenarioStartCount < 2 {
		t.Fatalf("expected two multi_window starts (voice + focus action), got %d", scenarioStartCount)
	}
	if mixStopCount < 2 {
		t.Fatalf("expected focus action to stop both audio_mix routes, got %d", mixStopCount)
	}
	if focusAudioStartCount == 0 {
		t.Fatalf("expected focus action to start focused audio route")
	}
	if !sawFocusedLabel {
		t.Fatalf("expected re-rendered UI with focused label Hearing d2")
	}
}

func TestGeneratedSessionMultiWindowEndActionRestoresPriorUIAndTransition(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
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
							DeviceId: "d1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "terminal-start",
						DeviceId:  "d1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
						Intent:    "terminal",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "multi-window-start",
						DeviceId:  "d1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "all cameras",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Input{
					Input: &iov1.InputEvent{
						DeviceId: "d1",
						Payload: &iov1.InputEvent_UiAction{
							UiAction: &iov1.UIAction{
								ComponentId: "multi_window_end",
								Action:      "multi_window_end",
							},
						},
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawMultiWindowStop bool
	var sawRestoredTerminalUI bool
	var sawTerminalEnterTransition bool
	var sawVideoStop bool
	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if result := resp.GetCommandResult(); result != nil && result.GetScenarioStop() == "multi_window" {
			sawMultiWindowStop = true
		}
		if set := resp.GetSetUi(); set != nil {
			if root := set.GetRoot(); root != nil &&
				(root.GetProps()["id"] == "terminal_root" || strings.HasSuffix(root.GetProps()["id"], "/terminal_root")) {
				sawRestoredTerminalUI = true
			}
		}
		if transition := resp.GetTransitionUi(); transition != nil && transition.GetTransition() == "terminal_enter" {
			sawTerminalEnterTransition = true
		}
		if stop := resp.GetStopStream(); stop != nil && stop.GetStreamId() == "route:d2|d1|video" {
			sawVideoStop = true
		}
	}

	if !sawMultiWindowStop {
		t.Fatalf("expected multi_window scenario stop from UI end action")
	}
	if !sawRestoredTerminalUI {
		t.Fatalf("expected restored terminal SetUI after multi-window end")
	}
	if !sawTerminalEnterTransition {
		t.Fatalf("expected terminal_enter transition restored after multi-window end")
	}
	if !sawVideoStop {
		t.Fatalf("expected video stop_stream for multi-window teardown")
	}
}

func TestGeneratedSessionInternalVideoCallStartSetUIAndHangupFlow(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
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
							DeviceId: "d1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "video-call-start",
						DeviceId:  "d1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
						Text:      "video call d2",
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Input{
					Input: &iov1.InputEvent{
						DeviceId: "d1",
						Payload: &iov1.InputEvent_UiAction{
							UiAction: &iov1.UIAction{
								ComponentId: "internal_video_call_hangup",
								Action:      "internal_video_call_end",
							},
						},
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var sawScenarioStart bool
	var sawScenarioStop bool
	var sawHangupAction bool
	var sawEnterTransition bool
	var sawExitTransition bool
	startStreams := map[string]bool{}
	stopStreams := map[string]bool{}
	walkNode := func(_ *uiv1.Node, _ func(*uiv1.Node)) {}
	walkNode = func(node *uiv1.Node, fn func(*uiv1.Node)) {
		if node == nil {
			return
		}
		fn(node)
		for _, child := range node.GetChildren() {
			walkNode(child, fn)
		}
	}

	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if result := resp.GetCommandResult(); result != nil {
			if result.GetScenarioStart() == "internal_video_call" {
				sawScenarioStart = true
			}
			if result.GetScenarioStop() == "internal_video_call" {
				sawScenarioStop = true
			}
		}
		if start := resp.GetStartStream(); start != nil {
			startStreams[start.GetStreamId()] = true
		}
		if stop := resp.GetStopStream(); stop != nil {
			stopStreams[stop.GetStreamId()] = true
		}
		if set := resp.GetSetUi(); set != nil {
			walkNode(set.GetRoot(), func(node *uiv1.Node) {
				if button := node.GetButton(); button != nil && button.GetAction() == "internal_video_call_end" {
					sawHangupAction = true
				}
			})
		}
		if transition := resp.GetTransitionUi(); transition != nil {
			if transition.GetTransition() == "internal_video_call_enter" {
				sawEnterTransition = true
			}
			if transition.GetTransition() == "internal_video_call_exit" {
				sawExitTransition = true
			}
		}
	}

	if !sawScenarioStart {
		t.Fatalf("expected internal_video_call scenario start")
	}
	if !sawScenarioStop {
		t.Fatalf("expected internal_video_call scenario stop from hangup action")
	}
	if !sawHangupAction {
		t.Fatalf("expected internal video call SetUI to include hangup action")
	}
	if !sawEnterTransition {
		t.Fatalf("expected internal_video_call_enter transition")
	}
	if !sawExitTransition {
		t.Fatalf("expected internal_video_call_exit transition")
	}
	for _, streamID := range []string{
		"route:d1|d2|audio",
		"route:d2|d1|audio",
		"route:d1|d2|video",
		"route:d2|d1|video",
	} {
		if !startStreams[streamID] {
			t.Fatalf("expected start_stream for %s", streamID)
		}
		if !stopStreams[streamID] {
			t.Fatalf("expected stop_stream for %s", streamID)
		}
	}
}
