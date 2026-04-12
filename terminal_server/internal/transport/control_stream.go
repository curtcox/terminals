package transport

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

var (
	// ErrInvalidClientMessage indicates an unsupported or empty client payload.
	ErrInvalidClientMessage = errors.New("invalid client message")
	// ErrInvalidCommandAction indicates an unsupported command action.
	ErrInvalidCommandAction = errors.New("invalid command action")
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
	RequestID string
	DeviceID  string
	Action    string // "start" (default) or "stop"
	Kind      string // "voice" or "manual"
	Text      string // voice transcript
	Intent    string // explicit scenario intent
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
	CommandAck    string
	SetUI         *ui.Descriptor
	Notification  string
	ScenarioStart string
	ScenarioStop  string
	Data          map[string]string
	Error         string
}

// StreamHandler processes control stream messages.
type StreamHandler struct {
	control   *ControlService
	runtime   *scenario.Runtime
	mu        sync.Mutex
	seen      map[string]ServerMessage
	seenOrder []string
	seenLimit int
}

// NewStreamHandler creates a handler for control stream messages.
func NewStreamHandler(control *ControlService) *StreamHandler {
	return &StreamHandler{
		control:   control,
		seen:      map[string]ServerMessage{},
		seenLimit: 1024,
	}
}

// NewStreamHandlerWithRuntime creates a handler with scenario runtime support.
func NewStreamHandlerWithRuntime(control *ControlService, runtime *scenario.Runtime) *StreamHandler {
	return &StreamHandler{
		control:   control,
		runtime:   runtime,
		seen:      map[string]ServerMessage{},
		seenLimit: 1024,
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
		if msg.Command.RequestID != "" {
			h.mu.Lock()
			if prior, ok := h.seen[msg.Command.RequestID]; ok {
				h.mu.Unlock()
				return []ServerMessage{prior}, nil
			}
			h.mu.Unlock()
		}
		commandResult, err := h.handleCommand(ctx, msg.Command)
		if err != nil {
			return []ServerMessage{{Error: err.Error()}}, err
		}
		if msg.Command.RequestID != "" {
			commandResult.CommandAck = msg.Command.RequestID
			h.mu.Lock()
			h.seen[msg.Command.RequestID] = commandResult
			h.seenOrder = append(h.seenOrder, msg.Command.RequestID)
			if len(h.seenOrder) > h.seenLimit {
				evict := h.seenOrder[0]
				h.seenOrder = h.seenOrder[1:]
				delete(h.seen, evict)
			}
			h.mu.Unlock()
		}
		return []ServerMessage{commandResult}, nil
	default:
		return []ServerMessage{{Error: ErrInvalidClientMessage.Error()}}, ErrInvalidClientMessage
	}
}

func (h *StreamHandler) handleCommand(ctx context.Context, cmd *CommandRequest) (ServerMessage, error) {
	if cmd == nil {
		return ServerMessage{}, ErrInvalidClientMessage
	}
	if cmd.Kind == "system" {
		return h.handleSystemCommand(cmd)
	}

	action := cmd.Action
	if action == "" {
		action = "start"
	}
	if action != "start" && action != "stop" {
		return ServerMessage{}, ErrInvalidCommandAction
	}

	switch cmd.Kind {
	case "voice":
		if action == "stop" {
			name, err := h.runtime.StopVoiceText(ctx, cmd.DeviceID, cmd.Text, h.control.now().UTC())
			if err != nil {
				return ServerMessage{}, err
			}
			return ServerMessage{
				ScenarioStop: name,
				Notification: "Scenario stopped: " + name,
			}, nil
		}
		name, err := h.runtime.HandleVoiceText(ctx, cmd.DeviceID, cmd.Text, h.control.now().UTC())
		if err != nil {
			return ServerMessage{}, err
		}
		return ServerMessage{
			ScenarioStart: name,
			Notification:  "Scenario started: " + name,
		}, nil
	default:
		trigger := scenario.Trigger{
			Kind:      scenario.TriggerManual,
			SourceID:  cmd.DeviceID,
			Intent:    cmd.Intent,
			Arguments: map[string]string{},
		}
		if action == "stop" {
			name, err := h.runtime.StopTrigger(ctx, trigger)
			if err != nil {
				return ServerMessage{}, err
			}
			return ServerMessage{
				ScenarioStop: name,
				Notification: "Scenario stopped: " + name,
			}, nil
		}
		name, err := h.runtime.HandleTrigger(ctx, trigger)
		if err != nil {
			return ServerMessage{}, err
		}
		return ServerMessage{
			ScenarioStart: name,
			Notification:  "Scenario started: " + name,
		}, nil
	}
}

func (h *StreamHandler) handleSystemCommand(cmd *CommandRequest) (ServerMessage, error) {
	if cmd == nil {
		return ServerMessage{}, ErrInvalidClientMessage
	}
	switch cmd.Intent {
	case "list_devices":
		data := map[string]string{}
		for _, d := range h.control.devices.List() {
			data[d.DeviceID] = fmt.Sprintf("%s|%s|%s", d.DeviceName, d.Platform, d.State)
		}
		return ServerMessage{
			Notification: "System query: list_devices",
			Data:         data,
		}, nil
	case "active_scenarios":
		data := map[string]string{}
		if h.runtime != nil && h.runtime.Engine != nil {
			for deviceID, scenarioName := range h.runtime.Engine.ActiveSnapshot() {
				data[deviceID] = scenarioName
			}
		}
		return ServerMessage{
			Notification: "System query: active_scenarios",
			Data:         data,
		}, nil
	default:
		return ServerMessage{}, fmt.Errorf("unknown system intent: %s", strings.TrimSpace(cmd.Intent))
	}
}
