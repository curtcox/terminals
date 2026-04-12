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
	"time"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/terminal"
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

// InputRequest carries client input events relevant to active scenarios.
type InputRequest struct {
	DeviceID    string
	ComponentID string
	Action      string
	Value       string
	KeyText     string
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
	Input      *InputRequest
	Command    *CommandRequest
}

// ServerMessage is a one-of control stream message from server to client.
type ServerMessage struct {
	RegisterAck   *RegisterResponse
	CommandAck    string
	SetUI         *ui.Descriptor
	UpdateUI      *UIUpdate
	TransitionUI  *UITransition
	Notification  string
	ScenarioStart string
	ScenarioStop  string
	Data          map[string]string
	ErrorCode     string
	Error         string
}

// UIUpdate carries a server-driven patch to a specific UI component.
type UIUpdate struct {
	ComponentID string
	Node        ui.Descriptor
}

// UITransition carries a UI transition hint for the active device UI.
type UITransition struct {
	Transition string
	DurationMS int32
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

	terminals              *terminal.Manager
	terminalByDevice       map[string]string
	terminalOutputByDevice map[string]string
	terminalDraftByDevice  map[string]string
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
		control:                control,
		metrics:                &Metrics{},
		seen:                   map[string]ServerMessage{},
		seenLimit:              1024,
		recent:                 []CommandEvent{},
		recentLimit:            200,
		terminals:              terminal.NewManager(),
		terminalByDevice:       map[string]string{},
		terminalOutputByDevice: map[string]string{},
		terminalDraftByDevice:  map[string]string{},
	}
}

// NewStreamHandlerWithRuntime creates a handler with scenario runtime support.
func NewStreamHandlerWithRuntime(control *ControlService, runtime *scenario.Runtime) *StreamHandler {
	return &StreamHandler{
		control:                control,
		runtime:                runtime,
		metrics:                &Metrics{},
		seen:                   map[string]ServerMessage{},
		seenLimit:              1024,
		recent:                 []CommandEvent{},
		recentLimit:            200,
		terminals:              terminal.NewManager(),
		terminalByDevice:       map[string]string{},
		terminalOutputByDevice: map[string]string{},
		terminalDraftByDevice:  map[string]string{},
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
			return []ServerMessage{{ErrorCode: errorCodeFor(err), Error: err.Error()}}, err
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
			return []ServerMessage{{ErrorCode: errorCodeFor(err), Error: err.Error()}}, err
		}
		return nil, nil
	case msg.Heartbeat != nil:
		h.metrics.heartbeatReceived.Add(1)
		err := h.control.Heartbeat(ctx, msg.Heartbeat.DeviceID)
		if err != nil {
			h.metrics.protocolErrors.Add(1)
			return []ServerMessage{{ErrorCode: errorCodeFor(err), Error: err.Error()}}, err
		}
		return nil, nil
	case msg.Input != nil:
		out, err := h.handleInput(ctx, msg.Input)
		if err != nil {
			h.metrics.protocolErrors.Add(1)
			return []ServerMessage{{ErrorCode: errorCodeFor(err), Error: err.Error()}}, err
		}
		return out, nil
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
			return []ServerMessage{{ErrorCode: errorCodeFor(err), Error: err.Error()}}, err
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
		postResponses := h.commandResponses(ctx, msg.Command, commandResult)
		return postResponses, nil
	default:
		h.metrics.protocolErrors.Add(1)
		return []ServerMessage{{ErrorCode: errorCodeFor(ErrInvalidClientMessage), Error: ErrInvalidClientMessage.Error()}}, ErrInvalidClientMessage
	}
}

func (h *StreamHandler) commandResponses(ctx context.Context, cmd *CommandRequest, commandResult ServerMessage) []ServerMessage {
	responses := []ServerMessage{commandResult}
	if cmd == nil {
		return responses
	}
	if commandResult.ScenarioStop == "terminal" {
		h.terminateTerminalForDevice(cmd.DeviceID)
		responses = append(responses, ServerMessage{
			TransitionUI: &UITransition{
				Transition: "terminal_exit",
				DurationMS: 220,
			},
		})
		return responses
	}
	if cmd.Action != "" && cmd.Action != CommandActionStart {
		return responses
	}
	if commandResult.ScenarioStart != "terminal" {
		return responses
	}

	output, err := h.ensureTerminalSession(ctx, cmd.DeviceID)
	if err != nil {
		responses = append(responses, ServerMessage{
			Notification: "Terminal session failed: " + err.Error(),
		})
		return responses
	}
	terminalUI := ui.TerminalViewWithOutput(cmd.DeviceID, output)
	responses = append(responses, ServerMessage{SetUI: &terminalUI})
	responses = append(responses, ServerMessage{
		TransitionUI: &UITransition{
			Transition: "terminal_enter",
			DurationMS: 220,
		},
	})
	return responses
}

func (h *StreamHandler) ensureTerminalSession(ctx context.Context, deviceID string) (string, error) {
	if strings.TrimSpace(deviceID) == "" {
		return "", ErrMissingCommandDeviceID
	}

	h.mu.Lock()
	sessionID := h.terminalByDevice[deviceID]
	h.mu.Unlock()
	if sessionID != "" {
		h.mu.Lock()
		output := h.terminalOutputByDevice[deviceID]
		h.mu.Unlock()
		return output, nil
	}

	session, err := h.terminals.Start(ctx, terminal.StartOptions{DeviceID: deviceID})
	if err != nil {
		return "", err
	}

	h.mu.Lock()
	h.terminalByDevice[deviceID] = session.ID
	h.terminalOutputByDevice[deviceID] = ""
	h.mu.Unlock()
	return "", nil
}

func (h *StreamHandler) terminateTerminalForDevice(deviceID string) {
	if strings.TrimSpace(deviceID) == "" {
		return
	}

	h.mu.Lock()
	sessionID := h.terminalByDevice[deviceID]
	delete(h.terminalByDevice, deviceID)
	delete(h.terminalOutputByDevice, deviceID)
	delete(h.terminalDraftByDevice, deviceID)
	h.mu.Unlock()
	if sessionID != "" {
		_ = h.terminals.Close(sessionID)
	}
}

func (h *StreamHandler) handleInput(_ context.Context, in *InputRequest) ([]ServerMessage, error) {
	if in == nil {
		return nil, ErrInvalidClientMessage
	}
	deviceID := strings.TrimSpace(in.DeviceID)
	if deviceID == "" {
		return nil, ErrMissingCommandDeviceID
	}

	h.mu.Lock()
	sessionID := h.terminalByDevice[deviceID]
	h.mu.Unlock()
	if sessionID == "" {
		return nil, nil
	}

	action := strings.ToLower(strings.TrimSpace(in.Action))
	componentID := strings.TrimSpace(in.ComponentID)

	switch action {
	case "change":
		if componentID == "terminal_input" {
			h.mu.Lock()
			h.terminalDraftByDevice[deviceID] = in.Value
			h.mu.Unlock()
			return nil, nil
		}
		if update, ok := h.renderTerminalUIAction(deviceID, componentID, action, in.Value); ok {
			return []ServerMessage{{UpdateUI: update}}, nil
		}
		return nil, nil
	case "toggle", "select":
		if update, ok := h.renderTerminalUIAction(deviceID, componentID, action, in.Value); ok {
			return []ServerMessage{{UpdateUI: update}}, nil
		}
		return nil, nil
	}

	text := strings.TrimSpace(in.Value)
	if text == "" && componentID == "terminal_input" {
		h.mu.Lock()
		text = strings.TrimSpace(h.terminalDraftByDevice[deviceID])
		h.mu.Unlock()
	}
	if text == "" {
		text = in.KeyText
	}
	if text == "" {
		return nil, nil
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if err := h.terminals.Write(sessionID, []byte(text)); err != nil {
		return nil, err
	}
	if componentID == "terminal_input" {
		h.mu.Lock()
		h.terminalDraftByDevice[deviceID] = ""
		h.mu.Unlock()
	}

	combinedOutput := h.readTerminalOutput(deviceID, sessionID)
	return []ServerMessage{{
		UpdateUI: &UIUpdate{
			ComponentID: "terminal_output",
			Node:        ui.TerminalOutputPatch(combinedOutput),
		},
	}}, nil
}

func (h *StreamHandler) renderTerminalUIAction(deviceID, componentID, action, value string) (*UIUpdate, bool) {
	if strings.TrimSpace(componentID) == "" {
		return nil, false
	}
	line := fmt.Sprintf("[ui_action] %s %s = %s\n", componentID, action, value)

	h.mu.Lock()
	existing := h.terminalOutputByDevice[deviceID]
	existing += line
	if len(existing) > 12000 {
		existing = existing[len(existing)-12000:]
	}
	h.terminalOutputByDevice[deviceID] = existing
	h.mu.Unlock()

	return &UIUpdate{
		ComponentID: "terminal_output",
		Node:        ui.TerminalOutputPatch(existing),
	}, true
}

func (h *StreamHandler) readTerminalOutput(deviceID, sessionID string) string {
	deadline := time.Now().Add(350 * time.Millisecond)
	var chunk []byte
	for time.Now().Before(deadline) {
		out, err := h.terminals.ReadAvailable(sessionID, 4096)
		if err != nil {
			break
		}
		if len(out) > 0 {
			chunk = append(chunk, out...)
		}
		time.Sleep(25 * time.Millisecond)
	}

	h.mu.Lock()
	existing := h.terminalOutputByDevice[deviceID]
	existing += string(chunk)
	if len(existing) > 12000 {
		existing = existing[len(existing)-12000:]
	}
	h.terminalOutputByDevice[deviceID] = existing
	h.mu.Unlock()
	return existing
}

func (h *StreamHandler) handleCommand(ctx context.Context, cmd *CommandRequest) (ServerMessage, error) {
	if cmd == nil {
		return ServerMessage{}, ErrInvalidClientMessage
	}
	kind := cmd.Kind
	if kind == "" {
		kind = CommandKindManual
	}

	if kind == CommandKindSystem {
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
		action = CommandActionStart
	}
	if action != CommandActionStart && action != CommandActionStop {
		return ServerMessage{}, ErrInvalidCommandAction
	}

	switch kind {
	case CommandKindVoice:
		if strings.TrimSpace(cmd.Text) == "" {
			return ServerMessage{}, ErrMissingCommandText
		}
		if action == CommandActionStop {
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
	case CommandKindManual:
		if strings.TrimSpace(cmd.Intent) == "" {
			return ServerMessage{}, ErrMissingCommandIntent
		}
		trigger := scenario.Trigger{
			Kind:      scenario.TriggerManual,
			SourceID:  cmd.DeviceID,
			Intent:    cmd.Intent,
			Arguments: map[string]string{},
		}
		if action == CommandActionStop {
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
	parsed, err := ParseSystemIntent(cmd.Intent)
	if err != nil {
		return ServerMessage{}, err
	}
	switch parsed.Name {
	case SystemIntentHelp:
		return ServerMessage{
			Notification: "System query: system_help",
			Data: map[string]string{
				"system_intents":  SystemHelpIntentsString(),
				"command_kinds":   "voice,manual,system",
				"command_actions": "start,stop",
			},
		}, nil
	case SystemIntentServerStatus:
		return ServerMessage{
			Notification: "System query: server_status",
			Data:         h.control.StatusData(),
		}, nil
	case SystemIntentRuntimeStatus:
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
	case SystemIntentScenarioRegistry:
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
	case SystemIntentRunDueTimers:
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
	case SystemIntentReconcileLiveness:
		timeout := 2 * time.Minute
		timeoutSeconds := "120"
		if parsed.Arg != "" {
			seconds, convErr := strconv.Atoi(parsed.Arg)
			if convErr != nil || seconds < 0 {
				return ServerMessage{}, fmt.Errorf("invalid reconcile_liveness seconds: %s", parsed.Arg)
			}
			timeout = time.Duration(seconds) * time.Second
			timeoutSeconds = parsed.Arg
		}
		updated := h.control.ReconcileLiveness(timeout)
		return ServerMessage{
			Notification: "System query: reconcile_liveness",
			Data: map[string]string{
				"updated":         toString(int64(updated)),
				"timeout_seconds": timeoutSeconds,
			},
		}, nil
	case SystemIntentTransportMetrics:
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
	case SystemIntentListDevices:
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
	case SystemIntentActiveScenarios:
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
	case SystemIntentPendingTimers:
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
	case SystemIntentRecentCommands:
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
		if parsed.Name == SystemIntentDeviceStatus && parsed.Arg != "" {
			deviceState, ok := h.control.devices.Get(parsed.Arg)
			if !ok {
				return ServerMessage{}, fmt.Errorf("device not found: %s", parsed.Arg)
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
		return ServerMessage{}, fmt.Errorf("unknown system intent: %s", cmd.Intent)
	}
}

// NoteProtocolError increments protocol error counters from session-level validation.
func (h *StreamHandler) NoteProtocolError() {
	if h.metrics != nil {
		h.metrics.protocolErrors.Add(1)
	}
}

// HandleDisconnect releases stream-scoped resources for a disconnected device.
func (h *StreamHandler) HandleDisconnect(deviceID string) {
	h.terminateTerminalForDevice(deviceID)
	h.disconnectRoutesForDevice(deviceID)
}

func (h *StreamHandler) disconnectRoutesForDevice(deviceID string) {
	if h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.IO == nil {
		return
	}
	routeIO, ok := h.runtime.Env.IO.(interface {
		RoutesForDevice(string) []iorouter.Route
		Disconnect(sourceID, targetID, streamKind string) error
	})
	if !ok {
		return
	}

	routes := routeIO.RoutesForDevice(deviceID)
	for _, route := range routes {
		_ = routeIO.Disconnect(route.SourceID, route.TargetID, route.StreamKind)
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
		return CommandActionStart
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
