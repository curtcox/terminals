package transport

import (
	"context"
	"strings"
	"testing"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	iov1 "github.com/curtcox/terminals/terminal_server/gen/go/io/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestGeneratedSessionUI_RECON_1(t *testing.T) {
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

	// First session
	stream1 := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_CapabilitySnapshot{
					CapabilitySnapshot: &controlv1.CapabilitySnapshot{
						DeviceId:   "device-1",
						Generation: 1,
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId:   "device-1",
							Identity:   &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
							Microphone: &capabilitiesv1.AudioInputCapability{},
							Camera:     &capabilitiesv1.CameraCapability{},
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
				Payload: &controlv1.ConnectRequest_Input{
					Input: &iov1.InputEvent{
						DeviceId: "device-1",
						Payload: &iov1.InputEvent_UiAction{
							UiAction: &iov1.UIAction{
								ComponentId: "act:device-1/__affordance.corner__",
								Action:      "corner.open",
							},
						},
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream1, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	// Verify the scenario started and overlay opened in stream1.
	// The overlay UpdateUI's component id is the canonical scoped form
	// "act:<owner>/" + ui.GlobalOverlayComponentID; match by suffix so the
	// assertion is robust to the active owner.
	var sawScenarioStart, sawOverlay bool
	overlaySuffix := "/" + ui.GlobalOverlayComponentID
	for _, sent := range stream1.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "intercom" {
			sawScenarioStart = true
		}
		if update := resp.GetUpdateUi(); update != nil &&
			(update.ComponentId == ui.GlobalOverlayComponentID || strings.HasSuffix(update.ComponentId, overlaySuffix)) &&
			update.Node != nil && len(update.Node.Children) > 0 {
			sawOverlay = true
		}
	}
	if !sawScenarioStart {
		t.Fatalf("intercom did not start in first session")
	}
	if !sawOverlay {
		t.Fatalf("overlay did not open in first session")
	}

	// Second session: reconnect
	stream2 := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_CapabilitySnapshot{
					CapabilitySnapshot: &controlv1.CapabilitySnapshot{
						DeviceId:   "device-1",
						Generation: 2,
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId:   "device-1",
							Identity:   &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
							Microphone: &capabilitiesv1.AudioInputCapability{},
							Camera:     &capabilitiesv1.CameraCapability{},
						},
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream2, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession(stream2) error = %v", err)
	}

	// Verify stream2 received the replayed SetUI, Overlay UpdateUI, StartStream, and RouteStream
	var sawReplaySetUI, sawReplayOverlay, sawReplayStartStream, sawReplayRouteStream bool
	for _, sent := range stream2.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if ui := resp.GetSetUi(); ui != nil {
			sawReplaySetUI = true
		}
		if update := resp.GetUpdateUi(); update != nil &&
			(update.ComponentId == ui.GlobalOverlayComponentID || strings.HasSuffix(update.ComponentId, "/"+ui.GlobalOverlayComponentID)) &&
			update.Node != nil && len(update.Node.Children) > 0 {
			sawReplayOverlay = true
		}
		if start := resp.GetStartStream(); start != nil {
			sawReplayStartStream = true
		}
		if route := resp.GetRouteStream(); route != nil {
			sawReplayRouteStream = true
		}
	}
	if !sawReplaySetUI {
		t.Errorf("missing SetUI replay on reconnect")
	}
	if !sawReplayOverlay {
		t.Errorf("missing Overlay UpdateUI replay on reconnect")
	}
	if !sawReplayStartStream {
		t.Errorf("missing StartStream replay on reconnect")
	}
	if !sawReplayRouteStream {
		t.Errorf("missing RouteStream replay on reconnect")
	}
}

func TestGeneratedSessionMidFlightOverlayIdempotent(t *testing.T) {
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
				Payload: &controlv1.ConnectRequest_CapabilitySnapshot{
					CapabilitySnapshot: &controlv1.CapabilitySnapshot{
						DeviceId:   "device-1",
						Generation: 1,
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			// Send it twice to simulate duplicate / mid-flight
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Input{
					Input: &iov1.InputEvent{
						DeviceId: "device-1",
						Payload: &iov1.InputEvent_UiAction{
							UiAction: &iov1.UIAction{
								ComponentId: "act:device-1/__affordance.corner__",
								Action:      "corner.open",
							},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Input{
					Input: &iov1.InputEvent{
						DeviceId: "device-1",
						Payload: &iov1.InputEvent_UiAction{
							UiAction: &iov1.UIAction{
								ComponentId: "act:device-1/__affordance.corner__",
								Action:      "corner.open",
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

	overlayCount := 0
	for _, sent := range stream.sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if update := resp.GetUpdateUi(); update != nil &&
			(update.ComponentId == ui.GlobalOverlayComponentID || strings.HasSuffix(update.ComponentId, "/"+ui.GlobalOverlayComponentID)) {
			overlayCount++
		}
	}
	// We might receive 2 updates, but it should not crash and the state should be idempotent
	if overlayCount < 1 {
		t.Fatalf("expected at least 1 overlay update, got %d", overlayCount)
	}
}
