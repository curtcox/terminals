package transport

import "sync"

type relaySender func(ServerMessage) error

type sessionRelayRegistry struct {
	mu       sync.RWMutex
	byDevice map[string]relaySender
}

func newSessionRelayRegistry() *sessionRelayRegistry {
	return &sessionRelayRegistry{
		byDevice: map[string]relaySender{},
	}
}

func (r *sessionRelayRegistry) Register(deviceID string, sender relaySender) {
	if deviceID == "" || sender == nil {
		return
	}
	r.mu.Lock()
	r.byDevice[deviceID] = sender
	r.mu.Unlock()
}

func (r *sessionRelayRegistry) Unregister(deviceID string) {
	if deviceID == "" {
		return
	}
	r.mu.Lock()
	delete(r.byDevice, deviceID)
	r.mu.Unlock()
}

func (r *sessionRelayRegistry) Relay(deviceID string, msg ServerMessage) error {
	if deviceID == "" {
		return nil
	}
	r.mu.RLock()
	sender, ok := r.byDevice[deviceID]
	r.mu.RUnlock()
	if !ok || sender == nil {
		return nil
	}
	return sender(msg)
}

var globalSessionRelayRegistry = newSessionRelayRegistry()
