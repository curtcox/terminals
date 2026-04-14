// Package ai contains scenario-facing AI backend interfaces and placeholders.
package ai

import "context"

// NoopBackend is the legacy single-method AI backend used by scenarios that
// only need a generic text-in / text-out adapter. It is kept around for
// backward compatibility while scenarios migrate to the capability-specific
// interfaces (LLM, SpeechToText, etc.).
type NoopBackend struct{}

// Query returns the deterministic sentinel response so callers can detect
// that no real backend is configured.
func (NoopBackend) Query(context.Context, string) (string, error) {
	return noopSentinel, nil
}

// LLMQueryAdapter exposes an LLM as the legacy `AIBackend.Query` shape so
// scenarios that have not yet migrated continue to work when only an LLM
// has been configured. The first message is sent as a single user turn.
type LLMQueryAdapter struct {
	LLM     LLM
	Options LLMOptions
}

// Query forwards the prompt to the wrapped LLM, returning the response
// text. A nil LLM yields the sentinel response so the adapter is safe to
// use as a default placeholder.
func (a LLMQueryAdapter) Query(ctx context.Context, input string) (string, error) {
	if a.LLM == nil {
		return noopSentinel, nil
	}
	resp, err := a.LLM.Query(ctx, []Message{{Role: "user", Content: input}}, a.Options)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", nil
	}
	return resp.Text, nil
}
