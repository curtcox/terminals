package transport

import (
	"context"
	"strings"

	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func (h *StreamHandler) commandResponsesForScenarioStop(
	ctx context.Context,
	cmd *CommandRequest,
	commandResult ServerMessage,
) ([]ServerMessage, bool) {
	switch commandResult.ScenarioStop {
	case "terminal":
		return h.commandResponsesTerminalStop(ctx, cmd), true
	case "photo_frame":
		return h.commandResponsesPhotoFrameStop(ctx, cmd), true
	case "internal_video_call":
		return h.commandResponsesInternalVideoCallStop(ctx, cmd), true
	case "multi_window":
		return h.commandResponsesMultiWindowStop(cmd), true
	case "chat":
		return h.commandResponsesChatStop(ctx, cmd), true
	default:
		return nil, false
	}
}

func (h *StreamHandler) commandResponsesTerminalStop(ctx context.Context, cmd *CommandRequest) []ServerMessage {
	h.terminateTerminalForDevice(cmd.DeviceID)
	responses := []ServerMessage{{
		TransitionUI: &UITransition{
			Transition: "terminal_exit",
			DurationMS: 220,
		},
	}}
	if restored := h.resumedScenarioUI(ctx, cmd.DeviceID, "terminal"); len(restored) > 0 {
		responses = append(responses, restored...)
	}
	return responses
}

func (h *StreamHandler) commandResponsesPhotoFrameStop(ctx context.Context, cmd *CommandRequest) []ServerMessage {
	for _, deviceID := range h.commandTargetDeviceIDs(cmd) {
		h.clearPhotoFrameState(deviceID)
	}
	responses := []ServerMessage{{
		TransitionUI: &UITransition{
			Transition: "photo_frame_exit",
			DurationMS: 220,
		},
	}}
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

func (h *StreamHandler) commandResponsesInternalVideoCallStop(ctx context.Context, cmd *CommandRequest) []ServerMessage {
	responses := []ServerMessage{{
		TransitionUI: &UITransition{
			Transition: "internal_video_call_exit",
			DurationMS: 220,
		},
	}}
	if restored := h.resumedScenarioUI(ctx, cmd.DeviceID, "internal_video_call"); len(restored) > 0 {
		responses = append(responses, restored...)
	}
	return responses
}

func (h *StreamHandler) commandResponsesMultiWindowStop(cmd *CommandRequest) []ServerMessage {
	if restoredUI, restoredTransition, ok := h.restoreMultiWindowResume(cmd.DeviceID); ok {
		responses := make([]ServerMessage, 0, 2)
		if restoredUI != nil {
			responses = append(responses, ServerMessage{SetUI: restoredUI})
		}
		if restoredTransition != nil {
			responses = append(responses, ServerMessage{TransitionUI: restoredTransition})
		}
		return responses
	}
	return nil
}

func (h *StreamHandler) commandResponsesChatStop(ctx context.Context, cmd *CommandRequest) []ServerMessage {
	responses := make([]ServerMessage, 0, 4)
	if restored := h.resumedScenarioUI(ctx, cmd.DeviceID, "chat"); len(restored) > 0 {
		responses = append(responses, restored...)
	}
	broadcast := h.chatBroadcastMessagesUpdate(cmd.DeviceID)
	if len(broadcast) > 1 {
		responses = append(responses, broadcast[1:]...)
	}
	return responses
}

func (h *StreamHandler) commandResponsesForScenarioStart(
	_ context.Context,
	cmd *CommandRequest,
	commandResult ServerMessage,
) ([]ServerMessage, bool) {
	switch commandResult.ScenarioStart {
	case "chat":
		return h.commandResponsesChatStart(cmd), true
	case "photo_frame":
		return h.commandResponsesPhotoFrameStart(cmd), true
	case "internal_video_call":
		return h.commandResponsesInternalVideoCallStart(cmd), true
	default:
		return nil, false
	}
}

func (h *StreamHandler) commandResponsesChatStart(cmd *CommandRequest) []ServerMessage {
	chatUI := h.chatEntryUI(cmd.DeviceID)
	responses := []ServerMessage{{SetUI: &chatUI}}
	broadcast := h.chatBroadcastMessagesUpdate(cmd.DeviceID)
	if len(broadcast) > 1 {
		responses = append(responses, broadcast[1:]...)
	}
	return responses
}

func (h *StreamHandler) commandResponsesPhotoFrameStart(cmd *CommandRequest) []ServerMessage {
	photoFrameUI := h.photoFrameSetUI(cmd.DeviceID, true)
	responses := []ServerMessage{
		{SetUI: &photoFrameUI},
		{TransitionUI: &UITransition{Transition: "photo_frame_enter", DurationMS: 220}},
	}
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

func (h *StreamHandler) commandResponsesInternalVideoCallStart(cmd *CommandRequest) []ServerMessage {
	responses := make([]ServerMessage, 0, 2)
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

func (h *StreamHandler) commandResponsesTerminalStart(ctx context.Context, cmd *CommandRequest) []ServerMessage {
	output, err := h.ensureTerminalSession(ctx, cmd.DeviceID)
	if err != nil {
		return []ServerMessage{{Notification: "Terminal session failed: " + err.Error()}}
	}
	terminalUI := ui.TerminalViewWithOutput(cmd.DeviceID, output)
	return []ServerMessage{
		{SetUI: &terminalUI},
		{TransitionUI: &UITransition{Transition: "terminal_enter", DurationMS: 220}},
	}
}

func (h *StreamHandler) appendMultiWindowStartUI(cmd *CommandRequest, responses []ServerMessage) []ServerMessage {
	peerIDs, focusedPeerID := h.multiWindowPeersAndFocus(cmd.DeviceID)
	multiWindowUI := ui.MultiWindowView(cmd.DeviceID, peerIDs, focusedPeerID)
	return append(responses, ServerMessage{SetUI: &multiWindowUI})
}
