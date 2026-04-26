package placement

import (
	"context"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
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

func TestFindRequiredCapsMissingFieldIsUnsupported(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{
		DeviceID:     "camera-available",
		Capabilities: device.CapabilitySet{"camera.front": "true"},
	})
	_, _ = devices.Register(device.Manifest{
		DeviceID:     "camera-absent",
		Capabilities: device.CapabilitySet{"screen.width": "1024"},
	})

	engine := NewManagerBackedEngine(devices)
	got, err := engine.Find(context.Background(), scenario.PlacementQuery{
		RequiredCaps: []string{"camera"},
	})
	if err != nil {
		t.Fatalf("Find(camera) error = %v", err)
	}
	if len(got) != 1 || got[0].DeviceID != "camera-available" {
		t.Fatalf("Find(camera) = %+v, want [camera-available]", got)
	}
}

func TestFindRequiredCapsFalseValuesAreUnsupported(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{
		DeviceID:     "camera-true",
		Capabilities: device.CapabilitySet{"camera.front": "true"},
	})
	_, _ = devices.Register(device.Manifest{
		DeviceID: "camera-false",
		Capabilities: device.CapabilitySet{
			"camera": "false",
		},
	})
	_, _ = devices.Register(device.Manifest{
		DeviceID: "camera-zero",
		Capabilities: device.CapabilitySet{
			"camera": "0",
		},
	})
	_, _ = devices.Register(device.Manifest{
		DeviceID: "camera-off",
		Capabilities: device.CapabilitySet{
			"camera": "off",
		},
	})

	engine := NewManagerBackedEngine(devices)
	got, err := engine.Find(context.Background(), scenario.PlacementQuery{
		RequiredCaps: []string{"camera"},
	})
	if err != nil {
		t.Fatalf("Find(camera) error = %v", err)
	}
	if len(got) != 1 || got[0].DeviceID != "camera-true" {
		t.Fatalf("Find(camera) = %+v, want [camera-true]", got)
	}
}

func TestFindRequiredCapsMissingAndFalseValuesAreUnsupportedAcrossMediaCaps(t *testing.T) {
	testCases := []struct {
		name           string
		requiredCap    string
		availableField string
	}{
		{
			name:           "camera",
			requiredCap:    "camera",
			availableField: "camera.front",
		},
		{
			name:           "microphone",
			requiredCap:    "microphone",
			availableField: "microphone.endpoint.main",
		},
		{
			name:           "speakers",
			requiredCap:    "speakers",
			availableField: "speakers.endpoint.main",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			devices := device.NewManager()
			_, _ = devices.Register(device.Manifest{
				DeviceID:     tc.name + "-available",
				Capabilities: device.CapabilitySet{tc.availableField: "true"},
			})
			_, _ = devices.Register(device.Manifest{
				DeviceID:     tc.name + "-absent",
				Capabilities: device.CapabilitySet{"screen.width": "1024"},
			})
			_, _ = devices.Register(device.Manifest{
				DeviceID:     tc.name + "-false",
				Capabilities: device.CapabilitySet{tc.requiredCap: "false"},
			})

			engine := NewManagerBackedEngine(devices)
			got, err := engine.Find(context.Background(), scenario.PlacementQuery{
				RequiredCaps: []string{tc.requiredCap},
			})
			if err != nil {
				t.Fatalf("Find(%s) error = %v", tc.requiredCap, err)
			}
			if len(got) != 1 || got[0].DeviceID != tc.name+"-available" {
				t.Fatalf("Find(%s) = %+v, want [%s-available]", tc.requiredCap, got, tc.name)
			}
		})
	}
}

func TestFindBackgroundRoleSkipsForegroundOnlyDevices(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{
		DeviceID: "foreground-only",
		Capabilities: device.CapabilitySet{
			"monitor.support_tier":       "foreground_only",
			"monitor.background_capable": "false",
		},
	})
	_, _ = devices.Register(device.Manifest{
		DeviceID: "background-capable",
		Capabilities: device.CapabilitySet{
			"monitor.support_tier":       "background_capable",
			"monitor.background_capable": "true",
		},
	})
	_ = devices.UpdatePlacement("foreground-only", device.PlacementMetadata{
		Roles: []string{"background_monitor"},
	})
	_ = devices.UpdatePlacement("background-capable", device.PlacementMetadata{
		Roles: []string{"background_monitor"},
	})

	engine := NewManagerBackedEngine(devices)
	got, err := engine.DevicesWithRole(context.Background(), "background_monitor")
	if err != nil {
		t.Fatalf("DevicesWithRole(background_monitor) error = %v", err)
	}
	if len(got) != 1 || got[0].DeviceID != "background-capable" {
		t.Fatalf("DevicesWithRole(background_monitor) = %+v, want [background-capable]", got)
	}
}

func TestFindBackgroundRoleExplicitFalseOverridesSupportTier(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{
		DeviceID: "background-false",
		Capabilities: device.CapabilitySet{
			"monitor.background_capable": "false",
			"monitor.support_tier":       "background_capable",
		},
	})
	_, _ = devices.Register(device.Manifest{
		DeviceID: "background-true",
		Capabilities: device.CapabilitySet{
			"monitor.background_capable": "true",
			"monitor.support_tier":       "background_capable",
		},
	})
	_ = devices.UpdatePlacement("background-false", device.PlacementMetadata{
		Roles: []string{"background_monitor"},
	})
	_ = devices.UpdatePlacement("background-true", device.PlacementMetadata{
		Roles: []string{"background_monitor"},
	})

	engine := NewManagerBackedEngine(devices)
	got, err := engine.DevicesWithRole(context.Background(), "background_monitor")
	if err != nil {
		t.Fatalf("DevicesWithRole(background_monitor) error = %v", err)
	}
	if len(got) != 1 || got[0].DeviceID != "background-true" {
		t.Fatalf("DevicesWithRole(background_monitor) = %+v, want [background-true]", got)
	}
}

func TestFindExcludeBusyUsesActiveClaims(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{
		DeviceID:     "busy-screen",
		Capabilities: device.CapabilitySet{"screen.width": "1920"},
	})
	_, _ = devices.Register(device.Manifest{
		DeviceID:     "idle-screen",
		Capabilities: device.CapabilitySet{"screen.width": "1280"},
	})

	claims := iorouter.NewClaimManager()
	_, err := claims.Request(context.Background(), []iorouter.Claim{{
		ActivationID: "act-1",
		DeviceID:     "busy-screen",
		Resource:     iorouter.ResourceComputeCPUShared,
		Mode:         iorouter.ClaimShared,
		Priority:     1,
	}})
	if err != nil {
		t.Fatalf("claims.Request() error = %v", err)
	}

	engine := NewManagerBackedEngine(devices, claims)
	got, err := engine.Find(context.Background(), scenario.PlacementQuery{
		RequiredCaps: []string{"screen"},
		ExcludeBusy:  true,
	})
	if err != nil {
		t.Fatalf("Find(screen, excludeBusy=true) error = %v", err)
	}
	if len(got) != 1 || got[0].DeviceID != "idle-screen" {
		t.Fatalf("Find(screen, excludeBusy=true) = %+v, want [idle-screen]", got)
	}
}
