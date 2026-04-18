package transport

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"google.golang.org/protobuf/proto"
)

func TestTCPServerRoundTripRegisterAndHeartbeat(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	grpcServer := NewServer(mustAvailableTCPAddress(t))
	grpcServer.ConfigureControl(control, GeneratedProtoAdapter{})

	tcpServer := NewTCPServer(mustAvailableTCPAddress(t), grpcServer)
	if err := tcpServer.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tcpServer.Stop(ctx)
	})

	conn, err := net.Dial("tcp", tcpServer.Address())
	if err != nil {
		t.Fatalf("net.Dial() error = %v", err)
	}
	defer func() { _ = conn.Close() }()

	mustSendTCPEnvelope(t, conn, &controlv1.WireEnvelope{
		ProtocolVersion: currentWireProtocolVersion,
		SessionId:       "tcp-session-1",
		Payload: &controlv1.WireEnvelope_TransportHello{
			TransportHello: &controlv1.TransportHello{ProtocolVersion: currentWireProtocolVersion},
		},
	})
	mustReceiveTCPEnvelope(t, conn)

	mustSendTCPEnvelope(t, conn, &controlv1.WireEnvelope{
		ProtocolVersion: currentWireProtocolVersion,
		SessionId:       "tcp-session-1",
		Sequence:        1,
		Payload: &controlv1.WireEnvelope_ClientMessage{
			ClientMessage: &controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen TCP"},
						},
					},
				},
			},
		},
	})
	registerAck := mustReceiveTCPEnvelope(t, conn)
	if registerAck.GetServerMessage().GetRegisterAck().GetServerId() != "srv-1" {
		t.Fatalf("register ack server_id = %q, want srv-1", registerAck.GetServerMessage().GetRegisterAck().GetServerId())
	}

	mustSendTCPEnvelope(t, conn, &controlv1.WireEnvelope{
		ProtocolVersion: currentWireProtocolVersion,
		SessionId:       "tcp-session-1",
		Sequence:        2,
		Payload: &controlv1.WireEnvelope_ClientMessage{
			ClientMessage: &controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Heartbeat{Heartbeat: &controlv1.Heartbeat{DeviceId: "device-1"}},
			},
		},
	})

	time.Sleep(50 * time.Millisecond)
	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected registered device")
	}
	if got.LastHeartbeat.IsZero() {
		t.Fatalf("expected heartbeat to be recorded")
	}
}

func TestHTTPControlServerRoundTripRegisterAndHeartbeat(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	grpcServer := NewServer(mustAvailableTCPAddress(t))
	grpcServer.ConfigureControl(control, GeneratedProtoAdapter{})

	httpServer := NewHTTPControlServer(mustAvailableTCPAddress(t), grpcServer)
	if err := httpServer.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = httpServer.Stop(ctx)
	})

	sessionID := "http-session-1"
	base := fmt.Sprintf("http://%s", httpServer.Address())

	mustPostHTTPEnvelope(t, base, sessionID, &controlv1.WireEnvelope{
		ProtocolVersion: currentWireProtocolVersion,
		SessionId:       sessionID,
		Payload: &controlv1.WireEnvelope_TransportHello{
			TransportHello: &controlv1.TransportHello{ProtocolVersion: currentWireProtocolVersion},
		},
	})
	mustGetHTTPEnvelope(t, base, sessionID)

	mustPostHTTPEnvelope(t, base, sessionID, &controlv1.WireEnvelope{
		ProtocolVersion: currentWireProtocolVersion,
		SessionId:       sessionID,
		Sequence:        1,
		Payload: &controlv1.WireEnvelope_ClientMessage{
			ClientMessage: &controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen HTTP"},
						},
					},
				},
			},
		},
	})
	registerAck := mustGetHTTPEnvelope(t, base, sessionID)
	if registerAck.GetServerMessage().GetRegisterAck().GetServerId() != "srv-1" {
		t.Fatalf("register ack server_id = %q, want srv-1", registerAck.GetServerMessage().GetRegisterAck().GetServerId())
	}

	mustPostHTTPEnvelope(t, base, sessionID, &controlv1.WireEnvelope{
		ProtocolVersion: currentWireProtocolVersion,
		SessionId:       sessionID,
		Sequence:        2,
		Payload: &controlv1.WireEnvelope_ClientMessage{
			ClientMessage: &controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Heartbeat{Heartbeat: &controlv1.Heartbeat{DeviceId: "device-1"}},
			},
		},
	})

	time.Sleep(50 * time.Millisecond)
	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected registered device")
	}
	if got.LastHeartbeat.IsZero() {
		t.Fatalf("expected heartbeat to be recorded")
	}
}

func mustSendTCPEnvelope(t *testing.T, conn net.Conn, envelope *controlv1.WireEnvelope) {
	t.Helper()
	payload, err := proto.Marshal(envelope)
	if err != nil {
		t.Fatalf("proto.Marshal() error = %v", err)
	}
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(payload)))
	if _, err := conn.Write(lenBuf); err != nil {
		t.Fatalf("conn.Write(length) error = %v", err)
	}
	if _, err := conn.Write(payload); err != nil {
		t.Fatalf("conn.Write(payload) error = %v", err)
	}
}

func mustReceiveTCPEnvelope(t *testing.T, conn net.Conn) *controlv1.WireEnvelope {
	t.Helper()
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		t.Fatalf("io.ReadFull(length) error = %v", err)
	}
	size := binary.BigEndian.Uint32(lenBuf)
	payload := make([]byte, size)
	if _, err := io.ReadFull(conn, payload); err != nil {
		t.Fatalf("io.ReadFull(payload) error = %v", err)
	}
	envelope := &controlv1.WireEnvelope{}
	if err := proto.Unmarshal(payload, envelope); err != nil {
		t.Fatalf("proto.Unmarshal() error = %v", err)
	}
	return envelope
}

func mustPostHTTPEnvelope(t *testing.T, base, sessionID string, envelope *controlv1.WireEnvelope) {
	t.Helper()
	payload, err := proto.Marshal(envelope)
	if err != nil {
		t.Fatalf("proto.Marshal() error = %v", err)
	}
	url := fmt.Sprintf("%s/v1/control/poll/%s", base, sessionID)
	resp, err := http.Post(url, "application/x-protobuf", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("http.Post() error = %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("poll status = %d, body=%s", resp.StatusCode, string(body))
	}
}

func mustGetHTTPEnvelope(t *testing.T, base, sessionID string) *controlv1.WireEnvelope {
	t.Helper()
	url := fmt.Sprintf("%s/v1/control/stream/%s?wait_ms=2000", base, sessionID)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("http.Get() error = %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("stream status = %d, body=%s", resp.StatusCode, string(body))
	}
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	envelope := &controlv1.WireEnvelope{}
	if err := proto.Unmarshal(payload, envelope); err != nil {
		t.Fatalf("proto.Unmarshal() error = %v", err)
	}
	return envelope
}
