package transport

import (
	"context"
	"errors"

	"github.com/curtcox/terminals/terminal_server/internal/scenario"
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

// CommandRequest carries a client-issued scenario command.
type CommandRequest struct {
	DeviceID string
	Kind     string // "voice" or "manual"
	Text     string // voice transcript
	Intent   string // explicit scenario intent
}

// ClientMessage is a one-of control stream message from client to server.
type ClientMessage struct {
	Register   *RegisterRequest
	Capability *CapabilityUpdateRequest
	Heartbeat  *HeartbeatRequest
	Command    *CommandRequest
}

// ServerMessage is a one-of control stream message from server to client.
type ServerMessage struct {
	RegisterAck   *RegisterResponse
	SetUI         *ui.Descriptor
	Notification  string
	ScenarioStart string
	Error         string
}

// StreamHandler processes control stream messages.
type StreamHandler struct {
	control *ControlService
	runtime *scenario.Runtime
}

// NewStreamHandler creates a handler for control stream messages.
func NewStreamHandler(control *ControlService) *StreamHandler {
	return &StreamHandler{control: control}
}

// NewStreamHandlerWithRuntime creates a handler with scenario runtime support.
func NewStreamHandlerWithRuntime(control *ControlService, runtime *scenario.Runtime) *StreamHandler {
	return &StreamHandler{
		control: control,
		runtime: runtime,
	}
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
	case msg.Command != nil:
		if h.runtime == nil {
			err := errors.New("scenario runtime not configured")
			return []ServerMessage{{Error: err.Error()}}, err
		}
		name, err := h.handleCommand(ctx, msg.Command)
		if err != nil {
			return []ServerMessage{{Error: err.Error()}}, err
		}
		return []ServerMessage{{
			ScenarioStart: name,
			Notification:  "Scenario started: " + name,
		}}, nil
	default:
		return []ServerMessage{{Error: ErrInvalidClientMessage.Error()}}, ErrInvalidClientMessage
	}
}

func (h *StreamHandler) handleCommand(ctx context.Context, cmd *CommandRequest) (string, error) {
	if cmd == nil {
		return "", ErrInvalidClientMessage
	}
	switch cmd.Kind {
	case "voice":
		return h.runtime.HandleVoiceText(ctx, cmd.DeviceID, cmd.Text, h.control.now().UTC())
	default:
		return h.runtime.HandleTrigger(ctx, scenario.Trigger{
			Kind:      scenario.TriggerManual,
			SourceID:  cmd.DeviceID,
			Intent:    cmd.Intent,
			Arguments: map[string]string{},
		})
	}
}
