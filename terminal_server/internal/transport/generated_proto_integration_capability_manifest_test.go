package transport

import (
	"context"
	"testing"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
)

// TestGeneratedSessionCapabilityManifestReportedOnConnection validates I3:
// a device reports its full capability manifest (screen, mic, camera, sensors)
// on connection and the server records every declared capability for routing.
func TestGeneratedSessionCapabilityManifestReportedOnConnection(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "tablet-kitchen",
							Identity: &capabilitiesv1.DeviceIdentity{
								DeviceName: "Kitchen Tablet",
								DeviceType: "tablet",
								Platform:   "android",
							},
							Screen: &capabilitiesv1.ScreenCapability{
								Width:  1920,
								Height: 1200,
								Touch:  true,
							},
							Microphone: &capabilitiesv1.AudioInputCapability{
								Channels: 1,
							},
							Camera: &capabilitiesv1.CameraCapability{
								Front: &capabilitiesv1.CameraLens{Width: 1280, Height: 720, Fps: 30},
							},
							Sensors: &capabilitiesv1.SensorCapability{
								AmbientLight: true,
								Proximity:    true,
							},
						},
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	d, ok := devices.Get("tablet-kitchen")
	if !ok {
		t.Fatal("device tablet-kitchen not found in registry after registration")
	}
	if d.DeviceName != "Kitchen Tablet" {
		t.Errorf("DeviceName = %q, want %q", d.DeviceName, "Kitchen Tablet")
	}
	caps := d.Capabilities
	if caps["screen.width"] != "1920" {
		t.Errorf("screen.width = %q, want 1920", caps["screen.width"])
	}
	if caps["screen.height"] != "1200" {
		t.Errorf("screen.height = %q, want 1200", caps["screen.height"])
	}
	if caps["screen.touch"] != "true" {
		t.Errorf("screen.touch = %q, want true", caps["screen.touch"])
	}
	if caps["camera.front.width"] == "" {
		t.Errorf("camera.front.width not recorded; caps = %v", caps)
	}
	if caps["sensors.ambient_light"] != "true" {
		t.Errorf("sensors.ambient_light = %q, want true", caps["sensors.ambient_light"])
	}
	if caps["sensors.proximity"] != "true" {
		t.Errorf("sensors.proximity = %q, want true", caps["sensors.proximity"])
	}
}

// TestWireSessionCapabilityManifestReportedOnConnection validates I3 over the
// wire (JSON) adapter: capability keys sent on registration are stored in the
// device registry for downstream routing and scenario decisions.
func TestWireSessionCapabilityManifestReportedOnConnection(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			WireClientMessage{Register: &WireRegisterRequest{
				DeviceID:   "tablet-living",
				DeviceName: "Living Room Tablet",
				DeviceType: "tablet",
				Platform:   "android",
				Capabilities: []DataEntry{
					{Key: "screen.width", Value: "1280"},
					{Key: "screen.height", Value: "800"},
					{Key: "screen.touch", Value: "true"},
					{Key: "microphone.channels", Value: "1"},
					{Key: "camera.front.width", Value: "640"},
					{Key: "sensors.ambient_light", Value: "true"},
				},
			}},
		},
	}

	if err := RunProtoSession(handler, control, stream, WireProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	d, ok := devices.Get("tablet-living")
	if !ok {
		t.Fatal("device tablet-living not found in registry after registration")
	}
	caps := d.Capabilities
	for _, wantKey := range []string{
		"screen.width", "screen.height", "screen.touch",
		"microphone.channels", "camera.front.width", "sensors.ambient_light",
	} {
		if caps[wantKey] == "" {
			t.Errorf("capability %q not stored; caps = %v", wantKey, caps)
		}
	}
}
