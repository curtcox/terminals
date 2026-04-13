package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
)

// ErrNilStream indicates the caller passed a nil stream.
var ErrNilStream = errors.New("nil control stream")

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
	var sendMu sync.Mutex
	send := func(msg ServerMessage) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		return stream.Send(msg)
	}
	registeredRelayDeviceID := ""
	defer func() {
		if registeredRelayDeviceID != "" {
			globalSessionRelayRegistry.Unregister(registeredRelayDeviceID)
		}
	}()
	for {
		in, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if connectedDeviceID != "" {
					s.handler.HandleDisconnect(connectedDeviceID)
					_ = s.control.Disconnect(stream.Context(), connectedDeviceID)
				}
				return nil
			}
			return err
		}

		if sessionErr := validateSessionMessage(connectedDeviceID, in); sessionErr != nil {
			s.handler.NoteProtocolError()
			if sendErr := send(ServerMessage{
				ErrorCode: ErrorCodeProtocolViolation,
				Error:     sessionErr.Error(),
			}); sendErr != nil {
				return sendErr
			}
			continue
		}

		if in.Register != nil {
			connectedDeviceID = in.Register.DeviceID
			if connectedDeviceID != "" && connectedDeviceID != registeredRelayDeviceID {
				if registeredRelayDeviceID != "" {
					globalSessionRelayRegistry.Unregister(registeredRelayDeviceID)
				}
				globalSessionRelayRegistry.Register(connectedDeviceID, send)
				registeredRelayDeviceID = connectedDeviceID
			}
		}
		if in.Heartbeat != nil && connectedDeviceID == "" {
			connectedDeviceID = in.Heartbeat.DeviceID
		}
		in.SessionDeviceID = connectedDeviceID

		out, handleErr := s.handler.HandleMessage(stream.Context(), in)
		for _, msg := range out {
			if msg.RelayToDeviceID != "" {
				relayMsg := msg
				relayMsg.RelayToDeviceID = ""
				if sendErr := globalSessionRelayRegistry.Relay(msg.RelayToDeviceID, relayMsg); sendErr != nil {
					return sendErr
				}
				continue
			}
			if sendErr := send(msg); sendErr != nil {
				return sendErr
			}
		}
		if handleErr != nil {
			// If we emitted an explicit error response, keep the session alive
			// so a malformed client message does not force reconnect.
			if hasStructuredError(out) {
				continue
			}
			return handleErr
		}
	}
}

func validateSessionMessage(connectedDeviceID string, in ClientMessage) error {
	if in.Register != nil {
		if in.Register.DeviceID == "" {
			return fmt.Errorf("register requires device id")
		}
		if connectedDeviceID != "" && in.Register.DeviceID != connectedDeviceID {
			return fmt.Errorf("register device id mismatch: connected=%s requested=%s", connectedDeviceID, in.Register.DeviceID)
		}
		return nil
	}

	if connectedDeviceID == "" {
		return fmt.Errorf("register required before other messages")
	}

	msgDeviceID, hasDeviceID := extractMessageDeviceID(in)
	if hasDeviceID && msgDeviceID != "" && msgDeviceID != connectedDeviceID {
		return fmt.Errorf("message device id mismatch: connected=%s message=%s", connectedDeviceID, msgDeviceID)
	}
	return nil
}

func extractMessageDeviceID(in ClientMessage) (string, bool) {
	switch {
	case in.Capability != nil:
		return in.Capability.DeviceID, true
	case in.Heartbeat != nil:
		return in.Heartbeat.DeviceID, true
	case in.Command != nil:
		return in.Command.DeviceID, true
	case in.Input != nil:
		return in.Input.DeviceID, true
	default:
		return "", false
	}
}

func hasStructuredError(messages []ServerMessage) bool {
	for _, msg := range messages {
		if msg.Error != "" {
			return true
		}
	}
	return false
}
