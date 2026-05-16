package usecasevalidation_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/usecasevalidation"
)

// TestUseCaseT1WithEvidence validates the cook-sets-a-timer use case via the
// voice path: the simulated terminal says "set a timer for 10 minutes",
// synthetic time advances past the due point, and ProcessDueTimers fires the
// notification — no real elapsed time required.
func TestUseCaseT1WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)

	// Pin the synthetic start time so fire times are deterministic.
	startTime := time.Date(2026, 5, 16, 18, 0, 0, 0, time.UTC)
	h.Clock().SetNow(startTime)

	// Inject the fake clock into ControlService so voice-command parsing uses
	// synthetic time when computing fire_unix_ms.
	h.StartServer()
	h.Control.SetNowForTest(h.Clock().Now)

	const (
		waitTimeout   = 2 * time.Second
		timerDuration = 10 * time.Minute
	)

	// Connect the kitchen terminal that will speak and receive the timer.
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
	if !kitchen.WaitForAny(waitTimeout) {
		t.Fatal("kitchen terminal: timed out waiting for session establishment")
	}

	// Send voice command through the voice parser — same path as a real terminal.
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

	// Wait for the timer_reminder scenario to start.
	_, sawScenarioStart := kitchen.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "timer_reminder"
	}, waitTimeout)
	h.Assert("T1-timer-set-voice", "timer_reminder scenario started via voice command",
		sawScenarioStart,
		fmt.Sprintf("kitchen received %d messages", len(kitchen.Received())))

	// At synthetic time T+0 no timer should fire yet.
	processed0, err := h.ProcessDueTimers(context.Background())
	h.Assert("T1-no-premature-fire", "no timers fire before the due time",
		err == nil && processed0 == 0,
		fmt.Sprintf("processed=%d err=%v", processed0, err))

	// Advance synthetic time past the 10-minute fire point.
	h.Clock().AdvanceTo(startTime.Add(timerDuration + time.Second))

	// Fire due timers: exactly one should fire.
	processed1, err := h.ProcessDueTimers(context.Background())
	h.Assert("T1-timer-fired", "exactly one timer processed after clock advance",
		err == nil && processed1 == 1,
		fmt.Sprintf("processed=%d err=%v", processed1, err))

	// The broadcast should include a "Timer done!" notification.
	events := h.Broadcast.Events()
	sawDone := false
	for _, ev := range events {
		if ev.Message == "Timer done!" {
			sawDone = true
			break
		}
	}
	h.Assert("T1-done-notification", "broadcast emits 'Timer done!' after timer fires",
		sawDone,
		fmt.Sprintf("broadcast events: %d", len(events)))

	if err := kitchen.Disconnect(); err != nil {
		t.Logf("kitchen disconnect: %v", err)
	}

	h.Evidence("T1")
}
