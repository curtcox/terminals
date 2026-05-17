package usecasevalidation_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/usecasevalidation"
)

// TestUseCaseM5WithEvidence validates the camera activity watch use case:
// a camera-equipped terminal arms monitoring with a time window, activity
// markers inside the window trigger alerts, and markers outside are suppressed
// — no real elapsed time required.
//
// Architecture note: sensor readings are dispatched to the active scenario for
// the reading device (see runtime.ProcessSensorReading). The child-room device
// therefore arms its own camera monitor so that its sensor readings route to
// the scenario. A parent-facing alert channel would require a broadcast policy
// change; that cross-device routing is a Phase 5 enhancement.
func TestUseCaseM5WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)

	// Pin synthetic time to a school-morning window: 7 AM – 8 AM.
	monitorDay := time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC)
	windowStart := monitorDay.Add(7 * time.Hour)
	windowEnd := monitorDay.Add(8 * time.Hour)

	h.Clock().SetNow(windowStart)
	h.StartServer()

	const waitTimeout = 2 * time.Second

	// Connect the child-room camera device that arms and reports camera activity.
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

	h.RecordInteraction("command", "Arm camera monitor on child-room device with window 7:00–8:00 AM.", "child-room")

	// Arm the camera monitor from the child-room device, restricted to the morning window.
	// The scenario's SourceID will be "child-room" so it processes readings from this device.
	childRoom.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "arm-camera-monitor",
				DeviceId:  "child-room",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "camera monitor",
				Arguments: map[string]string{
					"window_start_ms": strconv.FormatInt(windowStart.UnixMilli(), 10),
					"window_end_ms":   strconv.FormatInt(windowEnd.UnixMilli(), 10),
					"cooldown_ms":     "0",
				},
			},
		},
	})

	// Wait for camera_monitor scenario to confirm activation.
	_, sawMonitorActive := childRoom.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "camera_monitor"
	}, waitTimeout)
	h.Assert("M5-monitor-armed", "camera_monitor scenario started after arm command",
		sawMonitorActive,
		fmt.Sprintf("child-room received %d messages", len(childRoom.Received())))

	// --- Activity inside window: synthetic time T+15min (7:15 AM) ---
	// Camera activity at 7:15 AM is inside the 7–8 AM window and should fire.
	h.RecordInteraction("sensor", "Camera activity detected at 7:15 AM (inside monitored window — triggers alert).", "child-room")
	insideWindowMS := windowStart.Add(15 * time.Minute).UnixMilli()
	childRoom.Send(usecasevalidation.SensorDataRequest("child-room", insideWindowMS, map[string]float64{
		"camera_activity": 1.0,
	}))

	// Wait for the alert broadcast.
	time.Sleep(50 * time.Millisecond)

	broadcastEvents1 := h.Broadcast.Events()
	sawInsideAlert := len(broadcastEvents1) > 0
	h.Assert("M5-inside-window-alert", "camera activity inside window triggers alert",
		sawInsideAlert,
		fmt.Sprintf("broadcast events after inside reading: %d", len(broadcastEvents1)))

	// --- Activity outside window: synthetic time T+90min (8:30 AM, past window end) ---
	// Camera activity at 8:30 AM is outside the window and must be suppressed.
	outsideWindowMS := windowStart.Add(90 * time.Minute).UnixMilli()
	beforeCount := len(h.Broadcast.Events())
	childRoom.Send(usecasevalidation.SensorDataRequest("child-room", outsideWindowMS, map[string]float64{
		"camera_activity": 1.0,
	}))

	time.Sleep(50 * time.Millisecond)

	broadcastEvents2 := h.Broadcast.Events()
	h.Assert("M5-outside-window-suppressed", "camera activity outside window is suppressed",
		len(broadcastEvents2) == beforeCount,
		fmt.Sprintf("broadcast events before=%d after=%d", beforeCount, len(broadcastEvents2)))

	// --- Zero activity: should never trigger ---
	beforeCount2 := len(h.Broadcast.Events())
	childRoom.Send(usecasevalidation.SensorDataRequest("child-room", insideWindowMS, map[string]float64{
		"camera_activity": 0.0,
	}))

	time.Sleep(50 * time.Millisecond)
	h.Assert("M5-zero-activity-suppressed", "zero camera_activity value is suppressed",
		len(h.Broadcast.Events()) == beforeCount2,
		fmt.Sprintf("broadcast events before=%d after=%d", beforeCount2, len(h.Broadcast.Events())))

	if err := childRoom.Disconnect(); err != nil {
		t.Logf("child-room disconnect: %v", err)
	}

	h.Evidence("M5")
}

