// Package placement resolves semantic target scopes (zone/role/nearest)
// to concrete device ids.
package placement

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

// ErrNoMatchingDevices indicates a placement query produced no candidates.
var ErrNoMatchingDevices = errors.New("no matching devices")

// Engine resolves semantic placement queries.
type Engine interface {
	Find(ctx context.Context, q scenario.PlacementQuery) ([]scenario.DeviceRef, error)
	NearestWith(ctx context.Context, source scenario.DeviceRef, capability string) (scenario.DeviceRef, error)
	DevicesInZone(ctx context.Context, zone string) ([]scenario.DeviceRef, error)
	DevicesWithRole(ctx context.Context, role string) ([]scenario.DeviceRef, error)
}

// ManagerBackedEngine implements placement lookups using device.Manager.
type ManagerBackedEngine struct {
	devices *device.Manager
}

// NewManagerBackedEngine creates a placement engine over the shared device registry.
func NewManagerBackedEngine(devices *device.Manager) *ManagerBackedEngine {
	return &ManagerBackedEngine{devices: devices}
}

// Find resolves semantic scope plus capability constraints to concrete devices.
func (e *ManagerBackedEngine) Find(_ context.Context, q scenario.PlacementQuery) ([]scenario.DeviceRef, error) {
	candidates := e.filteredDevices(q)
	if len(candidates) == 0 {
		return nil, ErrNoMatchingDevices
	}
	if q.Count > 0 && len(candidates) > q.Count {
		candidates = candidates[:q.Count]
	}
	out := make([]scenario.DeviceRef, 0, len(candidates))
	for _, d := range candidates {
		out = append(out, scenario.DeviceRef{DeviceID: d.DeviceID})
	}
	return out, nil
}

// NearestWith returns the nearest matching device for the requested capability.
// Current distance heuristic is zone proximity: prefer same-zone devices.
func (e *ManagerBackedEngine) NearestWith(_ context.Context, source scenario.DeviceRef, capability string) (scenario.DeviceRef, error) {
	sourceDevice, sourceOK := e.devices.Get(strings.TrimSpace(source.DeviceID))
	query := scenario.PlacementQuery{
		RequiredCaps: []string{strings.TrimSpace(capability)},
		Count:        1,
	}
	if sourceOK && strings.TrimSpace(sourceDevice.Placement.Zone) != "" {
		query.Scope.Zone = sourceDevice.Placement.Zone
	}
	refs, err := e.Find(context.Background(), query)
	if err == nil && len(refs) > 0 {
		if refs[0].DeviceID != strings.TrimSpace(source.DeviceID) {
			return refs[0], nil
		}
		if len(refs) > 1 {
			return refs[1], nil
		}
	}

	refs, err = e.Find(context.Background(), scenario.PlacementQuery{
		RequiredCaps: []string{strings.TrimSpace(capability)},
		Count:        2,
	})
	if err != nil {
		return scenario.DeviceRef{}, err
	}
	for _, ref := range refs {
		if ref.DeviceID != strings.TrimSpace(source.DeviceID) {
			return ref, nil
		}
	}
	return scenario.DeviceRef{}, ErrNoMatchingDevices
}

// DevicesInZone returns all devices in the provided zone.
func (e *ManagerBackedEngine) DevicesInZone(_ context.Context, zone string) ([]scenario.DeviceRef, error) {
	return e.Find(context.Background(), scenario.PlacementQuery{
		Scope: scenario.TargetScope{Zone: strings.TrimSpace(zone), Broadcast: true},
	})
}

// DevicesWithRole returns all devices tagged with the provided role.
func (e *ManagerBackedEngine) DevicesWithRole(_ context.Context, role string) ([]scenario.DeviceRef, error) {
	return e.Find(context.Background(), scenario.PlacementQuery{
		Scope: scenario.TargetScope{Role: strings.TrimSpace(role), Broadcast: true},
	})
}

func (e *ManagerBackedEngine) filteredDevices(q scenario.PlacementQuery) []device.Device {
	if e == nil || e.devices == nil {
		return nil
	}
	devs := e.devices.List()
	out := make([]device.Device, 0, len(devs))
	for _, d := range devs {
		if d.State != device.StateConnected {
			continue
		}
		if q.ExcludeBusy && strings.EqualFold(d.Capabilities["liveness"], "busy") {
			continue
		}
		if !matchesScope(d, q.Scope) {
			continue
		}
		if !matchesRequiredCaps(d, q.RequiredCaps) {
			continue
		}
		out = append(out, d)
	}

	// Prefer candidates with more preferred capabilities.
	sort.Slice(out, func(i, j int) bool {
		scoreI := preferredScore(out[i], q.PreferredCaps)
		scoreJ := preferredScore(out[j], q.PreferredCaps)
		if scoreI == scoreJ {
			return out[i].DeviceID < out[j].DeviceID
		}
		return scoreI > scoreJ
	})
	return out
}

func preferredScore(d device.Device, preferred []string) int {
	if len(preferred) == 0 {
		return 0
	}
	score := 0
	for _, capName := range preferred {
		if hasCapability(d, capName) {
			score++
		}
	}
	return score
}

func matchesRequiredCaps(d device.Device, required []string) bool {
	for _, capName := range required {
		if !hasCapability(d, capName) {
			return false
		}
	}
	return true
}

func hasCapability(d device.Device, capName string) bool {
	capName = strings.TrimSpace(capName)
	if capName == "" {
		return true
	}
	value, ok := d.Capabilities[capName]
	if ok {
		v := strings.ToLower(strings.TrimSpace(value))
		return v != "" && v != "false" && v != "0" && v != "off" && v != "no"
	}
	// Allow "screen" to match capability keys like "screen.width".
	prefix := capName + "."
	for key, value := range d.Capabilities {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		v := strings.ToLower(strings.TrimSpace(value))
		if v != "" && v != "false" && v != "0" && v != "off" && v != "no" {
			return true
		}
	}
	return false
}

func matchesScope(d device.Device, scope scenario.TargetScope) bool {
	if scope.DeviceID != "" && d.DeviceID != strings.TrimSpace(scope.DeviceID) {
		return false
	}
	if scope.Zone != "" && !strings.EqualFold(d.Placement.Zone, strings.TrimSpace(scope.Zone)) {
		return false
	}
	if scope.Role != "" {
		role := strings.TrimSpace(scope.Role)
		found := false
		for _, item := range d.Placement.Roles {
			if strings.EqualFold(item, role) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
