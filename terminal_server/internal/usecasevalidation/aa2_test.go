package usecasevalidation_test

import (
	"fmt"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
	"github.com/curtcox/terminals/terminal_server/internal/usecasevalidation"
)

// TestUseCaseAA2WithEvidence validates the monitoring-agent use case:
// an external monitoring agent arms audio monitoring via the server API, a sound
// classification event fires (dryer beep), and the server broadcasts the
// notification targeted at the agent's device ID. The agent can then route this
// notification to external systems (Slack, email, etc.) — the test proves the
// server-side notification was generated with the correct device target and message.
//
// AA2: Monitoring agent (program) subscribes to sound classification events and
// routes notifications to other systems (Slack, email).
func TestUseCaseAA2WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)

	// Inject a fake sound classifier that immediately emits a dryer_beep event.
	// This simulates the server's audio classifier detecting the target sound
	// without requiring a real audio device or elapsed time.
	h.SetSound(&usecasevalidation.FakeSoundClassifier{
		Events: []scenario.SoundEvent{
			{Label: "dryer_beep", Confidence: 0.91, AtMS: 1000},
		},
	})
	h.StartServer()

	const waitTimeout = 2 * time.Second

	// The monitoring agent connects with a stable agent device ID.
	// In production this would be a persistent automation service.
	agent := h.ConnectTerminal("monitoring-agent", &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "monitoring-agent",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Sound Monitoring Agent"},
				},
			},
		},
	})
	if !agent.WaitForAny(waitTimeout) {
		t.Fatal("monitoring-agent terminal: timed out waiting for session establishment")
	}

	// --- Step 1: agent arms audio monitoring for the dryer target. ---
	// In production this would be triggered by configuration or an external event.
	agent.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "aa2-arm-monitor",
				DeviceId:  "monitoring-agent",
				Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
				Intent:    "audio monitor",
				Arguments: map[string]string{
					"target": "dryer",
				},
			},
		},
	})

	// Verify the audio_monitor scenario started.
	_, sawMonitorArmed := agent.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "audio_monitor"
	}, waitTimeout)
	h.Assert("AA2-monitor-armed", "monitoring agent armed audio monitoring via manual API",
		sawMonitorArmed,
		fmt.Sprintf("agent received %d messages", len(agent.Received())))

	// --- Step 2: wait for the dryer_beep classification event to be broadcast. ---
	// The fake sound classifier emits the event immediately after arming; the
	// server broadcasts "Audio monitor detected: dryer_beep" targeted at the agent.
	deadline := time.Now().Add(waitTimeout)
	sawDetection := false
	var detectionEvent ui.BroadcastEvent
	for time.Now().Before(deadline) && !sawDetection {
		for _, ev := range h.Broadcast.Events() {
			if ev.Message == "Audio monitor detected: dryer_beep" {
				sawDetection = true
				detectionEvent = ev
				break
			}
		}
		if !sawDetection {
			time.Sleep(10 * time.Millisecond)
		}
	}
	h.Assert("AA2-detection-broadcast", "dryer_beep classification event broadcast to agent",
		sawDetection,
		fmt.Sprintf("broadcast events: %d", len(h.Broadcast.Events())))

	// --- Step 3: verify the broadcast is directed to the monitoring agent. ---
	// The notification must target the agent's device ID so it can forward to
	// external systems (Slack, email) on behalf of the correct source.
	agentTargeted := false
	for _, id := range detectionEvent.DeviceIDs {
		if id == "monitoring-agent" {
			agentTargeted = true
			break
		}
	}
	h.Assert("AA2-agent-targeted", "detection broadcast is targeted at the monitoring agent device",
		agentTargeted,
		fmt.Sprintf("broadcast device IDs: %v", detectionEvent.DeviceIDs))

	if err := agent.Disconnect(); err != nil {
		t.Logf("monitoring-agent disconnect: %v", err)
	}

	h.Evidence("AA2")
}
