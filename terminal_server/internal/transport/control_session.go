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

	state := controlSessionState{}
	capabilityReady := false
	var sendMu sync.Mutex
	send := func(msg ServerMessage) error {
		sendMu.Lock()
		defer sendMu.Unlock()
		return stream.Send(msg)
	}
	defer func() {
		if state.registeredRelayDeviceID != "" {
			globalSessionRelayRegistry.Unregister(state.registeredRelayDeviceID)
		}
	}()
	for {
		in, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if state.connectedDeviceID != "" {
					s.handler.HandleDisconnect(state.connectedDeviceID)
					_ = s.control.Disconnect(stream.Context(), state.connectedDeviceID)
				}
				return nil
			}
			return err
		}

		if sessionErr := validateSessionMessage(state.connectedDeviceID, capabilityReady, in); sessionErr != nil {
			s.handler.NoteProtocolError()
			if sendErr := send(ServerMessage{
				ErrorCode: ErrorCodeProtocolViolation,
				Error:     sessionErr.Error(),
			}); sendErr != nil {
				return sendErr
			}
			continue
		}

		capabilityReady = state.observeClientMessage(in, capabilityReady, send)
		in.SessionDeviceID = state.connectedDeviceID

		out, handleErr := s.handler.HandleMessage(stream.Context(), in)
		if err := s.sendSessionMessages(out, state.connectedDeviceID, send); err != nil {
			return err
		}
		if shouldContinueSession(out, handleErr) {
			continue
		}
		if handleErr != nil {
			return handleErr
		}
	}
}

type controlSessionState struct {
	connectedDeviceID       string
	registeredRelayDeviceID string
}

func (s *controlSessionState) observeClientMessage(in ClientMessage, capabilityReady bool, send func(ServerMessage) error) bool {
	if in.Register != nil {
		s.connectedDeviceID = in.Register.DeviceID
		capabilityReady = true
	}
	if in.Hello != nil && s.connectedDeviceID == "" {
		s.connectedDeviceID = in.Hello.DeviceID
	}
	if in.CapabilitySnap != nil {
		if s.connectedDeviceID == "" {
			s.connectedDeviceID = in.CapabilitySnap.DeviceID
		}
		capabilityReady = true
	}
	if in.Heartbeat != nil && s.connectedDeviceID == "" {
		s.connectedDeviceID = in.Heartbeat.DeviceID
	}
	s.registerRelay(send)
	return capabilityReady
}

func (s *controlSessionState) registerRelay(send func(ServerMessage) error) {
	if s.connectedDeviceID == "" || s.connectedDeviceID == s.registeredRelayDeviceID {
		return
	}
	if s.registeredRelayDeviceID != "" {
		globalSessionRelayRegistry.Unregister(s.registeredRelayDeviceID)
	}
	globalSessionRelayRegistry.Register(s.connectedDeviceID, send)
	s.registeredRelayDeviceID = s.connectedDeviceID
}

func (s *Session) sendSessionMessages(out []ServerMessage, connectedDeviceID string, send func(ServerMessage) error) error {
	for _, msg := range out {
		targetDeviceID := connectedDeviceID
		if relayTarget := msg.RelayToDeviceID; relayTarget != "" {
			targetDeviceID = relayTarget
		}
		msg = s.handler.decorateBugReportAffordance(targetDeviceID, msg)
		prepared, err := s.handler.prepareOutboundUI(targetDeviceID, msg)
		if err != nil {
			return err
		}
		if prepared.RelayToDeviceID != "" {
			relayMsg := prepared
			relayMsg.RelayToDeviceID = ""
			if err := globalSessionRelayRegistry.Relay(prepared.RelayToDeviceID, relayMsg); err != nil {
				return err
			}
			continue
		}
		if err := send(prepared); err != nil {
			return err
		}
	}
	return nil
}

func shouldContinueSession(out []ServerMessage, handleErr error) bool {
	return handleErr != nil && hasStructuredError(out)
}

func validateSessionMessage(connectedDeviceID string, capabilityReady bool, in ClientMessage) error {
	if in.Register != nil {
		return validateSessionDeviceID("register", "requested", connectedDeviceID, in.Register.DeviceID)
	}

	if in.Hello != nil {
		return validateSessionDeviceID("hello", "hello", connectedDeviceID, in.Hello.DeviceID)
	}

	if in.CapabilitySnap != nil {
		return validateSessionGenerationMessage("capability snapshot", "snapshot", connectedDeviceID, in.CapabilitySnap.DeviceID, in.CapabilitySnap.Generation)
	}

	if in.CapabilityDelta != nil {
		return validateCapabilityDelta(connectedDeviceID, capabilityReady, in.CapabilityDelta)
	}

	return validateEstablishedSessionMessage(connectedDeviceID, capabilityReady, in)
}

func validateSessionDeviceID(kind, messageLabel, connectedDeviceID, messageDeviceID string) error {
	if messageDeviceID == "" {
		return fmt.Errorf("%s requires device id", kind)
	}
	if connectedDeviceID != "" && messageDeviceID != connectedDeviceID {
		return fmt.Errorf("%s device id mismatch: connected=%s %s=%s", kind, connectedDeviceID, messageLabel, messageDeviceID)
	}
	return nil
}

func validateSessionGenerationMessage(kind, messageLabel, connectedDeviceID, messageDeviceID string, generation uint64) error {
	if err := validateSessionDeviceID(kind, messageLabel, connectedDeviceID, messageDeviceID); err != nil {
		return err
	}
	if generation == 0 {
		return fmt.Errorf("%s requires generation > 0", kind)
	}
	return nil
}

func validateCapabilityDelta(connectedDeviceID string, capabilityReady bool, delta *CapabilityDeltaRequest) error {
	if err := validateSessionReady(connectedDeviceID, capabilityReady, "capability delta"); err != nil {
		return err
	}
	return validateSessionGenerationMessage("capability delta", "delta", connectedDeviceID, delta.DeviceID, delta.Generation)
}

func validateEstablishedSessionMessage(connectedDeviceID string, capabilityReady bool, in ClientMessage) error {
	if err := validateSessionReady(connectedDeviceID, capabilityReady, "other messages"); err != nil {
		return err
	}
	msgDeviceID, hasDeviceID := extractMessageDeviceID(in)
	if hasDeviceID && msgDeviceID != "" && msgDeviceID != connectedDeviceID {
		return fmt.Errorf("message device id mismatch: connected=%s message=%s", connectedDeviceID, msgDeviceID)
	}
	return nil
}

func validateSessionReady(connectedDeviceID string, capabilityReady bool, messageKind string) error {
	if connectedDeviceID == "" {
		return fmt.Errorf("hello or register required before %s", messageKind)
	}
	if !capabilityReady {
		return fmt.Errorf("capability snapshot or register required before %s", messageKind)
	}
	return nil
}

func extractMessageDeviceID(in ClientMessage) (string, bool) {
	switch {
	case in.Hello != nil:
		return in.Hello.DeviceID, true
	case in.CapabilitySnap != nil:
		return in.CapabilitySnap.DeviceID, true
	case in.CapabilityDelta != nil:
		return in.CapabilityDelta.DeviceID, true
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
