package transport

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
)

type fakeProtoClientEnvelope struct {
	client ClientMessage
}

type fakeProtoServerEnvelope struct {
	server ServerMessage
}

type fakeProtoAdapter struct{}

func (fakeProtoAdapter) ToInternal(env ProtoClientEnvelope) (ClientMessage, error) {
	typed, ok := env.(fakeProtoClientEnvelope)
	if !ok {
		return ClientMessage{}, errors.New("unexpected proto client envelope")
	}
	return typed.client, nil
}

func (fakeProtoAdapter) FromInternal(msg ServerMessage) (ProtoServerEnvelope, error) {
	return fakeProtoServerEnvelope{server: msg}, nil
}

type fakeProtoStream struct {
	ctx        context.Context
	recvQueue  []ProtoClientEnvelope
	sent       []ProtoServerEnvelope
	failOnSend bool
}

func (f *fakeProtoStream) RecvProto() (ProtoClientEnvelope, error) {
	if len(f.recvQueue) == 0 {
		return nil, io.EOF
	}
	env := f.recvQueue[0]
	f.recvQueue = f.recvQueue[1:]
	return env, nil
}

func (f *fakeProtoStream) SendProto(env ProtoServerEnvelope) error {
	if f.failOnSend {
		return errors.New("send failed")
	}
	f.sent = append(f.sent, env)
	return nil
}

func (f *fakeProtoStream) Context() context.Context { return f.ctx }

func TestRunProtoSession(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			fakeProtoClientEnvelope{
				client: ClientMessage{
					Register: &RegisterRequest{
						DeviceID:   "device-1",
						DeviceName: "Kitchen Chromebook",
					},
				},
			},
			fakeProtoClientEnvelope{
				client: ClientMessage{
					Heartbeat: &HeartbeatRequest{DeviceID: "device-1"},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, fakeProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}
	if len(stream.sent) != 2 {
		t.Fatalf("len(sent) = %d, want 2", len(stream.sent))
	}

	got, ok := devices.Get("device-1")
	if !ok {
		t.Fatalf("expected device to exist")
	}
	if got.State != device.StateDisconnected {
		t.Fatalf("state = %q, want %q", got.State, device.StateDisconnected)
	}
}

func TestRunProtoSessionNilGuards(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)

	err := RunProtoSession(handler, control, nil, fakeProtoAdapter{})
	if err != ErrNilProtoStream {
		t.Fatalf("err = %v, want %v", err, ErrNilProtoStream)
	}

	stream := &fakeProtoStream{ctx: context.Background()}
	err = RunProtoSession(handler, control, stream, nil)
	if err != ErrNilProtoAdapter {
		t.Fatalf("err = %v, want %v", err, ErrNilProtoAdapter)
	}
}
