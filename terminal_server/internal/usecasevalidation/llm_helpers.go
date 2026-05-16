package usecasevalidation

import (
	"context"

	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

// FakeLLM is a test double for scenario.LLM that returns a pre-configured
// JSON response for every query. Use this to inject deterministic LLM intent
// resolution into harness-based scenario tests without calling a real API.
type FakeLLM struct {
	// Response is the raw JSON string returned for every Query call.
	// It must be parseable as an llmIntentEnvelope (action, object, slots, scope).
	Response string
}

// Query returns the configured response regardless of the messages or options.
func (f *FakeLLM) Query(_ context.Context, _ []scenario.LLMMessage, _ scenario.LLMOptions) (*scenario.LLMResponse, error) {
	return &scenario.LLMResponse{Text: f.Response, FinishReason: "stop"}, nil
}
