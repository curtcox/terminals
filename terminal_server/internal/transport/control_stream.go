package transport

import (
	"context"
	"errors"

	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

var (
	// ErrInvalidClientMessage indicates an unsupported or empty client payload.
	ErrInvalidClientMessage = errors.New("invalid client message")
)

// CapabilityUpdateRequest is a transport-neutral capability update payload.
type CapabilityUpdateRequest struct {
	DeviceID     string
	Capabilities map[string]string
}

// HeartbeatRequest is a transport-neutral heartbeat payload.
type HeartbeatRequest struct {
	DeviceID string
}

// ClientMessage is a one-of control stream message from client to server.
type ClientMessage struct {
	Register   *RegisterRequest
	Capability *CapabilityUpdateRequest
	Heartbeat  *HeartbeatRequest
}

// ServerMessage is a one-of control stream message from server to client.
type ServerMessage struct {
	RegisterAck *RegisterResponse
	SetUI       *ui.Descriptor
	Error       string
}

// StreamHandler processes control stream messages.
type StreamHandler struct {
	control *ControlService
}

// NewStreamHandler creates a handler for control stream messages.
func NewStreamHandler(control *ControlService) *StreamHandler {
	return &StreamHandler{control: control}
}

// HandleMessage processes one incoming control message and returns responses.
func (h *StreamHandler) HandleMessage(ctx context.Context, msg ClientMessage) ([]ServerMessage, error) {
	switch {
	case msg.Register != nil:
		resp, err := h.control.Register(ctx, *msg.Register)
		if err != nil {
			return []ServerMessage{{Error: err.Error()}}, err
		}
		return []ServerMessage{
			{RegisterAck: &resp},
			{SetUI: &resp.Initial},
		}, nil
	case msg.Capability != nil:
		err := h.control.UpdateCapabilities(ctx, msg.Capability.DeviceID, msg.Capability.Capabilities)
		if err != nil {
			return []ServerMessage{{Error: err.Error()}}, err
		}
		return nil, nil
	case msg.Heartbeat != nil:
		err := h.control.Heartbeat(ctx, msg.Heartbeat.DeviceID)
		if err != nil {
			return []ServerMessage{{Error: err.Error()}}, err
		}
		return nil, nil
	default:
		return []ServerMessage{{Error: ErrInvalidClientMessage.Error()}}, ErrInvalidClientMessage
	}
}
