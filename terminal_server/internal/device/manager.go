// Package device manages device registration and state tracking.
package device

import (
	"errors"
	"sort"
	"sync"
	"time"
)

var (
	// ErrMissingDeviceID is returned when a manifest lacks an ID.
	ErrMissingDeviceID = errors.New("missing device id")
	// ErrDeviceNotFound is returned when a referenced device does not exist.
	ErrDeviceNotFound = errors.New("device not found")
)

// Manager stores all registered devices and their evolving state.
type Manager struct {
	mu      sync.RWMutex
	devices map[string]Device
	now     func() time.Time
}

// NewManager creates a ready-to-use device manager.
func NewManager() *Manager {
	return &Manager{
		devices: make(map[string]Device),
		now:     time.Now,
	}
}

// Register creates or refreshes a device record.
func (m *Manager) Register(manifest Manifest) (Device, error) {
	if manifest.DeviceID == "" {
		return Device{}, ErrMissingDeviceID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	current, exists := m.devices[manifest.DeviceID]
	if !exists {
		current.RegisteredAt = m.now().UTC()
	}
	current.DeviceID = manifest.DeviceID
	current.DeviceName = manifest.DeviceName
	current.DeviceType = manifest.DeviceType
	current.Platform = manifest.Platform
	current.Capabilities = cloneCapabilities(manifest.Capabilities)
	current.State = StateConnected
	current.LastHeartbeat = m.now().UTC()
	m.devices[manifest.DeviceID] = current

	return current, nil
}

// UpdateCapabilities replaces the capability set for an existing device.
func (m *Manager) UpdateCapabilities(deviceID string, caps CapabilitySet) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, exists := m.devices[deviceID]
	if !exists {
		return ErrDeviceNotFound
	}
	current.Capabilities = cloneCapabilities(caps)
	m.devices[deviceID] = current
	return nil
}

// Heartbeat updates a device's liveness timestamp.
func (m *Manager) Heartbeat(deviceID string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, exists := m.devices[deviceID]
	if !exists {
		return ErrDeviceNotFound
	}
	current.LastHeartbeat = at.UTC()
	current.State = StateConnected
	m.devices[deviceID] = current
	return nil
}

// MarkDisconnected marks a device as disconnected without deleting it.
func (m *Manager) MarkDisconnected(deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, exists := m.devices[deviceID]
	if !exists {
		return ErrDeviceNotFound
	}
	current.State = StateDisconnected
	m.devices[deviceID] = current
	return nil
}

// Get returns a copy of a device record by ID.
func (m *Manager) Get(deviceID string) (Device, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	current, exists := m.devices[deviceID]
	if !exists {
		return Device{}, false
	}
	current.Capabilities = cloneCapabilities(current.Capabilities)
	return current, true
}

// List returns all devices sorted by device ID for deterministic behavior.
func (m *Manager) List() []Device {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Device, 0, len(m.devices))
	for _, d := range m.devices {
		d.Capabilities = cloneCapabilities(d.Capabilities)
		result = append(result, d)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].DeviceID < result[j].DeviceID
	})
	return result
}

// ListDeviceIDs returns sorted device IDs for scenario targeting.
func (m *Manager) ListDeviceIDs() []string {
	devices := m.List()
	ids := make([]string, 0, len(devices))
	for _, d := range devices {
		ids = append(ids, d.DeviceID)
	}
	return ids
}

// MarkStaleDisconnected marks connected devices as disconnected when
// their last heartbeat is older than cutoff. Returns the number updated.
func (m *Manager) MarkStaleDisconnected(cutoff time.Time) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	updated := 0
	cutoff = cutoff.UTC()
	for id, current := range m.devices {
		if current.State != StateConnected {
			continue
		}
		if current.LastHeartbeat.IsZero() || current.LastHeartbeat.Before(cutoff) {
			current.State = StateDisconnected
			m.devices[id] = current
			updated++
		}
	}
	return updated
}

func cloneCapabilities(caps CapabilitySet) CapabilitySet {
	if caps == nil {
		return CapabilitySet{}
	}
	out := make(CapabilitySet, len(caps))
	for k, v := range caps {
		out[k] = v
	}
	return out
}
