package transport

import (
	"context"
	"sync"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/placement"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

// TestGeneratedSessionDeviceRegistryCapabilityQueryRoutesScenarioToMatchingDevices
// validates I4: after multiple devices register via the generated proto adapter,
// the placement engine returns only devices whose declared capabilities match the
// query — routing scenarios to appropriate devices without client changes.
func TestGeneratedSessionDeviceRegistryCapabilityQueryRoutesScenarioToMatchingDevices(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	streamA := &asyncFakeProtoStream{
		ctx:    context.Background(),
		recvCh: make(chan ProtoClientEnvelope, 4),
		sentCh: make(chan ProtoServerEnvelope, 8),
	}
	streamB := &asyncFakeProtoStream{
		ctx:    context.Background(),
		recvCh: make(chan ProtoClientEnvelope, 4),
		sentCh: make(chan ProtoServerEnvelope, 8),
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = RunProtoSession(handler, control, streamA, GeneratedProtoAdapter{})
	}()
	go func() {
		defer wg.Done()
		_ = RunProtoSession(handler, control, streamB, GeneratedProtoAdapter{})
	}()

	waitAck := func(label string, ch <-chan ProtoServerEnvelope) {
		deadline := time.After(2 * time.Second)
		for {
			select {
			case <-ch:
				return
			case <-deadline:
				t.Fatalf("timed out waiting for registration ACK from %s", label)
			}
		}
	}

	// Register device A: camera + screen.
	streamA.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "camera-tablet",
					Identity: &capabilitiesv1.DeviceIdentity{
						DeviceName: "Camera Tablet",
						DeviceType: "tablet",
						Platform:   "android",
					},
					Screen: &capabilitiesv1.ScreenCapability{
						Width: 1920, Height: 1200, Touch: true,
					},
					Camera: &capabilitiesv1.CameraCapability{
						Front: &capabilitiesv1.CameraLens{Width: 1280, Height: 720, Fps: 30},
					},
				},
			},
		},
	}
	waitAck("camera-tablet", streamA.sentCh)

	// Register device B: screen only, no camera.
	streamB.recvCh <- &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "screen-only",
					Identity: &capabilitiesv1.DeviceIdentity{
						DeviceName: "Screen Only",
						DeviceType: "tablet",
						Platform:   "android",
					},
					Screen: &capabilitiesv1.ScreenCapability{
						Width: 1280, Height: 800, Touch: true,
					},
				},
			},
		},
	}
	waitAck("screen-only", streamB.sentCh)

	// Both devices are now connected; query the placement engine.
	engine := placement.NewManagerBackedEngine(devices)

	// Camera query must return only the device that has a camera.
	cameraDevices, err := engine.Find(context.Background(), scenario.PlacementQuery{
		RequiredCaps: []string{"camera"},
	})
	if err != nil {
		t.Errorf("Find(camera) error = %v", err)
	} else if len(cameraDevices) != 1 || cameraDevices[0].DeviceID != "camera-tablet" {
		t.Errorf("Find(camera) = %+v, want [camera-tablet]", cameraDevices)
	}

	// Screen query must return both devices.
	screenDevices, err := engine.Find(context.Background(), scenario.PlacementQuery{
		RequiredCaps: []string{"screen"},
	})
	if err != nil {
		t.Errorf("Find(screen) error = %v", err)
	} else if len(screenDevices) != 2 {
		t.Errorf("Find(screen) returned %d device(s), want 2; got %+v", len(screenDevices), screenDevices)
	}

	// Multi-cap query: camera AND screen — only camera-tablet qualifies.
	multiDevices, err := engine.Find(context.Background(), scenario.PlacementQuery{
		RequiredCaps: []string{"camera", "screen"},
	})
	if err != nil {
		t.Errorf("Find(camera+screen) error = %v", err)
	} else if len(multiDevices) != 1 || multiDevices[0].DeviceID != "camera-tablet" {
		t.Errorf("Find(camera+screen) = %+v, want [camera-tablet]", multiDevices)
	}

	close(streamA.recvCh)
	close(streamB.recvCh)
	wg.Wait()
}

// TestWireSessionDeviceRegistryCapabilityQueryRoutesScenarioToMatchingDevices
// validates I4 over the wire (JSON) adapter: capabilities declared on connection
// are stored in the device registry and the placement engine routes based on them.
func TestWireSessionDeviceRegistryCapabilityQueryRoutesScenarioToMatchingDevices(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	streamA := &asyncFakeProtoStream{
		ctx:    context.Background(),
		recvCh: make(chan ProtoClientEnvelope, 4),
		sentCh: make(chan ProtoServerEnvelope, 8),
	}
	streamB := &asyncFakeProtoStream{
		ctx:    context.Background(),
		recvCh: make(chan ProtoClientEnvelope, 4),
		sentCh: make(chan ProtoServerEnvelope, 8),
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = RunProtoSession(handler, control, streamA, WireProtoAdapter{})
	}()
	go func() {
		defer wg.Done()
		_ = RunProtoSession(handler, control, streamB, WireProtoAdapter{})
	}()

	waitAck := func(label string, ch <-chan ProtoServerEnvelope) {
		deadline := time.After(2 * time.Second)
		for {
			select {
			case <-ch:
				return
			case <-deadline:
				t.Fatalf("timed out waiting for registration ACK from %s", label)
			}
		}
	}

	// Register device A: microphone + speaker + screen.
	streamA.recvCh <- WireClientMessage{Register: &WireRegisterRequest{
		DeviceID:   "mic-speaker",
		DeviceName: "Mic Speaker",
		DeviceType: "tablet",
		Platform:   "android",
		Capabilities: []DataEntry{
			{Key: "microphone.channels", Value: "2"},
			{Key: "speaker.main", Value: "true"},
			{Key: "screen.width", Value: "1024"},
		},
	}}
	waitAck("mic-speaker", streamA.sentCh)

	// Register device B: speaker only.
	streamB.recvCh <- WireClientMessage{Register: &WireRegisterRequest{
		DeviceID:   "speaker-only",
		DeviceName: "Speaker Only",
		DeviceType: "speaker",
		Platform:   "linux",
		Capabilities: []DataEntry{
			{Key: "speaker.main", Value: "true"},
		},
	}}
	waitAck("speaker-only", streamB.sentCh)

	// Both devices connected; query via the placement engine.
	engine := placement.NewManagerBackedEngine(devices)

	// Microphone query returns only the device that declared one.
	micDevices, err := engine.Find(context.Background(), scenario.PlacementQuery{
		RequiredCaps: []string{"microphone"},
	})
	if err != nil {
		t.Errorf("Find(microphone) error = %v", err)
	} else if len(micDevices) != 1 || micDevices[0].DeviceID != "mic-speaker" {
		t.Errorf("Find(microphone) = %+v, want [mic-speaker]", micDevices)
	}

	// Speaker query returns both devices.
	speakerDevices, err := engine.Find(context.Background(), scenario.PlacementQuery{
		RequiredCaps: []string{"speaker"},
	})
	if err != nil {
		t.Errorf("Find(speaker) error = %v", err)
	} else if len(speakerDevices) != 2 {
		t.Errorf("Find(speaker) returned %d device(s), want 2; got %+v", len(speakerDevices), speakerDevices)
	}

	close(streamA.recvCh)
	close(streamB.recvCh)
	wg.Wait()
}
