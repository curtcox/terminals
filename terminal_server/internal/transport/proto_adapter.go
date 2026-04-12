package transport

import (
	"context"
	"errors"
	"io"
)

var (
	// ErrNilProtoStream indicates a nil proto stream was provided.
	ErrNilProtoStream = errors.New("nil proto stream")
	// ErrNilProtoAdapter indicates a nil proto adapter was provided.
	ErrNilProtoAdapter = errors.New("nil proto adapter")
)

// ProtoClientEnvelope is the proto-facing message container.
// A concrete implementation can wrap generated protobuf messages later.
type ProtoClientEnvelope interface{}

// ProtoServerEnvelope is the proto-facing message container.
// A concrete implementation can wrap generated protobuf messages later.
type ProtoServerEnvelope interface{}

// ProtoStream represents the bidirectional gRPC Connect stream shape.
type ProtoStream interface {
	RecvProto() (ProtoClientEnvelope, error)
	SendProto(ProtoServerEnvelope) error
	Context() context.Context
}

// ProtoAdapter maps between proto envelopes and internal transport messages.
type ProtoAdapter interface {
	ToInternal(ProtoClientEnvelope) (ClientMessage, error)
	FromInternal(ServerMessage) (ProtoServerEnvelope, error)
}

// RunProtoSession bridges a proto-shaped stream into the internal session handler.
func RunProtoSession(handler *StreamHandler, control *ControlService, stream ProtoStream, adapter ProtoAdapter) error {
	if stream == nil {
		return ErrNilProtoStream
	}
	if adapter == nil {
		return ErrNilProtoAdapter
	}

	session := NewSession(handler, control)
	return session.Run(&protoBackedStream{
		stream:   stream,
		adapter:  adapter,
		lastRecv: nil,
	})
}

type protoBackedStream struct {
	stream   ProtoStream
	adapter  ProtoAdapter
	lastRecv *ClientMessage
}

func (p *protoBackedStream) Recv() (ClientMessage, error) {
	envelope, err := p.stream.RecvProto()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return ClientMessage{}, io.EOF
		}
		return ClientMessage{}, err
	}
	internal, err := p.adapter.ToInternal(envelope)
	if err != nil {
		return ClientMessage{}, err
	}
	p.lastRecv = &internal
	return internal, nil
}

func (p *protoBackedStream) Send(msg ServerMessage) error {
	envelope, err := p.adapter.FromInternal(msg)
	if err != nil {
		return err
	}
	return p.stream.SendProto(envelope)
}

func (p *protoBackedStream) Context() context.Context {
	return p.stream.Context()
}
