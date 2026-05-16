package usecasevalidation_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	iov1 "github.com/curtcox/terminals/terminal_server/gen/go/io/v1"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/usecasevalidation"
)

// TestUseCaseAA5WithEvidence validates the vision-analysis-agent use case:
// an external agent arms vision analysis via the server API, a camera
// activity marker fires, the vision analyzer is called, and the resulting
// caption and labels are broadcast back to the agent device. The agent can
// then surface the alert to users (overlays, annotations, notifications).
//
// AA5: Vision analysis agent (program) processes camera frames and generates
// alerts or annotations on the viewing device.
func TestUseCaseAA5WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)

	// Inject a fake vision analyzer that returns a pre-configured analysis.
	// This simulates a real vision backend without requiring actual image data.
	h.SetVision(&usecasevalidation.FakeVisionAnalyzer{
		Caption: "package at front door",
		Labels:  []string{"package", "door", "porch"},
	})
	h.StartServer()

	const waitTimeout = 2 * time.Second

	// The vision agent connects with a stable agent device ID.
	// In production this would be a persistent automation service.
	agent := h.ConnectTerminal("vision-agent", &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "vision-agent",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Vision Analysis Agent"},
				},
			},
		},
	})
	if !agent.WaitForAny(waitTimeout) {
		t.Fatal("vision-agent terminal: timed out waiting for session establishment")
	}

	// --- Step 1: agent arms vision analysis for the front door camera. ---
	agent.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "aa5-arm-vision",
				DeviceId:  "vision-agent",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "vision analysis",
				Arguments: map[string]string{
					"prompt":      "Identify packages, people, or vehicles at the front door",
					"cooldown_ms": "0",
				},
			},
		},
	})

	// Verify the vision_analysis scenario started.
	_, sawVisionArmed := agent.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "vision_analysis"
	}, waitTimeout)
	h.Assert("AA5-vision-armed", "vision agent armed vision analysis via manual API",
		sawVisionArmed,
		fmt.Sprintf("agent received %d messages", len(agent.Received())))

	// --- Step 2: camera activity is detected. ---
	// Inject a camera_activity sensor reading from the agent device to trigger
	// the vision analyzer without requiring physical camera hardware.
	frameMS := time.Now().UnixMilli()
	agent.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Sensor{
			Sensor: &iov1.SensorData{
				DeviceId: "vision-agent",
				UnixMs:   frameMS,
				Values:   map[string]float64{"camera_activity": 1.0},
			},
		},
	})

	// Wait for the vision analysis broadcast.
	deadline := time.Now().Add(waitTimeout)
	sawAnalysis := false
	var analysisMsg string
	for time.Now().Before(deadline) && !sawAnalysis {
		for _, ev := range h.Broadcast.Events() {
			if strings.Contains(ev.Message, "package at front door") {
				sawAnalysis = true
				analysisMsg = ev.Message
				break
			}
		}
		if !sawAnalysis {
			time.Sleep(10 * time.Millisecond)
		}
	}
	h.Assert("AA5-analysis-broadcast", "vision analysis result broadcast after camera activity",
		sawAnalysis,
		fmt.Sprintf("broadcast events: %d; last message seen: %q", len(h.Broadcast.Events()), analysisMsg))

	// --- Step 3: verify the broadcast is targeted at the vision agent. ---
	agentTargeted := false
	for _, ev := range h.Broadcast.Events() {
		if !strings.Contains(ev.Message, "package at front door") {
			continue
		}
		for _, id := range ev.DeviceIDs {
			if id == "vision-agent" {
				agentTargeted = true
				break
			}
		}
	}
	h.Assert("AA5-agent-targeted", "vision analysis broadcast is targeted at the vision agent device",
		agentTargeted,
		fmt.Sprintf("broadcast events: %d", len(h.Broadcast.Events())))

	// --- Step 4: verify labels are included in the broadcast message. ---
	h.Assert("AA5-labels-included", "broadcast message includes camera labels",
		strings.Contains(analysisMsg, "package") && strings.Contains(analysisMsg, "door"),
		fmt.Sprintf("message: %q", analysisMsg))

	// --- Step 5: zero activity is suppressed. ---
	beforeCount := len(h.Broadcast.Events())
	agent.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Sensor{
			Sensor: &iov1.SensorData{
				DeviceId: "vision-agent",
				UnixMs:   frameMS + 1000,
				Values:   map[string]float64{"camera_activity": 0.0},
			},
		},
	})
	time.Sleep(50 * time.Millisecond)
	h.Assert("AA5-zero-activity-suppressed", "zero camera_activity value does not trigger analysis",
		len(h.Broadcast.Events()) == beforeCount,
		fmt.Sprintf("broadcast events before=%d after=%d", beforeCount, len(h.Broadcast.Events())))

	if err := agent.Disconnect(); err != nil {
		t.Logf("vision-agent disconnect: %v", err)
	}

	h.Evidence("AA5")
}
