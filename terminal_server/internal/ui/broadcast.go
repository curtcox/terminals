// Package ui contains descriptor and broadcast utilities.
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

// HostEvent captures one UI host operation.
type HostEvent struct {
	Kind        string
	DeviceID    string
	Root        string
	ComponentID string
	Node        Descriptor
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

// MemoryHost records Set/Patch/Clear UI operations in memory.
type MemoryHost struct {
	mu     sync.RWMutex
	events []HostEvent
	roots  map[string]Descriptor
}

// NewMemoryHost creates an in-memory UI host.
func NewMemoryHost() *MemoryHost {
	return &MemoryHost{roots: map[string]Descriptor{}}
}

// Set records and stores a full UI replacement.
func (h *MemoryHost) Set(_ context.Context, deviceID string, root Descriptor) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.roots[deviceID] = root
	h.events = append(h.events, HostEvent{
		Kind:     "set",
		DeviceID: deviceID,
		Node:     root,
	})
	return nil
}

// Patch records a targeted UI update.
func (h *MemoryHost) Patch(_ context.Context, deviceID, componentID string, node Descriptor) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, HostEvent{
		Kind:        "patch",
		DeviceID:    deviceID,
		ComponentID: componentID,
		Node:        node,
	})
	return nil
}

// Clear records a UI clear operation and forgets the stored root.
func (h *MemoryHost) Clear(_ context.Context, deviceID, root string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.roots, deviceID)
	h.events = append(h.events, HostEvent{
		Kind:     "clear",
		DeviceID: deviceID,
		Root:     root,
	})
	return nil
}

// Events returns a copy of recorded UI host events.
func (h *MemoryHost) Events() []HostEvent {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]HostEvent, len(h.events))
	copy(out, h.events)
	return out
}

// Root returns the last full UI root stored for a device.
func (h *MemoryHost) Root(deviceID string) (Descriptor, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	root, ok := h.roots[deviceID]
	return root, ok
}
