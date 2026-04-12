package transport

import (
	"context"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/device"
)

func TestHandleMessageRegisterSendsAckAndUI(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
			DeviceType: "laptop",
			Platform:   "chromeos",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(register) error = %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("len(out) = %d, want 2", len(out))
	}
	if out[0].RegisterAck == nil {
		t.Fatalf("first response should contain register ack")
	}
	if out[1].SetUI == nil {
		t.Fatalf("second response should contain SetUI")
	}
}

func TestHandleMessageCapabilityAndHeartbeat(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	now := time.Date(2026, 4, 11, 20, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	handler := NewStreamHandler(service)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	})

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Capability: &CapabilityUpdateRequest{
			DeviceID: "device-1",
			Capabilities: map[string]string{
				"screen.width":  "1920",
				"screen.height": "1080",
			},
		},
	}); err != nil {
		t.Fatalf("HandleMessage(capability) error = %v", err)
	}

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Heartbeat: &HeartbeatRequest{DeviceID: "device-1"},
	}); err != nil {
		t.Fatalf("HandleMessage(heartbeat) error = %v", err)
	}

	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected registered device")
	}
	if got.Capabilities["screen.width"] != "1920" {
		t.Fatalf("screen.width = %q, want 1920", got.Capabilities["screen.width"])
	}
	if got.LastHeartbeat != now {
		t.Fatalf("LastHeartbeat = %v, want %v", got.LastHeartbeat, now)
	}
}

func TestHandleMessageInvalid(t *testing.T) {
	manager := device.NewManager()
	service := NewControlService("srv-1", manager)
	handler := NewStreamHandler(service)

	out, err := handler.HandleMessage(context.Background(), ClientMessage{})
	if err != ErrInvalidClientMessage {
		t.Fatalf("err = %v, want %v", err, ErrInvalidClientMessage)
	}
	if len(out) != 1 || out[0].Error == "" {
		t.Fatalf("expected one error response")
	}
}
