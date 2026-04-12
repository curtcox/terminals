package ui

import (
	"context"
	"sync"
)

// BroadcastEvent captures one outbound notification.
type BroadcastEvent struct {
	DeviceIDs []string
	Message   string
}

// MemoryBroadcaster records notifications in memory.
type MemoryBroadcaster struct {
	mu     sync.RWMutex
	events []BroadcastEvent
}

// NewMemoryBroadcaster creates an empty broadcaster.
func NewMemoryBroadcaster() *MemoryBroadcaster {
	return &MemoryBroadcaster{}
}

// Notify records the broadcast event.
func (b *MemoryBroadcaster) Notify(_ context.Context, deviceIDs []string, message string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	copied := make([]string, len(deviceIDs))
	copy(copied, deviceIDs)
	b.events = append(b.events, BroadcastEvent{
		DeviceIDs: copied,
		Message:   message,
	})
	return nil
}

// Events returns a copy of recorded events.
func (b *MemoryBroadcaster) Events() []BroadcastEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	out := make([]BroadcastEvent, len(b.events))
	copy(out, b.events)
	return out
}
