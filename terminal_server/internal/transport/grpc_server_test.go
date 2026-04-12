package transport

import (
	"context"
	"testing"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
)

func TestServerLifecycle(t *testing.T) {
	s := NewServer("127.0.0.1:50051")
	if s.Running() {
		t.Fatalf("expected server to start stopped")
	}
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !s.Running() {
		t.Fatalf("expected server running after Start")
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if s.Running() {
		t.Fatalf("expected server stopped after Stop")
	}
}

func TestServerConnectRequiresConfiguration(t *testing.T) {
	s := NewServer("127.0.0.1:50051")
	err := s.Connect(&fakeProtoStream{ctx: context.Background()})
	if err != ErrControlNotConfigured {
		t.Fatalf("Connect() error = %v, want %v", err, ErrControlNotConfigured)
	}
}

func TestServerConnectRunsSession(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	s := NewServer("127.0.0.1:50051")
	s.ConfigureControl(control, PassthroughProtoAdapter{})

	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			ClientMessage{
				Register: &RegisterRequest{
					DeviceID:   "device-1",
					DeviceName: "Kitchen Chromebook",
				},
			},
			ClientMessage{
				Heartbeat: &HeartbeatRequest{
					DeviceID: "device-1",
				},
			},
		},
	}

	if err := s.Connect(stream); err != nil {
		t.Fatalf("Connect() error = %v", err)
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

func TestServerConnectRunsSessionWithWireAdapter(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	s := NewServer("127.0.0.1:50051")
	s.ConfigureControl(control, WireProtoAdapter{})

	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			WireClientMessage{
				Register: &WireRegisterRequest{
					DeviceID:   "device-1",
					DeviceName: "Kitchen Chromebook",
				},
			},
			WireClientMessage{
				Heartbeat: &WireHeartbeatRequest{
					DeviceID: "device-1",
				},
			},
		},
	}

	if err := s.Connect(stream); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if len(stream.sent) != 2 {
		t.Fatalf("len(sent) = %d, want 2", len(stream.sent))
	}
	first, ok := stream.sent[0].(WireServerMessage)
	if !ok {
		t.Fatalf("first sent envelope type = %T, want WireServerMessage", stream.sent[0])
	}
	if first.RegisterAck == nil || first.RegisterAck.ServerID != "srv-1" {
		t.Fatalf("unexpected register ack payload: %+v", first.RegisterAck)
	}

	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected registered device")
	}
	if got.State != device.StateDisconnected {
		t.Fatalf("state = %q, want %q", got.State, device.StateDisconnected)
	}
}

func TestServerConnectRunsSessionWithGeneratedAdapter(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	s := NewServer("127.0.0.1:50051")
	s.ConfigureControl(control, GeneratedProtoAdapter{})

	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{
								DeviceName: "Kitchen Chromebook",
							},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Heartbeat{
					Heartbeat: &controlv1.Heartbeat{DeviceId: "device-1"},
				},
			},
		},
	}

	if err := s.Connect(stream); err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if len(stream.sent) != 2 {
		t.Fatalf("len(sent) = %d, want 2", len(stream.sent))
	}
	first, ok := stream.sent[0].(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("first sent envelope type = %T, want *controlv1.ConnectResponse", stream.sent[0])
	}
	if first.GetRegisterAck() == nil || first.GetRegisterAck().GetServerId() != "srv-1" {
		t.Fatalf("unexpected register ack payload: %+v", first.GetRegisterAck())
	}

	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected registered device")
	}
	if got.State != device.StateDisconnected {
		t.Fatalf("state = %q, want %q", got.State, device.StateDisconnected)
	}
}
