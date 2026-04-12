// Package telephony contains scenario-facing telephony bridge abstractions.
package telephony

import "context"

// NoopBridge is a placeholder telephony bridge.
type NoopBridge struct{}

// Call is currently a no-op.
func (NoopBridge) Call(context.Context, string) error {
	return nil
}

// Hangup is currently a no-op.
func (NoopBridge) Hangup(context.Context, string) error {
	return nil
}
