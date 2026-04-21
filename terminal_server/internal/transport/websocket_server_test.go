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
	mustSendTransportHello(t, conn, "test-session-1")

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

func TestSameOriginHostAllowsDifferentPorts(t *testing.T) {
	if !sameOriginHost("http://192.168.0.138:60739", "192.168.0.138:50054") {
		t.Fatalf("sameOriginHost should allow same host with different ports")
	}
	if !sameOriginHost("http://localhost:60739", "localhost:50054") {
		t.Fatalf("sameOriginHost should allow loopback host with different ports")
	}
	if sameOriginHost("http://evil.example:60739", "192.168.0.138:50054") {
		t.Fatalf("sameOriginHost should reject different hosts")
	}
}

func TestIsLoopbackHostPort(t *testing.T) {
	cases := []struct {
		hostPort string
		want     bool
	}{
		{"localhost", true},
		{"localhost:60739", true},
		{"127.0.0.1", true},
		{"127.0.0.1:50054", true},
		{"127.1.2.3:8080", true},
		{"[::1]", true},
		{"[::1]:50054", true},
		{"example.com", false},
		{"example.com:443", false},
		{"10.0.0.1:8080", false},
		{"0.0.0.0:50054", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isLoopbackHostPort(c.hostPort); got != c.want {
			t.Errorf("isLoopbackHostPort(%q) = %v, want %v", c.hostPort, got, c.want)
		}
	}
}

func TestWebSocketServerRejectsNonLoopbackOriginOnLoopbackBind(t *testing.T) {
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

	endpoint := url.URL{Scheme: "ws", Host: wsServer.Address(), Path: wsServer.Path()}
	config, err := websocket.NewConfig(endpoint.String(), "http://evil.example")
	if err != nil {
		t.Fatalf("websocket.NewConfig() error = %v", err)
	}
	if _, err := websocket.DialConfig(config); err == nil {
		t.Fatalf("DialConfig() error = nil, want cross-origin rejection even on loopback bind")
	}
}

func TestWebSocketServerAllowsLoopbackOriginWithoutExplicitAllowList(t *testing.T) {
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	grpcServer := NewServer(mustAvailableTCPAddress(t))
	grpcServer.ConfigureControl(control, GeneratedProtoAdapter{})
	grpcServer.ConfigureBugReportIntake(bugReportIntakeStub{
		ack: &diagnosticsv1.BugReportAck{
			ReportId:      "bug-loopback-ack-1",
			CorrelationId: "bug:bug-loopback-ack-1",
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

	// Simulate the production browser flow: the Flutter web client at
	// http://localhost:60739 connects to the control WebSocket on a
	// different loopback port. Without an explicit allow list, both
	// endpoints are still on the same host and should be permitted.
	conn := mustDialWebSocket(t, wsServer.Address(), wsServer.Path(), "http://localhost:60739")
	defer func() { _ = conn.Close() }()
	mustSendTransportHello(t, conn, "test-session-2")

	register := &controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "device-1",
					Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Browser Client"},
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
				ReportId:         "bug-loopback-1",
				ReporterDeviceId: "device-1",
				SubjectDeviceId:  "device-1",
				Source:           diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_SCREEN_BUTTON,
				Description:      "cross-port loopback bug report",
				TimestampUnixMs:  time.Now().UTC().UnixMilli(),
			},
		},
	}

	const ackDeadline = 1 * time.Second
	mustSendProtoMessage(t, conn, bugReport)
	ack := mustReceiveBugReportAckWithin(t, conn, ackDeadline)
	if ack.GetReportId() != "bug-loopback-ack-1" {
		t.Fatalf("bug_report_ack report_id = %q, want bug-loopback-ack-1", ack.GetReportId())
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
	mustSendTransportHello(t, conn, "test-session-3")

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

	request, ok := message.(*controlv1.ConnectRequest)
	if !ok {
		t.Fatalf("message type = %T, want *controlv1.ConnectRequest", message)
	}
	envelope := &controlv1.WireEnvelope{
		ProtocolVersion: currentWireProtocolVersion,
		Payload: &controlv1.WireEnvelope_ClientMessage{
			ClientMessage: request,
		},
	}
	payload, err := proto.Marshal(envelope)
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
	envelope := &controlv1.WireEnvelope{}
	if err := proto.Unmarshal(payload, envelope); err != nil {
		t.Fatalf("proto.Unmarshal() error = %v", err)
	}
	if ack := envelope.GetTransportHelloAck(); ack != nil {
		if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Fatalf("conn.SetReadDeadline() error = %v", err)
		}
		defer func() { _ = conn.SetReadDeadline(time.Time{}) }()
		if err := websocket.Message.Receive(conn, &payload); err != nil {
			t.Fatalf("websocket.Message.Receive() error = %v", err)
		}
		envelope = &controlv1.WireEnvelope{}
		if err := proto.Unmarshal(payload, envelope); err != nil {
			t.Fatalf("proto.Unmarshal() error = %v", err)
		}
	}
	response := envelope.GetServerMessage()
	if response == nil {
		t.Fatalf("envelope payload = %T, want server_message", envelope.Payload)
	}
	return response
}

func mustSendTransportHello(t *testing.T, conn *websocket.Conn, sessionID string) {
	t.Helper()
	envelope := &controlv1.WireEnvelope{
		ProtocolVersion: currentWireProtocolVersion,
		SessionId:       sessionID,
		Payload: &controlv1.WireEnvelope_TransportHello{
			TransportHello: &controlv1.TransportHello{
				ProtocolVersion: currentWireProtocolVersion,
				SupportedCarriers: []controlv1.CarrierKind{
					controlv1.CarrierKind_CARRIER_KIND_WEBSOCKET,
				},
			},
		},
	}
	payload, err := proto.Marshal(envelope)
	if err != nil {
		t.Fatalf("proto.Marshal() error = %v", err)
	}
	if err := websocket.Message.Send(conn, payload); err != nil {
		t.Fatalf("websocket.Message.Send() error = %v", err)
	}
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
