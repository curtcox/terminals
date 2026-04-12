package transport

import (
	"context"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
)

func TestWireProtoAdapterSession(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			WireClientMessage{
				Register: &WireRegisterRequest{
					DeviceID:   "device-1",
					DeviceName: "Kitchen Chromebook",
					Capabilities: []DataEntry{
						{Key: "screen.width", Value: "1920"},
					},
				},
			},
			WireClientMessage{
				Heartbeat: &WireHeartbeatRequest{DeviceID: "device-1"},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, WireProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}
	if len(stream.sent) != 2 {
		t.Fatalf("len(sent) = %d, want 2", len(stream.sent))
	}

	got, ok := devices.Get("device-1")
	if !ok {
		t.Fatalf("expected device")
	}
	if got.Capabilities["screen.width"] != "1920" {
		t.Fatalf("screen.width = %q, want 1920", got.Capabilities["screen.width"])
	}
}
