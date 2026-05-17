package usecasevalidation_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	iov1 "github.com/curtcox/terminals/terminal_server/gen/go/io/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
	"github.com/curtcox/terminals/terminal_server/internal/usecasevalidation"
)

// TestUseCaseUI9WithEvidence validates that a reconnecting terminal has its
// main UI layer, overlay, and active stream routes replayed on the new session.
// This wraps the transport-level RECON-1 coverage with harness evidence capture.
func TestUseCaseUI9WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)
	h.StartServer()

	// Register a peer so intercom has a target to route to.
	_, _ = h.Devices.Register(device.Manifest{DeviceID: "device-2", DeviceName: "Hall"})

	overlaySuffix := "/" + ui.GlobalOverlayComponentID

	// --- First session: connect, start intercom, open overlay ---
	stream1 := usecasevalidation.NewMemStream(context.Background(), []transport.ProtoClientEnvelope{
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
	})

	err := transport.RunProtoSession(h.NewStreamHandler(), h.Control, stream1, transport.GeneratedProtoAdapter{})
	h.Assert("UI9-session1-no-error", "first session RunProtoSession returns nil", err == nil,
		fmt.Sprintf("err=%v", err))

	var sawScenarioStart, sawOverlay bool
	for _, sent := range stream1.Sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "intercom" {
			sawScenarioStart = true
		}
		if upd := resp.GetUpdateUi(); upd != nil &&
			(upd.ComponentId == ui.GlobalOverlayComponentID || strings.HasSuffix(upd.ComponentId, overlaySuffix)) &&
			upd.Node != nil && len(upd.Node.Children) > 0 {
			sawOverlay = true
		}
	}
	h.Assert("UI9-session1-scenario-start", "intercom started in first session",
		sawScenarioStart, fmt.Sprintf("sent=%d messages", len(stream1.Sent)))
	h.Assert("UI9-session1-overlay", "corner overlay opened in first session",
		sawOverlay, fmt.Sprintf("sent=%d messages", len(stream1.Sent)))

	// --- Second session: reconnect with generation+1 ---
	stream2 := usecasevalidation.NewMemStream(context.Background(), []transport.ProtoClientEnvelope{
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
	})

	err = transport.RunProtoSession(h.NewStreamHandler(), h.Control, stream2, transport.GeneratedProtoAdapter{})
	h.Assert("UI9-session2-no-error", "reconnect session RunProtoSession returns nil", err == nil,
		fmt.Sprintf("err=%v", err))

	var sawReplaySetUI, sawReplayOverlay, sawReplayStartStream, sawReplayRouteStream bool
	for _, sent := range stream2.Sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if resp.GetSetUi() != nil {
			sawReplaySetUI = true
		}
		if upd := resp.GetUpdateUi(); upd != nil &&
			(upd.ComponentId == ui.GlobalOverlayComponentID || strings.HasSuffix(upd.ComponentId, overlaySuffix)) &&
			upd.Node != nil && len(upd.Node.Children) > 0 {
			sawReplayOverlay = true
		}
		if resp.GetStartStream() != nil {
			sawReplayStartStream = true
		}
		if resp.GetRouteStream() != nil {
			sawReplayRouteStream = true
		}
	}

	h.RecordInteraction("command", "Connect Kitchen device (first session) and start intercom + open overlay.", "device-1")
	h.RecordInteraction("command", "Disconnect and reconnect the Kitchen device (simulate drop/rejoin).", "device-1")

	h.CaptureFrame("UI9-reconnect-replay", "device-1", stream2.Sent)

	h.Assert("UI9-replay-set-ui", "SetUI replayed on reconnect",
		sawReplaySetUI, fmt.Sprintf("sent=%d messages", len(stream2.Sent)))
	h.Assert("UI9-replay-overlay", "overlay UpdateUI replayed on reconnect",
		sawReplayOverlay, fmt.Sprintf("sent=%d messages", len(stream2.Sent)))
	h.Assert("UI9-replay-start-stream", "StartStream replayed on reconnect",
		sawReplayStartStream, fmt.Sprintf("sent=%d messages", len(stream2.Sent)))
	h.Assert("UI9-replay-route-stream", "RouteStream replayed on reconnect",
		sawReplayRouteStream, fmt.Sprintf("sent=%d messages", len(stream2.Sent)))

	h.Evidence("UI9")
}
