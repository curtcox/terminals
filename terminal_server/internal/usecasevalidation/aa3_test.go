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

// TestUseCaseAA3WithEvidence validates the AI-agent voice interpretation use case:
// an ambiguous voice command that does not match any fixed trigger keyword is
// resolved by the LLM backend into a typed intent, and the corresponding
// scenario is activated. The test injects a FakeLLM so no real API call is made.
//
// AA3: AI agent (program) uses the LLM backend to interpret ambiguous voice
// commands and map them to scenarios.
func TestUseCaseAA3WithEvidence(t *testing.T) {
	h := usecasevalidation.New(t)

	// Inject a FakeLLM that resolves any query to an "announce" intent.
	// The JSON matches the llmIntentEnvelope expected by resolveVoiceIntentWithLLM.
	h.SetLLM(&usecasevalidation.FakeLLM{
		Response: `{"action":"announce","object":"dinner","slots":{},"scope":{"broadcast":true}}`,
	})
	h.StartServer()

	const waitTimeout = 2 * time.Second

	// The kitchen terminal utters an ambiguous command that has no matching
	// fixed-keyword trigger. The server falls through to LLM resolution.
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

	// A second terminal in the living room should receive the announcement route.
	living := h.ConnectTerminal("living", &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "living",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Living Room"},
				},
			},
		},
	})
	if !living.WaitForAny(waitTimeout) {
		t.Fatal("living terminal: timed out waiting for session establishment")
	}

	h.RecordInteraction("voice", "Say \"please broadcast a message that dinner is ready\" on the Kitchen device.", "kitchen")

	// --- Step 1: send ambiguous voice command. ---
	// "please broadcast a message that dinner is ready" does not match any
	// fixed keyword in ParseVoiceTrigger, so shouldResolveWithLLM returns true.
	kitchen.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "aa3-ambiguous-voice",
				DeviceId:  "kitchen",
				Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
				Text:      "please broadcast a message that dinner is ready",
			},
		},
	})

	// --- Step 2: verify the LLM-resolved intent started the announcement scenario. ---
	_, sawStart := kitchen.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "announcement"
	}, waitTimeout)
	h.Assert("AA3-llm-resolved-scenario",
		"LLM resolved ambiguous voice command to announcement scenario",
		sawStart,
		fmt.Sprintf("kitchen received %d messages", len(kitchen.Received())))

	// --- Step 3: verify the announcement route reached the living room. ---
	_, sawRoute := living.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		if !ok {
			return false
		}
		r := resp.GetRouteStream()
		return r != nil && r.GetKind() == "announcement_audio"
	}, waitTimeout)
	h.Assert("AA3-announcement-routed",
		"announcement audio route delivered to non-speaking terminal after LLM resolution",
		sawRoute,
		fmt.Sprintf("living received %d messages", len(living.Received())))

	h.CaptureFrame("AA3-llm-resolved", "kitchen", kitchen.Received())
	h.CaptureFrame("AA3-route-delivered", "living", living.Received())

	for _, term := range []*usecasevalidation.SimTerminal{kitchen, living} {
		if err := term.Disconnect(); err != nil {
			t.Logf("%s disconnect: %v", term.DeviceID, err)
		}
	}

	h.Evidence("AA3")
}
