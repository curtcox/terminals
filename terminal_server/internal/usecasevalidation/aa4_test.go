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

// TestUseCaseAA4WithEvidence validates the scheduling-agent use case:
// an external automation agent creates a timer via the manual-command API,
// then cancels it before it fires. After cancellation, advancing the clock
// past the due time must produce no "Timer done!" broadcast.
//
// AA4: Scheduling agent (program) creates, modifies, and cancels timers and
// reminders via the server API.
func TestUseCaseAA4WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)

	startTime := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	h.Clock().SetNow(startTime)
	h.StartServer()

	const waitTimeout = 2 * time.Second
	const timerDuration = 10 * time.Minute

	// The automation agent uses a dedicated device ID distinct from human users.
	agent := h.ConnectTerminal("automation-agent", &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "automation-agent",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Automation Agent"},
				},
			},
		},
	})
	if !agent.WaitForAny(waitTimeout) {
		t.Fatal("automation-agent terminal: timed out waiting for session establishment")
	}

	fireUnixMS := startTime.Add(timerDuration).UnixMilli()

	// --- Step 1: agent creates a timer via COMMAND_KIND_MANUAL (API call). ---
	agent.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "aa4-create-timer",
				DeviceId:  "automation-agent",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "set timer",
				Arguments: map[string]string{
					"duration_seconds": strconv.Itoa(int(timerDuration.Seconds())),
					"fire_unix_ms":     strconv.FormatInt(fireUnixMS, 10),
					"label":            "meeting-reminder",
				},
			},
		},
	})

	_, sawTimerSet := agent.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "timer_reminder"
	}, waitTimeout)
	h.Assert("AA4-timer-created", "scheduling agent created a timer via manual API",
		sawTimerSet,
		fmt.Sprintf("agent received %d messages", len(agent.Received())))

	// --- Step 2: verify timer is pending (would fire if clock advanced). ---
	processedBefore, err := h.ProcessDueTimers(context.Background())
	h.Assert("AA4-no-premature-fire", "timer does not fire before due time",
		err == nil && processedBefore == 0,
		fmt.Sprintf("processed=%d err=%v", processedBefore, err))

	broadcastCountBefore := len(h.Broadcast.Events())

	// --- Step 3: agent cancels the timer via COMMAND_KIND_MANUAL. ---
	agent.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "aa4-cancel-timer",
				DeviceId:  "automation-agent",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "cancel timer",
			},
		},
	})

	// Wait briefly for the cancel command to be processed, then check broadcast.
	time.Sleep(50 * time.Millisecond)

	cancelEvents := h.Broadcast.Events()
	sawCancelled := false
	for _, ev := range cancelEvents {
		if ev.Message == "Timer cancelled" {
			sawCancelled = true
			break
		}
	}
	h.Assert("AA4-timer-cancelled", "scheduling agent cancelled the timer via manual API",
		sawCancelled,
		fmt.Sprintf("broadcast events after cancel: %d", len(cancelEvents)))

	// --- Step 4: advance clock past due time — no "Timer done!" should fire. ---
	h.Clock().AdvanceTo(startTime.Add(timerDuration + time.Second))

	processedAfter, err := h.ProcessDueTimers(context.Background())
	h.Assert("AA4-no-fire-after-cancel", "no timer fires after cancellation",
		err == nil && processedAfter == 0,
		fmt.Sprintf("processed=%d err=%v", processedAfter, err))

	events := h.Broadcast.Events()
	sawTimerDone := false
	for _, ev := range events[broadcastCountBefore:] {
		if ev.Message == "Timer done!" {
			sawTimerDone = true
			break
		}
	}
	h.Assert("AA4-done-suppressed", "cancelled timer does not produce 'Timer done!' broadcast",
		!sawTimerDone,
		fmt.Sprintf("new broadcast events after clock advance: %d", len(events)-broadcastCountBefore))

	if err := agent.Disconnect(); err != nil {
		t.Logf("automation-agent disconnect: %v", err)
	}

	h.Evidence("AA4")
}

// TestUseCaseAA1WithEvidence validates the automation-agent trigger use case:
// an external agent (program) triggers a server scenario via the manual-command
// API, based on a simulated external event, and the system responds correctly.
//
// AA1: Automation agent (program) triggers scenarios via the server API based
// on external events (calendar, webhook, sensor).
func TestUseCaseAA1WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)
	h.StartServer()

	const waitTimeout = 2 * time.Second

	// An automation agent connects with a well-known agent device ID.
	agent := h.ConnectTerminal("webhook-agent", &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "webhook-agent",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Webhook Automation Agent"},
				},
			},
		},
	})
	if !agent.WaitForAny(waitTimeout) {
		t.Fatal("webhook-agent terminal: timed out waiting for session establishment")
	}

	// Connect a display terminal that will receive the announcement.
	display := h.ConnectTerminal("living-room", &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "living-room",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Living Room"},
				},
			},
		},
	})
	if !display.WaitForAny(waitTimeout) {
		t.Fatal("living-room terminal: timed out waiting for session establishment")
	}

	// The webhook agent triggers an announcement via COMMAND_KIND_MANUAL.
	// In production this would be initiated by a calendar event, webhook, or sensor.
	agent.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "aa1-webhook-trigger",
				DeviceId:  "webhook-agent",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "announce",
			},
		},
	})

	// Verify the announcement scenario started on the agent terminal.
	_, sawScenarioStart := agent.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "announcement"
	}, waitTimeout)
	h.Assert("AA1-scenario-triggered", "external agent triggered announcement scenario via manual API",
		sawScenarioStart,
		fmt.Sprintf("agent received %d messages", len(agent.Received())))

	// Verify the display terminal received an announcement audio route — this is
	// the observable system effect of the agent's API trigger.
	_, sawRoute := display.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		if !ok {
			return false
		}
		r := resp.GetRouteStream()
		return r != nil && r.GetKind() == "announcement_audio"
	}, waitTimeout)
	h.Assert("AA1-route-delivered", "display terminal received announcement_audio route from agent trigger",
		sawRoute,
		fmt.Sprintf("display received %d messages", len(display.Received())))

	_ = agent.Disconnect()
	_ = display.Disconnect()

	h.Evidence("AA1")
}
