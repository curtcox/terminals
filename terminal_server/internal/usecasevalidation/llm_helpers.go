package usecasevalidation

import (
	"context"
	"sync"

	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

// FakeLLM is a test double for scenario.LLM that returns a pre-configured
// response for every query. When used with intent resolution (AA3), Response
// must be a valid JSON intent envelope. When used with VoiceAssistantScenario
// (V1/V2/V3), Response is returned verbatim as the assistant's reply text.
type FakeLLM struct {
	// Response is returned as LLMResponse.Text for every Query call.
	Response string

	mu      sync.Mutex
	queries [][]scenario.LLMMessage
}

// Query returns the configured response and records the incoming messages.
func (f *FakeLLM) Query(_ context.Context, msgs []scenario.LLMMessage, _ scenario.LLMOptions) (*scenario.LLMResponse, error) {
	f.mu.Lock()
	f.queries = append(f.queries, msgs)
	f.mu.Unlock()
	return &scenario.LLMResponse{Text: f.Response, FinishReason: "stop"}, nil
}

// Queries returns a copy of every message list that was passed to Query.
func (f *FakeLLM) Queries() [][]scenario.LLMMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([][]scenario.LLMMessage, len(f.queries))
	copy(out, f.queries)
	return out
}
