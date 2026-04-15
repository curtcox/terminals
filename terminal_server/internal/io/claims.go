// Package io manages logical stream-routing state.
package io //nolint:revive

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
)

// ClaimMode controls resource sharing semantics.
type ClaimMode string

const (
	// ClaimExclusive allows only one activation to own a resource.
	ClaimExclusive ClaimMode = "exclusive"
	// ClaimShared allows multiple activations to share a resource.
	ClaimShared ClaimMode = "shared"
)

// Claimable resource names used by scenarios and flow planners.
const (
	ResourceComputeCPUShared = "compute.cpu.shared"
	ResourceComputeGPUShared = "compute.gpu.shared"
	ResourceComputeNPUShared = "compute.npu.shared"
	ResourceBufferAudio      = "buffer.audio.recent"
	ResourceBufferVideo      = "buffer.video.recent"
	ResourceBufferSensor     = "buffer.sensor.recent"
	ResourceBufferRadio      = "buffer.radio.recent"
	ResourceRadioBLEScan     = "radio.ble.scan"
	ResourceRadioWiFiScan    = "radio.wifi.scan"
)

// Claim describes one activation's request for a resource on a device.
type Claim struct {
	ActivationID string
	DeviceID     string
	Resource     string
	Mode         ClaimMode
	Priority     int
}

// Grant reports claims granted and lower-priority claims that were preempted.
type Grant struct {
	Granted   []Claim
	Preempted []Claim
}

var (
	// ErrClaimConflict indicates a lower/equal-priority conflict.
	ErrClaimConflict = errors.New("claim conflict")
)

type resourceKey struct {
	deviceID string
	resource string
}

// ClaimManager provides in-memory claim arbitration.
type ClaimManager struct {
	mu sync.Mutex

	activeByResource map[resourceKey][]Claim
	parkedByResource map[resourceKey][]Claim
	activeByAct      map[string][]Claim
	parkedByAct      map[string][]Claim
}

// NewClaimManager returns an empty claim manager.
func NewClaimManager() *ClaimManager {
	return &ClaimManager{
		activeByResource: make(map[resourceKey][]Claim),
		parkedByResource: make(map[resourceKey][]Claim),
		activeByAct:      make(map[string][]Claim),
		parkedByAct:      make(map[string][]Claim),
	}
}

// Request arbitrates a batch of claims atomically.
func (m *ClaimManager) Request(_ context.Context, claims []Claim) (Grant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	normalized := make([]Claim, 0, len(claims))
	for _, claim := range claims {
		claim = normalizeClaim(claim)
		if claim.ActivationID == "" || claim.DeviceID == "" || claim.Resource == "" {
			continue
		}
		normalized = append(normalized, claim)
	}
	if len(normalized) == 0 {
		return Grant{}, nil
	}

	// First pass: ensure every claim is grantable.
	for _, request := range normalized {
		key := resourceKey{deviceID: request.DeviceID, resource: request.Resource}
		active := m.activeByResource[key]
		if !isGrantable(request, active) {
			return Grant{}, ErrClaimConflict
		}
	}

	grant := Grant{
		Granted: make([]Claim, 0, len(normalized)),
	}
	for _, request := range normalized {
		key := resourceKey{deviceID: request.DeviceID, resource: request.Resource}
		active := m.activeByResource[key]

		stillActive := make([]Claim, 0, len(active))
		for _, current := range active {
			if conflicts(request, current) && request.Priority > current.Priority {
				m.parkClaim(current)
				grant.Preempted = append(grant.Preempted, current)
				continue
			}
			stillActive = append(stillActive, current)
		}

		stillActive = append(stillActive, request)
		m.activeByResource[key] = stillActive
		m.activeByAct[request.ActivationID] = append(m.activeByAct[request.ActivationID], request)
		grant.Granted = append(grant.Granted, request)
	}

	return grant, nil
}

// Release removes all active claims for activationID and restores parked
// claims when resources become available.
func (m *ClaimManager) Release(_ context.Context, activationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	activationID = strings.TrimSpace(activationID)
	if activationID == "" {
		return nil
	}

	affected := map[resourceKey]struct{}{}
	for _, claim := range m.activeByAct[activationID] {
		key := resourceKey{deviceID: claim.DeviceID, resource: claim.Resource}
		active := m.activeByResource[key]
		next := make([]Claim, 0, len(active))
		for _, existing := range active {
			if existing.ActivationID == activationID {
				continue
			}
			next = append(next, existing)
		}
		m.activeByResource[key] = next
		affected[key] = struct{}{}
	}
	delete(m.activeByAct, activationID)

	// Drop any parked claims owned by the released activation too.
	for key, parked := range m.parkedByResource {
		next := parked[:0]
		for _, existing := range parked {
			if existing.ActivationID == activationID {
				continue
			}
			next = append(next, existing)
		}
		if len(next) == 0 {
			delete(m.parkedByResource, key)
			continue
		}
		m.parkedByResource[key] = append([]Claim(nil), next...)
		affected[key] = struct{}{}
	}
	delete(m.parkedByAct, activationID)

	for key := range affected {
		m.restoreParkedForResource(key)
	}
	return nil
}

// Snapshot returns active claims for one device sorted by resource then activation.
func (m *ClaimManager) Snapshot(deviceID string) []Claim {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshotLocked(deviceID, m.activeByResource)
}

// SuspendedSnapshot returns currently parked (preempted) claims for one device.
func (m *ClaimManager) SuspendedSnapshot(deviceID string) []Claim {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshotLocked(deviceID, m.parkedByResource)
}

func (m *ClaimManager) snapshotLocked(deviceID string, source map[resourceKey][]Claim) []Claim {
	deviceID = strings.TrimSpace(deviceID)
	out := make([]Claim, 0)
	for key, claims := range source {
		if key.deviceID != deviceID {
			continue
		}
		out = append(out, claims...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Resource == out[j].Resource {
			if out[i].Priority == out[j].Priority {
				return out[i].ActivationID < out[j].ActivationID
			}
			return out[i].Priority > out[j].Priority
		}
		return out[i].Resource < out[j].Resource
	})
	return out
}

func (m *ClaimManager) parkClaim(claim Claim) {
	key := resourceKey{deviceID: claim.DeviceID, resource: claim.Resource}
	m.parkedByResource[key] = append(m.parkedByResource[key], claim)
	m.parkedByAct[claim.ActivationID] = append(m.parkedByAct[claim.ActivationID], claim)

	active := m.activeByAct[claim.ActivationID]
	next := active[:0]
	for _, existing := range active {
		if existing.DeviceID == claim.DeviceID && existing.Resource == claim.Resource {
			continue
		}
		next = append(next, existing)
	}
	if len(next) == 0 {
		delete(m.activeByAct, claim.ActivationID)
	} else {
		m.activeByAct[claim.ActivationID] = append([]Claim(nil), next...)
	}
}

func (m *ClaimManager) restoreParkedForResource(key resourceKey) {
	parked := append([]Claim(nil), m.parkedByResource[key]...)
	if len(parked) == 0 {
		return
	}

	sort.Slice(parked, func(i, j int) bool {
		if parked[i].Priority == parked[j].Priority {
			if parked[i].Mode == parked[j].Mode {
				return parked[i].ActivationID < parked[j].ActivationID
			}
			// Prefer shared claims when tied so analyzer taps can come back together.
			return parked[i].Mode == ClaimShared
		}
		return parked[i].Priority > parked[j].Priority
	})

	active := append([]Claim(nil), m.activeByResource[key]...)
	remaining := make([]Claim, 0, len(parked))
	for _, claim := range parked {
		if isGrantable(claim, active) {
			active = append(active, claim)
			m.activeByAct[claim.ActivationID] = append(m.activeByAct[claim.ActivationID], claim)
			m.dropParkedClaimLocked(claim)
			continue
		}
		remaining = append(remaining, claim)
	}
	m.activeByResource[key] = active
	if len(remaining) == 0 {
		delete(m.parkedByResource, key)
	} else {
		m.parkedByResource[key] = remaining
	}
}

func (m *ClaimManager) dropParkedClaimLocked(claim Claim) {
	parked := m.parkedByAct[claim.ActivationID]
	if len(parked) == 0 {
		return
	}
	next := parked[:0]
	for _, existing := range parked {
		if existing.DeviceID == claim.DeviceID && existing.Resource == claim.Resource {
			continue
		}
		next = append(next, existing)
	}
	if len(next) == 0 {
		delete(m.parkedByAct, claim.ActivationID)
	} else {
		m.parkedByAct[claim.ActivationID] = append([]Claim(nil), next...)
	}
}

func normalizeClaim(claim Claim) Claim {
	claim.ActivationID = strings.TrimSpace(claim.ActivationID)
	claim.DeviceID = strings.TrimSpace(claim.DeviceID)
	claim.Resource = strings.TrimSpace(claim.Resource)
	if claim.Mode == "" {
		claim.Mode = ClaimExclusive
	}
	return claim
}

func conflicts(a, b Claim) bool {
	if a.DeviceID != b.DeviceID || a.Resource != b.Resource {
		return false
	}
	if a.Mode == ClaimShared && b.Mode == ClaimShared {
		return false
	}
	return true
}

func isGrantable(request Claim, active []Claim) bool {
	for _, current := range active {
		if !conflicts(request, current) {
			continue
		}
		if request.Priority <= current.Priority {
			return false
		}
	}
	return true
}
