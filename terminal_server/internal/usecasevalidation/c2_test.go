package usecasevalidation_test

import (
	"fmt"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/usecasevalidation"
)

// TestUseCaseC2WithEvidence validates the whole-house announcement use case:
// one speaking terminal triggers an announcement that fans out to three
// receiving terminals via audio routes, with no duplicate delivery.
func TestUseCaseC2WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)
	h.StartServer()

	const waitTimeout = 2 * time.Second

	registerMsg := func(deviceID, name string) transport.ProtoClientEnvelope {
		return &controlv1.ConnectRequest{
			Payload: &controlv1.ConnectRequest_Register{
				Register: &controlv1.RegisterDevice{
					Capabilities: &capabilitiesv1.DeviceCapabilities{
						DeviceId: deviceID,
						Identity: &capabilitiesv1.DeviceIdentity{DeviceName: name},
					},
				},
			},
		}
	}

	// Connect 1 speaker and 3 receivers.
	speaker := h.ConnectTerminal("speaker", registerMsg("speaker", "Kitchen"))
	hall := h.ConnectTerminal("hall", registerMsg("hall", "Hall"))
	living := h.ConnectTerminal("living", registerMsg("living", "Living Room"))
	bedroom := h.ConnectTerminal("bedroom", registerMsg("bedroom", "Bedroom"))

	// Wait for all sessions to be established before issuing the command.
	for _, term := range []*usecasevalidation.SimTerminal{speaker, hall, living, bedroom} {
		if !term.WaitForAny(waitTimeout) {
			t.Fatalf("terminal %s: timed out waiting for session establishment", term.DeviceID)
		}
	}

	h.RecordInteraction("command", "Say \"announce: dinner is ready\" or press the announce button.", "speaker")

	// Trigger announcement from speaker.
	speaker.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "announce-start",
				DeviceId:  "speaker",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "announcement",
			},
		},
	})

	// Wait for the speaker to see the scenario-start command result.
	_, sawStart := speaker.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "announcement"
	}, waitTimeout)
	h.Assert("C2-scenario-start", "announcement scenario started on speaker",
		sawStart, fmt.Sprintf("speaker received %d messages", len(speaker.Received())))

	// Each receiver should observe a RouteStream for announcement_audio.
	isAnnouncementRoute := func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		if !ok {
			return false
		}
		r := resp.GetRouteStream()
		return r != nil && r.GetKind() == "announcement_audio"
	}

	for _, term := range []*usecasevalidation.SimTerminal{hall, living, bedroom} {
		_, sawRoute := term.WaitFor(isAnnouncementRoute, waitTimeout)
		h.Assert(
			fmt.Sprintf("C2-receiver-%s-route", term.DeviceID),
			fmt.Sprintf("receiver %s sees announcement_audio RouteStream", term.DeviceID),
			sawRoute,
			fmt.Sprintf("%s received %d messages", term.DeviceID, len(term.Received())),
		)
	}

	// Verify no receiver gets duplicate announcement_audio RouteStream messages.
	for _, term := range []*usecasevalidation.SimTerminal{hall, living, bedroom} {
		count := 0
		for _, env := range term.Received() {
			resp, ok := env.(*controlv1.ConnectResponse)
			if !ok {
				continue
			}
			if r := resp.GetRouteStream(); r != nil && r.GetKind() == "announcement_audio" {
				count++
			}
		}
		h.Assert(
			fmt.Sprintf("C2-no-duplicate-%s", term.DeviceID),
			fmt.Sprintf("receiver %s gets exactly one announcement_audio RouteStream", term.DeviceID),
			count <= 1,
			fmt.Sprintf("got %d announcement_audio RouteStream messages", count),
		)
	}

	// Verify the broadcaster recorded notifications for all receiver device IDs.
	broadcastEvents := h.Broadcast.Events()
	receiverIDs := map[string]bool{"hall": false, "living": false, "bedroom": false}
	for _, ev := range broadcastEvents {
		for _, id := range ev.DeviceIDs {
			if _, ok := receiverIDs[id]; ok {
				receiverIDs[id] = true
			}
		}
	}
	for id, notified := range receiverIDs {
		h.Assert(
			fmt.Sprintf("C2-broadcast-%s", id),
			fmt.Sprintf("receiver %s received broadcast notification", id),
			notified,
			fmt.Sprintf("broadcast events: %d", len(broadcastEvents)),
		)
	}

	h.CaptureFrame("C2-announcement-routes-sent", "hall", hall.Received())

	// Disconnect all terminals cleanly.
	for _, term := range []*usecasevalidation.SimTerminal{speaker, hall, living, bedroom} {
		if err := term.Disconnect(); err != nil {
			t.Logf("terminal %s disconnect: %v", term.DeviceID, err)
		}
	}

	h.Evidence("C2")
}
