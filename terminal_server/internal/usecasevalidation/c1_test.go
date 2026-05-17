package usecasevalidation_test

import (
	"context"
	"fmt"
	"testing"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/usecasevalidation"
)

// TestUseCaseC1WithEvidence wraps the existing C1 intercom transport coverage
// with evidence capture. It exercises the same production server code paths as
// the transport-level intercom tests and writes a manifest.json on every run.
func TestUseCaseC1WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)
	h.StartServer()

	// Pre-register the Hall device so the intercom scenario can route to it.
	_, _ = h.Devices.Register(device.Manifest{DeviceID: "device-2", DeviceName: "Hall"})

	stream := usecasevalidation.NewMemStream(context.Background(), []transport.ProtoClientEnvelope{
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
	})

	err := transport.RunProtoSession(h.NewStreamHandler(), h.Control, stream, transport.GeneratedProtoAdapter{})

	h.Assert("C1-no-session-error", "RunProtoSession returns nil", err == nil,
		fmt.Sprintf("err=%v", err))

	h.RecordInteraction("command", "Press intercom button or say \"intercom to kitchen\" from the Kitchen device.", "device-1")
	h.RecordInteraction("command", "End the intercom call (stop command).", "device-1")

	var sawRoute, sawStop bool
	for _, sent := range stream.Sent {
		resp, ok := sent.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if r := resp.GetRouteStream(); r != nil &&
			r.GetSourceDeviceId() == "device-1" &&
			r.GetTargetDeviceId() == "device-2" &&
			r.GetKind() == "audio" {
			sawRoute = true
		}
		if s := resp.GetStopStream(); s != nil &&
			s.GetStreamId() == "route:device-1|device-2|audio" {
			sawStop = true
		}
	}

	h.Assert("C1-route-stream", "intercom start emits audio route stream to peer",
		sawRoute, fmt.Sprintf("sent=%d messages", len(stream.Sent)))
	h.Assert("C1-stop-stream", "intercom stop emits stop stream for audio route",
		sawStop, fmt.Sprintf("sent=%d messages", len(stream.Sent)))

	h.CaptureFrame("C1-intercom-routed", "device-1", stream.Sent)

	h.Evidence("C1")
}
