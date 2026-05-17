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

// TestUseCaseT3T4WithEvidence validates the school-morning routine use cases:
//
//   - T3: parent arms a camera-based morning monitor; if the child is not seen
//     by the alert time, the parent is notified ("child is running late").
//   - T4: the system warns the child at a configured time ("The bus comes in 10 minutes").
//
// Synthetic time is used throughout so the full morning window runs in milliseconds.
func TestUseCaseT3T4WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)

	// Synthetic morning: 7:00 AM school day.
	monitorDay := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	windowStart := monitorDay.Add(7 * time.Hour)
	alertTime := monitorDay.Add(7*time.Hour + 30*time.Minute)   // alert if no activity by 7:30
	warningTime := monitorDay.Add(7*time.Hour + 50*time.Minute) // warn child at 7:50

	h.Clock().SetNow(windowStart)
	h.StartServer()

	const waitTimeout = 2 * time.Second

	// Connect the parent terminal that arms monitoring and receives alerts.
	parent := h.ConnectTerminal("parent", &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "parent",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Parent Bedroom"},
				},
			},
		},
	})
	if !parent.WaitForAny(waitTimeout) {
		t.Fatal("parent terminal: timed out waiting for session establishment")
	}

	// Connect the child-room terminal that receives the bus warning.
	childRoom := h.ConnectTerminal("child-room", &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "child-room",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Child Room"},
				},
			},
		},
	})
	if !childRoom.WaitForAny(waitTimeout) {
		t.Fatal("child-room terminal: timed out waiting for session establishment")
	}

	h.RecordInteraction("command", "Parent arms school morning monitor: alert at 7:30 AM if child not seen, warn child at 7:50 AM (bus in 10 min).", "parent")

	// Parent arms morning routine monitor (T3): alert at 7:30, warn child at 7:50.
	parent.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "arm-morning-routine",
				DeviceId:  "parent",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "morning routine monitor",
				Arguments: map[string]string{
					"alert_time_ms":     strconv.FormatInt(alertTime.UnixMilli(), 10),
					"warning_time_ms":   strconv.FormatInt(warningTime.UnixMilli(), 10),
					"alert_device_id":   "parent",
					"warning_device_id": "child-room",
					"alert_message":     "Morning routine: no activity detected",
					"warning_message":   "The bus comes in 10 minutes",
				},
			},
		},
	})

	// Confirm monitor activation.
	_, sawMonitorActive := parent.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil &&
			resp.GetCommandResult().GetScenarioStart() == "morning_routine_monitor"
	}, waitTimeout)
	h.Assert("T3-monitor-armed", "morning_routine_monitor scenario started after arm command",
		sawMonitorActive,
		fmt.Sprintf("parent received %d messages", len(parent.Received())))

	h.CaptureFrame("T3-monitor-armed", "parent", parent.Received())

	// No camera activity sent — child is not yet up.

	// Advance clock to alert time and fire due jobs.
	h.Clock().AdvanceTo(alertTime.Add(time.Second))
	ctx := context.Background()

	processed, err := h.ProcessDueTimers(ctx)
	h.Assert("T3-alert-fired", "morning_routine.alert job processed at alert time",
		err == nil && processed >= 1,
		fmt.Sprintf("processed=%d err=%v", processed, err))

	// Verify parent broadcast received the "no activity" alert (T3).
	events := h.Broadcast.Events()
	sawParentAlert := false
	for _, ev := range events {
		if ev.Message == "Morning routine: no activity detected" {
			sawParentAlert = true
			break
		}
	}
	h.Assert("T3-parent-notified", "parent notified that child has not been seen",
		sawParentAlert,
		fmt.Sprintf("broadcast events after alert time: %d", len(events)))

	// Advance clock to warning time and fire due jobs (T4).
	h.Clock().AdvanceTo(warningTime.Add(time.Second))

	processed2, err := h.ProcessDueTimers(ctx)
	h.Assert("T4-warning-fired", "morning_routine.warning job processed at warning time",
		err == nil && processed2 >= 1,
		fmt.Sprintf("processed=%d err=%v", processed2, err))

	// Verify child-room broadcast received the bus warning (T4).
	events2 := h.Broadcast.Events()
	sawChildWarning := false
	for _, ev := range events2 {
		if ev.Message == "The bus comes in 10 minutes" {
			sawChildWarning = true
			break
		}
	}
	h.Assert("T4-child-warned", "child-room notified 'The bus comes in 10 minutes'",
		sawChildWarning,
		fmt.Sprintf("broadcast events after warning time: %d", len(events2)))

	h.CaptureFrame("T3-parent-alert", "parent", parent.Received())
	h.CaptureFrame("T4-child-warned", "child-room", childRoom.Received())

	if err := parent.Disconnect(); err != nil {
		t.Logf("parent disconnect: %v", err)
	}
	if err := childRoom.Disconnect(); err != nil {
		t.Logf("child-room disconnect: %v", err)
	}

	h.Evidence("T3/T4")
}

// TestUseCaseT3ActivityCancelsAlert validates that when camera activity IS
// detected before the alert time, the parent alert is suppressed (child is up).
func TestUseCaseT3ActivityCancelsAlert(t *testing.T) {
	h := usecasevalidation.New(t)

	monitorDay := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	windowStart := monitorDay.Add(7 * time.Hour)
	alertTime := monitorDay.Add(7*time.Hour + 30*time.Minute)

	h.Clock().SetNow(windowStart)
	h.StartServer()

	const waitTimeout = 2 * time.Second

	parent := h.ConnectTerminal("parent2", &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "parent2",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Parent Bedroom"},
				},
			},
		},
	})
	if !parent.WaitForAny(waitTimeout) {
		t.Fatal("parent terminal: timed out waiting for session establishment")
	}

	childRoom := h.ConnectTerminal("child-room2", &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "child-room2",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Child Room"},
				},
			},
		},
	})
	if !childRoom.WaitForAny(waitTimeout) {
		t.Fatal("child-room terminal: timed out waiting for session establishment")
	}

	parent.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "arm-morning-routine2",
				DeviceId:  "parent2",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "morning routine monitor",
				Arguments: map[string]string{
					"alert_time_ms":   strconv.FormatInt(alertTime.UnixMilli(), 10),
					"alert_device_id": "parent2",
					"alert_message":   "Morning routine: no activity detected",
				},
			},
		},
	})

	_, sawMonitorActive := parent.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil &&
			resp.GetCommandResult().GetScenarioStart() == "morning_routine_monitor"
	}, waitTimeout)
	h.Assert("T3-cancel-armed", "morning_routine_monitor started",
		sawMonitorActive,
		fmt.Sprintf("parent received %d messages", len(parent.Received())))

	h.CaptureFrame("T3-monitor-armed", "parent2", parent.Received())

	h.RecordInteraction("sensor", "Camera activity detected in child's room at 7:15 AM (child is up).", "child-room2")

	// Child gets up at 7:15 — camera activity detected before alert time.
	activityTimeMS := windowStart.Add(15 * time.Minute).UnixMilli()
	childRoom.Send(usecasevalidation.SensorDataRequest("child-room2", activityTimeMS, map[string]float64{
		"camera_activity": 1.0,
	}))

	// Small yield to let the sensor reading be processed.
	time.Sleep(50 * time.Millisecond)

	// Advance clock to alert time — alert should have been cancelled.
	h.Clock().AdvanceTo(alertTime.Add(time.Second))
	beforeCount := len(h.Broadcast.Events())

	processed, err := h.ProcessDueTimers(context.Background())
	h.Assert("T3-no-alert-when-active", "no alert fires when camera activity was seen",
		err == nil && processed == 0,
		fmt.Sprintf("processed=%d err=%v", processed, err))

	events := h.Broadcast.Events()
	sawSpuriousAlert := false
	for _, ev := range events[beforeCount:] {
		if ev.Message == "Morning routine: no activity detected" {
			sawSpuriousAlert = true
			break
		}
	}
	h.Assert("T3-alert-suppressed", "no 'no activity' alert when child was seen active",
		!sawSpuriousAlert,
		fmt.Sprintf("new broadcast events after alert time: %d", len(events)-beforeCount))

	h.CaptureFrame("T3-alert-suppressed", "parent2", parent.Received())

	_ = parent.Disconnect()
	_ = childRoom.Disconnect()

	h.Evidence("T3")
}
