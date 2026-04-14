package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/recording"
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

func (h *StreamHandler) activeScenarioName(deviceID string) string {
	if h.runtime == nil || h.runtime.Engine == nil {
		return ""
	}
	name, ok := h.runtime.Engine.Active(strings.TrimSpace(deviceID))
	if !ok {
		return ""
	}
	return name
}

func (h *StreamHandler) captureMultiWindowResume(deviceID, priorScenario string) {
	deviceID = strings.TrimSpace(deviceID)
	priorScenario = strings.TrimSpace(priorScenario)
	if deviceID == "" || priorScenario == "multi_window" {
		return
	}

	h.mu.Lock()
	if _, exists := h.multiWindowResume[deviceID]; exists {
		h.mu.Unlock()
		return
	}
	storedUI, hasUI := h.lastSetUIByDevice[deviceID]
	h.multiWindowResume[deviceID] = multiWindowResumeState{
		PriorScenario: priorScenario,
		PriorUI:       storedUI,
		HasPriorUI:    hasUI,
	}
	h.mu.Unlock()
}

func (h *StreamHandler) restoreMultiWindowResume(deviceID string) (*ui.Descriptor, *UITransition, bool) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, nil, false
	}

	h.mu.Lock()
	state, exists := h.multiWindowResume[deviceID]
	if exists {
		delete(h.multiWindowResume, deviceID)
	}
	h.mu.Unlock()
	if !exists {
		return nil, nil, false
	}

	var restoredUI *ui.Descriptor
	if state.HasPriorUI {
		copyUI := state.PriorUI
		restoredUI = &copyUI
	}

	var restoredTransition *UITransition
	if transition, ok := enterTransitionForScenario(state.PriorScenario); ok {
		copyTransition := transition
		restoredTransition = &copyTransition
	}
	return restoredUI, restoredTransition, true
}

func enterTransitionForScenario(name string) (UITransition, bool) {
	switch strings.TrimSpace(name) {
	case "terminal":
		return UITransition{Transition: "terminal_enter", DurationMS: 220}, true
	case "photo_frame":
		return UITransition{Transition: "photo_frame_enter", DurationMS: 220}, true
	default:
		return UITransition{}, false
	}
}

func defaultPhotoFrameSlides() []string {
	return []string{
		"https://picsum.photos/id/1015/1920/1080",
		"https://picsum.photos/id/1016/1920/1080",
		"https://picsum.photos/id/1025/1920/1080",
		"https://picsum.photos/id/1035/1920/1080",
		"https://picsum.photos/id/1043/1920/1080",
	}
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

// WebRTCSignalRequest carries client-originated WebRTC signaling payloads.
type WebRTCSignalRequest struct {
	StreamID   string
	SignalType string
	Payload    string
}

// WebRTCSignalResponse carries server-originated WebRTC signaling payloads.
type WebRTCSignalResponse struct {
	StreamID   string
	SignalType string
	Payload    string
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
	Arguments map[string]string
}

// VoiceAudioRequest carries a chunk of raw microphone audio from a device.
// Chunks are accumulated per device; on IsFinal the server runs STT on the
// assembled buffer and drives the voice command pipeline.
type VoiceAudioRequest struct {
	DeviceID   string
	Audio      []byte
	SampleRate int32
	IsFinal    bool
}

// PlayAudioResponse instructs a specific device to play synthesized audio.
type PlayAudioResponse struct {
	RequestID string
	DeviceID  string
	Audio     []byte
	Format    string
}

// ClientMessage is a one-of control stream message from client to server.
type ClientMessage struct {
	Register        *RegisterRequest
	Capability      *CapabilityUpdateRequest
	Heartbeat       *HeartbeatRequest
	Sensor          *SensorDataRequest
	StreamReady     *StreamReadyRequest
	WebRTCSignal    *WebRTCSignalRequest
	Input           *InputRequest
	Command         *CommandRequest
	VoiceAudio      *VoiceAudioRequest
	SessionDeviceID string
}

// ServerMessage is a one-of control stream message from server to client.
type ServerMessage struct {
	RegisterAck     *RegisterResponse
	CommandAck      string
	SetUI           *ui.Descriptor
	UpdateUI        *UIUpdate
	StartStream     *StartStreamResponse
	StopStream      *StopStreamResponse
	RouteStream     *RouteStreamResponse
	WebRTCSignal    *WebRTCSignalResponse
	TransitionUI    *UITransition
	PlayAudio       *PlayAudioResponse
	Notification    string
	ScenarioStart   string
	ScenarioStop    string
	Data            map[string]string
	ErrorCode       string
	Error           string
	RelayToDeviceID string
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

// DeviceAudioPublisher receives live mic-audio chunks keyed by device id so
// scenarios subscribed via scenario.Environment.DeviceAudio can analyze the
// live stream alongside any voice-command pipeline already consuming the
// buffered audio.
type DeviceAudioPublisher interface {
	Publish(deviceID string, chunk []byte)
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
	lastSetUIByDevice      map[string]ui.Descriptor
	multiWindowResume      map[string]multiWindowResumeState
	photoFrameSlides       []string
	photoFrameIndexByDev   map[string]int
	photoFrameLastByDev    map[string]time.Time
	photoFrameInterval     time.Duration

	mediaStreams      map[string]mediaStreamState
	sensorsByDevice   map[string]sensorSnapshot
	voiceAudioBuffers map[string][]byte

	deviceAudio DeviceAudioPublisher
	recording   recording.Manager
}

type mediaStreamState struct {
	StreamID       string
	Kind           string
	SourceDeviceID string
	TargetDeviceID string
	Metadata       map[string]string
	Ready          bool
}

type multiWindowResumeState struct {
	PriorScenario string
	PriorUI       ui.Descriptor
	HasPriorUI    bool
}

type sensorSnapshot struct {
	UnixMS int64
	Values map[string]float64
}

const (
	defaultTerminalReadDeadline = 180 * time.Millisecond
	defaultTerminalReadInterval = 10 * time.Millisecond
	defaultTerminalUIInterval   = 800 * time.Millisecond
	defaultPhotoFrameInterval   = 12 * time.Second
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
		lastSetUIByDevice:      map[string]ui.Descriptor{},
		multiWindowResume:      map[string]multiWindowResumeState{},
		photoFrameSlides:       defaultPhotoFrameSlides(),
		photoFrameIndexByDev:   map[string]int{},
		photoFrameLastByDev:    map[string]time.Time{},
		photoFrameInterval:     defaultPhotoFrameInterval,
		mediaStreams:           map[string]mediaStreamState{},
		sensorsByDevice:        map[string]sensorSnapshot{},
		voiceAudioBuffers:      map[string][]byte{},
		recording:              recording.NoopManager{},
	}
}

// SetDeviceAudioPublisher wires a live audio publisher so incoming VoiceAudio
// chunks are fanned out to scenarios that need to analyze the device's
// mic stream in real time. Safe to call once before any control streams are
// handled; subsequent calls replace the publisher.
func (h *StreamHandler) SetDeviceAudioPublisher(pub DeviceAudioPublisher) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.deviceAudio = pub
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
		lastSetUIByDevice:      map[string]ui.Descriptor{},
		multiWindowResume:      map[string]multiWindowResumeState{},
		photoFrameSlides:       defaultPhotoFrameSlides(),
		photoFrameIndexByDev:   map[string]int{},
		photoFrameLastByDev:    map[string]time.Time{},
		photoFrameInterval:     defaultPhotoFrameInterval,
		mediaStreams:           map[string]mediaStreamState{},
		sensorsByDevice:        map[string]sensorSnapshot{},
		voiceAudioBuffers:      map[string][]byte{},
		recording:              recording.NoopManager{},
	}
}

// SetRecordingManager wires stream recording lifecycle hooks used when routes
// start and stop. Passing nil restores the no-op manager.
func (h *StreamHandler) SetRecordingManager(mgr recording.Manager) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if mgr == nil {
		h.recording = recording.NoopManager{}
		return
	}
	h.recording = mgr
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
		out := []ServerMessage{
			{RegisterAck: &resp},
			{SetUI: &resp.Initial},
		}
		h.rememberSetUI(msg.Register.DeviceID, out)
		return out, nil
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
		out := make([]ServerMessage, 0, 2)
		update, pollErr := h.pollTerminalOutput(msg.Heartbeat.DeviceID, false)
		if pollErr != nil {
			h.metrics.protocolErrors.Add(1)
			return []ServerMessage{{ErrorCode: errorCodeFor(pollErr), Error: pollErr.Error()}}, pollErr
		}
		if update != nil {
			out = append(out, *update)
		}
		if photoRotate := h.photoFrameHeartbeatUpdate(msg.Heartbeat.DeviceID); photoRotate != nil {
			out = append(out, *photoRotate)
		}
		if len(out) > 0 {
			return out, nil
		}
		return nil, nil
	case msg.Sensor != nil:
		h.metrics.sensorReceived.Add(1)
		beforeBroadcastEvents := h.broadcastEventCount()
		h.recordSensorData(msg.Sensor)
		if h.runtime != nil {
			values := map[string]float64{}
			for key, value := range msg.Sensor.Values {
				values[key] = value
			}
			if err := h.runtime.ProcessSensorReading(ctx, scenario.SensorReading{
				DeviceID: strings.TrimSpace(msg.Sensor.DeviceID),
				UnixMS:   msg.Sensor.UnixMS,
				Values:   values,
			}); err != nil {
				h.metrics.protocolErrors.Add(1)
				return []ServerMessage{{ErrorCode: errorCodeFor(err), Error: err.Error()}}, err
			}
		}
		out := h.broadcastNotificationsSince(beforeBroadcastEvents, msg.Sensor.DeviceID, true)
		if len(out) == 0 {
			return nil, nil
		}
		return out, nil
	case msg.StreamReady != nil:
		h.metrics.streamReadyReceived.Add(1)
		h.markStreamReady(msg.StreamReady.StreamID)
		return nil, nil
	case msg.WebRTCSignal != nil:
		h.metrics.webrtcSignalReceived.Add(1)
		return h.relayWebRTCSignal(msg.WebRTCSignal, msg.SessionDeviceID), nil
	case msg.VoiceAudio != nil:
		h.metrics.voiceAudioReceived.Add(1)
		out, err := h.handleVoiceAudio(ctx, msg.VoiceAudio)
		if err != nil {
			h.metrics.protocolErrors.Add(1)
			return []ServerMessage{{ErrorCode: errorCodeFor(err), Error: err.Error()}}, err
		}
		return out, nil
	case msg.Input != nil:
		out, err := h.handleInput(ctx, msg.Input)
		if err != nil {
			h.metrics.protocolErrors.Add(1)
			return []ServerMessage{{ErrorCode: errorCodeFor(err), Error: err.Error()}}, err
		}
		h.rememberSetUI(msg.Input.DeviceID, out)
		return out, nil
	case msg.Command != nil:
		h.metrics.commandReceived.Add(1)
		priorActiveScenario := h.activeScenarioName(msg.Command.DeviceID)
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
		beforeBroadcastEvents := h.broadcastEventCount()
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
		if commandResult.ScenarioStart == "multi_window" && defaultAction(msg.Command.Action) == CommandActionStart {
			h.captureMultiWindowResume(msg.Command.DeviceID, priorActiveScenario)
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
		postResponses := h.commandResponses(ctx, msg.Command, commandResult)
		afterRoutes := h.routeSnapshotForDevice(msg.Command.DeviceID)
		routeUpdates := h.routeUpdatesForCommand(msg.Command, commandResult, beforeRoutes, afterRoutes)
		if len(routeUpdates) > 0 {
			postResponses = append(postResponses, routeUpdates...)
		}
		paTransitions := h.paTransitionsForCommand(msg.Command, commandResult, beforeRoutes, afterRoutes)
		if len(paTransitions) > 0 {
			postResponses = append(postResponses, paTransitions...)
		}
		overlayClears := h.paOverlayClearsForCommand(msg.Command, commandResult, beforeRoutes)
		if len(overlayClears) > 0 {
			postResponses = append(postResponses, overlayClears...)
		}
		broadcastNotifications := h.broadcastNotificationsForCommand(msg.Command, commandResult, beforeBroadcastEvents)
		if len(broadcastNotifications) > 0 {
			postResponses = append(postResponses, broadcastNotifications...)
		}
		h.rememberSetUI(msg.Command.DeviceID, postResponses)
		return postResponses, nil
	default:
		h.metrics.protocolErrors.Add(1)
		return []ServerMessage{{ErrorCode: errorCodeFor(ErrInvalidClientMessage), Error: ErrInvalidClientMessage.Error()}}, ErrInvalidClientMessage
	}
}

func (h *StreamHandler) broadcastEventCount() int {
	if h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.Broadcast == nil {
		return 0
	}
	eventReader, ok := h.runtime.Env.Broadcast.(interface {
		Events() []ui.BroadcastEvent
	})
	if !ok {
		return 0
	}
	return len(eventReader.Events())
}

func (h *StreamHandler) broadcastNotificationsForCommand(
	cmd *CommandRequest,
	commandResult ServerMessage,
	beforeCount int,
) []ServerMessage {
	if cmd == nil {
		return nil
	}
	if commandResult.ScenarioStart == "" && commandResult.ScenarioStop == "" {
		return nil
	}
	return h.broadcastNotificationsSince(beforeCount, cmd.DeviceID, false)
}

func (h *StreamHandler) broadcastNotificationsSince(
	beforeCount int,
	sessionDeviceID string,
	includeSession bool,
) []ServerMessage {
	if h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.Broadcast == nil {
		return nil
	}
	eventReader, ok := h.runtime.Env.Broadcast.(interface {
		Events() []ui.BroadcastEvent
	})
	if !ok {
		return nil
	}
	events := eventReader.Events()
	if beforeCount < 0 {
		beforeCount = 0
	}
	if beforeCount > len(events) {
		beforeCount = len(events)
	}
	newEvents := events[beforeCount:]
	if len(newEvents) == 0 {
		return nil
	}

	trimmedSessionDeviceID := strings.TrimSpace(sessionDeviceID)
	out := make([]ServerMessage, 0, len(newEvents))
	for _, event := range newEvents {
		if len(event.DeviceIDs) == 0 {
			continue
		}
		for _, targetDeviceID := range event.DeviceIDs {
			targetDeviceID = strings.TrimSpace(targetDeviceID)
			if targetDeviceID == "" {
				continue
			}
			if targetDeviceID == trimmedSessionDeviceID && !includeSession {
				continue
			}
			msg := ServerMessage{
				Notification: event.Message,
			}
			if targetDeviceID != trimmedSessionDeviceID {
				msg.RelayToDeviceID = targetDeviceID
			}
			out = append(out, msg)

			if strings.HasPrefix(event.Message, "PA from ") {
				overlayMsg := ServerMessage{
					UpdateUI: &UIUpdate{
						ComponentID: ui.GlobalOverlayComponentID,
						Node:        ui.PAReceiverOverlayPatch(event.Message),
					},
				}
				if targetDeviceID != trimmedSessionDeviceID {
					overlayMsg.RelayToDeviceID = targetDeviceID
				}
				out = append(out, overlayMsg)
			}
		}
	}
	return out
}

func (h *StreamHandler) paOverlayClearsForCommand(
	cmd *CommandRequest,
	commandResult ServerMessage,
	beforeRoutes []iorouter.Route,
) []ServerMessage {
	if cmd == nil {
		return nil
	}
	if commandResult.ScenarioStop != "pa_system" {
		return nil
	}
	if defaultAction(cmd.Action) != CommandActionStop {
		return nil
	}
	sessionDeviceID := strings.TrimSpace(cmd.DeviceID)
	targets := map[string]struct{}{}
	for _, route := range beforeRoutes {
		if route.StreamKind != "pa_audio" {
			continue
		}
		if strings.TrimSpace(route.SourceID) != sessionDeviceID {
			continue
		}
		targetID := strings.TrimSpace(route.TargetID)
		if targetID == "" || targetID == sessionDeviceID {
			continue
		}
		targets[targetID] = struct{}{}
	}
	if len(targets) == 0 {
		return nil
	}

	out := make([]ServerMessage, 0, len(targets))
	for targetID := range targets {
		out = append(out, ServerMessage{
			UpdateUI: &UIUpdate{
				ComponentID: ui.GlobalOverlayComponentID,
				Node:        ui.GlobalOverlaySlot(),
			},
			RelayToDeviceID: targetID,
		})
	}
	return out
}

func (h *StreamHandler) paTransitionsForCommand(
	cmd *CommandRequest,
	commandResult ServerMessage,
	beforeRoutes []iorouter.Route,
	afterRoutes []iorouter.Route,
) []ServerMessage {
	if cmd == nil {
		return nil
	}
	sourceID := strings.TrimSpace(cmd.DeviceID)
	if sourceID == "" {
		return nil
	}

	if commandResult.ScenarioStart == "pa_system" && defaultAction(cmd.Action) == CommandActionStart {
		targets := paTargetsFromRoutes(afterRoutes, sourceID)
		return paTransitionMessages(targets, "pa_source_enter", "pa_receive_enter")
	}
	if commandResult.ScenarioStop == "pa_system" && defaultAction(cmd.Action) == CommandActionStop {
		targets := paTargetsFromRoutes(beforeRoutes, sourceID)
		return paTransitionMessages(targets, "pa_source_exit", "pa_receive_exit")
	}
	return nil
}

func paTargetsFromRoutes(routes []iorouter.Route, sourceID string) []string {
	set := map[string]struct{}{}
	for _, route := range routes {
		if route.StreamKind != "pa_audio" {
			continue
		}
		if strings.TrimSpace(route.SourceID) != sourceID {
			continue
		}
		targetID := strings.TrimSpace(route.TargetID)
		if targetID == "" || targetID == sourceID {
			continue
		}
		set[targetID] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for targetID := range set {
		out = append(out, targetID)
	}
	sort.Strings(out)
	return out
}

func paTransitionMessages(
	targetIDs []string,
	sourceTransition string,
	receiverTransition string,
) []ServerMessage {
	out := make([]ServerMessage, 0, len(targetIDs)+1)
	out = append(out, ServerMessage{
		TransitionUI: &UITransition{
			Transition: sourceTransition,
			DurationMS: 180,
		},
	})
	for _, targetID := range targetIDs {
		out = append(out, ServerMessage{
			TransitionUI: &UITransition{
				Transition: receiverTransition,
				DurationMS: 180,
			},
			RelayToDeviceID: targetID,
		})
	}
	return out
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
		if restored := h.resumedScenarioUI(ctx, cmd.DeviceID, "terminal"); len(restored) > 0 {
			responses = append(responses, restored...)
		}
		return responses
	}
	if commandResult.ScenarioStop == "photo_frame" {
		for _, deviceID := range h.commandTargetDeviceIDs(cmd) {
			h.clearPhotoFrameState(deviceID)
		}
		responses = append(responses, ServerMessage{
			TransitionUI: &UITransition{
				Transition: "photo_frame_exit",
				DurationMS: 220,
			},
		})
		for _, targetDeviceID := range h.commandTargetDeviceIDs(cmd) {
			targetDeviceID = strings.TrimSpace(targetDeviceID)
			if targetDeviceID == "" || targetDeviceID == strings.TrimSpace(cmd.DeviceID) {
				continue
			}
			responses = append(responses, ServerMessage{
				TransitionUI: &UITransition{
					Transition: "photo_frame_exit",
					DurationMS: 220,
				},
				RelayToDeviceID: targetDeviceID,
			})
		}
		if restored := h.resumedScenarioUI(ctx, cmd.DeviceID, "photo_frame"); len(restored) > 0 {
			responses = append(responses, restored...)
		}
		return responses
	}
	if commandResult.ScenarioStop == "internal_video_call" {
		responses = append(responses, ServerMessage{
			TransitionUI: &UITransition{
				Transition: "internal_video_call_exit",
				DurationMS: 220,
			},
		})
		if restored := h.resumedScenarioUI(ctx, cmd.DeviceID, "internal_video_call"); len(restored) > 0 {
			responses = append(responses, restored...)
		}
		return responses
	}
	if commandResult.ScenarioStop == "multi_window" {
		if restoredUI, restoredTransition, ok := h.restoreMultiWindowResume(cmd.DeviceID); ok {
			if restoredUI != nil {
				responses = append(responses, ServerMessage{SetUI: restoredUI})
			}
			if restoredTransition != nil {
				responses = append(responses, ServerMessage{TransitionUI: restoredTransition})
			}
		}
		return responses
	}
	if commandResult.ScenarioStart == "multi_window" {
		peerIDs, focusedPeerID := h.multiWindowPeersAndFocus(cmd.DeviceID)
		multiWindowUI := ui.MultiWindowView(cmd.DeviceID, peerIDs, focusedPeerID)
		responses = append(responses, ServerMessage{SetUI: &multiWindowUI})
	}
	if commandResult.ScenarioStart == "photo_frame" {
		photoFrameUI := h.photoFrameSetUI(cmd.DeviceID, true)
		responses = append(responses, ServerMessage{SetUI: &photoFrameUI})
		responses = append(responses, ServerMessage{
			TransitionUI: &UITransition{
				Transition: "photo_frame_enter",
				DurationMS: 220,
			},
		})
		for _, targetDeviceID := range h.commandTargetDeviceIDs(cmd) {
			targetDeviceID = strings.TrimSpace(targetDeviceID)
			if targetDeviceID == "" || targetDeviceID == strings.TrimSpace(cmd.DeviceID) {
				continue
			}
			peerUI := h.photoFrameSetUI(targetDeviceID, true)
			responses = append(responses, ServerMessage{
				SetUI:           &peerUI,
				RelayToDeviceID: targetDeviceID,
			})
			responses = append(responses, ServerMessage{
				TransitionUI: &UITransition{
					Transition: "photo_frame_enter",
					DurationMS: 220,
				},
				RelayToDeviceID: targetDeviceID,
			})
		}
		return responses
	}
	if commandResult.ScenarioStart == "internal_video_call" {
		if peerID, ok := h.internalVideoCallPeer(cmd.DeviceID); ok {
			internalVideoCallUI := ui.InternalVideoCallView(cmd.DeviceID, peerID)
			responses = append(responses, ServerMessage{SetUI: &internalVideoCallUI})
		}
		responses = append(responses, ServerMessage{
			TransitionUI: &UITransition{
				Transition: "internal_video_call_enter",
				DurationMS: 220,
			},
		})
		return responses
	}
	if cmd.Action != "" && cmd.Action != CommandActionStart {
		if commandResult.ScenarioStop != "" {
			if restored := h.resumedScenarioUI(ctx, cmd.DeviceID, commandResult.ScenarioStop); len(restored) > 0 {
				responses = append(responses, restored...)
			}
		}
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
			startMsg := ServerMessage{
				StartStream: &StartStreamResponse{
					StreamID:       routeID,
					Kind:           route.StreamKind,
					SourceDeviceID: route.SourceID,
					TargetDeviceID: route.TargetID,
					Metadata: map[string]string{
						"origin": "route_delta",
					},
				},
			}
			out = h.appendRouteMessageForPeers(out, cmd.DeviceID, route.SourceID, route.TargetID, startMsg)
			h.registerMediaStream(StartStreamResponse{
				StreamID:       routeID,
				Kind:           route.StreamKind,
				SourceDeviceID: route.SourceID,
				TargetDeviceID: route.TargetID,
				Metadata: map[string]string{
					"origin": "route_delta",
				},
			})
			routeMsg := ServerMessage{
				RouteStream: &RouteStreamResponse{
					StreamID:       routeID,
					SourceDeviceID: route.SourceID,
					TargetDeviceID: route.TargetID,
					Kind:           route.StreamKind,
				},
			}
			out = h.appendRouteMessageForPeers(out, cmd.DeviceID, route.SourceID, route.TargetID, routeMsg)
		}
		for _, route := range before {
			routeID := routeStreamID(route)
			if _, exists := afterSet[routeID]; exists {
				continue
			}
			stopMsg := ServerMessage{
				StopStream: &StopStreamResponse{
					StreamID: routeID,
				},
			}
			out = h.appendRouteMessageForPeers(out, cmd.DeviceID, route.SourceID, route.TargetID, stopMsg)
			h.unregisterMediaStream(routeID)
		}
	}
	if action == CommandActionStop {
		for _, route := range before {
			routeID := routeStreamID(route)
			if _, exists := afterSet[routeID]; exists {
				continue
			}
			stopMsg := ServerMessage{
				StopStream: &StopStreamResponse{
					StreamID: routeID,
				},
			}
			out = h.appendRouteMessageForPeers(out, cmd.DeviceID, route.SourceID, route.TargetID, stopMsg)
			h.unregisterMediaStream(routeID)
		}
	}
	return out
}

func (h *StreamHandler) resumedScenarioUI(ctx context.Context, deviceID, stoppedScenario string) []ServerMessage {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" || h.runtime == nil || h.runtime.Engine == nil {
		return nil
	}
	resumedName, active := h.runtime.Engine.Active(deviceID)
	if !active || resumedName == "" || resumedName == strings.TrimSpace(stoppedScenario) {
		return nil
	}

	switch resumedName {
	case "photo_frame":
		view := h.photoFrameSetUI(deviceID, false)
		return []ServerMessage{
			{SetUI: &view},
			{TransitionUI: &UITransition{Transition: "photo_frame_enter", DurationMS: 220}},
		}
	case "terminal":
		output, err := h.ensureTerminalSession(ctx, deviceID)
		if err != nil {
			return []ServerMessage{{Notification: "Terminal session failed: " + err.Error()}}
		}
		view := ui.TerminalViewWithOutput(deviceID, output)
		return []ServerMessage{
			{SetUI: &view},
			{TransitionUI: &UITransition{Transition: "terminal_enter", DurationMS: 220}},
		}
	default:
		return nil
	}
}

func (h *StreamHandler) clearPhotoFrameState(deviceID string) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return
	}
	h.mu.Lock()
	delete(h.photoFrameIndexByDev, deviceID)
	delete(h.photoFrameLastByDev, deviceID)
	h.mu.Unlock()
}

func (h *StreamHandler) photoFrameSetUI(deviceID string, reset bool) ui.Descriptor {
	deviceID = strings.TrimSpace(deviceID)
	now := h.nowUTC()

	h.mu.Lock()
	slides := append([]string(nil), h.photoFrameSlides...)
	if len(slides) == 0 {
		slides = defaultPhotoFrameSlides()
	}
	index := h.photoFrameIndexByDev[deviceID]
	if reset {
		index = 0
	}
	if index < 0 || index >= len(slides) {
		index = 0
	}
	h.photoFrameIndexByDev[deviceID] = index
	h.photoFrameLastByDev[deviceID] = now
	h.mu.Unlock()

	url := slides[index]
	caption := "Photo frame: " + deviceID
	return ui.PhotoFrameView(url, caption, index, len(slides))
}

func (h *StreamHandler) photoFrameHeartbeatUpdate(deviceID string) *ServerMessage {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil
	}
	if h.activeScenarioName(deviceID) != "photo_frame" {
		return nil
	}

	now := h.nowUTC()
	h.mu.Lock()
	slides := append([]string(nil), h.photoFrameSlides...)
	if len(slides) == 0 {
		slides = defaultPhotoFrameSlides()
	}

	last, hasLast := h.photoFrameLastByDev[deviceID]
	interval := h.photoFrameInterval
	if interval <= 0 {
		interval = defaultPhotoFrameInterval
	}
	if !hasLast {
		h.photoFrameLastByDev[deviceID] = now
		h.mu.Unlock()
		return nil
	}
	if now.Sub(last) < interval {
		h.mu.Unlock()
		return nil
	}

	index := h.photoFrameIndexByDev[deviceID]
	if index < 0 || index >= len(slides) {
		index = 0
	}
	index = (index + 1) % len(slides)
	h.photoFrameIndexByDev[deviceID] = index
	h.photoFrameLastByDev[deviceID] = now
	h.mu.Unlock()

	view := ui.PhotoFrameView(slides[index], "Photo frame: "+deviceID, index, len(slides))
	return &ServerMessage{SetUI: &view}
}

func (h *StreamHandler) commandTargetDeviceIDs(cmd *CommandRequest) []string {
	if cmd == nil {
		return nil
	}

	args := cmd.Arguments
	if len(args) > 0 {
		if rawList := strings.TrimSpace(args["device_ids"]); rawList != "" {
			parts := strings.Split(rawList, ",")
			out := make([]string, 0, len(parts))
			seen := map[string]struct{}{}
			for _, part := range parts {
				deviceID := strings.TrimSpace(part)
				if deviceID == "" {
					continue
				}
				if _, exists := seen[deviceID]; exists {
					continue
				}
				seen[deviceID] = struct{}{}
				out = append(out, deviceID)
			}
			if len(out) > 0 {
				return out
			}
		}
		if one := strings.TrimSpace(args["device_id"]); one != "" {
			return []string{one}
		}
	}

	if h.runtime != nil && h.runtime.Env != nil && h.runtime.Env.Devices != nil {
		all := h.runtime.Env.Devices.ListDeviceIDs()
		if len(all) > 0 {
			return all
		}
	}

	if source := strings.TrimSpace(cmd.DeviceID); source != "" {
		return []string{source}
	}
	return nil
}

func (h *StreamHandler) appendRouteMessageForPeers(
	out []ServerMessage,
	sessionDeviceID string,
	sourceDeviceID string,
	targetDeviceID string,
	msg ServerMessage,
) []ServerMessage {
	peers := []string{}
	seen := map[string]struct{}{}
	for _, deviceID := range []string{sourceDeviceID, targetDeviceID} {
		deviceID = strings.TrimSpace(deviceID)
		if deviceID == "" {
			continue
		}
		if _, exists := seen[deviceID]; exists {
			continue
		}
		seen[deviceID] = struct{}{}
		peers = append(peers, deviceID)
	}
	sessionDeviceID = strings.TrimSpace(sessionDeviceID)
	for _, peerDeviceID := range peers {
		next := msg
		if peerDeviceID != sessionDeviceID {
			next.RelayToDeviceID = peerDeviceID
		}
		out = append(out, next)
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

func (h *StreamHandler) relayWebRTCSignal(signal *WebRTCSignalRequest, sourceDeviceID string) []ServerMessage {
	if signal == nil {
		return nil
	}
	streamID := strings.TrimSpace(signal.StreamID)
	signalType := strings.TrimSpace(signal.SignalType)
	if streamID == "" || signalType == "" {
		return nil
	}
	peerDeviceID := h.peerDeviceForStream(streamID, strings.TrimSpace(sourceDeviceID))
	if peerDeviceID == "" {
		return nil
	}
	return []ServerMessage{
		{
			WebRTCSignal: &WebRTCSignalResponse{
				StreamID:   streamID,
				SignalType: signalType,
				Payload:    signal.Payload,
			},
			RelayToDeviceID: peerDeviceID,
		},
	}
}

func (h *StreamHandler) peerDeviceForStream(streamID, sourceDeviceID string) string {
	const prefix = "route:"
	if strings.HasPrefix(streamID, prefix) {
		parts := strings.SplitN(strings.TrimPrefix(streamID, prefix), "|", 3)
		if len(parts) == 3 {
			if sourceDeviceID == parts[0] {
				return parts[1]
			}
			if sourceDeviceID == parts[1] {
				return parts[0]
			}
		}
	}

	h.mu.Lock()
	state, ok := h.mediaStreams[streamID]
	h.mu.Unlock()
	if !ok {
		return ""
	}
	if sourceDeviceID == state.SourceDeviceID {
		return state.TargetDeviceID
	}
	if sourceDeviceID == state.TargetDeviceID {
		return state.SourceDeviceID
	}
	return ""
}

func (h *StreamHandler) registerMediaStream(start StartStreamResponse) {
	streamID := strings.TrimSpace(start.StreamID)
	if streamID == "" {
		return
	}
	metadata := map[string]string{}
	for k, v := range start.Metadata {
		metadata[k] = v
	}
	h.mu.Lock()
	h.mediaStreams[streamID] = mediaStreamState{
		StreamID:       streamID,
		Kind:           start.Kind,
		SourceDeviceID: start.SourceDeviceID,
		TargetDeviceID: start.TargetDeviceID,
		Metadata:       metadata,
		Ready:          false,
	}
	recorder := h.recording
	h.mu.Unlock()
	if recorder != nil {
		_ = recorder.Start(context.Background(), recording.Stream{
			StreamID:       streamID,
			Kind:           start.Kind,
			SourceDeviceID: start.SourceDeviceID,
			TargetDeviceID: start.TargetDeviceID,
			Metadata:       metadata,
		})
	}
}

func (h *StreamHandler) unregisterMediaStream(streamID string) {
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		return
	}
	h.mu.Lock()
	delete(h.mediaStreams, streamID)
	recorder := h.recording
	h.mu.Unlock()
	if recorder != nil {
		_ = recorder.Stop(context.Background(), streamID)
	}
}

func (h *StreamHandler) markStreamReady(streamID string) {
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		return
	}
	h.mu.Lock()
	state, ok := h.mediaStreams[streamID]
	if !ok {
		state = mediaStreamState{
			StreamID: streamID,
			Kind:     "unknown",
		}
	}
	state.Ready = true
	h.mediaStreams[streamID] = state
	h.mu.Unlock()
}

func (h *StreamHandler) mediaStreamStatusData() map[string]string {
	h.mu.Lock()
	streams := make([]mediaStreamState, 0, len(h.mediaStreams))
	for _, state := range h.mediaStreams {
		streams = append(streams, state)
	}
	h.mu.Unlock()

	sort.Slice(streams, func(i, j int) bool {
		return streams[i].StreamID < streams[j].StreamID
	})

	ready := 0
	details := make([]string, 0, len(streams))
	for _, state := range streams {
		if state.Ready {
			ready++
		}
		details = append(details, fmt.Sprintf(
			"%s|%s|%s->%s|ready=%t",
			state.StreamID,
			state.Kind,
			state.SourceDeviceID,
			state.TargetDeviceID,
			state.Ready,
		))
	}

	return map[string]string{
		"media_streams_active":  strconv.Itoa(len(streams)),
		"media_streams_ready":   strconv.Itoa(ready),
		"media_streams_pending": strconv.Itoa(len(streams) - ready),
		"media_streams":         strings.Join(details, ";"),
	}
}

func (h *StreamHandler) recordSensorData(sensor *SensorDataRequest) {
	if sensor == nil {
		return
	}
	deviceID := strings.TrimSpace(sensor.DeviceID)
	if deviceID == "" {
		return
	}
	values := map[string]float64{}
	for key, value := range sensor.Values {
		values[key] = value
	}
	h.mu.Lock()
	h.sensorsByDevice[deviceID] = sensorSnapshot{
		UnixMS: sensor.UnixMS,
		Values: values,
	}
	h.mu.Unlock()
}

func (h *StreamHandler) sensorDataForDevice(deviceID string) (sensorSnapshot, bool) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return sensorSnapshot{}, false
	}
	h.mu.Lock()
	snapshot, ok := h.sensorsByDevice[deviceID]
	h.mu.Unlock()
	if !ok {
		return sensorSnapshot{}, false
	}
	values := map[string]float64{}
	for key, value := range snapshot.Values {
		values[key] = value
	}
	return sensorSnapshot{
		UnixMS: snapshot.UnixMS,
		Values: values,
	}, true
}

func (h *StreamHandler) sensorStatusData() map[string]string {
	h.mu.Lock()
	byDevice := make(map[string]sensorSnapshot, len(h.sensorsByDevice))
	for deviceID, snapshot := range h.sensorsByDevice {
		values := map[string]float64{}
		for key, value := range snapshot.Values {
			values[key] = value
		}
		byDevice[deviceID] = sensorSnapshot{
			UnixMS: snapshot.UnixMS,
			Values: values,
		}
	}
	h.mu.Unlock()

	deviceIDs := make([]string, 0, len(byDevice))
	latestUnixMS := int64(0)
	details := make([]string, 0, len(byDevice))
	for deviceID, snapshot := range byDevice {
		deviceIDs = append(deviceIDs, deviceID)
		if snapshot.UnixMS > latestUnixMS {
			latestUnixMS = snapshot.UnixMS
		}
		keys := make([]string, 0, len(snapshot.Values))
		for key := range snapshot.Values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		details = append(details, fmt.Sprintf(
			"%s|unix_ms=%d|keys=%s",
			deviceID,
			snapshot.UnixMS,
			strings.Join(keys, ","),
		))
	}
	sort.Strings(deviceIDs)
	sort.Strings(details)

	return map[string]string{
		"sensor_devices_reporting": strconv.Itoa(len(deviceIDs)),
		"sensor_latest_unix_ms":    strconv.FormatInt(latestUnixMS, 10),
		"sensor_device_ids":        strings.Join(deviceIDs, ","),
		"sensor_summaries":         strings.Join(details, ";"),
	}
}

func (h *StreamHandler) recordingStatusData() map[string]string {
	h.mu.Lock()
	recorder := h.recording
	h.mu.Unlock()
	activeReader, ok := recorder.(interface {
		Active() map[string]recording.Stream
	})
	if !ok {
		return map[string]string{
			"recording_active_streams": "0",
			"recording_stream_ids":     "",
		}
	}
	active := activeReader.Active()
	streamIDs := make([]string, 0, len(active))
	for streamID := range active {
		streamIDs = append(streamIDs, streamID)
	}
	sort.Strings(streamIDs)
	return map[string]string{
		"recording_active_streams": strconv.Itoa(len(streamIDs)),
		"recording_stream_ids":     strings.Join(streamIDs, ","),
	}
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
		return route.StreamKind == "audio" && (route.SourceID == deviceID || route.TargetID == deviceID)
	case "internal_video_call":
		if route.StreamKind != "audio" && route.StreamKind != "video" {
			return false
		}
		return route.SourceID == deviceID || route.TargetID == deviceID
	case "pa_system":
		return route.SourceID == deviceID && route.StreamKind == "pa_audio"
	case "multi_window":
		if route.TargetID != deviceID {
			return false
		}
		return route.StreamKind == "video" || route.StreamKind == "audio_mix" || route.StreamKind == "audio"
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
	beforeRoutes := h.routeSnapshotForDevice(deviceID)
	beforeBroadcastEvents := h.broadcastEventCount()

	intent := ""
	commandAction := CommandActionStart
	triggerArgs := map[string]string{
		"device_id": deviceID,
	}
	switch {
	case action == "stop_active":
		intent = activeName
		commandAction = CommandActionStop
	case action == "internal_video_call_end":
		if activeName != "internal_video_call" {
			return nil, false, nil
		}
		intent = "internal_video_call"
		commandAction = CommandActionStop
	case action == "multi_window_end":
		if activeName != "multi_window" {
			return nil, false, nil
		}
		intent = "multi_window"
		commandAction = CommandActionStop
	case strings.HasPrefix(action, "multi_window_focus:"):
		if activeName != "multi_window" {
			return nil, false, nil
		}
		focusDeviceID := strings.TrimSpace(strings.TrimPrefix(action, "multi_window_focus:"))
		if focusDeviceID == "" {
			return nil, true, nil
		}
		intent = "multi_window"
		triggerArgs["audio_focus_device_id"] = focusDeviceID
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
		Kind:      scenario.TriggerManual,
		SourceID:  deviceID,
		Intent:    intent,
		Arguments: triggerArgs,
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
			DeviceID:  deviceID,
			Action:    commandAction,
			Kind:      CommandKindManual,
			Intent:    intent,
			Arguments: copyStringMap(triggerArgs),
		}
		responses := h.commandResponses(ctx, cmd, result)
		afterRoutes := h.routeSnapshotForDevice(deviceID)
		routeUpdates := h.routeUpdatesForCommand(cmd, result, beforeRoutes, afterRoutes)
		if len(routeUpdates) > 0 {
			responses = append(responses, routeUpdates...)
		}
		paTransitions := h.paTransitionsForCommand(cmd, result, beforeRoutes, afterRoutes)
		if len(paTransitions) > 0 {
			responses = append(responses, paTransitions...)
		}
		overlayClears := h.paOverlayClearsForCommand(cmd, result, beforeRoutes)
		if len(overlayClears) > 0 {
			responses = append(responses, overlayClears...)
		}
		broadcastNotifications := h.broadcastNotificationsForCommand(cmd, result, beforeBroadcastEvents)
		if len(broadcastNotifications) > 0 {
			responses = append(responses, broadcastNotifications...)
		}
		return responses, true, nil
	}
	name, err := h.runtime.HandleTrigger(ctx, trigger)
	if err != nil {
		return nil, true, err
	}
	result := ServerMessage{
		ScenarioStart: name,
		Notification:  "Scenario started: " + name,
	}
	if result.ScenarioStart == "multi_window" && commandAction == CommandActionStart {
		h.captureMultiWindowResume(deviceID, activeName)
	}
	cmd := &CommandRequest{
		DeviceID:  deviceID,
		Action:    commandAction,
		Kind:      CommandKindManual,
		Intent:    intent,
		Arguments: copyStringMap(triggerArgs),
	}
	responses := h.commandResponses(ctx, cmd, result)
	afterRoutes := h.routeSnapshotForDevice(deviceID)
	routeUpdates := h.routeUpdatesForCommand(cmd, result, beforeRoutes, afterRoutes)
	if len(routeUpdates) > 0 {
		responses = append(responses, routeUpdates...)
	}
	paTransitions := h.paTransitionsForCommand(cmd, result, beforeRoutes, afterRoutes)
	if len(paTransitions) > 0 {
		responses = append(responses, paTransitions...)
	}
	overlayClears := h.paOverlayClearsForCommand(cmd, result, beforeRoutes)
	if len(overlayClears) > 0 {
		responses = append(responses, overlayClears...)
	}
	broadcastNotifications := h.broadcastNotificationsForCommand(cmd, result, beforeBroadcastEvents)
	if len(broadcastNotifications) > 0 {
		responses = append(responses, broadcastNotifications...)
	}
	return responses, true, nil
}

func (h *StreamHandler) multiWindowPeersAndFocus(deviceID string) ([]string, string) {
	routes := h.routeSnapshotForDevice(deviceID)
	peers := make([]string, 0)
	seenPeers := map[string]struct{}{}
	focusedPeerID := ""
	for _, route := range routes {
		if strings.TrimSpace(route.TargetID) != strings.TrimSpace(deviceID) {
			continue
		}
		sourceID := strings.TrimSpace(route.SourceID)
		if sourceID == "" || sourceID == strings.TrimSpace(deviceID) {
			continue
		}
		if route.StreamKind == "video" {
			if _, exists := seenPeers[sourceID]; !exists {
				seenPeers[sourceID] = struct{}{}
				peers = append(peers, sourceID)
			}
		}
		if route.StreamKind == "audio" {
			focusedPeerID = sourceID
		}
	}
	sort.Strings(peers)
	return peers, focusedPeerID
}

func (h *StreamHandler) internalVideoCallPeer(deviceID string) (string, bool) {
	selfID := strings.TrimSpace(deviceID)
	if selfID == "" {
		return "", false
	}
	routes := h.routeSnapshotForDevice(selfID)
	for _, route := range routes {
		if route.StreamKind != "video" {
			continue
		}
		sourceID := strings.TrimSpace(route.SourceID)
		targetID := strings.TrimSpace(route.TargetID)
		if sourceID == selfID && targetID != "" {
			return targetID, true
		}
		if targetID == selfID && sourceID != "" {
			return sourceID, true
		}
	}
	return "", false
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

func (h *StreamHandler) rememberSetUI(deviceID string, responses []ServerMessage) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" || len(responses) == 0 {
		return
	}

	for _, response := range responses {
		if response.SetUI == nil {
			continue
		}
		if relayTarget := strings.TrimSpace(response.RelayToDeviceID); relayTarget != "" {
			continue
		}
		h.mu.Lock()
		h.lastSetUIByDevice[deviceID] = *response.SetUI
		h.mu.Unlock()
	}
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

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
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

// handleVoiceAudio accumulates inbound mic audio per device and, on IsFinal,
// runs STT on the assembled buffer, drives the voice command pipeline through
// Runtime.HandleVoiceText, then synthesizes the resulting response via TTS and
// returns it as a PlayAudio server message targeted at the source device.
func (h *StreamHandler) handleVoiceAudio(ctx context.Context, va *VoiceAudioRequest) ([]ServerMessage, error) {
	if va == nil {
		return nil, ErrInvalidClientMessage
	}
	deviceID := strings.TrimSpace(va.DeviceID)
	if deviceID == "" {
		return nil, ErrMissingCommandDeviceID
	}

	h.mu.Lock()
	existing := h.voiceAudioBuffers[deviceID]
	buf := make([]byte, 0, len(existing)+len(va.Audio))
	buf = append(buf, existing...)
	buf = append(buf, va.Audio...)
	publisher := h.deviceAudio
	recorder := h.recording
	if !va.IsFinal {
		h.voiceAudioBuffers[deviceID] = buf
		h.mu.Unlock()
		if publisher != nil && len(va.Audio) > 0 {
			publisher.Publish(deviceID, va.Audio)
		}
		if len(va.Audio) > 0 {
			h.recordVoiceAudioChunk(recorder, deviceID, va.Audio)
		}
		return nil, nil
	}
	delete(h.voiceAudioBuffers, deviceID)
	h.mu.Unlock()

	if publisher != nil && len(va.Audio) > 0 {
		publisher.Publish(deviceID, va.Audio)
	}
	if len(va.Audio) > 0 {
		h.recordVoiceAudioChunk(recorder, deviceID, va.Audio)
	}

	if h.runtime == nil || h.runtime.Env == nil {
		return nil, errors.New("scenario runtime not configured")
	}
	if h.runtime.Env.STT == nil {
		return nil, errors.New("speech-to-text backend not configured")
	}

	source := &voiceAudioReader{buf: buf}
	transcripts, err := h.runtime.Env.STT.Transcribe(ctx, source)
	if err != nil {
		return nil, err
	}
	var spoken string
	for tr := range transcripts {
		if tr.IsFinal && tr.Text != "" {
			spoken = tr.Text
		} else if spoken == "" && tr.Text != "" {
			spoken = tr.Text
		}
	}
	spoken = strings.TrimSpace(spoken)
	if spoken == "" {
		return nil, ErrMissingCommandText
	}
	if h.runtime.Env.WakeWord != nil {
		detection, err := h.runtime.Env.WakeWord.Detect(ctx, spoken)
		if err != nil {
			return nil, err
		}
		if !detection.Detected {
			return nil, nil
		}
		if normalized := strings.TrimSpace(detection.Command); normalized != "" {
			spoken = normalized
		}
	}

	beforeCount := h.broadcastEventCount()
	scenarioName, err := h.runtime.HandleVoiceText(ctx, deviceID, spoken, h.control.now().UTC())
	if err != nil {
		return nil, err
	}

	out := []ServerMessage{
		{ScenarioStart: scenarioName, Notification: "Scenario started: " + scenarioName},
	}

	responseText := h.latestBroadcastForDevice(deviceID, beforeCount)
	if responseText == "" {
		return out, nil
	}
	responseView := ui.VoiceAssistantResponseView(deviceID, spoken, responseText)
	out = append(out, ServerMessage{
		SetUI: &responseView,
	})
	if h.runtime.Env.TTS == nil {
		return out, nil
	}

	playback, err := h.runtime.Env.TTS.Synthesize(ctx, responseText, scenario.TTSOptions{
		Voice:  "default",
		Format: "pcm16",
	})
	if err != nil {
		return nil, err
	}
	audio, err := readAudioPlayback(playback)
	if err != nil {
		return nil, err
	}

	out = append(out, ServerMessage{
		PlayAudio: &PlayAudioResponse{
			DeviceID: deviceID,
			Audio:    audio,
			Format:   "pcm16",
		},
	})
	return out, nil
}

func (h *StreamHandler) recordVoiceAudioChunk(recorder recording.Manager, deviceID string, chunk []byte) {
	writer, ok := recorder.(interface {
		WriteDeviceAudio(deviceID string, chunk []byte) error
	})
	if !ok {
		return
	}
	_ = writer.WriteDeviceAudio(deviceID, chunk)
}

// latestBroadcastForDevice returns the most recent broadcast message emitted
// after beforeCount that targets deviceID (or the most recent message overall
// if none explicitly target the device). Returns "" if no new events exist.
func (h *StreamHandler) latestBroadcastForDevice(deviceID string, beforeCount int) string {
	if h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.Broadcast == nil {
		return ""
	}
	eventReader, ok := h.runtime.Env.Broadcast.(interface {
		Events() []ui.BroadcastEvent
	})
	if !ok {
		return ""
	}
	events := eventReader.Events()
	if beforeCount < 0 {
		beforeCount = 0
	}
	if beforeCount > len(events) {
		beforeCount = len(events)
	}
	newEvents := events[beforeCount:]
	if len(newEvents) == 0 {
		return ""
	}
	deviceID = strings.TrimSpace(deviceID)
	fallback := ""
	for _, event := range newEvents {
		fallback = event.Message
		for _, target := range event.DeviceIDs {
			if strings.TrimSpace(target) == deviceID {
				return event.Message
			}
		}
	}
	return fallback
}

// voiceAudioReader is a simple io.Reader over an accumulated voice buffer.
type voiceAudioReader struct {
	buf []byte
	off int
}

// Read consumes bytes from the buffered voice audio.
func (r *voiceAudioReader) Read(p []byte) (int, error) {
	if r.off >= len(r.buf) {
		return 0, io.EOF
	}
	n := copy(p, r.buf[r.off:])
	r.off += n
	return n, nil
}

// readAudioPlayback drains a scenario.AudioPlayback into a byte slice.
func readAudioPlayback(playback scenario.AudioPlayback) ([]byte, error) {
	if playback == nil {
		return nil, nil
	}
	buf := make([]byte, 0, 256)
	chunk := make([]byte, 256)
	for {
		n, err := playback.Read(chunk)
		if n > 0 {
			buf = append(buf, chunk[:n]...)
		}
		if err == io.EOF {
			return buf, nil
		}
		if err != nil {
			return nil, err
		}
	}
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

	action := cmd.Action
	if action == "" {
		action = CommandActionStart
	}
	if action != CommandActionStart && action != CommandActionStop {
		return ServerMessage{}, ErrInvalidCommandAction
	}
	manualIntent := strings.TrimSpace(cmd.Intent)
	if h.runtime == nil {
		if kind != CommandKindManual ||
			(manualIntent != SystemIntentTerminalRefresh && manualIntent != ManualIntentPlaybackMetadata) {
			return ServerMessage{}, errors.New("scenario runtime not configured")
		}
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
		if manualIntent == "" {
			return ServerMessage{}, ErrMissingCommandIntent
		}
		if manualIntent == SystemIntentTerminalRefresh {
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
		if manualIntent == ManualIntentPlaybackMetadata {
			if action == CommandActionStop {
				return ServerMessage{}, ErrInvalidCommandAction
			}
			artifactID := strings.TrimSpace(cmd.Arguments["artifact_id"])
			if artifactID == "" {
				return ServerMessage{}, fmt.Errorf("playback_metadata requires artifact_id")
			}
			targetDeviceID := strings.TrimSpace(cmd.Arguments["target_device_id"])
			if targetDeviceID == "" {
				targetDeviceID = strings.TrimSpace(cmd.DeviceID)
			}
			if targetDeviceID == "" {
				return ServerMessage{}, ErrMissingCommandDeviceID
			}
			metadata, ok := h.playbackMetadataForTarget(artifactID, targetDeviceID)
			if !ok {
				return ServerMessage{}, fmt.Errorf("playback artifact not found: %s", artifactID)
			}
			return ServerMessage{
				Notification: "Playback metadata ready",
				Data:         metadata,
			}, nil
		}
		trigger := scenario.Trigger{
			Kind:      scenario.TriggerManual,
			SourceID:  cmd.DeviceID,
			Intent:    cmd.Intent,
			Arguments: copyStringMap(cmd.Arguments),
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
		for k, v := range h.mediaStreamStatusData() {
			data[k] = v
		}
		for k, v := range h.sensorStatusData() {
			data[k] = v
		}
		for k, v := range h.recordingStatusData() {
			data[k] = v
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
	case SystemIntentRecordingEvents:
		data := map[string]string{}
		h.mu.Lock()
		recorder := h.recording
		h.mu.Unlock()
		eventReader, ok := recorder.(interface {
			RecentEvents(limit int) []recording.Event
		})
		if ok {
			events := eventReader.RecentEvents(50)
			for i, event := range events {
				key := fmt.Sprintf("%03d", i)
				data[key] = strings.Join([]string{
					strconv.FormatInt(event.AtUnixMS, 10),
					event.Action,
					event.StreamID,
					event.Kind,
					event.SourceID,
					event.TargetID,
				}, "|")
			}
		}
		return ServerMessage{
			Notification: "System query: recording_events",
			Data:         data,
		}, nil
	case SystemIntentListPlaybackFiles:
		data := map[string]string{}
		for i, artifact := range h.listPlaybackArtifacts() {
			key := fmt.Sprintf("%03d", i)
			data[key] = strings.Join([]string{
				artifact.ArtifactID,
				artifact.StreamID,
				artifact.Kind,
				artifact.SourceDeviceID,
				artifact.TargetDeviceID,
				strconv.FormatInt(artifact.SizeBytes, 10),
				strconv.FormatInt(artifact.UpdatedUnixMS, 10),
				artifact.AudioPath,
			}, "|")
		}
		return ServerMessage{
			Notification: "System query: list_playback_artifacts",
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
			if snapshot, ok := h.sensorDataForDevice(parsed.Arg); ok {
				data["sensor.unix_ms"] = strconv.FormatInt(snapshot.UnixMS, 10)
				keys := make([]string, 0, len(snapshot.Values))
				for key := range snapshot.Values {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				for _, key := range keys {
					data["sensor."+key] = strconv.FormatFloat(snapshot.Values[key], 'f', -1, 64)
				}
			}
			return ServerMessage{
				Notification: "System query: device_status",
				Data:         data,
			}, nil
		}
		return ServerMessage{}, fmt.Errorf("unknown system intent: %s", cmd.Intent)
	}
}

func (h *StreamHandler) listPlaybackArtifacts() []recording.Artifact {
	h.mu.Lock()
	recorder := h.recording
	h.mu.Unlock()
	lister, ok := recorder.(interface {
		ListPlayableArtifacts() []recording.Artifact
	})
	if !ok {
		return nil
	}
	artifacts := lister.ListPlayableArtifacts()
	out := make([]recording.Artifact, len(artifacts))
	copy(out, artifacts)
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedUnixMS == out[j].UpdatedUnixMS {
			return out[i].ArtifactID < out[j].ArtifactID
		}
		return out[i].UpdatedUnixMS > out[j].UpdatedUnixMS
	})
	return out
}

func (h *StreamHandler) playbackMetadataForTarget(artifactID, targetDeviceID string) (map[string]string, bool) {
	h.mu.Lock()
	recorder := h.recording
	h.mu.Unlock()
	provider, ok := recorder.(interface {
		PlaybackMetadata(artifactID, targetDeviceID string) (recording.PlaybackMetadata, bool)
	})
	if !ok {
		return nil, false
	}
	metadata, ok := provider.PlaybackMetadata(artifactID, targetDeviceID)
	if !ok {
		return nil, false
	}
	return map[string]string{
		"artifact_id":      metadata.Artifact.ArtifactID,
		"stream_id":        metadata.Artifact.StreamID,
		"kind":             metadata.Artifact.Kind,
		"source_device_id": metadata.Artifact.SourceDeviceID,
		"target_device_id": metadata.TargetDeviceID,
		"audio_path":       metadata.Artifact.AudioPath,
		"size_bytes":       strconv.FormatInt(metadata.Artifact.SizeBytes, 10),
		"updated_unix_ms":  strconv.FormatInt(metadata.Artifact.UpdatedUnixMS, 10),
	}, true
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
		h.unregisterMediaStream(routeStreamID(route))
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
