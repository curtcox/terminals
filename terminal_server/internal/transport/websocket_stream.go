package transport

import (
	"context"
	"fmt"

	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"golang.org/x/net/websocket"
	"google.golang.org/protobuf/proto"
)

// WebSocketEnvelopeStream adapts a websocket connection to WireEnvelope framing.
type WebSocketEnvelopeStream struct {
	conn *websocket.Conn
	ctx  context.Context
}

// NewWebSocketEnvelopeStream creates a websocket envelope stream adapter.
func NewWebSocketEnvelopeStream(ctx context.Context, conn *websocket.Conn) WebSocketEnvelopeStream {
	return WebSocketEnvelopeStream{conn: conn, ctx: ctx}
}

// ReadEnvelope reads one websocket binary frame and decodes a WireEnvelope.
func (s WebSocketEnvelopeStream) ReadEnvelope() (*controlv1.WireEnvelope, error) {
	var payload []byte
	if err := websocket.Message.Receive(s.conn, &payload); err != nil {
		return nil, fmt.Errorf("receive websocket envelope: %w", err)
	}
	envelope := &controlv1.WireEnvelope{}
	if err := proto.Unmarshal(payload, envelope); err != nil {
		return nil, fmt.Errorf("decode websocket envelope: %w", err)
	}
	return envelope, nil
}

// WriteEnvelope encodes one WireEnvelope and writes it as a websocket binary frame.
func (s WebSocketEnvelopeStream) WriteEnvelope(envelope *controlv1.WireEnvelope) error {
	payload, err := proto.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("encode websocket envelope: %w", err)
	}
	if err := websocket.Message.Send(s.conn, payload); err != nil {
		return fmt.Errorf("send websocket envelope: %w", err)
	}
	return nil
}

// Context returns the parent request context.
func (s WebSocketEnvelopeStream) Context() context.Context {
	return s.ctx
}

// Carrier returns websocket carrier metadata for hello negotiation.
func (s WebSocketEnvelopeStream) Carrier() controlv1.CarrierKind {
	return controlv1.CarrierKind_CARRIER_KIND_WEBSOCKET
}
