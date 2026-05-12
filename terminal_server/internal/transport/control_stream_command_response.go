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
	if commandResult.ScenarioStart == "chat" {
		chatUI := h.chatEntryUI(cmd.DeviceID)
		responses = append(responses, ServerMessage{SetUI: &chatUI})
		broadcast := h.chatBroadcastMessagesUpdate(cmd.DeviceID)
		// skip index 0 (self) since we already pushed the full SetUI
		if len(broadcast) > 1 {
			responses = append(responses, broadcast[1:]...)
		}
		return responses
	}
	if commandResult.ScenarioStop == "chat" {
		if restored := h.resumedScenarioUI(ctx, cmd.DeviceID, "chat"); len(restored) > 0 {
			responses = append(responses, restored...)
		}
		broadcast := h.chatBroadcastMessagesUpdate(cmd.DeviceID)
		if len(broadcast) > 1 {
			responses = append(responses, broadcast[1:]...)
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
			if restored := h.resumedScenarioUIForTargets(ctx, cmd, commandResult.ScenarioStop); len(restored) > 0 {
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
			routing := routeDeltaStreamRouting()
			startMsg := ServerMessage{
				StartStream: &StartStreamResponse{
					StreamID:       routeID,
					Kind:           route.StreamKind,
					SourceDeviceID: route.SourceID,
					TargetDeviceID: route.TargetID,
					Metadata: map[string]string{
						"origin":      "route_delta",
						"webrtc_mode": "server_managed",
					},
					Routing: routing,
				},
			}
			out = h.appendRouteMessageForPeers(out, cmd.DeviceID, route.SourceID, route.TargetID, startMsg)
			h.registerMediaStream(StartStreamResponse{
				StreamID:       routeID,
				Kind:           route.StreamKind,
				SourceDeviceID: route.SourceID,
				TargetDeviceID: route.TargetID,
				Metadata: map[string]string{
					"origin":      "route_delta",
					"webrtc_mode": "server_managed",
				},
				Routing: routing,
			})
			routeMsg := ServerMessage{
				RouteStream: &RouteStreamResponse{
					StreamID:       routeID,
					SourceDeviceID: route.SourceID,
					TargetDeviceID: route.TargetID,
					Kind:           route.StreamKind,
					Routing:        routing,
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
