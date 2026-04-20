package transport

import (
	"strings"

	"github.com/curtcox/terminals/terminal_server/internal/chat"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

// chatRoom returns the process-global chat room via the scenario package,
// which owns the singleton.
func (h *StreamHandler) chatRoom() *chat.Room {
	return scenario.SharedRoom()
}

// chatEntryUI builds the initial SetUI for a device entering the chat
// scenario. A device that has not declared a name sees the identity view;
// a device with a known name sees the full chat view.
func (h *StreamHandler) chatEntryUI(deviceID string) ui.Descriptor {
	room := h.chatRoom()
	name := room.Name(deviceID)
	if strings.TrimSpace(name) == "" {
		return ui.ChatIdentityView(deviceID)
	}
	return ui.ChatView(deviceID, name, chatUIMessages(room.Messages()))
}

// chatMessagesUpdate returns an UpdateUI patch for the chat message list.
func chatMessagesUpdate(messages []chat.Message) *UIUpdate {
	return &UIUpdate{
		ComponentID: ui.ChatMessagesComponentID,
		Node:        ui.ChatMessageList(chatUIMessages(messages)),
	}
}

func chatUIMessages(messages []chat.Message) []ui.ChatMessage {
	out := make([]ui.ChatMessage, 0, len(messages))
	for _, msg := range messages {
		out = append(out, ui.ChatMessage{
			ID:       msg.ID,
			DeviceID: msg.DeviceID,
			Name:     msg.Name,
			Text:     msg.Text,
			At:       msg.At,
		})
	}
	return out
}

// chatActiveDevices returns device IDs currently running the chat scenario.
// If the engine is unavailable it falls back to the room participant set,
// which is updated on Start/Stop.
func (h *StreamHandler) chatActiveDevices() []string {
	if h.runtime != nil && h.runtime.Engine != nil {
		active := h.runtime.Engine.ActiveSnapshot()
		out := make([]string, 0, len(active))
		for deviceID, name := range active {
			if name == "chat" {
				out = append(out, deviceID)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return h.chatRoom().Participants()
}

// chatBroadcastMessagesUpdate returns ServerMessages that deliver a refreshed
// chat message list to every participant. The requesting device gets a local
// UpdateUI; other participants are reached via RelayToDeviceID.
func (h *StreamHandler) chatBroadcastMessagesUpdate(requesterDeviceID string) []ServerMessage {
	requester := strings.TrimSpace(requesterDeviceID)
	room := h.chatRoom()
	messages := room.Messages()
	peers := h.chatActiveDevices()
	// Include the requester even if not in the engine snapshot yet.
	seen := map[string]struct{}{}
	if requester != "" {
		seen[requester] = struct{}{}
	}
	responses := make([]ServerMessage, 0, len(peers)+1)
	responses = append(responses, ServerMessage{UpdateUI: chatMessagesUpdate(messages)})
	for _, deviceID := range peers {
		deviceID = strings.TrimSpace(deviceID)
		if deviceID == "" || deviceID == requester {
			continue
		}
		if _, ok := seen[deviceID]; ok {
			continue
		}
		seen[deviceID] = struct{}{}
		responses = append(responses, ServerMessage{
			UpdateUI:        chatMessagesUpdate(messages),
			RelayToDeviceID: deviceID,
		})
	}
	return responses
}

// BroadcastChatMessagesUpdate pushes a refreshed chat message list to every
// connected device currently registered on the session relay. It is intended
// for out-of-band posters (e.g. the admin HTTP endpoint) that need to nudge
// clients whose scenario engine shows chat as active.
func BroadcastChatMessagesUpdate() {
	messages := scenario.SharedRoom().Messages()
	update := chatMessagesUpdate(messages)
	participants := scenario.SharedRoom().Participants()
	for _, deviceID := range participants {
		deviceID = strings.TrimSpace(deviceID)
		if deviceID == "" {
			continue
		}
		_ = globalSessionRelayRegistry.Relay(deviceID, ServerMessage{UpdateUI: update})
	}
}

// handleChatInput processes chat-related input actions. Returns the messages
// to emit and a bool indicating whether the input was consumed here. When
// not consumed, the caller proceeds with existing handling.
func (h *StreamHandler) handleChatInput(deviceID, componentID, action, value string) ([]ServerMessage, bool) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, false
	}
	// Only handle when chat is the active scenario for this device.
	if h.activeScenarioName(deviceID) != "chat" {
		return nil, false
	}
	room := h.chatRoom()
	action = strings.ToLower(strings.TrimSpace(action))
	componentID = strings.TrimSpace(componentID)

	switch {
	case componentID == ui.ChatNameInputID && (action == "submit" || action == ui.ChatActionSend):
		name := strings.TrimSpace(value)
		if name == "" {
			return nil, true
		}
		room.SetName(deviceID, name)
		messages := room.Messages()
		view := ui.ChatView(deviceID, name, chatUIMessages(messages))
		return []ServerMessage{{SetUI: &view}}, true

	case componentID == ui.ChatMessageInputID && action == "submit":
		text := strings.TrimSpace(value)
		if text == "" {
			return nil, true
		}
		name := room.Name(deviceID)
		if _, ok := room.Post(deviceID, name, text); !ok {
			return nil, true
		}
		return h.chatBroadcastMessagesUpdate(deviceID), true

	case action == ui.ChatActionChangeName:
		view := ui.ChatIdentityView(deviceID)
		return []ServerMessage{{SetUI: &view}}, true
	}

	return nil, false
}
