package transport

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/device"
)

func TestRegisterReturnsInitialUI(t *testing.T) {
	m := device.NewManager()
	s := NewControlService("srv-1", m)
	s.SetRegisterMetadata(map[string]string{
		"photo_frame_asset_base_url": "http://photos.example.test/photo-frame",
	})

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
	if got := resp.Metadata["photo_frame_asset_base_url"]; got != "http://photos.example.test/photo-frame" {
		t.Fatalf("metadata photo_frame_asset_base_url = %q, want configured value", got)
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

func TestStatusData(t *testing.T) {
	m := device.NewManager()
	s := NewControlService("srv-1", m)

	started := time.Date(2026, 4, 12, 8, 0, 0, 0, time.UTC)
	now := started.Add(90 * time.Second)
	s.started = started
	s.now = func() time.Time { return now }

	_, _ = s.Register(context.Background(), RegisterRequest{
		DeviceID:   "d1",
		DeviceName: "Kitchen",
	})
	_, _ = s.Register(context.Background(), RegisterRequest{
		DeviceID:   "d2",
		DeviceName: "Hall",
	})
	_ = s.Disconnect(context.Background(), "d2")

	status := s.StatusData()
	if status["server_id"] != "srv-1" {
		t.Fatalf("server_id = %q, want srv-1", status["server_id"])
	}
	if status["devices_total"] != "2" {
		t.Fatalf("devices_total = %q, want 2", status["devices_total"])
	}
	if status["devices_connected"] != "1" || status["devices_disconnected"] != "1" {
		t.Fatalf("unexpected connected/disconnected values: %+v", status)
	}
	gotUptime, err := strconv.Atoi(status["uptime_seconds"])
	if err != nil {
		t.Fatalf("uptime_seconds parse error: %v", err)
	}
	if gotUptime != 90 {
		t.Fatalf("uptime_seconds = %d, want 90", gotUptime)
	}
}

func TestReconcileLiveness(t *testing.T) {
	m := device.NewManager()
	s := NewControlService("srv-1", m)
	base := time.Date(2026, 4, 12, 10, 0, 0, 0, time.UTC)
	stale := base.Add(-10 * time.Minute)
	s.now = func() time.Time { return stale }

	_, _ = s.Register(context.Background(), RegisterRequest{
		DeviceID:   "d1",
		DeviceName: "Kitchen",
	})
	_, _ = s.Register(context.Background(), RegisterRequest{
		DeviceID:   "d2",
		DeviceName: "Hall",
	})
	_ = s.Heartbeat(context.Background(), "d1")
	_ = s.Heartbeat(context.Background(), "d2")

	s.now = func() time.Time { return base }
	// Refresh d2 heartbeat to current base time so only d1 is stale.
	_ = s.Heartbeat(context.Background(), "d2")

	updated := s.ReconcileLiveness(5 * time.Minute)
	if updated != 1 {
		t.Fatalf("updated = %d, want 1", updated)
	}
	d1, _ := m.Get("d1")
	d2, _ := m.Get("d2")
	if d1.State != device.StateDisconnected {
		t.Fatalf("d1 state = %q, want disconnected", d1.State)
	}
	if d2.State != device.StateConnected {
		t.Fatalf("d2 state = %q, want connected", d2.State)
	}
}

func TestReconcileLivenessDeviceIDs(t *testing.T) {
	m := device.NewManager()
	s := NewControlService("srv-1", m)
	base := time.Date(2026, 4, 12, 10, 0, 0, 0, time.UTC)
	stale := base.Add(-10 * time.Minute)
	s.now = func() time.Time { return stale }

	_, _ = s.Register(context.Background(), RegisterRequest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = s.Register(context.Background(), RegisterRequest{DeviceID: "d2", DeviceName: "Hall"})
	_ = s.Heartbeat(context.Background(), "d1")
	_ = s.Heartbeat(context.Background(), "d2")

	s.now = func() time.Time { return base }
	_ = s.Heartbeat(context.Background(), "d2")

	updated := s.ReconcileLivenessDeviceIDs(5 * time.Minute)
	if len(updated) != 1 || updated[0] != "d1" {
		t.Fatalf("updated = %+v, want [d1]", updated)
	}
}
