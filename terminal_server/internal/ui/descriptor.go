package ui

import "strconv"

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
	}), New("scroll", map[string]string{
		"id": "terminal_output_scroll",
	}, New("text", map[string]string{
		"id":    "terminal_output",
		"value": output,
		"style": "monospace",
		"color": "#E8E8E8",
	})), New("text_input", map[string]string{
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
	}), New("button", map[string]string{
		"id":     "multi_window_end",
		"label":  "End multi-window",
		"action": "multi_window_end",
	}), New("grid", map[string]string{
		"id":      "multi_window_grid",
		"columns": strconv.Itoa(columns),
	}, gridChildren...), GlobalOverlaySlot())
}
