package device

import (
	"testing"
	"time"
)

func TestRegisterAndGet(t *testing.T) {
	m := NewManager()
	fixedNow := time.Date(2026, 4, 11, 10, 30, 0, 0, time.UTC)
	m.now = func() time.Time { return fixedNow }

	got, err := m.Register(Manifest{
		DeviceID:   "device-1",
		DeviceName: "Kitchen Chromebook",
		DeviceType: "laptop",
		Platform:   "chromeos",
		Capabilities: CapabilitySet{
			"screen.width": "1920",
			"mic.channels": "1",
		},
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if got.DeviceID != "device-1" {
		t.Fatalf("DeviceID = %q, want %q", got.DeviceID, "device-1")
	}
	if got.RegisteredAt != fixedNow {
		t.Fatalf("RegisteredAt = %v, want %v", got.RegisteredAt, fixedNow)
	}

	found, ok := m.Get("device-1")
	if !ok {
		t.Fatalf("Get() did not find registered device")
	}
	if found.Capabilities["screen.width"] != "1920" {
		t.Fatalf("capability screen.width = %q", found.Capabilities["screen.width"])
	}
}

func TestRegisterMissingID(t *testing.T) {
	m := NewManager()
	if _, err := m.Register(Manifest{}); err != ErrMissingDeviceID {
		t.Fatalf("Register() error = %v, want %v", err, ErrMissingDeviceID)
	}
}

func TestUpdateCapabilities(t *testing.T) {
	m := NewManager()
	_, _ = m.Register(Manifest{DeviceID: "device-1"})

	if err := m.UpdateCapabilities("device-1", CapabilitySet{"camera.front": "true"}); err != nil {
		t.Fatalf("UpdateCapabilities() error = %v", err)
	}

	found, _ := m.Get("device-1")
	if found.Capabilities["camera.front"] != "true" {
		t.Fatalf("camera.front = %q", found.Capabilities["camera.front"])
	}
}

func TestHeartbeatAndDisconnect(t *testing.T) {
	m := NewManager()
	_, _ = m.Register(Manifest{DeviceID: "device-1"})

	pulse := time.Date(2026, 4, 11, 11, 45, 0, 0, time.UTC)
	if err := m.Heartbeat("device-1", pulse); err != nil {
		t.Fatalf("Heartbeat() error = %v", err)
	}
	if err := m.MarkDisconnected("device-1"); err != nil {
		t.Fatalf("MarkDisconnected() error = %v", err)
	}

	found, _ := m.Get("device-1")
	if found.LastHeartbeat != pulse {
		t.Fatalf("LastHeartbeat = %v, want %v", found.LastHeartbeat, pulse)
	}
	if found.State != StateDisconnected {
		t.Fatalf("State = %q, want %q", found.State, StateDisconnected)
	}
}

func TestListSorted(t *testing.T) {
	m := NewManager()
	_, _ = m.Register(Manifest{DeviceID: "b"})
	_, _ = m.Register(Manifest{DeviceID: "a"})

	list := m.List()
	if len(list) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(list))
	}
	if list[0].DeviceID != "a" || list[1].DeviceID != "b" {
		t.Fatalf("List() order = %q, %q", list[0].DeviceID, list[1].DeviceID)
	}
}

func TestUpdatePlacement(t *testing.T) {
	m := NewManager()
	_, _ = m.Register(Manifest{DeviceID: "device-1"})

	err := m.UpdatePlacement("device-1", PlacementMetadata{
		Zone:     "kitchen",
		Roles:    []string{"kitchen_display", "screen", "screen"},
		Mobility: "fixed",
		Affinity: "home",
	})
	if err != nil {
		t.Fatalf("UpdatePlacement() error = %v", err)
	}

	found, ok := m.Get("device-1")
	if !ok {
		t.Fatalf("Get() did not find device-1")
	}
	if found.Placement.Zone != "kitchen" {
		t.Fatalf("Placement.Zone = %q, want kitchen", found.Placement.Zone)
	}
	if len(found.Placement.Roles) != 2 || found.Placement.Roles[0] != "kitchen_display" || found.Placement.Roles[1] != "screen" {
		t.Fatalf("Placement.Roles = %+v, want [kitchen_display screen]", found.Placement.Roles)
	}
	if found.Placement.Mobility != "fixed" {
		t.Fatalf("Placement.Mobility = %q, want fixed", found.Placement.Mobility)
	}
	if found.Placement.Affinity != "home" {
		t.Fatalf("Placement.Affinity = %q, want home", found.Placement.Affinity)
	}
}

func TestMarkStaleDisconnectedDevices(t *testing.T) {
	m := NewManager()
	base := time.Date(2026, 4, 12, 10, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return base.Add(-10 * time.Minute) }
	_, _ = m.Register(Manifest{DeviceID: "stale"})
	_, _ = m.Register(Manifest{DeviceID: "fresh"})
	m.now = func() time.Time { return base }
	_ = m.Heartbeat("fresh", base)

	updated := m.MarkStaleDisconnectedDevices(base.Add(-5 * time.Minute))
	if len(updated) != 1 || updated[0] != "stale" {
		t.Fatalf("updated = %+v, want [stale]", updated)
	}
	stale, _ := m.Get("stale")
	if stale.State != StateDisconnected {
		t.Fatalf("stale state = %q, want disconnected", stale.State)
	}
}
