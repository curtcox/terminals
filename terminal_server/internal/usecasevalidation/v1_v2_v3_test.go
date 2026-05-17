package usecasevalidation_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/usecasevalidation"
)

// voiceAssistantHarness starts a server with a FakeLLM, connects a single
// terminal, sends the given voice command text, and returns the terminal and
// the fake LLM so callers can assert on both the terminal output and LLM calls.
func voiceAssistantHarness(t *testing.T, deviceID, voiceText, llmResponse string) (
	h *usecasevalidation.Harness,
	term *usecasevalidation.SimTerminal,
	fakeLLM *usecasevalidation.FakeLLM,
) {
	t.Helper()
	h = usecasevalidation.New(t)
	fakeLLM = &usecasevalidation.FakeLLM{Response: llmResponse}
	h.SetLLM(fakeLLM)
	h.StartServer()

	const waitTimeout = 2 * time.Second
	term = h.ConnectTerminal(deviceID, &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: deviceID,
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: deviceID},
				},
			},
		},
	})
	if !term.WaitForAny(waitTimeout) {
		t.Fatalf("%s: timed out waiting for session establishment", deviceID)
	}

	h.RecordInteraction("voice", fmt.Sprintf("Say %q on the %s device.", voiceText, deviceID), deviceID)

	term.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Command{
			Command: &controlv1.CommandRequest{
				RequestId: "va-cmd",
				DeviceId:  deviceID,
				Kind:      controlv1.CommandKind_COMMAND_KIND_VOICE,
				Text:      voiceText,
			},
		},
	})
	return h, term, fakeLLM
}

// TestUseCaseV1WithEvidence validates the voice assistant wake-and-answer use case:
// a home user says "assistant <question>" on any device, the server invokes the
// LLM backend, and the response is broadcast back to the source device.
//
// V1: Home user says a wake word and asks a question on any device to get a
// spoken and visual answer without touching a keyboard.
func TestUseCaseV1WithEvidence(t *testing.T) {
	const (
		deviceID    = "living-room"
		voiceText   = "assistant what is the weather today"
		llmResponse = "It is sunny and 72 degrees outside."
		waitTimeout = 2 * time.Second
	)

	h, term, fakeLLM := voiceAssistantHarness(t, deviceID, voiceText, llmResponse)

	// --- Step 1: verify the voice_assistant scenario started. ---
	_, sawStart := term.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil &&
			resp.GetCommandResult().GetScenarioStart() == "voice_assistant"
	}, waitTimeout)
	h.Assert("V1-scenario-started",
		"voice_assistant scenario starts in response to 'assistant' wake word",
		sawStart,
		fmt.Sprintf("%s received %d messages", deviceID, len(term.Received())))

	// --- Step 2: verify the LLM was queried with the spoken question. ---
	queries := fakeLLM.Queries()
	llmCalled := len(queries) > 0
	h.Assert("V1-llm-queried",
		"LLM backend was queried to generate the voice assistant response",
		llmCalled,
		fmt.Sprintf("LLM query count: %d", len(queries)))

	queryContainsWeather := false
	for _, msgs := range queries {
		for _, msg := range msgs {
			if strings.Contains(strings.ToLower(msg.Content), "weather") {
				queryContainsWeather = true
			}
		}
	}
	h.Assert("V1-llm-query-content",
		"LLM query contains the spoken question text",
		queryContainsWeather,
		fmt.Sprintf("queries: %v", queries))

	// --- Step 3: verify the response was broadcast to the source device. ---
	events := h.Broadcast.Events()
	sawResponse := false
	for _, ev := range events {
		if ev.Message == llmResponse {
			sawResponse = true
			break
		}
	}
	h.Assert("V1-response-broadcast",
		"LLM response is broadcast back to the requesting device",
		sawResponse,
		fmt.Sprintf("broadcast events: %d", len(events)))

	h.CaptureFrame("V1-assistant-response", deviceID, term.Received())

	if err := term.Disconnect(); err != nil {
		t.Logf("%s disconnect: %v", deviceID, err)
	}

	h.Evidence("V1")
}

// TestUseCaseV2WithEvidence validates the recipe-by-voice use case: a user in
// the kitchen asks for a recipe by voice and receives it on the nearest screen.
//
// V2: Cook in the kitchen asks for a recipe by voice and sees it displayed on
// the nearest screen to follow instructions hands-free while cooking.
func TestUseCaseV2WithEvidence(t *testing.T) {
	const (
		deviceID    = "kitchen"
		voiceText   = "assistant how do I make pasta carbonara"
		llmResponse = "Pasta carbonara: cook spaghetti, mix eggs and pecorino, combine with pancetta."
		waitTimeout = 2 * time.Second
	)

	h, term, fakeLLM := voiceAssistantHarness(t, deviceID, voiceText, llmResponse)

	// --- Step 1: verify the voice_assistant scenario started. ---
	_, sawStart := term.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil &&
			resp.GetCommandResult().GetScenarioStart() == "voice_assistant"
	}, waitTimeout)
	h.Assert("V2-scenario-started",
		"voice_assistant scenario starts for recipe query",
		sawStart,
		fmt.Sprintf("%s received %d messages", deviceID, len(term.Received())))

	// --- Step 2: verify the LLM was queried. ---
	queries := fakeLLM.Queries()
	h.Assert("V2-llm-queried",
		"LLM backend was queried for the recipe",
		len(queries) > 0,
		fmt.Sprintf("LLM query count: %d", len(queries)))

	// --- Step 3: verify the recipe response is available in the broadcast stream. ---
	// The server broadcasts the response to the source device; in the audio path
	// this also drives a VoiceAssistantResponsePatch UI overlay, but the
	// authoritative content signal is the broadcast event itself.
	events := h.Broadcast.Events()
	sawRecipe := false
	for _, ev := range events {
		if ev.Message == llmResponse {
			sawRecipe = true
			break
		}
	}
	h.Assert("V2-recipe-broadcast",
		"recipe response is broadcast back to the kitchen device",
		sawRecipe,
		fmt.Sprintf("broadcast events: %d", len(events)))

	h.CaptureFrame("V2-recipe-response", deviceID, term.Received())

	if err := term.Disconnect(); err != nil {
		t.Logf("%s disconnect: %v", deviceID, err)
	}

	h.Evidence("V2")
}

// TestUseCaseV3WithEvidence validates the general-knowledge voice query use case:
// a household member asks about the weather, news, or general knowledge and
// receives a quick spoken answer from any room.
//
// V3: Household member asks the system about the weather, news, or general
// knowledge to get quick answers from any room.
func TestUseCaseV3WithEvidence(t *testing.T) {
	const (
		deviceID    = "bedroom"
		voiceText   = "assistant what is the capital of France"
		llmResponse = "The capital of France is Paris."
		waitTimeout = 2 * time.Second
	)

	h, term, fakeLLM := voiceAssistantHarness(t, deviceID, voiceText, llmResponse)

	// --- Step 1: verify the voice_assistant scenario started. ---
	_, sawStart := term.WaitFor(func(env transport.ProtoServerEnvelope) bool {
		resp, ok := env.(*controlv1.ConnectResponse)
		return ok && resp.GetCommandResult() != nil &&
			resp.GetCommandResult().GetScenarioStart() == "voice_assistant"
	}, waitTimeout)
	h.Assert("V3-scenario-started",
		"voice_assistant scenario starts for general-knowledge query",
		sawStart,
		fmt.Sprintf("%s received %d messages", deviceID, len(term.Received())))

	// --- Step 2: verify the LLM was queried. ---
	queries := fakeLLM.Queries()
	h.Assert("V3-llm-queried",
		"LLM backend was queried for the general-knowledge answer",
		len(queries) > 0,
		fmt.Sprintf("LLM query count: %d", len(queries)))

	// --- Step 3: verify the answer appears in the broadcast stream. ---
	events := h.Broadcast.Events()
	sawAnswer := false
	for _, ev := range events {
		if ev.Message == llmResponse {
			sawAnswer = true
			break
		}
	}
	h.Assert("V3-answer-broadcast",
		"general-knowledge answer is broadcast back to the requesting device",
		sawAnswer,
		fmt.Sprintf("broadcast events: %d", len(events)))

	h.CaptureFrame("V3-answer-response", deviceID, term.Received())

	if err := term.Disconnect(); err != nil {
		t.Logf("%s disconnect: %v", deviceID, err)
	}

	h.Evidence("V3")
}
