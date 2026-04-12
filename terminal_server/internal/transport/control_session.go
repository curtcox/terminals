package transport

import (
	"context"
	"errors"
	"io"
)

var (
	// ErrNilStream indicates the caller passed a nil stream.
	ErrNilStream = errors.New("nil control stream")
)

// ControlStream is the transport-neutral interface mapped to gRPC streams.
type ControlStream interface {
	Recv() (ClientMessage, error)
	Send(ServerMessage) error
	Context() context.Context
}

// Session runs the control-plane bidirectional stream loop.
type Session struct {
	handler *StreamHandler
	control *ControlService
}

// NewSession builds a stream session.
func NewSession(handler *StreamHandler, control *ControlService) *Session {
	return &Session{
		handler: handler,
		control: control,
	}
}

// Run processes incoming client messages until stream termination.
func (s *Session) Run(stream ControlStream) error {
	if stream == nil {
		return ErrNilStream
	}

	var connectedDeviceID string
	for {
		in, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if connectedDeviceID != "" {
					_ = s.control.Disconnect(stream.Context(), connectedDeviceID)
				}
				return nil
			}
			return err
		}

		if in.Register != nil {
			connectedDeviceID = in.Register.DeviceID
		}
		if in.Heartbeat != nil && connectedDeviceID == "" {
			connectedDeviceID = in.Heartbeat.DeviceID
		}

		out, handleErr := s.handler.HandleMessage(stream.Context(), in)
		for _, msg := range out {
			if sendErr := stream.Send(msg); sendErr != nil {
				return sendErr
			}
		}
		if handleErr != nil {
			return handleErr
		}
	}
}
