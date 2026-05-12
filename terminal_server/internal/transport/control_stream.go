package transport

import (
	"context"
	"log/slog"
	"strings"
	"time"

	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
	iov1 "github.com/curtcox/terminals/terminal_server/gen/go/io/v1"
	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/replsession"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

// SetMenuOverlayInputPolicyForTesting overrides menu-overlay routing policy in tests.
func (h *StreamHandler) SetMenuOverlayInputPolicyForTesting(mode string, overrides map[string]bool) {
	config := overlayInputPolicyConfig{
		Mode:      normalizeOverlayInputPolicy(mode),
		Overrides: map[overlayInputStream]bool{},
	}
	for key, value := range overrides {
		stream := normalizeOverlayInputStream(key)
		if stream == "" {
			continue
		}
		config.Overrides[stream] = value
	}
	h.mu.Lock()
	h.menuOverlayPolicy = mergeOverlayPolicy(defaultOverlayInputPolicy(), config)
	h.mu.Unlock()
}

// SetPhotoFrameSettings overrides photo-frame slide URLs and rotation
// interval. Empty slide input preserves existing/default slides.
func (h *StreamHandler) SetPhotoFrameSettings(slides []string, interval time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(slides) > 0 {
		h.photoFrameSlides = append([]string(nil), slides...)
	}
	if interval > 0 {
		h.photoFrameInterval = interval
	}
}

// SetTerminalREPLAdminURL configures the base URL used by terminal REPL
// sessions when they query server control-plane APIs.
func (h *StreamHandler) SetTerminalREPLAdminURL(baseURL string) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = defaultTerminalReplAdminURL
	}
	h.mu.Lock()
	h.terminalReplAdminURL = baseURL
	h.mu.Unlock()
}

// ReplSessions exposes typed REPL session lifecycle APIs used by admin and
// other control-plane callers.
func (h *StreamHandler) ReplSessions() *replsession.Service {
	return h.replSessions
}

// HandleMessage processes one incoming control message and returns responses.
func (h *StreamHandler) HandleMessage(ctx context.Context, msg ClientMessage) ([]ServerMessage, error) {
	switch {
	case msg.Hello != nil:
		return h.handleHelloMessage(ctx, msg.Hello)
	case msg.CapabilitySnap != nil:
		return h.handleCapabilitySnapshotMessage(ctx, msg.CapabilitySnap)
	case msg.CapabilityDelta != nil:
		return h.handleCapabilityDeltaMessage(ctx, msg.CapabilityDelta)
	case msg.Register != nil:
		return h.handleRegisterMessage(ctx, msg.Register)
	case msg.Capability != nil:
		return h.handleCapabilityUpdateMessage(ctx, msg.Capability)
	case msg.Heartbeat != nil:
		return h.handleHeartbeatMessage(ctx, msg.Heartbeat)
	case msg.Sensor != nil:
		return h.handleSensorMessage(ctx, msg.Sensor)
	case msg.Observation != nil:
		h.handleObservationMessage(ctx, msg.Observation)
		return nil, nil
	case msg.ArtifactReady != nil:
		h.handleArtifactReadyMessage(ctx, msg.ArtifactReady)
		return nil, nil
	case msg.FlowStats != nil:
		h.handleFlowStatsMessage(ctx, msg.FlowStats)
		return nil, nil
	case msg.ClockSample != nil:
		h.handleClockSampleMessage(ctx, msg.ClockSample)
		return nil, nil
	case msg.StreamReady != nil:
		return h.handleStreamReadyMessage(msg.StreamReady)
	case msg.WebRTCSignal != nil:
		return h.handleWebRTCSignalMessage(ctx, msg.WebRTCSignal, msg.SessionDeviceID)
	case msg.VoiceAudio != nil:
		return h.handleVoiceAudioMessage(ctx, msg.VoiceAudio)
	case msg.Input != nil:
		return h.handleInputMessage(ctx, msg.Input)
	case msg.Command != nil:
		return h.commandDispatcher.Dispatch(ctx, msg.Command)
	case msg.BugReport != nil:
		return h.handleBugReportMessage(ctx, msg.BugReport)
	default:
		return h.protocolError(ErrInvalidClientMessage)
	}
}

func (h *StreamHandler) protocolError(err error) ([]ServerMessage, error) {
	h.metrics.protocolErrors.Add(1)
	return []ServerMessage{{ErrorCode: errorCodeFor(err), Error: err.Error()}}, err
}

// Capability lifecycle dispatch stays in StreamHandler because reconnect UI,
// overlay replay, route replay, and capability-change effects cross subsystem
// boundaries even though capability persistence lives in CapabilityLifecycle.
func (h *StreamHandler) handleHelloMessage(ctx context.Context, req *HelloRequest) ([]ServerMessage, error) {
	out, err := h.capabilityLifecycle.HandleHello(ctx, *req)
	if err != nil {
		return h.protocolError(err)
	}
	return out, nil
}

func (h *StreamHandler) handleCapabilitySnapshotMessage(ctx context.Context, req *CapabilitySnapshotRequest) ([]ServerMessage, error) {
	h.metrics.capabilityReceived.Add(1)
	result, err := h.capabilityLifecycle.HandleSnapshot(ctx, *req)
	if err != nil {
		return h.protocolError(err)
	}
	out := result.Messages
	deviceID := result.DeviceID
	storedUI, hasUI := h.uiSession.LastSetUI(deviceID)
	hasOverlay := h.hasMenuOverlay(deviceID)

	if !result.HadPriorDevice || hasUI {
		if hasUI {
			out = append(out, ServerMessage{SetUI: &storedUI})
		} else {
			initial := ui.HelloWorld(result.AfterDeviceName)
			result.RegisterAck.Initial = initial
			out = append(out, ServerMessage{SetUI: &initial})
			h.uiSession.RememberSetUI(deviceID, out)
		}
		out = h.appendOverlayReplay(out, deviceID, hasOverlay)
		out = append(out, h.routeReplay.MessagesForDevice(deviceID, h.routeSnapshotForDevice(deviceID), true)...)
	}
	if !result.IsInitialBaseline {
		effects := h.handleCapabilityChangeEffects(ctx, deviceID, result.BeforeCaps, result.AfterCaps)
		if len(effects) > 0 {
			out = append(out, effects...)
		}
	}
	return out, nil
}

func (h *StreamHandler) handleCapabilityDeltaMessage(ctx context.Context, req *CapabilityDeltaRequest) ([]ServerMessage, error) {
	h.metrics.capabilityReceived.Add(1)
	result, err := h.capabilityLifecycle.HandleDelta(ctx, *req)
	if err != nil {
		return h.protocolError(err)
	}
	out := result.Messages
	effects := h.handleCapabilityChangeEffects(ctx, result.DeviceID, result.BeforeCaps, result.AfterCaps)
	if len(effects) > 0 {
		out = append(out, effects...)
	}
	return out, nil
}

func (h *StreamHandler) handleRegisterMessage(ctx context.Context, req *RegisterRequest) ([]ServerMessage, error) {
	h.metrics.registerReceived.Add(1)
	resp, err := h.capabilityLifecycle.HandleRegister(ctx, *req)
	if err != nil {
		return h.protocolError(err)
	}
	deviceID := req.DeviceID
	storedUI, hasUI := h.uiSession.LastSetUI(deviceID)
	hasOverlay := h.hasMenuOverlay(deviceID)

	out := []ServerMessage{{RegisterAck: &resp}}
	if hasUI {
		out = append(out, ServerMessage{SetUI: &storedUI})
	} else {
		out = append(out, ServerMessage{SetUI: &resp.Initial})
		h.uiSession.RememberSetUI(deviceID, out)
	}
	out = h.appendOverlayReplay(out, deviceID, hasOverlay)
	out = append(out, h.routeReplay.MessagesForDevice(deviceID, h.routeSnapshotForDevice(deviceID), false)...)
	return out, nil
}

func (h *StreamHandler) handleCapabilityUpdateMessage(ctx context.Context, req *CapabilityUpdateRequest) ([]ServerMessage, error) {
	h.metrics.capabilityReceived.Add(1)
	if err := h.capabilityLifecycle.HandleUpdateCapabilities(ctx, *req); err != nil {
		return h.protocolError(err)
	}
	return nil, nil
}

func (h *StreamHandler) hasMenuOverlay(deviceID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.menuOverlayByDevice[deviceID]
	return ok
}

func (h *StreamHandler) appendOverlayReplay(out []ServerMessage, deviceID string, hasOverlay bool) []ServerMessage {
	if !hasOverlay {
		return out
	}
	return append(out, ServerMessage{
		UpdateUI: &UIUpdate{
			ComponentID: ui.GlobalOverlayComponentID,
			Node:        h.menuOverlayDescriptor(deviceID),
		},
	})
}

// Heartbeat dispatch coordinates terminal polling, scenario UI host updates,
// and lightweight scenario heartbeat work. The durable per-device UI snapshot
// remains owned by UISessionState.
func (h *StreamHandler) handleHeartbeatMessage(ctx context.Context, req *HeartbeatRequest) ([]ServerMessage, error) {
	h.metrics.heartbeatReceived.Add(1)
	if err := h.control.Heartbeat(ctx, req.DeviceID); err != nil {
		return h.protocolError(err)
	}
	out := make([]ServerMessage, 0, 2)
	update, pollErr := h.pollTerminalOutput(req.DeviceID, false)
	if pollErr != nil {
		return h.protocolError(pollErr)
	}
	if update != nil {
		out = append(out, *update)
	}
	if photoRotate := h.photoFrameHeartbeatUpdate(req.DeviceID); photoRotate != nil {
		out = append(out, *photoRotate)
	}
	uiMessages := h.uiHostMessagesForDevice(req.DeviceID)
	if len(uiMessages) > 0 {
		out = append(out, uiMessages...)
	}
	if len(out) > 0 {
		h.uiSession.RememberSetUI(req.DeviceID, out)
		return out, nil
	}
	return nil, nil
}

// Telemetry and observation dispatch stay intentionally thin: StreamHandler
// translates transport messages into runtime sinks or event-log records, while
// the runtime/observer owns downstream semantics.
func (h *StreamHandler) handleSensorMessage(ctx context.Context, req *SensorDataRequest) ([]ServerMessage, error) {
	h.metrics.sensorReceived.Add(1)
	beforeBroadcastEvents := h.broadcastEventCount()
	h.recordSensorData(req)
	if h.runtime != nil {
		values := map[string]float64{}
		for key, value := range req.Values {
			values[key] = value
		}
		if err := h.runtime.ProcessSensorReading(ctx, scenario.SensorReading{
			DeviceID: strings.TrimSpace(req.DeviceID),
			UnixMS:   req.UnixMS,
			Values:   values,
		}); err != nil {
			return h.protocolError(err)
		}
	}
	out := h.broadcastNotificationsSince(beforeBroadcastEvents, req.DeviceID, true)
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func (h *StreamHandler) handleObservationMessage(ctx context.Context, req *ObservationRequest) {
	if sink, ok := h.observationSink(); ok {
		sink.AddObservation(ctx, req.Observation)
	}
}

func (h *StreamHandler) handleArtifactReadyMessage(ctx context.Context, req *ArtifactAvailableRequest) {
	if sink, ok := h.observationSink(); ok {
		sink.AddObservation(ctx, iorouter.Observation{
			Kind:       "artifact.available",
			OccurredAt: time.Now().UTC(),
			Evidence:   []iorouter.ArtifactRef{req.Artifact},
		})
	}
}

func (h *StreamHandler) observationSink() (observationSink, bool) {
	if h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.Observe == nil {
		return nil, false
	}
	sink, ok := h.runtime.Env.Observe.(observationSink)
	return sink, ok
}

func (h *StreamHandler) handleFlowStatsMessage(ctx context.Context, req *FlowStatsRequest) {
	eventlog.Emit(ctx, "io.flow.stats", slog.LevelInfo, "flow stats received",
		slog.String("component", "io.flow"),
		slog.String("flow_id", strings.TrimSpace(req.FlowID)),
		slog.Float64("cpu_pct", req.CPUPct),
		slog.Float64("mem_mb", req.MemMB),
		slog.Uint64("dropped_frames", req.DroppedFrames),
		slog.String("state", resolveFlowState(req)),
		slog.String("error", strings.TrimSpace(req.Error)),
	)
}

// resolveFlowState prefers the typed enum when set and falls back to the
// legacy string for older clients.
func resolveFlowState(req *FlowStatsRequest) string {
	switch req.StateEnum {
	case iov1.FlowState_FLOW_STATE_UNSPECIFIED:
		return strings.TrimSpace(req.State)
	case iov1.FlowState_FLOW_STATE_STARTING:
		return "starting"
	case iov1.FlowState_FLOW_STATE_RUNNING:
		return "running"
	case iov1.FlowState_FLOW_STATE_DEGRADED:
		return "degraded"
	case iov1.FlowState_FLOW_STATE_STOPPING:
		return "stopping"
	case iov1.FlowState_FLOW_STATE_STOPPED:
		return "stopped"
	case iov1.FlowState_FLOW_STATE_FAILED:
		return "failed"
	}
	return strings.TrimSpace(req.State)
}

func (h *StreamHandler) handleClockSampleMessage(ctx context.Context, req *ClockSampleRequest) {
	eventlog.Emit(ctx, "io.flow.stats", slog.LevelDebug, "clock sample received",
		slog.String("component", "io.flow"),
		slog.String("device_id", strings.TrimSpace(req.DeviceID)),
		slog.Float64("error_ms", req.ErrorMS),
		slog.Int64("client_unix_ms", req.ClientUnixMS),
		slog.Int64("server_unix_ms", req.ServerUnixMS),
	)
}

// Media, voice, and UI input dispatch routes transport events to the
// collaborators that now own mutable media and voice state. StreamHandler keeps
// protocol metrics, overlay admission policy, and wire error mapping.
func (h *StreamHandler) handleStreamReadyMessage(req *StreamReadyRequest) ([]ServerMessage, error) {
	h.metrics.streamReadyReceived.Add(1)
	h.markStreamReady(req.StreamID)
	return nil, nil
}

func (h *StreamHandler) handleWebRTCSignalMessage(ctx context.Context, req *WebRTCSignalRequest, sessionDeviceID string) ([]ServerMessage, error) {
	h.metrics.webrtcSignalReceived.Add(1)
	return h.handleWebRTCSignal(ctx, req, sessionDeviceID), nil
}

func (h *StreamHandler) handleVoiceAudioMessage(ctx context.Context, req *VoiceAudioRequest) ([]ServerMessage, error) {
	h.metrics.voiceAudioReceived.Add(1)
	if h.shouldDropMainStreamWhileOverlayOpen(strings.TrimSpace(req.DeviceID), overlayStreamAudio) {
		return nil, nil
	}
	out, err := h.handleVoiceAudio(ctx, req)
	if err != nil {
		return h.protocolError(err)
	}
	return out, nil
}

func (h *StreamHandler) handleInputMessage(ctx context.Context, req *InputRequest) ([]ServerMessage, error) {
	out, err := h.handleInput(ctx, req)
	if err != nil {
		return h.protocolError(err)
	}
	h.uiSession.RememberSetUI(req.DeviceID, out)
	return out, nil
}

func (h *StreamHandler) handleBugReportMessage(ctx context.Context, report *diagnosticsv1.BugReport) ([]ServerMessage, error) {
	response, err := h.diagnostics.HandleBugReport(ctx, report)
	if err != nil {
		return h.protocolError(err)
	}
	return []ServerMessage{response}, nil
}

func (h *StreamHandler) decorateBugReportAffordance(deviceID string, msg ServerMessage) ServerMessage {
	if msg.SetUI == nil {
		return msg
	}
	decorated := withBugReportAffordance(*msg.SetUI, strings.TrimSpace(deviceID))
	decorated = withCornerAffordance(decorated, strings.TrimSpace(deviceID))
	msg.SetUI = &decorated
	return msg
}

func withBugReportAffordance(root ui.Descriptor, subjectDeviceID string) ui.Descriptor {
	if hasBugReportAffordance(root) {
		return root
	}
	action := bugReportActionPrefix
	if subjectDeviceID != "" {
		action += ":" + subjectDeviceID
	}
	button := ui.New("button", map[string]string{
		"id":     bugReportButtonID,
		"label":  "Report a bug",
		"action": action,
	})
	if root.Type == "stack" {
		root.Children = append(root.Children, button)
		return root
	}
	return ui.New("stack", map[string]string{
		"id": "bug_report_affordance_root",
	}, root, button)
}

func hasBugReportAffordance(node ui.Descriptor) bool {
	nodeID := strings.TrimSpace(node.ID)
	if nodeID == "" {
		nodeID = strings.TrimSpace(node.Props["id"])
	}
	if nodeID == bugReportButtonID {
		return true
	}
	if strings.TrimSpace(node.Type) == "button" && strings.HasPrefix(strings.TrimSpace(node.Props["action"]), bugReportActionPrefix) {
		return true
	}
	for _, child := range node.Children {
		if hasBugReportAffordance(child) {
			return true
		}
	}
	return false
}

func scopedAffordanceID(ownerID, logicalID string) string {
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		ownerID = "main"
	}
	return "act:" + ownerID + "/" + strings.TrimSpace(logicalID)
}

func hasNodeID(node ui.Descriptor, id string) bool {
	nodeID := strings.TrimSpace(node.ID)
	if nodeID == "" {
		nodeID = strings.TrimSpace(node.Props["id"])
	}
	if nodeID == id {
		return true
	}
	for _, child := range node.Children {
		if hasNodeID(child, id) {
			return true
		}
	}
	return false
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

func (h *StreamHandler) uiHostEventCount() int {
	if h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.UI == nil {
		return 0
	}
	eventReader, ok := h.runtime.Env.UI.(interface {
		Events() []ui.HostEvent
	})
	if !ok {
		return 0
	}
	return len(eventReader.Events())
}

func (h *StreamHandler) uiHostMessagesForDevice(deviceID string) []ServerMessage {
	count := h.uiHostEventCount()
	beforeCount := h.uiSession.UIHostBeforeCountAndAdvance(deviceID, count)
	return h.uiHostMessagesSince(beforeCount, deviceID, true)
}

func (h *StreamHandler) uiHostMessagesSince(beforeCount int, sessionDeviceID string, includeSession bool) []ServerMessage {
	if h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.UI == nil {
		return nil
	}
	eventReader, ok := h.runtime.Env.UI.(interface {
		Events() []ui.HostEvent
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
	sessionDeviceID = strings.TrimSpace(sessionDeviceID)
	out := make([]ServerMessage, 0, len(events)-beforeCount)
	delivered := map[string]struct{}{}
	for _, event := range events[beforeCount:] {
		targetDeviceID := strings.TrimSpace(event.DeviceID)
		if targetDeviceID == "" {
			continue
		}
		if targetDeviceID == sessionDeviceID && !includeSession {
			continue
		}
		msg := ServerMessage{}
		switch event.Kind {
		case "set":
			node := event.Node
			msg.SetUI = &node
		case "patch":
			msg.UpdateUI = &UIUpdate{
				ComponentID: strings.TrimSpace(event.ComponentID),
				Node:        event.Node,
			}
		case "clear":
			node := ui.HelloWorld(targetDeviceID)
			msg.SetUI = &node
		default:
			continue
		}
		if targetDeviceID != sessionDeviceID {
			msg.RelayToDeviceID = targetDeviceID
		}
		out = append(out, msg)
		delivered[targetDeviceID] = struct{}{}
		delivered[sessionDeviceID] = struct{}{}
	}
	if len(delivered) > 0 {
		h.uiSession.MarkUIHostDelivered(delivered, len(events))
	}
	return out
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

// NoteProtocolError increments the transport protocol-error counter exposed via metrics.
func (h *StreamHandler) NoteProtocolError() {
	if h.metrics != nil {
		h.metrics.protocolErrors.Add(1)
	}
}

// HandleDisconnect releases stream-scoped resources for a disconnected device.
func (h *StreamHandler) HandleDisconnect(deviceID string) {
	h.uiSession.ForgetMainUIActivation(deviceID)
	h.uiOwners.ForgetDevice(deviceID)
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
	h.routeReplay.Capture(deviceID, routes)
	for _, route := range routes {
		_ = routeIO.Disconnect(route.SourceID, route.TargetID, route.StreamKind)
		h.unregisterMediaStream(routeStreamID(route))
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
