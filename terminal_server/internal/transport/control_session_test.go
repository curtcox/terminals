package transport

import (
	"context"
	"io"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
)

type fakeStream struct {
	ctx       context.Context
	recvQueue []ClientMessage
	sent      []ServerMessage
}

func (f *fakeStream) Recv() (ClientMessage, error) {
	if len(f.recvQueue) == 0 {
		return ClientMessage{}, io.EOF
	}
	msg := f.recvQueue[0]
	f.recvQueue = f.recvQueue[1:]
	return msg, nil
}

func (f *fakeStream) Send(msg ServerMessage) error {
	f.sent = append(f.sent, msg)
	return nil
}

func (f *fakeStream) Context() context.Context {
	return f.ctx
}

func TestSessionRunRegisterAndDisconnect(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	handler := NewStreamHandler(control)
	session := NewSession(handler, control)

	stream := &fakeStream{
		ctx: context.Background(),
		recvQueue: []ClientMessage{
			{
				Register: &RegisterRequest{
					DeviceID:   "device-1",
					DeviceName: "Kitchen Chromebook",
				},
			},
		},
	}

	if err := session.Run(stream); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(stream.sent) != 2 {
		t.Fatalf("len(sent) = %d, want 2", len(stream.sent))
	}

	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected registered device")
	}
	if got.State != device.StateDisconnected {
		t.Fatalf("state = %q, want %q", got.State, device.StateDisconnected)
	}
}

func TestSessionRunCapabilityAndHeartbeat(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	handler := NewStreamHandler(control)
	session := NewSession(handler, control)

	stream := &fakeStream{
		ctx: context.Background(),
		recvQueue: []ClientMessage{
			{Register: &RegisterRequest{DeviceID: "d1", DeviceName: "Kitchen"}},
			{Capability: &CapabilityUpdateRequest{
				DeviceID: "d1",
				Capabilities: map[string]string{
					"screen.width": "1920",
				},
			}},
			{Heartbeat: &HeartbeatRequest{DeviceID: "d1"}},
		},
	}

	if err := session.Run(stream); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got, ok := manager.Get("d1")
	if !ok {
		t.Fatalf("expected device d1")
	}
	if got.Capabilities["screen.width"] != "1920" {
		t.Fatalf("screen.width = %q, want 1920", got.Capabilities["screen.width"])
	}
}
