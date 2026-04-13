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

// SensorDataRequest carries sensor telemetry from device clients.
type SensorDataRequest struct {
	DeviceID string
	UnixMS   int64
	Values   map[string]float64
}

// StreamReadyRequest indicates a media stream is ready on the client side.
type StreamReadyRequest struct {
	StreamID string
}

// RouteStreamResponse instructs clients to establish or acknowledge media routing.
type RouteStreamResponse struct {
	StreamID       string
	SourceDeviceID string
	TargetDeviceID string
	Kind           string
}

// StartStreamResponse instructs clients to start an underlying media stream.
type StartStreamResponse struct {
	StreamID       string
	Kind           string
	SourceDeviceID string
	TargetDeviceID string
	Metadata       map[string]string
}

// StopStreamResponse instructs clients to stop an underlying media stream.
type StopStreamResponse struct {
	StreamID string
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
	Sensor     *SensorDataRequest
	StreamReady *StreamReadyRequest
	Input      *InputRequest
	Command    *CommandRequest
}

// ServerMessage is a one-of control stream message from server to client.
type ServerMessage struct {
	RegisterAck   *RegisterResponse
	CommandAck    string
	SetUI         *ui.Descriptor
	UpdateUI      *UIUpdate
	StartStream   *StartStreamResponse
	StopStream    *StopStreamResponse
	RouteStream   *RouteStreamResponse
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
	terminalOutputDirty    map[string]bool
	terminalLastUIFlush    map[string]time.Time
	terminalReadDeadline   time.Duration
	terminalReadInterval   time.Duration
	terminalUIInterval     time.Duration
}

const (
	defaultTerminalReadDeadline = 180 * time.Millisecond
	defaultTerminalReadInterval = 10 * time.Millisecond
	defaultTerminalUIInterval   = 800 * time.Millisecond
)

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
		terminalOutputDirty:    map[string]bool{},
		terminalLastUIFlush:    map[string]time.Time{},
		terminalReadDeadline:   defaultTerminalReadDeadline,
		terminalReadInterval:   defaultTerminalReadInterval,
		terminalUIInterval:     defaultTerminalUIInterval,
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
		terminalOutputDirty:    map[string]bool{},
		terminalLastUIFlush:    map[string]time.Time{},
		terminalReadDeadline:   defaultTerminalReadDeadline,
		terminalReadInterval:   defaultTerminalReadInterval,
		terminalUIInterval:     defaultTerminalUIInterval,
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
		update, pollErr := h.pollTerminalOutput(msg.Heartbeat.DeviceID, false)
		if pollErr != nil {
			h.metrics.protocolErrors.Add(1)
			return []ServerMessage{{ErrorCode: errorCodeFor(pollErr), Error: pollErr.Error()}}, pollErr
		}
		if update != nil {
			return []ServerMessage{*update}, nil
		}
		return nil, nil
	case msg.Sensor != nil:
		h.metrics.sensorReceived.Add(1)
		return nil, nil
	case msg.StreamReady != nil:
		h.metrics.streamReadyReceived.Add(1)
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
		beforeRoutes := h.routeSnapshotForDevice(msg.Command.DeviceID)
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
		afterRoutes := h.routeSnapshotForDevice(msg.Command.DeviceID)
		routeUpdates := h.routeUpdatesForCommand(msg.Command, commandResult, beforeRoutes, afterRoutes)
		if len(routeUpdates) > 0 {
			postResponses = append(postResponses, routeUpdates...)
		}
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
	if commandResult.ScenarioStop != "" {
		h.disconnectScenarioRoutes(cmd.DeviceID, commandResult.ScenarioStop)
	}
	if refresh, ok := h.commandTerminalRefresh(ctx, cmd); ok {
		responses = append(responses, refresh)
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

func (h *StreamHandler) routeUpdatesForCommand(
	cmd *CommandRequest,
	commandResult ServerMessage,
	before []iorouter.Route,
	after []iorouter.Route,
) []ServerMessage {
	if cmd == nil {
		return nil
	}
	if commandResult.ScenarioStart == "" && commandResult.ScenarioStop == "" {
		return nil
	}
	action := defaultAction(cmd.Action)
	if action != CommandActionStart && action != CommandActionStop {
		return nil
	}
	beforeSet := map[string]struct{}{}
	for _, route := range before {
		beforeSet[routeStreamID(route)] = struct{}{}
	}
	afterSet := map[string]iorouter.Route{}
	for _, route := range after {
		afterSet[routeStreamID(route)] = route
	}
	out := make([]ServerMessage, 0, len(after))
	if action == CommandActionStart {
		for _, route := range after {
			routeID := routeStreamID(route)
			if _, exists := beforeSet[routeID]; exists {
				continue
			}
			out = append(out, ServerMessage{
				StartStream: &StartStreamResponse{
					StreamID:       routeID,
					Kind:           route.StreamKind,
					SourceDeviceID: route.SourceID,
					TargetDeviceID: route.TargetID,
					Metadata: map[string]string{
						"origin": "route_delta",
					},
				},
			})
			out = append(out, ServerMessage{
				RouteStream: &RouteStreamResponse{
					StreamID:       routeID,
					SourceDeviceID: route.SourceID,
					TargetDeviceID: route.TargetID,
					Kind:           route.StreamKind,
				},
			})
		}
	}
	if action == CommandActionStop {
		for _, route := range before {
			routeID := routeStreamID(route)
			if _, exists := afterSet[routeID]; exists {
				continue
			}
			out = append(out, ServerMessage{
				StopStream: &StopStreamResponse{
					StreamID: routeID,
				},
			})
		}
	}
	return out
}

func (h *StreamHandler) routeSnapshotForDevice(deviceID string) []iorouter.Route {
	if h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.IO == nil {
		return nil
	}
	routeProvider, ok := h.runtime.Env.IO.(interface {
		RoutesForDevice(deviceID string) []iorouter.Route
	})
	if !ok {
		return nil
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil
	}
	return routeProvider.RoutesForDevice(deviceID)
}

func routeStreamID(route iorouter.Route) string {
	return "route:" + route.SourceID + "|" + route.TargetID + "|" + route.StreamKind
}

func (h *StreamHandler) disconnectScenarioRoutes(deviceID, scenarioName string) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return
	}
	if h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.IO == nil {
		return
	}
	routeProvider, ok := h.runtime.Env.IO.(interface {
		RoutesForDevice(deviceID string) []iorouter.Route
		Disconnect(sourceID, targetID, streamKind string) error
	})
	if !ok {
		return
	}

	for _, route := range routeProvider.RoutesForDevice(deviceID) {
		if !isScenarioOwnedRoute(deviceID, scenarioName, route) {
			continue
		}
		_ = routeProvider.Disconnect(route.SourceID, route.TargetID, route.StreamKind)
	}
}

func isScenarioOwnedRoute(deviceID, scenarioName string, route iorouter.Route) bool {
	switch scenarioName {
	case "intercom":
		return route.SourceID == deviceID && route.StreamKind == "audio"
	case "pa_system":
		return route.SourceID == deviceID && route.StreamKind == "pa_audio"
	case "multi_window":
		return route.TargetID == deviceID && route.StreamKind == "video"
	default:
		return false
	}
}

func (h *StreamHandler) commandTerminalRefresh(_ context.Context, cmd *CommandRequest) (ServerMessage, bool) {
	if cmd == nil {
		return ServerMessage{}, false
	}
	targetDeviceID := ""
	switch cmd.Kind {
	case CommandKindManual:
		if strings.TrimSpace(cmd.Intent) != SystemIntentTerminalRefresh {
			return ServerMessage{}, false
		}
		targetDeviceID = strings.TrimSpace(cmd.DeviceID)
	case CommandKindSystem:
		parsed, err := ParseSystemIntent(cmd.Intent)
		if err != nil || parsed.Name != SystemIntentTerminalRefresh {
			return ServerMessage{}, false
		}
		targetDeviceID = strings.TrimSpace(parsed.Arg)
		if targetDeviceID == "" {
			targetDeviceID = strings.TrimSpace(cmd.DeviceID)
		}
	default:
		return ServerMessage{}, false
	}
	if targetDeviceID == "" {
		return ServerMessage{}, false
	}
	update, err := h.pollTerminalOutput(targetDeviceID, true)
	if err != nil || update == nil {
		return ServerMessage{}, false
	}
	return *update, true
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
	delete(h.terminalOutputDirty, deviceID)
	delete(h.terminalLastUIFlush, deviceID)
	h.mu.Unlock()
	if sessionID != "" {
		_ = h.terminals.Close(sessionID)
	}
}

func (h *StreamHandler) handleInput(ctx context.Context, in *InputRequest) ([]ServerMessage, error) {
	if in == nil {
		return nil, ErrInvalidClientMessage
	}
	deviceID := strings.TrimSpace(in.DeviceID)
	if deviceID == "" {
		return nil, ErrMissingCommandDeviceID
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
	case SystemIntentTerminalRefresh:
		cmd := &CommandRequest{
			DeviceID: deviceID,
			Kind:     CommandKindManual,
			Intent:   SystemIntentTerminalRefresh,
		}
		commandResult, err := h.handleCommand(ctx, cmd)
		if err != nil {
			return nil, err
		}
		return h.commandResponses(ctx, cmd, commandResult), nil
	}

	if action != "" && (componentID != "terminal_input" || action != "submit") {
		if out, routed, err := h.routeScenarioUIAction(ctx, deviceID, action); routed {
			return out, err
		}
	}

	h.mu.Lock()
	sessionID := h.terminalByDevice[deviceID]
	h.mu.Unlock()
	if sessionID == "" {
		return nil, nil
	}

	text := in.Value
	fromKey := false
	if text == "" && componentID == "terminal_input" {
		h.mu.Lock()
		text = h.terminalDraftByDevice[deviceID]
		h.mu.Unlock()
	}
	if text == "" {
		text = in.KeyText
		fromKey = text != ""
	}
	if text == "" || (!fromKey && strings.TrimSpace(text) == "") {
		return nil, nil
	}
	if fromKey {
		text = normalizeTerminalKeyText(text)
	}
	if !fromKey && !strings.HasSuffix(text, "\n") {
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

	h.readTerminalOutput(deviceID, sessionID)
	return []ServerMessage{{
		UpdateUI: h.terminalOutputUpdate(deviceID),
	}}, nil
}

func (h *StreamHandler) routeScenarioUIAction(ctx context.Context, deviceID, action string) ([]ServerMessage, bool, error) {
	if h.runtime == nil || h.runtime.Engine == nil {
		return nil, false, nil
	}

	activeName, active := h.runtime.Engine.Active(deviceID)
	if !active {
		return nil, false, nil
	}

	action = strings.TrimSpace(action)
	if action == "" {
		return nil, false, nil
	}

	intent := ""
	commandAction := CommandActionStart
	switch {
	case action == "stop_active":
		intent = activeName
		commandAction = CommandActionStop
	case strings.HasPrefix(action, "start:"):
		intent = strings.TrimSpace(strings.TrimPrefix(action, "start:"))
	case strings.HasPrefix(action, "stop:"):
		intent = strings.TrimSpace(strings.TrimPrefix(action, "stop:"))
		commandAction = CommandActionStop
	default:
		intent = action
	}
	if intent == "" {
		return nil, true, nil
	}
	if commandAction == CommandActionStart && !h.isRegisteredScenario(intent) {
		return nil, false, nil
	}
	if commandAction == CommandActionStop && action != "stop_active" && !h.isRegisteredScenario(intent) {
		return nil, false, nil
	}

	trigger := scenario.Trigger{
		Kind:     scenario.TriggerManual,
		SourceID: deviceID,
		Intent:   intent,
		Arguments: map[string]string{
			"device_id": deviceID,
		},
	}
	if commandAction == CommandActionStop {
		name, err := h.runtime.StopTrigger(ctx, trigger)
		if err != nil {
			return nil, true, err
		}
		result := ServerMessage{
			ScenarioStop: name,
			Notification: "Scenario stopped: " + name,
		}
		cmd := &CommandRequest{
			DeviceID: deviceID,
			Action:   commandAction,
			Kind:     CommandKindManual,
			Intent:   intent,
		}
		return h.commandResponses(ctx, cmd, result), true, nil
	}
	name, err := h.runtime.HandleTrigger(ctx, trigger)
	if err != nil {
		return nil, true, err
	}
	result := ServerMessage{
		ScenarioStart: name,
		Notification:  "Scenario started: " + name,
	}
	cmd := &CommandRequest{
		DeviceID: deviceID,
		Action:   commandAction,
		Kind:     CommandKindManual,
		Intent:   intent,
	}
	return h.commandResponses(ctx, cmd, result), true, nil
}

func (h *StreamHandler) isRegisteredScenario(name string) bool {
	if h.runtime == nil || h.runtime.Engine == nil || strings.TrimSpace(name) == "" {
		return false
	}
	for _, item := range h.runtime.Engine.RegistrySnapshot() {
		if item.Name == name {
			return true
		}
	}
	return false
}

func (h *StreamHandler) renderTerminalUIAction(deviceID, componentID, action, value string) (*UIUpdate, bool) {
	if strings.TrimSpace(componentID) == "" {
		return nil, false
	}
	line := fmt.Sprintf("[ui_action] %s %s = %s\n", componentID, action, value)

	h.appendTerminalOutput(deviceID, line)
	return h.terminalOutputUpdate(deviceID), true
}

func (h *StreamHandler) pollTerminalOutput(deviceID string, force bool) (*ServerMessage, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, nil
	}

	h.mu.Lock()
	sessionID := h.terminalByDevice[deviceID]
	h.mu.Unlock()
	if sessionID == "" {
		return nil, nil
	}

	chunk, err := h.terminals.ReadAvailable(sessionID, 4096)
	if err != nil {
		return nil, err
	}
	if len(chunk) == 0 {
		if !h.shouldEmitTerminalUpdate(deviceID, force) {
			return nil, nil
		}
		return &ServerMessage{
			UpdateUI: h.terminalOutputUpdate(deviceID),
		}, nil
	}

	h.appendTerminalOutput(deviceID, string(chunk))
	if !h.shouldEmitTerminalUpdate(deviceID, force) {
		return nil, nil
	}
	return &ServerMessage{
		UpdateUI: h.terminalOutputUpdate(deviceID),
	}, nil
}

func (h *StreamHandler) appendTerminalOutput(deviceID, chunk string) string {
	h.mu.Lock()
	existing := h.terminalOutputByDevice[deviceID]
	if chunk != "" {
		existing += chunk
		h.terminalOutputDirty[deviceID] = true
	}
	if len(existing) > 12000 {
		existing = existing[len(existing)-12000:]
	}
	h.terminalOutputByDevice[deviceID] = existing
	h.mu.Unlock()
	return existing
}

func (h *StreamHandler) shouldEmitTerminalUpdate(deviceID string, force bool) bool {
	h.mu.Lock()
	dirty := h.terminalOutputDirty[deviceID]
	last := h.terminalLastUIFlush[deviceID]
	interval := h.terminalUIInterval
	h.mu.Unlock()

	if !dirty {
		return false
	}
	if force {
		return true
	}
	if interval <= 0 {
		interval = defaultTerminalUIInterval
	}
	if last.IsZero() {
		return true
	}
	return h.nowUTC().Sub(last) >= interval
}

func (h *StreamHandler) terminalOutputUpdate(deviceID string) *UIUpdate {
	h.mu.Lock()
	output := h.terminalOutputByDevice[deviceID]
	h.terminalOutputDirty[deviceID] = false
	h.terminalLastUIFlush[deviceID] = h.nowUTC()
	h.mu.Unlock()
	return &UIUpdate{
		ComponentID: "terminal_output",
		Node:        ui.TerminalOutputPatch(output),
	}
}

func (h *StreamHandler) nowUTC() time.Time {
	if h.control != nil && h.control.now != nil {
		return h.control.now().UTC()
	}
	return time.Now().UTC()
}

func normalizeTerminalKeyText(text string) string {
	if text == "" {
		return text
	}
	// PTY line discipline typically expects DEL (0x7f) for backward delete.
	return strings.ReplaceAll(text, "\b", "\x7f")
}

func (h *StreamHandler) readTerminalOutput(deviceID, sessionID string) string {
	readDeadline, readInterval := h.terminalReadSettings()
	deadline := time.Now().Add(readDeadline)
	var chunk []byte
	for time.Now().Before(deadline) {
		out, err := h.terminals.ReadAvailable(sessionID, 4096)
		if err != nil {
			break
		}
		if len(out) > 0 {
			chunk = append(chunk, out...)
		}
		time.Sleep(readInterval)
	}

	return h.appendTerminalOutput(deviceID, string(chunk))
}

func (h *StreamHandler) terminalReadSettings() (time.Duration, time.Duration) {
	readDeadline := h.terminalReadDeadline
	readInterval := h.terminalReadInterval

	if readDeadline <= 0 {
		readDeadline = defaultTerminalReadDeadline
	}
	if readInterval <= 0 {
		readInterval = defaultTerminalReadInterval
	}
	if readInterval > readDeadline {
		readInterval = readDeadline
	}
	return readDeadline, readInterval
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
		if strings.TrimSpace(cmd.Intent) == SystemIntentTerminalRefresh {
			if action == CommandActionStop {
				return ServerMessage{}, ErrInvalidCommandAction
			}
			return ServerMessage{
				Notification: "Terminal refresh requested",
				Data: map[string]string{
					"device_id": cmd.DeviceID,
				},
			}, nil
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
	case SystemIntentTerminalRefresh:
		targetDeviceID := strings.TrimSpace(parsed.Arg)
		if targetDeviceID == "" {
			targetDeviceID = strings.TrimSpace(cmd.DeviceID)
		}
		if targetDeviceID == "" {
			return ServerMessage{}, ErrMissingCommandDeviceID
		}
		return ServerMessage{
			Notification: "System query: terminal_refresh",
			Data: map[string]string{
				"device_id": targetDeviceID,
			},
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
