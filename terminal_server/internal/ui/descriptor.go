package ui

import (
	"strconv"
	"strings"
	"time"
)

const (
	// GlobalOverlayComponentID is the stable component id used for transient overlays.
	GlobalOverlayComponentID = "global_overlay"
)

// Descriptor is a generic server-driven UI node.
type Descriptor struct {
	ID       string            `json:"id,omitempty"`
	Type     string            `json:"type"`
	Props    map[string]string `json:"props,omitempty"`
	Children []Descriptor      `json:"children,omitempty"`
}

// New builds a descriptor node with type and optional props.
func New(nodeType string, props map[string]string, children ...Descriptor) Descriptor {
	if props == nil {
		props = map[string]string{}
	}
	return Descriptor{
		Type:     nodeType,
		Props:    props,
		Children: children,
	}
}

// HelloWorld returns a minimal initial screen used at connect time.
func HelloWorld(deviceName string) Descriptor {
	return New("stack", map[string]string{
		"background": "#101418",
	}, New("text", map[string]string{
		"value": "Connected: " + deviceName,
		"style": "headline",
		"color": "#E7F0F7",
	}), GlobalOverlaySlot())
}

// TerminalView returns a simple server-driven terminal layout.
func TerminalView(deviceID string) Descriptor {
	return TerminalViewWithOutput(deviceID, "")
}

// TerminalViewWithOutput returns a simple server-driven terminal layout with output.
func TerminalViewWithOutput(deviceID, output string) Descriptor {
	return New("stack", map[string]string{
		"id":         "terminal_root",
		"background": "#000000",
	}, New("text", map[string]string{
		"id":    "terminal_banner",
		"value": "Terminal session on " + deviceID,
		"style": "monospace",
		"color": "#39FF14",
	}), New("expand", nil, New("scroll", map[string]string{
		"id": "terminal_output_scroll",
	}, New("text", map[string]string{
		"id":    "terminal_output",
		"value": output,
		"style": "monospace",
		"color": "#E8E8E8",
	}))), New("text_input", map[string]string{
		"id":          "terminal_input",
		"placeholder": "Type command and press enter",
		"autofocus":   "true",
	}), New("button", map[string]string{
		"id":     "terminal_refresh_button",
		"label":  "Refresh",
		"action": "terminal_refresh",
	}), GlobalOverlaySlot())
}

// TerminalOutputPatch returns a descriptor for updating terminal output text only.
func TerminalOutputPatch(output string) Descriptor {
	return New("text", map[string]string{
		"id":    "terminal_output",
		"value": output,
		"style": "monospace",
		"color": "#E8E8E8",
	})
}

// GlobalOverlaySlot returns an empty overlay node used as a stable update target.
func GlobalOverlaySlot() Descriptor {
	return New("overlay", map[string]string{
		"id": GlobalOverlayComponentID,
	})
}

// PAReceiverOverlayPatch returns a global overlay descriptor for PA receive indicators.
func PAReceiverOverlayPatch(message string) Descriptor {
	return New("overlay", map[string]string{
		"id": GlobalOverlayComponentID,
	}, New("stack", map[string]string{
		"background": "#6A1B1A",
	}, New("text", map[string]string{
		"value": message,
		"style": "headline",
		"color": "#FFFFFF",
	})))
}

// VoiceAssistantResponseView renders a focused voice-assistant response screen.
func VoiceAssistantResponseView(deviceID, prompt, response string) Descriptor {
	prompt = "You said: " + prompt
	if prompt == "You said: " {
		prompt = "You said: (no prompt captured)"
	}
	if response == "" {
		response = "No response available"
	}

	return New("stack", map[string]string{
		"id":         "voice_assistant_response_root",
		"background": "#0D1220",
	}, New("text", map[string]string{
		"id":    "voice_assistant_response_title",
		"value": "Voice assistant",
		"style": "headline",
		"color": "#E7F0F7",
	}), New("text", map[string]string{
		"id":    "voice_assistant_response_device",
		"value": "Device: " + deviceID,
		"style": "body",
		"color": "#9CB4CF",
	}), New("stack", map[string]string{
		"id":         "voice_assistant_response_card",
		"background": "#131D33",
	}, New("text", map[string]string{
		"id":    "voice_assistant_response_prompt",
		"value": prompt,
		"style": "body",
		"color": "#CFE1F2",
	}), New("text", map[string]string{
		"id":    "voice_assistant_response_text",
		"value": response,
		"style": "headline",
		"color": "#FFFFFF",
	})), GlobalOverlaySlot())
}

// VoiceAssistantResponsePatch returns a global overlay patch with assistant text.
func VoiceAssistantResponsePatch(response string) Descriptor {
	if response == "" {
		response = "No response available"
	}
	return New("overlay", map[string]string{
		"id": GlobalOverlayComponentID,
	}, New("stack", map[string]string{
		"id":         "voice_assistant_response_overlay",
		"background": "#131D33",
	}, New("text", map[string]string{
		"id":    "voice_assistant_response_overlay_title",
		"value": "Assistant reply",
		"style": "headline",
		"color": "#E7F0F7",
	}), New("text", map[string]string{
		"id":    "voice_assistant_response_overlay_text",
		"value": response,
		"style": "body",
		"color": "#FFFFFF",
	})))
}

// InternalVideoCallView renders a two-surface video call layout with controls.
func InternalVideoCallView(sourceDeviceID, targetDeviceID string) Descriptor {
	remoteTrackID := "route:" + targetDeviceID + "|" + sourceDeviceID + "|video"
	localTrackID := "route:" + sourceDeviceID + "|" + targetDeviceID + "|video"
	return New("stack", map[string]string{
		"id":         "internal_video_call_root",
		"background": "#070B12",
	}, New("text", map[string]string{
		"id":    "internal_video_call_title",
		"value": "Video call with " + targetDeviceID,
		"style": "headline",
		"color": "#E7F0F7",
	}), New("video_surface", map[string]string{
		"id":       "internal_video_call_remote_video",
		"track_id": remoteTrackID,
	}), New("video_surface", map[string]string{
		"id":       "internal_video_call_local_preview",
		"track_id": localTrackID,
	}), New("button", map[string]string{
		"id":     "internal_video_call_hangup",
		"label":  "Hang up",
		"action": "internal_video_call_end",
	}), GlobalOverlaySlot())
}

// MultiWindowView renders an adaptive camera-grid descriptor for a viewer device.
func MultiWindowView(viewerDeviceID string, peerDeviceIDs []string, focusedPeerID string) Descriptor {
	gridChildren := make([]Descriptor, 0, len(peerDeviceIDs))
	for _, peerDeviceID := range peerDeviceIDs {
		if peerDeviceID == "" {
			continue
		}
		videoTrackID := "route:" + peerDeviceID + "|" + viewerDeviceID + "|video"
		focusLabel := "Hear " + peerDeviceID
		if focusedPeerID == peerDeviceID {
			focusLabel = "Hearing " + peerDeviceID
		}
		gridChildren = append(gridChildren, New("stack", map[string]string{
			"id":         "multi_window_tile_" + peerDeviceID,
			"background": "#111111",
		}, New("text", map[string]string{
			"id":    "multi_window_label_" + peerDeviceID,
			"value": peerDeviceID,
			"style": "headline",
			"color": "#FFFFFF",
		}), New("video_surface", map[string]string{
			"id":       "multi_window_video_" + peerDeviceID,
			"track_id": videoTrackID,
		}), New("button", map[string]string{
			"id":     "multi_window_focus_" + peerDeviceID,
			"label":  focusLabel,
			"action": "multi_window_focus:" + peerDeviceID,
		})))
	}

	var columns int
	switch count := len(gridChildren); {
	case count <= 1:
		columns = 1
	case count <= 4:
		columns = 2
	case count <= 9:
		columns = 3
	default:
		columns = 4
	}

	return New("stack", map[string]string{
		"id":         "multi_window_root",
		"background": "#090C10",
	}, New("text", map[string]string{
		"id":    "multi_window_title",
		"value": "Multi-window view",
		"style": "headline",
		"color": "#E7F0F7",
	}), New("grid", map[string]string{
		"id":      "multi_window_grid",
		"columns": strconv.Itoa(columns),
	}, gridChildren...), New("button", map[string]string{
		"id":     "multi_window_end",
		"label":  "End multi-window",
		"action": "multi_window_end",
	}), GlobalOverlaySlot())
}

// ChatMessage is one rendered chat log entry. Time is formatted as HH:MM:SS
// UTC in the view; the caller retains full timestamps.
type ChatMessage struct {
	ID       string
	DeviceID string
	Name     string
	Text     string
	At       time.Time
}

// ChatComponentIDs are the stable component ids used to patch the chat view.
const (
	ChatRootComponentID     = "chat_root"
	ChatMessagesComponentID = "chat_messages"
	ChatNameInputID         = "chat_name_input"
	ChatMessageInputID      = "chat_message_input"
	ChatHeaderComponentID   = "chat_header"
)

// Chat action strings recognized by the transport layer.
const (
	ChatActionSend       = "chat_send"
	ChatActionChangeName = "chat_change_name"
	ChatActionLeave      = "chat_leave"
)

// ChatIdentityView renders the name-entry step shown before a device has
// declared a chat identity.
func ChatIdentityView(deviceID string) Descriptor {
	return New("stack", map[string]string{
		"id":         ChatRootComponentID,
		"background": "#0B1622",
	}, New("text", map[string]string{
		"id":    "chat_identity_title",
		"value": "Chat",
		"style": "headline",
		"color": "#E7F0F7",
	}), New("text", map[string]string{
		"id":    "chat_identity_prompt",
		"value": "Choose a display name to join the chat.",
		"style": "body",
		"color": "#9CB4CF",
	}), New("text", map[string]string{
		"id":    "chat_identity_device",
		"value": "Terminal: " + deviceID,
		"style": "body",
		"color": "#6D839C",
	}), New("text_input", map[string]string{
		"id":          ChatNameInputID,
		"placeholder": "Your name",
		"autofocus":   "true",
	}), New("button", map[string]string{
		"id":     "chat_leave_from_identity",
		"label":  "Leave chat",
		"action": ChatActionLeave,
	}), GlobalOverlaySlot())
}

// ChatView renders the full chat screen for a joined device.
func ChatView(deviceID, displayName string, messages []ChatMessage) Descriptor {
	header := "Chat — " + displayName + " on " + deviceID
	return New("stack", map[string]string{
		"id":         ChatRootComponentID,
		"background": "#0B1622",
	}, New("text", map[string]string{
		"id":    ChatHeaderComponentID,
		"value": header,
		"style": "headline",
		"color": "#E7F0F7",
	}), New("expand", nil, New("scroll", map[string]string{
		"id":        "chat_messages_scroll",
		"direction": "vertical",
	}, ChatMessageList(messages))), New("text_input", map[string]string{
		"id":          ChatMessageInputID,
		"placeholder": "Type a message and press enter",
		"autofocus":   "true",
	}), New("button", map[string]string{
		"id":     "chat_change_name_button",
		"label":  "Change name",
		"action": ChatActionChangeName,
	}), New("button", map[string]string{
		"id":     "chat_leave_button",
		"label":  "Leave chat",
		"action": ChatActionLeave,
	}), GlobalOverlaySlot())
}

// ChatMessageList renders the scrollable messages column.
func ChatMessageList(messages []ChatMessage) Descriptor {
	children := make([]Descriptor, 0, len(messages)+1)
	if len(messages) == 0 {
		children = append(children, New("text", map[string]string{
			"id":    "chat_empty",
			"value": "No messages yet. Say hi!",
			"style": "body",
			"color": "#6D839C",
		}))
	}
	for _, msg := range messages {
		children = append(children, chatMessageRow(msg))
	}
	return New("stack", map[string]string{
		"id": ChatMessagesComponentID,
	}, children...)
}

// ChatHeaderPatch returns an UpdateUI node for the chat header line.
func ChatHeaderPatch(deviceID, displayName string) Descriptor {
	return New("text", map[string]string{
		"id":    ChatHeaderComponentID,
		"value": "Chat — " + displayName + " on " + deviceID,
		"style": "headline",
		"color": "#E7F0F7",
	})
}

func chatMessageRow(msg ChatMessage) Descriptor {
	name := strings.TrimSpace(msg.Name)
	if name == "" {
		name = msg.DeviceID
	}
	timestamp := msg.At.UTC().Format("15:04:05")
	meta := timestamp + "  " + name + " (" + msg.DeviceID + ")"
	return New("stack", map[string]string{
		"id":         "chat_message_" + msg.ID,
		"background": "#11202F",
	}, New("text", map[string]string{
		"id":    "chat_message_meta_" + msg.ID,
		"value": meta,
		"style": "body",
		"color": "#9CB4CF",
	}), New("text", map[string]string{
		"id":    "chat_message_text_" + msg.ID,
		"value": msg.Text,
		"style": "body",
		"color": "#E7F0F7",
	}))
}

// PhotoFrameView renders a fullscreen ambient photo layout with keep-awake hints.
func PhotoFrameView(photoURL, caption string, index, total int) Descriptor {
	if photoURL == "" {
		photoURL = "https://picsum.photos/1920/1080?grayscale"
	}
	if caption == "" {
		caption = "Photo frame"
	}
	if total < 1 {
		total = 1
	}
	if index < 0 {
		index = 0
	}
	if index >= total {
		index = total - 1
	}
	progress := strconv.Itoa(index+1) + " / " + strconv.Itoa(total)

	return New("stack", map[string]string{
		"id":         "photo_frame_root",
		"background": "#000000",
	}, New("keep_awake", map[string]string{
		"id":      "photo_frame_keep_awake",
		"enabled": "true",
	}, New("fullscreen", map[string]string{
		"id":      "photo_frame_fullscreen",
		"enabled": "true",
	}, New("image", map[string]string{
		"id":  "photo_frame_image",
		"url": photoURL,
	}))), New("stack", map[string]string{
		"id":         "photo_frame_overlay",
		"background": "#00000088",
	}, New("text", map[string]string{
		"id":    "photo_frame_caption",
		"value": caption,
		"style": "headline",
		"color": "#FFFFFF",
	}), New("text", map[string]string{
		"id":    "photo_frame_progress",
		"value": progress,
		"style": "body",
		"color": "#D7D7D7",
	})), GlobalOverlaySlot())
}
