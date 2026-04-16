// Package device manages connected device registry state.
package device

import (
	"errors"
	"sort"
	"strings"
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
	current.Placement = clonePlacementMetadata(current.Placement)
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
	current.Placement = clonePlacementMetadata(current.Placement)
	return current, true
}

// List returns all devices sorted by device ID for deterministic behavior.
func (m *Manager) List() []Device {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Device, 0, len(m.devices))
	for _, d := range m.devices {
		d.Capabilities = cloneCapabilities(d.Capabilities)
		d.Placement = clonePlacementMetadata(d.Placement)
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
	return len(m.MarkStaleDisconnectedDevices(cutoff))
}

// MarkStaleDisconnectedDevices marks stale connected devices as disconnected
// and returns the device ids that changed state.
func (m *Manager) MarkStaleDisconnectedDevices(cutoff time.Time) []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	updated := make([]string, 0, len(m.devices))
	cutoff = cutoff.UTC()
	for id, current := range m.devices {
		if current.State != StateConnected {
			continue
		}
		if current.LastHeartbeat.IsZero() || current.LastHeartbeat.Before(cutoff) {
			current.State = StateDisconnected
			m.devices[id] = current
			updated = append(updated, id)
		}
	}
	return updated
}

// UpdatePlacement replaces semantic placement metadata for an existing device.
func (m *Manager) UpdatePlacement(deviceID string, placement PlacementMetadata) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, exists := m.devices[strings.TrimSpace(deviceID)]
	if !exists {
		return ErrDeviceNotFound
	}
	current.Placement = clonePlacementMetadata(placement)
	m.devices[current.DeviceID] = current
	return nil
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

func clonePlacementMetadata(in PlacementMetadata) PlacementMetadata {
	out := PlacementMetadata{
		Zone:     strings.TrimSpace(in.Zone),
		Mobility: strings.TrimSpace(in.Mobility),
		Affinity: strings.TrimSpace(in.Affinity),
	}
	if len(in.Roles) == 0 {
		out.Roles = []string{}
		return out
	}
	seen := make(map[string]struct{}, len(in.Roles))
	out.Roles = make([]string, 0, len(in.Roles))
	for _, role := range in.Roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		if _, exists := seen[role]; exists {
			continue
		}
		seen[role] = struct{}{}
		out.Roles = append(out.Roles, role)
	}
	sort.Strings(out.Roles)
	return out
}
