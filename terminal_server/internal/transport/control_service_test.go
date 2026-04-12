package transport

import (
	"context"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/device"
)

func TestRegisterReturnsInitialUI(t *testing.T) {
	m := device.NewManager()
	s := NewControlService("srv-1", m)

	resp, err := s.Register(context.Background(), RegisterRequest{
		DeviceID:   "device-1",
		DeviceName: "Kitchen Chromebook",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if resp.ServerID != "srv-1" {
		t.Fatalf("ServerID = %q, want srv-1", resp.ServerID)
	}
	if resp.Initial.Type != "stack" {
		t.Fatalf("Initial.Type = %q, want stack", resp.Initial.Type)
	}
}

func TestHeartbeat(t *testing.T) {
	m := device.NewManager()
	s := NewControlService("srv-1", m)
	_, _ = s.Register(context.Background(), RegisterRequest{
		DeviceID:   "device-1",
		DeviceName: "Kitchen Chromebook",
	})

	fixed := time.Date(2026, 4, 11, 18, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return fixed }
	if err := s.Heartbeat(context.Background(), "device-1"); err != nil {
		t.Fatalf("Heartbeat() error = %v", err)
	}

	got, ok := m.Get("device-1")
	if !ok {
		t.Fatalf("expected device to exist")
	}
	if got.LastHeartbeat != fixed {
		t.Fatalf("LastHeartbeat = %v, want %v", got.LastHeartbeat, fixed)
	}
}
