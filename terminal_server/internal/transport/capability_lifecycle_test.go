package transport

import (
	"context"
	"errors"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
)

func newLifecycleForTest() *CapabilityLifecycle {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	return NewCapabilityLifecycle(service)
}

// TestHandleHelloReturnsHelloAck pins the Hello result shape.
func TestHandleHelloReturnsHelloAck(t *testing.T) {
	lc := newLifecycleForTest()
	out, err := lc.HandleHello(context.Background(), HelloRequest{
		DeviceID:   "device-hello",
		DeviceName: "Test Device",
	})
	if err != nil {
		t.Fatalf("HandleHello error = %v", err)
	}
	if len(out) != 1 || out[0].HelloAck == nil {
		t.Fatalf("expected single HelloAck, got %#v", out)
	}
	if out[0].HelloAck.SessionID == "" {
		t.Fatalf("HelloAck missing session id")
	}
}

// TestHandleRegisterReturnsRegisterAck pins the deprecated Register path.
func TestHandleRegisterReturnsRegisterAck(t *testing.T) {
	lc := newLifecycleForTest()
	resp, err := lc.HandleRegister(context.Background(), RegisterRequest{
		DeviceID:   "device-register",
		DeviceName: "Kitchen Chromebook",
		DeviceType: "laptop",
		Platform:   "chromeos",
	})
	if err != nil {
		t.Fatalf("HandleRegister error = %v", err)
	}
	if resp.ServerID != "srv-1" {
		t.Fatalf("ServerID = %q, want srv-1", resp.ServerID)
	}
	if resp.Initial.Type == "" {
		t.Fatalf("RegisterResponse missing Initial UI")
	}
}

// TestHandleSnapshotEstablishesBaseline verifies the first snapshot for an
// unknown device falls back through the implicit-Hello path and reports
// IsInitialBaseline=true with no prior caps.
func TestHandleSnapshotEstablishesBaseline(t *testing.T) {
	lc := newLifecycleForTest()
	result, err := lc.HandleSnapshot(context.Background(), CapabilitySnapshotRequest{
		DeviceID:   "device-snap",
		Generation: 1,
		Capabilities: map[string]string{
			"device_name":   "Snap Device",
			"screen.width":  "1920",
			"screen.height": "1080",
		},
	})
	if err != nil {
		t.Fatalf("HandleSnapshot error = %v", err)
	}
	if !result.IsInitialBaseline {
		t.Errorf("IsInitialBaseline = false, want true on first snapshot")
	}
	if result.HadPriorDevice {
		t.Errorf("HadPriorDevice = true, want false for unknown device")
	}
	if len(result.Messages) != 2 {
		t.Fatalf("len(Messages) = %d, want 2 (CapabilityAck + RegisterAck)", len(result.Messages))
	}
	if result.Messages[0].CapabilityAck == nil {
		t.Errorf("first message should be CapabilityAck")
	}
	if result.Messages[1].RegisterAck == nil {
		t.Errorf("second message should be RegisterAck")
	}
	if result.RegisterAck == nil || result.RegisterAck != result.Messages[1].RegisterAck {
		t.Errorf("RegisterAck pointer should match Messages[1].RegisterAck")
	}
	if got := result.AfterCaps["screen.width"]; got != "1920" {
		t.Errorf("AfterCaps[screen.width] = %q, want 1920", got)
	}
}

// TestHandleSnapshotForExistingDeviceIsNotInitialBaseline ensures a second
// snapshot over an existing device records HadPriorDevice and clears the
// IsInitialBaseline flag so capability-change effects fire downstream.
func TestHandleSnapshotForExistingDeviceIsNotInitialBaseline(t *testing.T) {
	lc := newLifecycleForTest()
	ctx := context.Background()
	if _, err := lc.HandleSnapshot(ctx, CapabilitySnapshotRequest{
		DeviceID:   "device-existing",
		Generation: 1,
		Capabilities: map[string]string{
			"device_name":  "Existing",
			"screen.width": "1024",
		},
	}); err != nil {
		t.Fatalf("seed snapshot error = %v", err)
	}
	result, err := lc.HandleSnapshot(ctx, CapabilitySnapshotRequest{
		DeviceID:   "device-existing",
		Generation: 2,
		Capabilities: map[string]string{
			"device_name":  "Existing",
			"screen.width": "1024",
		},
	})
	if err != nil {
		t.Fatalf("second snapshot error = %v", err)
	}
	if result.IsInitialBaseline {
		t.Errorf("IsInitialBaseline = true on second snapshot, want false")
	}
	if !result.HadPriorDevice {
		t.Errorf("HadPriorDevice = false, want true on second snapshot")
	}
}

// TestHandleDeltaUpdatesCapabilities verifies a delta returns CapabilityAck and
// before/after caps with the new values applied.
func TestHandleDeltaUpdatesCapabilities(t *testing.T) {
	lc := newLifecycleForTest()
	ctx := context.Background()
	if _, err := lc.HandleSnapshot(ctx, CapabilitySnapshotRequest{
		DeviceID:   "device-delta",
		Generation: 1,
		Capabilities: map[string]string{
			"device_name":  "Delta",
			"screen.width": "1024",
		},
	}); err != nil {
		t.Fatalf("seed snapshot error = %v", err)
	}
	result, err := lc.HandleDelta(ctx, CapabilityDeltaRequest{
		DeviceID:   "device-delta",
		Generation: 2,
		Capabilities: map[string]string{
			"screen.width": "2048",
		},
	})
	if err != nil {
		t.Fatalf("HandleDelta error = %v", err)
	}
	if len(result.Messages) != 1 || result.Messages[0].CapabilityAck == nil {
		t.Fatalf("expected single CapabilityAck, got %#v", result.Messages)
	}
	if result.BeforeCaps["screen.width"] != "1024" {
		t.Errorf("BeforeCaps[screen.width] = %q, want 1024", result.BeforeCaps["screen.width"])
	}
	if result.AfterCaps["screen.width"] != "2048" {
		t.Errorf("AfterCaps[screen.width] = %q, want 2048", result.AfterCaps["screen.width"])
	}
}

// TestHandleDeltaStaleGenerationReturnsError preserves the existing protocol
// violation behavior for stale capability deltas.
func TestHandleDeltaStaleGenerationReturnsError(t *testing.T) {
	lc := newLifecycleForTest()
	ctx := context.Background()
	if _, err := lc.HandleSnapshot(ctx, CapabilitySnapshotRequest{
		DeviceID:   "device-stale",
		Generation: 5,
		Capabilities: map[string]string{
			"device_name": "Stale",
		},
	}); err != nil {
		t.Fatalf("seed snapshot error = %v", err)
	}
	_, err := lc.HandleDelta(ctx, CapabilityDeltaRequest{
		DeviceID:   "device-stale",
		Generation: 2,
		Capabilities: map[string]string{
			"screen.width": "10",
		},
	})
	if err == nil {
		t.Fatalf("HandleDelta with stale generation should return an error")
	}
	if !errors.Is(err, device.ErrStaleGeneration) {
		t.Errorf("err = %v, want ErrStaleGeneration", err)
	}
}

// TestHandleSnapshotInvalidDeviceIDReturnsError preserves the protocol error
// mapping for malformed device IDs.
func TestHandleSnapshotInvalidDeviceIDReturnsError(t *testing.T) {
	lc := newLifecycleForTest()
	_, err := lc.HandleSnapshot(context.Background(), CapabilitySnapshotRequest{
		DeviceID:   "",
		Generation: 1,
	})
	if err == nil {
		t.Fatalf("HandleSnapshot with empty device id should return an error")
	}
}

// TestHandleUpdateCapabilitiesAppliesCapabilities exercises the deprecated
// Capability path and verifies it returns no ack messages on success.
func TestHandleUpdateCapabilitiesAppliesCapabilities(t *testing.T) {
	lc := newLifecycleForTest()
	ctx := context.Background()
	if _, err := lc.HandleRegister(ctx, RegisterRequest{
		DeviceID:   "device-legacy",
		DeviceName: "Legacy",
	}); err != nil {
		t.Fatalf("seed register error = %v", err)
	}
	if err := lc.HandleUpdateCapabilities(ctx, CapabilityUpdateRequest{
		DeviceID: "device-legacy",
		Capabilities: map[string]string{
			"keyboard.physical": "true",
		},
	}); err != nil {
		t.Fatalf("HandleUpdateCapabilities error = %v", err)
	}
}

// TestHandleSnapshotCapabilityInvalidations verifies the lifecycle reports
// capability invalidations on the CapabilityAck when resources are lost.
func TestHandleSnapshotCapabilityInvalidations(t *testing.T) {
	lc := newLifecycleForTest()
	ctx := context.Background()
	if _, err := lc.HandleSnapshot(ctx, CapabilitySnapshotRequest{
		DeviceID:   "device-inval",
		Generation: 1,
		Capabilities: map[string]string{
			"device_name":   "Inval",
			"screen.width":  "1024",
			"screen.height": "768",
		},
	}); err != nil {
		t.Fatalf("seed snapshot error = %v", err)
	}
	result, err := lc.HandleSnapshot(ctx, CapabilitySnapshotRequest{
		DeviceID:   "device-inval",
		Generation: 2,
		Capabilities: map[string]string{
			"device_name": "Inval",
		},
	})
	if err != nil {
		t.Fatalf("second snapshot error = %v", err)
	}
	if len(result.Messages) == 0 || result.Messages[0].CapabilityAck == nil {
		t.Fatalf("expected CapabilityAck in messages")
	}
	ack := result.Messages[0].CapabilityAck
	if len(ack.Invalidations) == 0 {
		t.Errorf("expected capability invalidations after losing screen resources")
	}
}
