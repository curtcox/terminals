package placement

import (
	"context"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

func TestFindByZoneAndRole(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{
		DeviceID:     "kitchen-display",
		Capabilities: device.CapabilitySet{"screen.width": "1920"},
	})
	_, _ = devices.Register(device.Manifest{
		DeviceID:     "living-speaker",
		Capabilities: device.CapabilitySet{"speaker.main": "true"},
	})
	_ = devices.UpdatePlacement("kitchen-display", device.PlacementMetadata{
		Zone:  "kitchen",
		Roles: []string{"kitchen_display", "screen"},
	})
	_ = devices.UpdatePlacement("living-speaker", device.PlacementMetadata{
		Zone:  "living_room",
		Roles: []string{"speaker"},
	})

	engine := NewManagerBackedEngine(devices)

	inZone, err := engine.DevicesInZone(context.Background(), "kitchen")
	if err != nil {
		t.Fatalf("DevicesInZone(kitchen) error = %v", err)
	}
	if len(inZone) != 1 || inZone[0].DeviceID != "kitchen-display" {
		t.Fatalf("DevicesInZone(kitchen) = %+v, want kitchen-display", inZone)
	}

	withRole, err := engine.DevicesWithRole(context.Background(), "speaker")
	if err != nil {
		t.Fatalf("DevicesWithRole(speaker) error = %v", err)
	}
	if len(withRole) != 1 || withRole[0].DeviceID != "living-speaker" {
		t.Fatalf("DevicesWithRole(speaker) = %+v, want living-speaker", withRole)
	}
}

func TestNearestWithPrefersSameZoneAndSkipsSource(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{
		DeviceID:     "source",
		Capabilities: device.CapabilitySet{"mic.capture": "true"},
	})
	_, _ = devices.Register(device.Manifest{
		DeviceID:     "kitchen-screen",
		Capabilities: device.CapabilitySet{"screen.width": "1280"},
	})
	_, _ = devices.Register(device.Manifest{
		DeviceID:     "bedroom-screen",
		Capabilities: device.CapabilitySet{"screen.width": "1920"},
	})
	_ = devices.UpdatePlacement("source", device.PlacementMetadata{Zone: "kitchen"})
	_ = devices.UpdatePlacement("kitchen-screen", device.PlacementMetadata{Zone: "kitchen"})
	_ = devices.UpdatePlacement("bedroom-screen", device.PlacementMetadata{Zone: "bedroom"})

	engine := NewManagerBackedEngine(devices)

	got, err := engine.NearestWith(context.Background(), scenario.DeviceRef{DeviceID: "source"}, "screen")
	if err != nil {
		t.Fatalf("NearestWith(source, screen) error = %v", err)
	}
	if got.DeviceID != "kitchen-screen" {
		t.Fatalf("NearestWith(source, screen) = %q, want kitchen-screen", got.DeviceID)
	}
}

func TestFindRequiredCaps(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{
		DeviceID:     "camera-1",
		Capabilities: device.CapabilitySet{"camera.front": "true"},
	})
	_, _ = devices.Register(device.Manifest{
		DeviceID:     "screen-1",
		Capabilities: device.CapabilitySet{"screen.width": "1024"},
	})
	engine := NewManagerBackedEngine(devices)

	got, err := engine.Find(context.Background(), scenario.PlacementQuery{
		RequiredCaps: []string{"camera"},
	})
	if err != nil {
		t.Fatalf("Find(camera) error = %v", err)
	}
	if len(got) != 1 || got[0].DeviceID != "camera-1" {
		t.Fatalf("Find(camera) = %+v, want camera-1", got)
	}
}
