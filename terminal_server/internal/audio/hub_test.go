package audio

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestHubPublishesChunksToAllSubscribers(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subA := hub.Subscribe(ctx, "device-1")
	subB := hub.Subscribe(ctx, "device-1")
	defer func() { _ = subA.Close() }()
	defer func() { _ = subB.Close() }()

	hub.Publish("device-1", []byte("abc"))
	hub.Publish("device-1", []byte("de"))

	gotA := readN(t, subA, 5)
	gotB := readN(t, subB, 5)

	if string(gotA) != "abcde" {
		t.Fatalf("subA bytes = %q, want abcde", string(gotA))
	}
	if string(gotB) != "abcde" {
		t.Fatalf("subB bytes = %q, want abcde", string(gotB))
	}
}

func TestHubPublishDoesNotReachOtherDevices(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subOther := hub.Subscribe(ctx, "device-2")
	defer func() { _ = subOther.Close() }()

	hub.Publish("device-1", []byte("abc"))

	done := make(chan struct{})
	go func() {
		buf := make([]byte, 4)
		_, _ = subOther.Read(buf)
		close(done)
	}()

	select {
	case <-done:
		t.Fatalf("unexpected read from subscriber of different device")
	case <-time.After(30 * time.Millisecond):
	}
}

func TestHubSubscriptionCloseReturnsEOF(t *testing.T) {
	hub := NewHub()
	sub := hub.Subscribe(context.Background(), "device-1")

	hub.Publish("device-1", []byte("xy"))
	if _ = sub.Close(); hub.SubscriberCount("device-1") != 0 {
		t.Fatalf("SubscriberCount = %d, want 0", hub.SubscriberCount("device-1"))
	}

	buf := make([]byte, 4)
	n, err := sub.Read(buf)
	if n != 2 || string(buf[:2]) != "xy" {
		t.Fatalf("drained read = %d %q, want 2 xy", n, string(buf[:2]))
	}
	if err != nil {
		t.Fatalf("drained read err = %v, want nil", err)
	}

	_, err = sub.Read(buf)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("post-drain read err = %v, want io.EOF", err)
	}
}

func TestHubSubscriptionCancelsWithContext(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	sub := hub.Subscribe(ctx, "device-1")

	cancel()

	// Wait for the hub cancel goroutine to close the subscription. Since
	// goroutine scheduling is async we poll briefly.
	deadline := time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(deadline) {
		if hub.SubscriberCount("device-1") == 0 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if hub.SubscriberCount("device-1") != 0 {
		t.Fatalf("SubscriberCount = %d, want 0 after context cancel", hub.SubscriberCount("device-1"))
	}

	buf := make([]byte, 4)
	_, err := sub.Read(buf)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("read err after cancel = %v, want io.EOF", err)
	}
}

func TestHubPublishIgnoresEmptyInputs(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub := hub.Subscribe(ctx, "device-1")
	defer func() { _ = sub.Close() }()

	hub.Publish("", []byte("abc"))
	hub.Publish("device-1", nil)

	done := make(chan struct{})
	go func() {
		buf := make([]byte, 4)
		_, _ = sub.Read(buf)
		close(done)
	}()

	select {
	case <-done:
		t.Fatalf("did not expect any data for empty publishes")
	case <-time.After(30 * time.Millisecond):
	}
}

func readN(t *testing.T, r io.Reader, n int) []byte {
	t.Helper()
	out := make([]byte, 0, n)
	buf := make([]byte, n)
	for len(out) < n {
		got, err := r.Read(buf[:n-len(out)])
		if got > 0 {
			out = append(out, buf[:got]...)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatalf("read error = %v", err)
		}
	}
	return out
}
