package transport

import (
	"context"
	"fmt"

	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"golang.org/x/net/websocket"
	"google.golang.org/protobuf/proto"
)

// WebSocketProtoStream adapts a websocket connection to the shared ProtoStream interface.
type WebSocketProtoStream struct {
	conn *websocket.Conn
	ctx  context.Context
}

// NewWebSocketProtoStream creates a protobuf websocket stream adapter.
func NewWebSocketProtoStream(conn *websocket.Conn, ctx context.Context) WebSocketProtoStream {
	return WebSocketProtoStream{conn: conn, ctx: ctx}
}

// RecvProto reads one websocket binary frame and decodes a ConnectRequest.
func (s WebSocketProtoStream) RecvProto() (ProtoClientEnvelope, error) {
	var payload []byte
	if err := websocket.Message.Receive(s.conn, &payload); err != nil {
		return nil, err
	}
	request := &controlv1.ConnectRequest{}
	if err := proto.Unmarshal(payload, request); err != nil {
		return nil, fmt.Errorf("decode websocket connect request: %w", err)
	}
	return request, nil
}

// SendProto encodes one ConnectResponse and writes it as a websocket binary frame.
func (s WebSocketProtoStream) SendProto(envelope ProtoServerEnvelope) error {
	response, ok := envelope.(*controlv1.ConnectResponse)
	if !ok {
		return fmt.Errorf("unexpected proto server envelope %T", envelope)
	}
	payload, err := proto.Marshal(response)
	if err != nil {
		return fmt.Errorf("encode websocket connect response: %w", err)
	}
	if err := websocket.Message.Send(s.conn, payload); err != nil {
		return fmt.Errorf("send websocket connect response: %w", err)
	}
	return nil
}

// Context returns the parent request context.
func (s WebSocketProtoStream) Context() context.Context {
	return s.ctx
}
