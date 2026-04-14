// Package audio provides a device-scoped audio pub/sub hub used to share
// inbound mic audio between the transport layer and scenarios that need
// to analyze or forward live audio (for example, AudioMonitorScenario).
package audio

import (
	"context"
	"io"
	"sync"
)

// Hub fans out audio chunks published for a device to every active
// subscriber for that device. Subscriptions block on Read until new audio
// arrives or the subscription is closed.
type Hub struct {
	mu          sync.Mutex
	subscribers map[string][]*Subscription
}

// NewHub returns an empty Hub.
func NewHub() *Hub {
	return &Hub{subscribers: make(map[string][]*Subscription)}
}

// Publish delivers a copy of chunk to every active subscriber for deviceID.
// Zero-length chunks and empty device IDs are ignored.
func (h *Hub) Publish(deviceID string, chunk []byte) {
	if deviceID == "" || len(chunk) == 0 {
		return
	}
	h.mu.Lock()
	subs := append([]*Subscription(nil), h.subscribers[deviceID]...)
	h.mu.Unlock()

	for _, sub := range subs {
		copied := make([]byte, len(chunk))
		copy(copied, chunk)
		sub.push(copied)
	}
}

// Subscribe returns a new Subscription for deviceID. Canceling ctx closes
// the subscription so subsequent Reads return io.EOF after draining any
// remaining buffered audio.
func (h *Hub) Subscribe(ctx context.Context, deviceID string) *Subscription {
	sub := newSubscription()
	sub.remove = func() { h.removeSub(deviceID, sub) }

	h.mu.Lock()
	h.subscribers[deviceID] = append(h.subscribers[deviceID], sub)
	h.mu.Unlock()

	if ctx != nil {
		go func() {
			select {
			case <-ctx.Done():
				_ = sub.Close()
			case <-sub.done:
			}
		}()
	}
	return sub
}

// SubscriberCount returns the number of active subscribers for deviceID.
func (h *Hub) SubscriberCount(deviceID string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subscribers[deviceID])
}

func (h *Hub) removeSub(deviceID string, target *Subscription) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs := h.subscribers[deviceID]
	for i, s := range subs {
		if s == target {
			h.subscribers[deviceID] = append(subs[:i], subs[i+1:]...)
			if len(h.subscribers[deviceID]) == 0 {
				delete(h.subscribers, deviceID)
			}
			return
		}
	}
}

// Subscription delivers published audio chunks to a scenario via Read.
type Subscription struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buf    []byte
	closed bool
	done   chan struct{}

	closeOnce sync.Once
	remove    func()
}

func newSubscription() *Subscription {
	s := &Subscription{done: make(chan struct{})}
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (s *Subscription) push(chunk []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}
	s.buf = append(s.buf, chunk...)
	s.cond.Broadcast()
}

// Read blocks until audio data is available or the subscription is closed.
// Once closed and drained, Read returns io.EOF.
func (s *Subscription) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for len(s.buf) == 0 && !s.closed {
		s.cond.Wait()
	}
	if len(s.buf) == 0 {
		return 0, io.EOF
	}
	n := copy(p, s.buf)
	s.buf = s.buf[n:]
	return n, nil
}

// Close releases resources held by the subscription. Subsequent Read calls
// return io.EOF after draining. Close is safe to call more than once.
func (s *Subscription) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.cond.Broadcast()
		s.mu.Unlock()

		close(s.done)
		if s.remove != nil {
			s.remove()
		}
	})
	return nil
}
