package transport

import (
	"context"
	"errors"
	"io"
)

func (s *Session) runOneClientMessage(
	ctx context.Context,
	stream ControlStream,
	state *controlSessionState,
	capabilityReady bool,
	send func(ServerMessage) error,
) (done bool, nextCapabilityReady bool, err error) {
	in, err := stream.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			s.disconnectSession(stream.Context(), state)
			return true, capabilityReady, nil
		}
		return false, capabilityReady, err
	}

	if sessionErr := validateSessionMessage(state.connectedDeviceID, capabilityReady, in); sessionErr != nil {
		s.handler.NoteProtocolError()
		if sendErr := send(ServerMessage{
			ErrorCode: ErrorCodeProtocolViolation,
			Error:     sessionErr.Error(),
		}); sendErr != nil {
			return false, capabilityReady, sendErr
		}
		return false, capabilityReady, nil
	}

	capabilityReady = state.observeClientMessage(in, capabilityReady, send)
	in.SessionDeviceID = state.connectedDeviceID

	out, handleErr := s.handler.HandleMessage(ctx, in)
	if err := s.sendSessionMessages(out, state.connectedDeviceID, send); err != nil {
		return false, capabilityReady, err
	}
	if shouldContinueSession(out, handleErr) {
		return false, capabilityReady, nil
	}
	if handleErr != nil {
		return false, capabilityReady, handleErr
	}
	return false, capabilityReady, nil
}

func (s *Session) disconnectSession(ctx context.Context, state *controlSessionState) {
	if state.connectedDeviceID != "" {
		s.handler.HandleDisconnect(state.connectedDeviceID)
		_ = s.control.Disconnect(ctx, state.connectedDeviceID)
	}
}
