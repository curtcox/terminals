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
	writes  []*controlv1.WireEnvelope
}

func (s *testEnvelopeStream) ReadEnvelope() (*controlv1.WireEnvelope, error) {
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
