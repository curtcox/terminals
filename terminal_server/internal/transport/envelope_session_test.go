package transport

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
)

type testEnvelopeStream struct {
	ctx     context.Context
	carrier controlv1.CarrierKind
	reads   []*controlv1.WireEnvelope
	readErr error
	writes  []*controlv1.WireEnvelope
}

func (s *testEnvelopeStream) ReadEnvelope() (*controlv1.WireEnvelope, error) {
	if len(s.reads) > 0 {
		env := s.reads[0]
		s.reads = s.reads[1:]
		return env, nil
	}
	if s.readErr != nil {
		return nil, s.readErr
	}
	return nil, io.EOF
}

func (s *testEnvelopeStream) WriteEnvelope(env *controlv1.WireEnvelope) error {
	s.writes = append(s.writes, env)
	return nil
}

func (s *testEnvelopeStream) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}

func (s *testEnvelopeStream) Carrier() controlv1.CarrierKind {
	return s.carrier
}

func TestEnvelopeHandleHelloRejectsUnsupportedProtocolVersion(t *testing.T) {
	stream := &testEnvelopeStream{carrier: controlv1.CarrierKind_CARRIER_KIND_TCP}
	eps := &envelopeProtoStream{stream: stream}

	err := eps.handleHello(&controlv1.TransportHello{ProtocolVersion: 77}, "session-x")
	if err == nil {
		t.Fatalf("handleHello() error = nil, want unsupported protocol version")
	}
	if !strings.Contains(err.Error(), "unsupported protocol version") {
		t.Fatalf("handleHello() error = %v, want unsupported protocol version", err)
	}
	if len(stream.writes) != 1 {
		t.Fatalf("len(writes) = %d, want 1", len(stream.writes))
	}
	terr := stream.writes[0].GetTransportError()
	if terr == nil {
		t.Fatalf("first response should be transport error")
	}
	if terr.GetCode() != "unsupported_protocol_version" {
		t.Fatalf("transport error code = %q, want unsupported_protocol_version", terr.GetCode())
	}
}

func TestEnvelopeHandleHelloRejectsUndeclaredCarrier(t *testing.T) {
	stream := &testEnvelopeStream{carrier: controlv1.CarrierKind_CARRIER_KIND_TCP}
	eps := &envelopeProtoStream{stream: stream}

	err := eps.handleHello(&controlv1.TransportHello{
		ProtocolVersion: currentWireProtocolVersion,
		SupportedCarriers: []controlv1.CarrierKind{
			controlv1.CarrierKind_CARRIER_KIND_WEBSOCKET,
		},
	}, "session-y")
	if err == nil {
		t.Fatalf("handleHello() error = nil, want unsupported carrier")
	}
	if !strings.Contains(err.Error(), "not declared") {
		t.Fatalf("handleHello() error = %v, want undeclared carrier", err)
	}
	if len(stream.writes) != 1 {
		t.Fatalf("len(writes) = %d, want 1", len(stream.writes))
	}
	terr := stream.writes[0].GetTransportError()
	if terr == nil {
		t.Fatalf("first response should be transport error")
	}
	if terr.GetCode() != "unsupported_carrier" {
		t.Fatalf("transport error code = %q, want unsupported_carrier", terr.GetCode())
	}
}

func TestEnvelopeHandleHelloIssuesAndReusesResumeToken(t *testing.T) {
	envelopeResumeRegistry.mu.Lock()
	envelopeResumeRegistry.tokens = map[string]time.Time{}
	envelopeResumeRegistry.mu.Unlock()

	stream1 := &testEnvelopeStream{carrier: controlv1.CarrierKind_CARRIER_KIND_TCP}
	eps1 := &envelopeProtoStream{stream: stream1}
	err := eps1.handleHello(&controlv1.TransportHello{
		ProtocolVersion: currentWireProtocolVersion,
		SupportedCarriers: []controlv1.CarrierKind{
			controlv1.CarrierKind_CARRIER_KIND_TCP,
		},
	}, "session-a")
	if err != nil {
		t.Fatalf("handleHello() error = %v", err)
	}
	if len(stream1.writes) != 1 {
		t.Fatalf("len(writes) = %d, want 1", len(stream1.writes))
	}
	ack1 := stream1.writes[0].GetTransportHelloAck()
	if ack1 == nil {
		t.Fatalf("expected transport hello ack")
	}
	if ack1.GetResumeToken() == "" {
		t.Fatalf("expected resume token to be issued")
	}

	stream2 := &testEnvelopeStream{carrier: controlv1.CarrierKind_CARRIER_KIND_WEBSOCKET}
	eps2 := &envelopeProtoStream{stream: stream2}
	err = eps2.handleHello(&controlv1.TransportHello{
		ProtocolVersion: currentWireProtocolVersion,
		SupportedCarriers: []controlv1.CarrierKind{
			controlv1.CarrierKind_CARRIER_KIND_WEBSOCKET,
		},
		ResumeToken: ack1.GetResumeToken(),
	}, "session-b")
	if err != nil {
		t.Fatalf("handleHello() with resume token error = %v", err)
	}
	if len(stream2.writes) != 1 {
		t.Fatalf("len(second writes) = %d, want 1", len(stream2.writes))
	}
	ack2 := stream2.writes[0].GetTransportHelloAck()
	if ack2 == nil {
		t.Fatalf("expected second transport hello ack")
	}
	if ack2.GetResumeToken() != ack1.GetResumeToken() {
		t.Fatalf("resume token = %q, want %q", ack2.GetResumeToken(), ack1.GetResumeToken())
	}
}

func TestEnvelopeHandleHelloReplacesUnknownResumeToken(t *testing.T) {
	envelopeResumeRegistry.mu.Lock()
	envelopeResumeRegistry.tokens = map[string]time.Time{}
	envelopeResumeRegistry.mu.Unlock()

	stream := &testEnvelopeStream{carrier: controlv1.CarrierKind_CARRIER_KIND_HTTP}
	eps := &envelopeProtoStream{stream: stream}
	err := eps.handleHello(&controlv1.TransportHello{
		ProtocolVersion: currentWireProtocolVersion,
		SupportedCarriers: []controlv1.CarrierKind{
			controlv1.CarrierKind_CARRIER_KIND_HTTP,
		},
		ResumeToken: "resume-unknown-token",
	}, "session-c")
	if err != nil {
		t.Fatalf("handleHello() error = %v", err)
	}
	if len(stream.writes) != 1 {
		t.Fatalf("len(writes) = %d, want 1", len(stream.writes))
	}
	ack := stream.writes[0].GetTransportHelloAck()
	if ack == nil {
		t.Fatalf("expected transport hello ack")
	}
	if ack.GetResumeToken() == "" {
		t.Fatalf("expected non-empty resume token")
	}
	if ack.GetResumeToken() == "resume-unknown-token" {
		t.Fatalf("unknown resume token should not be reused")
	}
}

func TestEnvelopeHandleHelloAllowsEmptySupportedCarriers(t *testing.T) {
	stream := &testEnvelopeStream{carrier: controlv1.CarrierKind_CARRIER_KIND_WEBSOCKET}
	eps := &envelopeProtoStream{stream: stream}
	err := eps.handleHello(&controlv1.TransportHello{
		ProtocolVersion: currentWireProtocolVersion,
	}, "session-d")
	if err != nil {
		t.Fatalf("handleHello() error = %v", err)
	}
	if len(stream.writes) != 1 {
		t.Fatalf("len(writes) = %d, want 1", len(stream.writes))
	}
	ack := stream.writes[0].GetTransportHelloAck()
	if ack == nil {
		t.Fatalf("expected transport hello ack")
	}
	if ack.GetNegotiatedCarrier() != controlv1.CarrierKind_CARRIER_KIND_WEBSOCKET {
		t.Fatalf("negotiated carrier = %s, want websocket", ack.GetNegotiatedCarrier().String())
	}
}

func TestEnvelopeRecvProtoRequiresHelloBeforeClientMessage(t *testing.T) {
	stream := &testEnvelopeStream{
		carrier: controlv1.CarrierKind_CARRIER_KIND_TCP,
		reads: []*controlv1.WireEnvelope{{
			Sequence: 1,
			Payload: &controlv1.WireEnvelope_ClientMessage{
				ClientMessage: &controlv1.ConnectRequest{},
			},
		}},
	}
	eps := &envelopeProtoStream{stream: stream}

	_, err := eps.RecvProto()
	if err == nil {
		t.Fatalf("RecvProto() error = nil, want hello-required error")
	}
	if !strings.Contains(err.Error(), "transport hello required") {
		t.Fatalf("RecvProto() error = %v, want hello-required error", err)
	}
}

func TestEnvelopeRecvProtoSkipsHeartbeatAndReturnsClientMessage(t *testing.T) {
	stream := &testEnvelopeStream{
		carrier: controlv1.CarrierKind_CARRIER_KIND_TCP,
		reads: []*controlv1.WireEnvelope{
			{
				SessionId: "session-z",
				Payload: &controlv1.WireEnvelope_TransportHello{
					TransportHello: &controlv1.TransportHello{
						ProtocolVersion: currentWireProtocolVersion,
						SupportedCarriers: []controlv1.CarrierKind{
							controlv1.CarrierKind_CARRIER_KIND_TCP,
						},
					},
				},
			},
			{
				Sequence: 1,
				Payload: &controlv1.WireEnvelope_TransportHeartbeat{
					TransportHeartbeat: &controlv1.TransportHeartbeat{UnixMs: 1},
				},
			},
			{
				Sequence: 2,
				Payload: &controlv1.WireEnvelope_ClientMessage{
					ClientMessage: &controlv1.ConnectRequest{},
				},
			},
		},
	}
	eps := &envelopeProtoStream{stream: stream}

	msg, err := eps.RecvProto()
	if err != nil {
		t.Fatalf("RecvProto() error = %v", err)
	}
	if _, ok := msg.(*controlv1.ConnectRequest); !ok {
		t.Fatalf("RecvProto() msg type = %T, want *controlv1.ConnectRequest", msg)
	}
	if len(stream.writes) != 1 || stream.writes[0].GetTransportHelloAck() == nil {
		t.Fatalf("expected transport hello ack write before returning message")
	}
}

func TestEnvelopeRecvProtoReturnsTransportError(t *testing.T) {
	stream := &testEnvelopeStream{
		carrier: controlv1.CarrierKind_CARRIER_KIND_HTTP,
		reads: []*controlv1.WireEnvelope{
			{
				SessionId: "session-te",
				Payload: &controlv1.WireEnvelope_TransportHello{
					TransportHello: &controlv1.TransportHello{
						ProtocolVersion: currentWireProtocolVersion,
						SupportedCarriers: []controlv1.CarrierKind{
							controlv1.CarrierKind_CARRIER_KIND_HTTP,
						},
					},
				},
			},
			{
				Sequence: 1,
				Payload: &controlv1.WireEnvelope_TransportError{
					TransportError: &controlv1.TransportError{Code: "x", Message: "bad"},
				},
			},
		},
	}
	eps := &envelopeProtoStream{stream: stream}

	_, err := eps.RecvProto()
	if err == nil {
		t.Fatalf("RecvProto() error = nil, want transport error")
	}
	if !strings.Contains(err.Error(), "transport error x: bad") {
		t.Fatalf("RecvProto() error = %v, want transport error", err)
	}
}

func TestEnvelopeSendProtoRequiresHelloAck(t *testing.T) {
	stream := &testEnvelopeStream{carrier: controlv1.CarrierKind_CARRIER_KIND_TCP}
	eps := &envelopeProtoStream{stream: stream}

	err := eps.SendProto(&controlv1.ConnectResponse{})
	if err == nil {
		t.Fatalf("SendProto() error = nil, want hello-required error")
	}
	if !strings.Contains(err.Error(), "transport hello required") {
		t.Fatalf("SendProto() error = %v, want hello-required error", err)
	}
}

func TestEnvelopeSendProtoWrapsServerMessageAfterHello(t *testing.T) {
	stream := &testEnvelopeStream{carrier: controlv1.CarrierKind_CARRIER_KIND_WEBSOCKET}
	eps := &envelopeProtoStream{stream: stream}

	err := eps.handleHello(&controlv1.TransportHello{
		ProtocolVersion: currentWireProtocolVersion,
		SupportedCarriers: []controlv1.CarrierKind{
			controlv1.CarrierKind_CARRIER_KIND_WEBSOCKET,
		},
	}, "session-send")
	if err != nil {
		t.Fatalf("handleHello() error = %v", err)
	}

	resp := &controlv1.ConnectResponse{
		Payload: &controlv1.ConnectResponse_RegisterAck{RegisterAck: &controlv1.RegisterAck{ServerId: "srv"}},
	}
	if err := eps.SendProto(resp); err != nil {
		t.Fatalf("SendProto() error = %v", err)
	}
	if len(stream.writes) != 2 {
		t.Fatalf("len(writes) = %d, want 2 (hello ack + server message)", len(stream.writes))
	}
	wrapped := stream.writes[1]
	if wrapped.GetServerMessage() == nil {
		t.Fatalf("second write should be server message envelope")
	}
	if wrapped.GetSessionId() != "session-send" {
		t.Fatalf("session_id = %q, want session-send", wrapped.GetSessionId())
	}
	if wrapped.GetSequence() != 2 {
		t.Fatalf("sequence = %d, want 2", wrapped.GetSequence())
	}
}
