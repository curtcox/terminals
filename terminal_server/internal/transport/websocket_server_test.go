package transport

import (
	"context"
	"net/url"
	"testing"
	"time"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"golang.org/x/net/websocket"
	"google.golang.org/protobuf/proto"
)

func TestWebSocketServerRoundTripRegisterAndHeartbeat(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	grpcServer := NewServer(mustAvailableTCPAddress(t))
	grpcServer.ConfigureControl(control, GeneratedProtoAdapter{})

	wsServer := NewWebSocketServer(mustAvailableTCPAddress(t), grpcServer, []string{})
	if err := wsServer.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = wsServer.Stop(ctx)
	})

	conn := mustDialWebSocket(t, wsServer.Address(), wsServer.Path(), "http://"+wsServer.Address())
	defer func() { _ = conn.Close() }()

	register := &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "device-1",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen Chromebook"},
				},
			},
		},
	}
	mustSendProtoMessage(t, conn, register)

	response := mustReceiveConnectResponse(t, conn)
	if response.GetRegisterAck() == nil || response.GetRegisterAck().GetServerId() != "srv-1" {
		t.Fatalf("register ack = %+v, want server_id srv-1", response.GetRegisterAck())
	}

	heartbeat := &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Heartbeat{Heartbeat: &controlv1.Heartbeat{DeviceId: "device-1"}},
	}
	mustSendProtoMessage(t, conn, heartbeat)
	time.Sleep(50 * time.Millisecond)

	got, ok := manager.Get("device-1")
	if !ok {
		t.Fatalf("expected registered device")
	}
	if got.LastHeartbeat.IsZero() {
		t.Fatalf("expected heartbeat to be recorded")
	}
}

func TestWebSocketServerRejectsDisallowedOrigin(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	grpcServer := NewServer(mustAvailableTCPAddress(t))
	grpcServer.ConfigureControl(control, GeneratedProtoAdapter{})
	wsServer := NewWebSocketServer(mustAvailableTCPAddress(t), grpcServer, []string{"http://localhost:60739"})

	if err := wsServer.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = wsServer.Stop(ctx)
	})

	endpoint := url.URL{Scheme: "ws", Host: wsServer.Address(), Path: wsServer.Path()}
	config, err := websocket.NewConfig(endpoint.String(), "http://evil.example")
	if err != nil {
		t.Fatalf("websocket.NewConfig() error = %v", err)
	}
	if _, err := websocket.DialConfig(config); err == nil {
		t.Fatalf("DialConfig() error = nil, want origin rejection")
	}
}

func TestWebSocketServerRoundTripBugReportAckWithinDeadline(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	grpcServer := NewServer(mustAvailableTCPAddress(t))
	grpcServer.ConfigureControl(control, GeneratedProtoAdapter{})
	grpcServer.ConfigureBugReportIntake(bugReportIntakeStub{
		ack: &diagnosticsv1.BugReportAck{
			ReportId:      "bug-ws-ack-1",
			CorrelationId: "bug:bug-ws-ack-1",
			Status:        diagnosticsv1.BugReportStatus_BUG_REPORT_STATUS_FILED,
		},
	})

	wsServer := NewWebSocketServer(mustAvailableTCPAddress(t), grpcServer, []string{})
	if err := wsServer.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = wsServer.Stop(ctx)
	})

	conn := mustDialWebSocket(t, wsServer.Address(), wsServer.Path(), "http://"+wsServer.Address())
	defer func() { _ = conn.Close() }()

	register := &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "device-1",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen Chromebook"},
				},
			},
		},
	}
	mustSendProtoMessage(t, conn, register)

	registerAck := mustReceiveConnectResponse(t, conn)
	if registerAck.GetRegisterAck() == nil {
		t.Fatalf("first register response should include register_ack")
	}

	bugReport := &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_BugReport{
			BugReport: &diagnosticsv1.BugReport{
				ReportId:         "bug-client-1",
				ReporterDeviceId: "device-1",
				SubjectDeviceId:  "device-1",
				Source:           diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_SCREEN_BUTTON,
				Description:      "websocket bug report roundtrip test",
				TimestampUnixMs:  time.Now().UTC().UnixMilli(),
			},
		},
	}

	const ackDeadline = 1 * time.Second
	start := time.Now()
	mustSendProtoMessage(t, conn, bugReport)
	ack := mustReceiveBugReportAckWithin(t, conn, ackDeadline)
	elapsed := time.Since(start)

	if ack.GetReportId() != "bug-ws-ack-1" {
		t.Fatalf("bug_report_ack report_id = %q, want bug-ws-ack-1", ack.GetReportId())
	}
	if elapsed > ackDeadline {
		t.Fatalf("bug_report_ack elapsed = %v, want <= %v", elapsed, ackDeadline)
	}
}

func mustDialWebSocket(t *testing.T, host, path, origin string) *websocket.Conn {
	t.Helper()
	endpoint := url.URL{Scheme: "ws", Host: host, Path: path}
	config, err := websocket.NewConfig(endpoint.String(), origin)
	if err != nil {
		t.Fatalf("websocket.NewConfig() error = %v", err)
	}
	conn, err := websocket.DialConfig(config)
	if err != nil {
		t.Fatalf("websocket.DialConfig() error = %v", err)
	}
	return conn
}

func mustSendProtoMessage(t *testing.T, conn *websocket.Conn, message proto.Message) {
	t.Helper()
	payload, err := proto.Marshal(message)
	if err != nil {
		t.Fatalf("proto.Marshal() error = %v", err)
	}
	if err := websocket.Message.Send(conn, payload); err != nil {
		t.Fatalf("websocket.Message.Send() error = %v", err)
	}
}

func mustReceiveConnectResponse(t *testing.T, conn *websocket.Conn) *controlv1.ConnectResponse {
	t.Helper()
	var payload []byte
	if err := websocket.Message.Receive(conn, &payload); err != nil {
		t.Fatalf("websocket.Message.Receive() error = %v", err)
	}
	response := &controlv1.ConnectResponse{}
	if err := proto.Unmarshal(payload, response); err != nil {
		t.Fatalf("proto.Unmarshal() error = %v", err)
	}
	return response
}

func mustReceiveBugReportAckWithin(t *testing.T, conn *websocket.Conn, timeout time.Duration) *diagnosticsv1.BugReportAck {
	t.Helper()
	deadline := time.Now().Add(timeout)
	if err := conn.SetDeadline(deadline); err != nil {
		t.Fatalf("conn.SetDeadline() error = %v", err)
	}
	defer func() { _ = conn.SetDeadline(time.Time{}) }()

	for {
		response := mustReceiveConnectResponse(t, conn)
		if ack := response.GetBugReportAck(); ack != nil {
			return ack
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for bug_report_ack payload")
		}
	}
}
