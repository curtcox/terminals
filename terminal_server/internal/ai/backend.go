// Package ai provides AI backend interfaces and placeholder implementations.
package ai

import "context"

// NoopBackend is a deterministic placeholder for early integration tests.
type NoopBackend struct{}

// Query returns a fixed response template for now.
func (NoopBackend) Query(context.Context, string) (string, error) {
	return "ai backend not configured", nil
}
