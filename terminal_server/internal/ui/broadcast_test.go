package ui

import (
	"context"
	"testing"
)

func TestMemoryBroadcasterNotify(t *testing.T) {
	b := NewMemoryBroadcaster()
	if err := b.Notify(context.Background(), []string{"a", "b"}, "hello"); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	events := b.Events()
	if len(events) != 1 {
		t.Fatalf("len(Events()) = %d, want 1", len(events))
	}
	if events[0].Message != "hello" {
		t.Fatalf("Message = %q, want hello", events[0].Message)
	}
}
