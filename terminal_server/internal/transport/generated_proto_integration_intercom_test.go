package transport

import (
	"context"
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
