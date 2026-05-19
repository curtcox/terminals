package transport

import (
	"context"
	"os"
	"sort"
	"strings"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

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
	if playback, ok := h.commandPlaybackDispatch(cmd, commandResult); ok {
		responses = append(responses, playback)
		return responses
	}
	if extra, handled := h.commandResponsesForScenarioStop(ctx, cmd, commandResult); handled {
		responses = append(responses, extra...)
		return responses
	}
	if extra, handled := h.commandResponsesForScenarioStart(ctx, cmd, commandResult); handled {
		responses = append(responses, extra...)
		return responses
	}
	if commandResult.ScenarioStart == "multi_window" {
		responses = h.appendMultiWindowStartUI(cmd, responses)
	}
	if cmd.Action != "" && cmd.Action != CommandActionStart {
		if commandResult.ScenarioStop != "" {
			if restored := h.resumedScenarioUIForTargets(ctx, cmd, commandResult.ScenarioStop); len(restored) > 0 {
				responses = append(responses, restored...)
			}
		}
		return responses
	}
	if commandResult.ScenarioStart != "terminal" {
		return responses
	}
	responses = append(responses, h.commandResponsesTerminalStart(ctx, cmd)...)
	return responses
}

func (h *StreamHandler) commandPlaybackDispatch(cmd *CommandRequest, commandResult ServerMessage) (ServerMessage, bool) {
	if cmd == nil {
		return ServerMessage{}, false
	}
	kind := strings.TrimSpace(cmd.Kind)
	if kind == "" {
		kind = CommandKindManual
	}
	if kind != CommandKindManual {
		return ServerMessage{}, false
	}
	if defaultAction(cmd.Action) != CommandActionStart {
		return ServerMessage{}, false
	}
	if strings.TrimSpace(cmd.Intent) != ManualIntentPlaybackMetadata {
		return ServerMessage{}, false
	}
	audioPath := strings.TrimSpace(commandResult.Data["audio_path"])
	targetDeviceID := strings.TrimSpace(commandResult.Data["target_device_id"])
	if audioPath == "" || targetDeviceID == "" {
		return ServerMessage{}, false
	}
	audio, err := os.ReadFile(audioPath)
	if err != nil || len(audio) == 0 {
		return ServerMessage{}, false
	}

	format := strings.TrimSpace(commandResult.Data["format"])
	if format == "" {
		format = "pcm16"
	}
	playAudio := ServerMessage{
		PlayAudio: &PlayAudioResponse{
			RequestID: cmd.RequestID,
			DeviceID:  targetDeviceID,
			Audio:     audio,
			Format:    format,
		},
	}
	if targetDeviceID != strings.TrimSpace(cmd.DeviceID) {
		playAudio.RelayToDeviceID = targetDeviceID
	}
	return playAudio, true
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
	switch action {
	case CommandActionStart:
		return h.routeStartUpdatesForCommand(cmd, before, after)
	case CommandActionStop:
		return h.routeStopUpdatesForCommand(cmd, before, after)
	default:
		return nil
	}
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

func (h *StreamHandler) resumedScenarioUIForTargets(
	ctx context.Context,
	cmd *CommandRequest,
	stoppedScenario string,
) []ServerMessage {
	if cmd == nil {
		return nil
	}
	targets := h.commandTargetDeviceIDs(cmd)
	if len(targets) == 0 {
		return h.resumedScenarioUI(ctx, cmd.DeviceID, stoppedScenario)
	}
	sourceDeviceID := strings.TrimSpace(cmd.DeviceID)
	out := make([]ServerMessage, 0, len(targets)*2)
	seen := map[string]struct{}{}
	for _, targetDeviceID := range targets {
		targetDeviceID = strings.TrimSpace(targetDeviceID)
		if targetDeviceID == "" {
			continue
		}
		if _, exists := seen[targetDeviceID]; exists {
			continue
		}
		seen[targetDeviceID] = struct{}{}
		resumed := h.resumedScenarioUI(ctx, targetDeviceID, stoppedScenario)
		for _, msg := range resumed {
			if targetDeviceID != sourceDeviceID {
				msg.RelayToDeviceID = targetDeviceID
			}
			out = append(out, msg)
		}
	}
	return out
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
	if targets := commandTargetDeviceIDsFromArgs(cmd.Arguments); len(targets) > 0 {
		return targets
	}
	return h.commandTargetDeviceIDsFallback(cmd)
}
