package usecasevalidation_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/usecasevalidation"
)

// TestUseCaseT2WithEvidence validates the timer reminder use case using
// deterministic synthetic time: a timer is scheduled at a known future epoch,
// the fake clock is advanced past that point, and ProcessDueTimers fires the
// notification without any real elapsed time.
func TestUseCaseT2WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)

	// Pin the synthetic start time so fire times are deterministic.
	startTime := time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC)
	h.Clock().SetNow(startTime)

	h.StartServer()

	const waitTimeout = 2 * time.Second

	// Connect the terminal that will set and receive the timer.
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

	h.RecordInteraction("command", "Set a 5-minute \"pasta\" timer on the Kitchen device.", "kitchen")

	// Schedule a 5-minute timer with an explicit fire_unix_ms so the scenario
	// uses our synthetic time rather than time.Now().
	const timerDuration = 5 * time.Minute
	fireUnixMS := startTime.Add(timerDuration).UnixMilli()

	kitchen.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "timer-set",
				DeviceId:  "kitchen",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "set timer",
				Arguments: map[string]string{
					"duration_seconds": strconv.Itoa(int(timerDuration.Seconds())),
					"fire_unix_ms":     strconv.FormatInt(fireUnixMS, 10),
					"label":            "pasta",
				},
			},
		},
	})

	// Wait for the timer-set confirmation broadcast.
	_, sawSetNotification := kitchen.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "timer_reminder"
	}, waitTimeout)
	h.Assert("T2-timer-set", "timer_reminder scenario started after set timer command",
		sawSetNotification,
		fmt.Sprintf("kitchen received %d messages", len(kitchen.Received())))

	h.CaptureFrame("T2-timer-set", "kitchen", kitchen.Received())

	// At synthetic time T+0: no timer should fire yet.
	processed0, err := h.ProcessDueTimers(context.Background())
	h.Assert("T2-no-premature-fire", "no timers fire before the due time",
		err == nil && processed0 == 0,
		fmt.Sprintf("processed=%d err=%v", processed0, err))

	// Advance synthetic time past the fire point.
	h.Clock().AdvanceTo(startTime.Add(timerDuration + time.Second))

	// ProcessDueTimers at the advanced time: exactly one timer fires.
	processed1, err := h.ProcessDueTimers(context.Background())
	h.Assert("T2-timer-fired", "exactly one timer processed after clock advance",
		err == nil && processed1 == 1,
		fmt.Sprintf("processed=%d err=%v", processed1, err))

	// The broadcast should now include a "Timer done!" notification.
	events := h.Broadcast.Events()
	sawDone := false
	for _, ev := range events {
		if ev.Message == "Timer done!" {
			sawDone = true
			break
		}
	}
	h.Assert("T2-done-notification", "broadcast emits 'Timer done!' after timer fires",
		sawDone,
		fmt.Sprintf("broadcast events: %d", len(events)))

	h.CaptureHostFrame("T2-timer-done", "kitchen")

	// Disconnect cleanly.
	if err := kitchen.Disconnect(); err != nil {
		t.Logf("kitchen disconnect: %v", err)
	}

	h.Evidence("T2")
}
