package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
)

const currentWireProtocolVersion = 1

// EnvelopeStream is the non-gRPC control stream contract carrying WireEnvelope frames.
type EnvelopeStream interface {
	ReadEnvelope() (*controlv1.WireEnvelope, error)
	WriteEnvelope(*controlv1.WireEnvelope) error
	Context() context.Context
	Carrier() controlv1.CarrierKind
}

// RunEnvelopeSession bridges a wire-envelope stream into the existing proto session runner.
func RunEnvelopeSession(handler *StreamHandler, control *ControlService, stream EnvelopeStream, adapter ProtoAdapter) error {
	if stream == nil {
		return ErrNilProtoStream
	}
	if adapter == nil {
		return ErrNilProtoAdapter
	}
	return RunProtoSession(handler, control, &envelopeProtoStream{stream: stream}, adapter)
}

type envelopeProtoStream struct {
	stream          EnvelopeStream
	mu              sync.Mutex
	helloAcked      bool
	sessionID       string
	protocolVersion uint32
	inSeq           uint64
	outSeq          uint64
}

func (e *envelopeProtoStream) RecvProto() (ProtoClientEnvelope, error) {
	for {
		env, err := e.stream.ReadEnvelope()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, io.EOF
			}
			return nil, err
		}
		if env == nil {
			continue
		}

		if err := e.processSequence(env.GetSequence()); err != nil {
			return nil, err
		}

		if hello := env.GetTransportHello(); hello != nil {
			if err := e.handleHello(hello, env.GetSessionId()); err != nil {
				return nil, err
			}
			continue
		}

		e.mu.Lock()
		helloAcked := e.helloAcked
		e.mu.Unlock()
		if !helloAcked {
			return nil, fmt.Errorf("transport hello required before client messages")
		}

		if msg := env.GetClientMessage(); msg != nil {
			return msg, nil
		}

		if env.GetTransportHeartbeat() != nil {
			continue
		}

		if terr := env.GetTransportError(); terr != nil {
			return nil, fmt.Errorf("transport error %s: %s", terr.GetCode(), terr.GetMessage())
		}
	}
}

func (e *envelopeProtoStream) SendProto(msg ProtoServerEnvelope) error {
	response, ok := msg.(*controlv1.ConnectResponse)
	if !ok {
		return fmt.Errorf("unexpected proto server envelope %T", msg)
	}

	e.mu.Lock()
	if !e.helloAcked {
		e.mu.Unlock()
		return fmt.Errorf("transport hello required before server messages")
	}
	e.outSeq++
	outSeq := e.outSeq
	sessionID := e.sessionID
	protocolVersion := e.protocolVersion
	e.mu.Unlock()

	return e.stream.WriteEnvelope(&controlv1.WireEnvelope{
		ProtocolVersion: protocolVersion,
		SessionId:       sessionID,
		Sequence:        outSeq,
		Payload: &controlv1.WireEnvelope_ServerMessage{
			ServerMessage: response,
		},
	})
}

func (e *envelopeProtoStream) Context() context.Context {
	return e.stream.Context()
}

func (e *envelopeProtoStream) processSequence(sequence uint64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if sequence == 0 {
		return nil
	}
	if sequence <= e.inSeq {
		return fmt.Errorf("non-monotonic client sequence: got=%d last=%d", sequence, e.inSeq)
	}
	e.inSeq = sequence
	return nil
}

func (e *envelopeProtoStream) handleHello(hello *controlv1.TransportHello, requestedSessionID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.helloAcked {
		return fmt.Errorf("transport hello already processed")
	}
	version := hello.GetProtocolVersion()
	if version == 0 {
		version = currentWireProtocolVersion
	}
	e.protocolVersion = version
	if requestedSessionID != "" {
		e.sessionID = requestedSessionID
	} else {
		e.sessionID = fmt.Sprintf("%s-%d", e.stream.Carrier().String(), time.Now().UTC().UnixNano())
	}
	e.outSeq++
	ackSeq := e.outSeq
	ack := &controlv1.WireEnvelope{
		ProtocolVersion: version,
		SessionId:       e.sessionID,
		Sequence:        ackSeq,
		Payload: &controlv1.WireEnvelope_TransportHelloAck{
			TransportHelloAck: &controlv1.TransportHelloAck{
				AcceptedProtocolVersion: version,
				NegotiatedCarrier:      e.stream.Carrier(),
				SessionId:              e.sessionID,
				HeartbeatIntervalMs:    30000,
			},
		},
	}
	if err := e.stream.WriteEnvelope(ack); err != nil {
		return err
	}
	e.helloAcked = true
	return nil
}
