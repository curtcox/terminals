package transport

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func (h *StreamHandler) routeScenarioUIAction(ctx context.Context, deviceID, action string) ([]ServerMessage, bool, error) {
	if h.runtime == nil || h.runtime.Engine == nil {
		return nil, false, nil
	}
	activeName, active := h.runtime.Engine.Active(deviceID)
	if !active {
		return nil, false, nil
	}
	uiCmd, resolved := resolveScenarioUICommand(activeName, action, deviceID)
	if !resolved {
		return nil, false, nil
	}
	if uiCmd.intent == "" {
		return nil, true, nil
	}
	if uiCmd.commandAction == CommandActionStart && !h.isRegisteredScenario(uiCmd.intent) {
		return nil, false, nil
	}
	if uiCmd.commandAction == CommandActionStop && action != "stop_active" && !h.isRegisteredScenario(uiCmd.intent) {
		return nil, false, nil
	}

	beforeRoutes := h.routeSnapshotForDevice(deviceID)
	beforeBroadcastEvents := h.broadcastEventCount()
	beforeUIEvents := h.uiHostEventCount()
	trigger := h.scenarioUITrigger(deviceID, uiCmd)
	cmd := h.scenarioUICommandRequest(deviceID, uiCmd)

	if uiCmd.commandAction == CommandActionStop {
		return h.finishScenarioUIStop(ctx, deviceID, trigger, cmd, beforeRoutes, beforeBroadcastEvents, beforeUIEvents)
	}
	return h.finishScenarioUIStart(ctx, deviceID, activeName, trigger, cmd, beforeRoutes, beforeBroadcastEvents, beforeUIEvents)
}

func (h *StreamHandler) finishScenarioUIStop(
	ctx context.Context,
	deviceID string,
	trigger scenario.Trigger,
	cmd *CommandRequest,
	beforeRoutes []iorouter.Route,
	beforeBroadcastEvents int,
	beforeUIEvents int,
) ([]ServerMessage, bool, error) {
	name, err := h.runtime.StopTrigger(ctx, trigger)
	if err != nil {
		return nil, true, err
	}
	result := ServerMessage{
		ScenarioStop: name,
		Notification: "Scenario stopped: " + name,
	}
	responses := h.commandResponses(ctx, cmd, result)
	responses = h.appendScenarioUICommandSideEffects(
		ctx, deviceID, cmd, result, beforeRoutes, beforeBroadcastEvents, beforeUIEvents, responses,
	)
	return responses, true, nil
}

func (h *StreamHandler) finishScenarioUIStart(
	ctx context.Context,
	deviceID, activeName string,
	trigger scenario.Trigger,
	cmd *CommandRequest,
	beforeRoutes []iorouter.Route,
	beforeBroadcastEvents int,
	beforeUIEvents int,
) ([]ServerMessage, bool, error) {
	name, err := h.runtime.HandleTrigger(ctx, trigger)
	if err != nil {
		return nil, true, err
	}
	result := ServerMessage{
		ScenarioStart: name,
		Notification:  "Scenario started: " + name,
	}
	if result.ScenarioStart == "multi_window" {
		h.captureMultiWindowResume(deviceID, activeName)
	}
	responses := h.commandResponses(ctx, cmd, result)
	responses = h.appendScenarioUICommandSideEffects(
		ctx, deviceID, cmd, result, beforeRoutes, beforeBroadcastEvents, beforeUIEvents, responses,
	)
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

func (h *StreamHandler) renderTerminalUIAction(deviceID, componentID, action, value string) (*UIUpdate, bool) {
	if strings.TrimSpace(componentID) == "" {
		return nil, false
	}
	line := fmt.Sprintf("[ui_action] %s %s = %s\n", componentID, action, value)
	sessionID, ok := h.replSessionIDForDevice(deviceID)
	if !ok {
		return nil, false
	}
	_, _ = h.replSessions.AppendOutput(sessionID, line)
	return h.terminalOutputUpdate(sessionID), true
}

func (h *StreamHandler) pollTerminalOutput(deviceID string, force bool) (*ServerMessage, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, nil
	}

	sessionID, ok := h.replSessionIDForDevice(deviceID)
	if !ok {
		return nil, nil
	}

	chunk, err := h.replSessions.ReadAvailable(sessionID, 4096)
	if err != nil {
		return nil, err
	}
	if len(chunk) == 0 {
		emit, emitErr := h.replSessions.ShouldEmitUpdate(sessionID, force, h.nowUTC(), h.terminalUIInterval)
		if emitErr != nil || !emit {
			return nil, nil
		}
		return &ServerMessage{
			UpdateUI: h.terminalOutputUpdate(sessionID),
		}, nil
	}

	emit, emitErr := h.replSessions.ShouldEmitUpdate(sessionID, force, h.nowUTC(), h.terminalUIInterval)
	if emitErr != nil || !emit {
		return nil, nil
	}
	return &ServerMessage{
		UpdateUI: h.terminalOutputUpdate(sessionID),
	}, nil
}

func (h *StreamHandler) terminalOutputUpdate(sessionID string) *UIUpdate {
	output, err := h.replSessions.MarkFlushed(sessionID, h.nowUTC())
	if err != nil {
		output = ""
	}
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

func (h *StreamHandler) prepareOutboundUI(targetDeviceID string, msg ServerMessage) (ServerMessage, error) {
	targetDeviceID = strings.TrimSpace(targetDeviceID)
	activationID := targetDeviceID
	if activationID == "" {
		activationID = "main"
	}

	if msg.SetUI != nil {
		rewritten, componentIDs, err := rewriteDescriptorIDsForActivation(*msg.SetUI, activationID)
		if err != nil {
			return ServerMessage{}, err
		}
		msg.SetUI = &rewritten
		mainActivationID := scopedActivationFromComponentIDs(componentIDs, activationID)
		if priorActivationID := h.swapMainUIActivation(targetDeviceID, mainActivationID); priorActivationID != "" && priorActivationID != mainActivationID {
			h.uiOwners.ForgetActivation(targetDeviceID, priorActivationID)
		}
		h.uiOwners.RecordSetUI(targetDeviceID, mainActivationID, componentIDs)
	}
	if msg.UpdateUI != nil {
		rewritten, err := rewriteAndValidateUpdateUI(targetDeviceID, msg.UpdateUI, nil)
		if err != nil {
			return ServerMessage{}, err
		}
		msg.UpdateUI = rewritten
		if _, activationID, _, ok := parseScopedComponentID(rewritten.ComponentID); ok {
			_, componentIDs, rewriteErr := rewriteDescriptorIDsForActivation(rewritten.Node, activationID)
			if rewriteErr == nil {
				h.uiOwners.RecordUpdate(targetDeviceID, activationID, componentIDs)
			}
		}
	}
	return msg, nil
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

func scopedActivationFromComponentIDs(componentIDs []string, fallback string) string {
	for _, componentID := range componentIDs {
		if _, activationID, _, ok := parseScopedComponentID(componentID); ok {
			return activationID
		}
	}
	return strings.TrimSpace(fallback)
}

func (h *StreamHandler) swapMainUIActivation(deviceID, activationID string) string {
	return h.uiSession.SwapMainUIActivation(deviceID, activationID)
}
