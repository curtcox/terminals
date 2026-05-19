package transport

import (
	"context"
	"strings"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

type scenarioUICommand struct {
	intent        string
	commandAction string
	triggerArgs   map[string]string
}

func resolveScenarioUICommand(activeName, action, deviceID string) (scenarioUICommand, bool) {
	action = strings.TrimSpace(action)
	if action == "" {
		return scenarioUICommand{}, false
	}
	triggerArgs := map[string]string{"device_id": deviceID}
	cmd := scenarioUICommand{
		commandAction: CommandActionStart,
		triggerArgs:   triggerArgs,
	}
	switch {
	case action == "stop_active":
		cmd.intent = activeName
		cmd.commandAction = CommandActionStop
	case action == "internal_video_call_end":
		if activeName != "internal_video_call" {
			return scenarioUICommand{}, false
		}
		cmd.intent = "internal_video_call"
		cmd.commandAction = CommandActionStop
	case action == "multi_window_end":
		if activeName != "multi_window" {
			return scenarioUICommand{}, false
		}
		cmd.intent = "multi_window"
		cmd.commandAction = CommandActionStop
	case strings.HasPrefix(action, "multi_window_focus:"):
		if activeName != "multi_window" {
			return scenarioUICommand{}, false
		}
		focusDeviceID := strings.TrimSpace(strings.TrimPrefix(action, "multi_window_focus:"))
		if focusDeviceID == "" {
			return scenarioUICommand{}, true
		}
		cmd.intent = "multi_window"
		cmd.triggerArgs["audio_focus_device_id"] = focusDeviceID
	case strings.HasPrefix(action, "start:"):
		cmd.intent = strings.TrimSpace(strings.TrimPrefix(action, "start:"))
	case strings.HasPrefix(action, "stop:"):
		cmd.intent = strings.TrimSpace(strings.TrimPrefix(action, "stop:"))
		cmd.commandAction = CommandActionStop
	default:
		cmd.intent = action
	}
	if cmd.intent == "" {
		return scenarioUICommand{}, true
	}
	return cmd, true
}

func (h *StreamHandler) scenarioUICommandRequest(deviceID string, uiCmd scenarioUICommand) *CommandRequest {
	return &CommandRequest{
		DeviceID:  deviceID,
		Action:    uiCmd.commandAction,
		Kind:      CommandKindManual,
		Intent:    uiCmd.intent,
		Arguments: copyStringMap(uiCmd.triggerArgs),
	}
}

func (h *StreamHandler) scenarioUITrigger(deviceID string, uiCmd scenarioUICommand) scenario.Trigger {
	return scenario.Trigger{
		Kind:      scenario.TriggerManual,
		SourceID:  deviceID,
		Intent:    uiCmd.intent,
		Arguments: uiCmd.triggerArgs,
		IntentV2: &scenario.IntentRecord{
			Action: uiCmd.intent,
			Slots:  copyStringMap(uiCmd.triggerArgs),
			Source: scenario.SourceUI,
		},
	}
}

func (h *StreamHandler) appendScenarioUICommandSideEffects(
	_ context.Context,
	deviceID string,
	cmd *CommandRequest,
	result ServerMessage,
	beforeRoutes []iorouter.Route,
	beforeBroadcastEvents int,
	beforeUIEvents int,
	responses []ServerMessage,
) []ServerMessage {
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
	broadcastNotifications := h.commandDispatcher.BroadcastNotificationsForCommand(cmd, result, beforeBroadcastEvents)
	if len(broadcastNotifications) > 0 {
		responses = append(responses, broadcastNotifications...)
	}
	uiMessages := h.uiHostMessagesSince(beforeUIEvents, deviceID, true)
	if len(uiMessages) > 0 {
		responses = append(responses, uiMessages...)
	}
	return responses
}
