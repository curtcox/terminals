package usecasevalidation_test

import (
	"context"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
	"github.com/curtcox/terminals/terminal_server/internal/usecasevalidation"
)

func TestDebugT1Events(t *testing.T) {
	h := usecasevalidation.New(t)
	startTime := time.Date(2026, 5, 16, 18, 0, 0, 0, time.UTC)
	h.Clock().SetNow(startTime)
	h.StartServer()
	h.Control.SetNowForTest(h.Clock().Now)

	const timerDuration = 10 * time.Minute

	kitchen := h.ConnectTerminal("kitchen", &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "kitchen",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
				},
			},
		},
	})
	if !kitchen.WaitForAny(2 * time.Second) {
		t.Fatal("timeout")
	}

	kitchen.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "voice-set-timer",
				DeviceId:  "kitchen",
				Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
				Text:      "set a timer for 10 minutes",
			},
		},
	})
	kitchen.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "timer_reminder"
	}, 2*time.Second)

	h.Clock().AdvanceTo(startTime.Add(timerDuration + time.Second))
	processed, _ := h.ProcessDueTimers(context.Background())
	t.Logf("processed=%d", processed)

	// Check MemoryHost events
	type eventer interface {
		Events() []ui.HostEvent
	}
	if ev, ok := h.Runtime.Env.UI.(eventer); ok {
		events := ev.Events()
		t.Logf("UI events count: %d", len(events))
		for i, e := range events {
			t.Logf("  event[%d]: kind=%s device=%s component=%s node.type=%s node.props=%v node.children=%d",
				i, e.Kind, e.DeviceID, e.ComponentID, e.Node.Type, e.Node.Props, len(e.Node.Children))
		}
	}

	// Now call CaptureHostFrame and check result
	h.CaptureHostFrame("T1-debug-done", "kitchen")
}
