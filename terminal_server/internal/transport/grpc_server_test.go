package transport

import (
	"context"
	"net"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestServerLifecycle(t *testing.T) {
	s := NewServer(mustAvailableTCPAddress(t))
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
	s := NewServer(mustAvailableTCPAddress(t))
	err := s.Connect(&fakeProtoStream{ctx: context.Background()})
	if err != ErrControlNotConfigured {
		t.Fatalf("Connect() error = %v, want %v", err, ErrControlNotConfigured)
	}
}

func TestServerConnectRunsSession(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	control.SetRegisterMetadata(map[string]string{
		"photo_frame_asset_base_url": "http://home.local:50052/photo-frame",
	})
	s := NewServer(mustAvailableTCPAddress(t))
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
	s := NewServer(mustAvailableTCPAddress(t))
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
	control.SetRegisterMetadata(map[string]string{
		"photo_frame_asset_base_url": "http://home.local:50052/photo-frame",
	})
	s := NewServer(mustAvailableTCPAddress(t))
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
	if got := first.GetRegisterAck().GetMetadata()["photo_frame_asset_base_url"]; got != "http://home.local:50052/photo-frame" {
		t.Fatalf("register ack metadata photo_frame_asset_base_url = %q, want configured value", got)
	}

	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected registered device")
	}
	if got.State != device.StateDisconnected {
		t.Fatalf("state = %q, want %q", got.State, device.StateDisconnected)
	}
}

func TestServerStartInvalidAddressLeavesStopped(t *testing.T) {
	s := NewServer("127.0.0.1:-1")
	if err := s.Start(context.Background()); err == nil {
		t.Fatalf("Start() error = nil, want bind error")
	}
	if s.Running() {
		t.Fatalf("expected server to remain stopped after failed Start")
	}
}

func TestServerGeneratedGRPCRoundTripRegisterAndHeartbeat(t *testing.T) {
	addr := mustAvailableTCPAddress(t)
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	s := NewServer(addr)
	s.ConfigureControl(control, GeneratedProtoAdapter{})

	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.Stop(stopCtx)
	})

	dialCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := grpc.NewClient(
		s.Address(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	client := controlv1.NewTerminalControlServiceClient(conn)
	stream, err := client.Connect(dialCtx)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}

	if err := stream.Send(&controlv1.ConnectRequest{
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
	}); err != nil {
		t.Fatalf("Send(register) error = %v", err)
	}

	first, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv(register_ack) error = %v", err)
	}
	if first.GetRegisterAck() == nil || first.GetRegisterAck().GetServerId() != "srv-1" {
		t.Fatalf("register ack = %+v, want server_id srv-1", first.GetRegisterAck())
	}

	if err := stream.Send(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Heartbeat{
			Heartbeat: &controlv1.Heartbeat{DeviceId: "device-1"},
		},
	}); err != nil {
		t.Fatalf("Send(heartbeat) error = %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected registered device")
	}
	if got.LastHeartbeat.IsZero() {
		t.Fatalf("expected heartbeat to be recorded")
	}
}

func mustAvailableTCPAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	defer func() { _ = listener.Close() }()
	return listener.Addr().String()
}
