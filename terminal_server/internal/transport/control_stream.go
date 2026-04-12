package transport

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
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
	// ErrInvalidCommandKind indicates an unsupported command kind.
	ErrInvalidCommandKind = errors.New("invalid command kind")
	// ErrMissingCommandIntent indicates required command intent is missing.
	ErrMissingCommandIntent = errors.New("missing command intent")
	// ErrMissingCommandText indicates required voice command text is missing.
	ErrMissingCommandText = errors.New("missing command text")
	// ErrMissingCommandDeviceID indicates required command device id is missing.
	ErrMissingCommandDeviceID = errors.New("missing command device id")
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
	control     *ControlService
	runtime     *scenario.Runtime
	metrics     *Metrics
	mu          sync.Mutex
	seen        map[string]ServerMessage
	seenOrder   []string
	seenLimit   int
	recent      []CommandEvent
	recentLimit int
}

// CommandEvent is a bounded audit record of command handling.
type CommandEvent struct {
	RequestID string
	DeviceID  string
	Kind      string
	Action    string
	Intent    string
	Outcome   string
	WhenUnix  int64
}

// NewStreamHandler creates a handler for control stream messages.
func NewStreamHandler(control *ControlService) *StreamHandler {
	return &StreamHandler{
		control:     control,
		metrics:     &Metrics{},
		seen:        map[string]ServerMessage{},
		seenLimit:   1024,
		recent:      []CommandEvent{},
		recentLimit: 200,
	}
}

// NewStreamHandlerWithRuntime creates a handler with scenario runtime support.
func NewStreamHandlerWithRuntime(control *ControlService, runtime *scenario.Runtime) *StreamHandler {
	return &StreamHandler{
		control:     control,
		runtime:     runtime,
		metrics:     &Metrics{},
		seen:        map[string]ServerMessage{},
		seenLimit:   1024,
		recent:      []CommandEvent{},
		recentLimit: 200,
	}
}

// HandleMessage processes one incoming control message and returns responses.
func (h *StreamHandler) HandleMessage(ctx context.Context, msg ClientMessage) ([]ServerMessage, error) {
	switch {
	case msg.Register != nil:
		h.metrics.registerReceived.Add(1)
		resp, err := h.control.Register(ctx, *msg.Register)
		if err != nil {
			h.metrics.protocolErrors.Add(1)
			return []ServerMessage{{Error: err.Error()}}, err
		}
		return []ServerMessage{
			{RegisterAck: &resp},
			{SetUI: &resp.Initial},
		}, nil
	case msg.Capability != nil:
		h.metrics.capabilityReceived.Add(1)
		err := h.control.UpdateCapabilities(ctx, msg.Capability.DeviceID, msg.Capability.Capabilities)
		if err != nil {
			h.metrics.protocolErrors.Add(1)
			return []ServerMessage{{Error: err.Error()}}, err
		}
		return nil, nil
	case msg.Heartbeat != nil:
		h.metrics.heartbeatReceived.Add(1)
		err := h.control.Heartbeat(ctx, msg.Heartbeat.DeviceID)
		if err != nil {
			h.metrics.protocolErrors.Add(1)
			return []ServerMessage{{Error: err.Error()}}, err
		}
		return nil, nil
	case msg.Command != nil:
		h.metrics.commandReceived.Add(1)
		if msg.Command.RequestID != "" {
			h.mu.Lock()
			if prior, ok := h.seen[msg.Command.RequestID]; ok {
				if h.metrics != nil {
					h.metrics.dedupeHits.Add(1)
				}
				h.appendCommandEventLocked(CommandEvent{
					RequestID: msg.Command.RequestID,
					DeviceID:  msg.Command.DeviceID,
					Kind:      msg.Command.Kind,
					Action:    defaultAction(msg.Command.Action),
					Intent:    msg.Command.Intent,
					Outcome:   "deduped",
					WhenUnix:  h.control.now().UTC().UnixMilli(),
				})
				h.mu.Unlock()
				return []ServerMessage{prior}, nil
			}
			h.mu.Unlock()
		}
		commandResult, err := h.handleCommand(ctx, msg.Command)
		if err != nil {
			h.metrics.commandErrors.Add(1)
			h.mu.Lock()
			h.appendCommandEventLocked(CommandEvent{
				RequestID: msg.Command.RequestID,
				DeviceID:  msg.Command.DeviceID,
				Kind:      msg.Command.Kind,
				Action:    defaultAction(msg.Command.Action),
				Intent:    msg.Command.Intent,
				Outcome:   "error:" + err.Error(),
				WhenUnix:  h.control.now().UTC().UnixMilli(),
			})
			h.mu.Unlock()
			return []ServerMessage{{Error: err.Error()}}, err
		}
		h.mu.Lock()
		h.appendCommandEventLocked(CommandEvent{
			RequestID: msg.Command.RequestID,
			DeviceID:  msg.Command.DeviceID,
			Kind:      msg.Command.Kind,
			Action:    defaultAction(msg.Command.Action),
			Intent:    msg.Command.Intent,
			Outcome:   commandOutcome(commandResult),
			WhenUnix:  h.control.now().UTC().UnixMilli(),
		})
		h.mu.Unlock()
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
		h.metrics.protocolErrors.Add(1)
		return []ServerMessage{{Error: ErrInvalidClientMessage.Error()}}, ErrInvalidClientMessage
	}
}

func (h *StreamHandler) handleCommand(ctx context.Context, cmd *CommandRequest) (ServerMessage, error) {
	if cmd == nil {
		return ServerMessage{}, ErrInvalidClientMessage
	}
	kind := cmd.Kind
	if kind == "" {
		kind = "manual"
	}

	if kind == "system" {
		return h.handleSystemCommand(ctx, cmd)
	}
	if strings.TrimSpace(cmd.DeviceID) == "" {
		return ServerMessage{}, ErrMissingCommandDeviceID
	}
	if h.runtime == nil {
		return ServerMessage{}, errors.New("scenario runtime not configured")
	}

	action := cmd.Action
	if action == "" {
		action = "start"
	}
	if action != "start" && action != "stop" {
		return ServerMessage{}, ErrInvalidCommandAction
	}

	switch kind {
	case "voice":
		if strings.TrimSpace(cmd.Text) == "" {
			return ServerMessage{}, ErrMissingCommandText
		}
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
	case "manual":
		if strings.TrimSpace(cmd.Intent) == "" {
			return ServerMessage{}, ErrMissingCommandIntent
		}
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
	default:
		return ServerMessage{}, ErrInvalidCommandKind
	}
}

func (h *StreamHandler) handleSystemCommand(ctx context.Context, cmd *CommandRequest) (ServerMessage, error) {
	if cmd == nil {
		return ServerMessage{}, ErrInvalidClientMessage
	}
	intent := strings.TrimSpace(cmd.Intent)
	if intent == "" {
		return ServerMessage{}, ErrMissingCommandIntent
	}
	switch intent {
	case "system_help":
		return ServerMessage{
			Notification: "System query: system_help",
			Data: map[string]string{
				"system_intents":  "server_status,runtime_status,scenario_registry,transport_metrics,list_devices,active_scenarios,pending_timers,recent_commands,device_status <device_id>,run_due_timers,system_help",
				"command_kinds":   "voice,manual,system",
				"command_actions": "start,stop",
			},
		}, nil
	case "server_status":
		return ServerMessage{
			Notification: "System query: server_status",
			Data:         h.control.StatusData(),
		}, nil
	case "runtime_status":
		data := map[string]string{}
		if h.runtime != nil {
			for k, v := range h.runtime.StatusData() {
				data[k] = v
			}
		}
		return ServerMessage{
			Notification: "System query: runtime_status",
			Data:         data,
		}, nil
	case "scenario_registry":
		data := map[string]string{}
		if h.runtime != nil && h.runtime.Engine != nil {
			for _, item := range h.runtime.Engine.RegistrySnapshot() {
				data[item.Name] = fmt.Sprintf("priority=%d", item.Priority)
			}
		}
		return ServerMessage{
			Notification: "System query: scenario_registry",
			Data:         data,
		}, nil
	case "run_due_timers":
		processed := 0
		if h.runtime != nil {
			count, err := h.runtime.ProcessDueTimers(ctx, h.control.now().UTC())
			if err != nil {
				return ServerMessage{}, err
			}
			processed = count
		}
		return ServerMessage{
			Notification: "System query: run_due_timers",
			Data: map[string]string{
				"processed": toString(int64(processed)),
			},
		}, nil
	case "transport_metrics":
		data := map[string]string{}
		if h.metrics != nil {
			for k, v := range h.metrics.Snapshot() {
				data[k] = v
			}
		}
		return ServerMessage{
			Notification: "System query: transport_metrics",
			Data:         data,
		}, nil
	case "list_devices":
		data := map[string]string{}
		devices := h.control.devices.List()
		sort.Slice(devices, func(i, j int) bool {
			return devices[i].DeviceID < devices[j].DeviceID
		})
		for _, d := range devices {
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
	case "pending_timers":
		data := map[string]string{}
		if h.runtime != nil && h.runtime.Env != nil && h.runtime.Env.Scheduler != nil {
			for _, key := range h.runtime.Env.Scheduler.Due(math.MaxInt64) {
				data[key] = "scheduled"
			}
		}
		return ServerMessage{
			Notification: "System query: pending_timers",
			Data:         data,
		}, nil
	case "recent_commands":
		data := map[string]string{}
		h.mu.Lock()
		events := make([]CommandEvent, len(h.recent))
		copy(events, h.recent)
		h.mu.Unlock()
		for i, ev := range events {
			key := fmt.Sprintf("%03d", i)
			data[key] = strings.Join([]string{
				ev.RequestID,
				ev.DeviceID,
				ev.Kind,
				ev.Action,
				ev.Intent,
				ev.Outcome,
				strconv.FormatInt(ev.WhenUnix, 10),
			}, "|")
		}
		return ServerMessage{
			Notification: "System query: recent_commands",
			Data:         data,
		}, nil
	default:
		if strings.HasPrefix(intent, "device_status ") {
			deviceID := strings.TrimSpace(strings.TrimPrefix(intent, "device_status "))
			if deviceID == "" {
				return ServerMessage{}, fmt.Errorf("device_status requires device id")
			}
			deviceState, ok := h.control.devices.Get(deviceID)
			if !ok {
				return ServerMessage{}, fmt.Errorf("device not found: %s", deviceID)
			}
			data := map[string]string{
				"device_id":   deviceState.DeviceID,
				"device_name": deviceState.DeviceName,
				"device_type": deviceState.DeviceType,
				"platform":    deviceState.Platform,
				"state":       string(deviceState.State),
			}
			for k, v := range deviceState.Capabilities {
				data["cap."+k] = v
			}
			return ServerMessage{
				Notification: "System query: device_status",
				Data:         data,
			}, nil
		}
		return ServerMessage{}, fmt.Errorf("unknown system intent: %s", intent)
	}
}

// NoteProtocolError increments protocol error counters from session-level validation.
func (h *StreamHandler) NoteProtocolError() {
	if h.metrics != nil {
		h.metrics.protocolErrors.Add(1)
	}
}

func (h *StreamHandler) appendCommandEventLocked(ev CommandEvent) {
	h.recent = append(h.recent, ev)
	if len(h.recent) > h.recentLimit {
		h.recent = h.recent[len(h.recent)-h.recentLimit:]
	}
}

func defaultAction(action string) string {
	if action == "" {
		return "start"
	}
	return action
}

func commandOutcome(msg ServerMessage) string {
	switch {
	case msg.ScenarioStart != "":
		return "started:" + msg.ScenarioStart
	case msg.ScenarioStop != "":
		return "stopped:" + msg.ScenarioStop
	case msg.Notification != "":
		return "notified"
	default:
		return "ok"
	}
}
