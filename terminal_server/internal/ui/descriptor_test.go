package ui

import "testing"

func TestHelloWorld(t *testing.T) {
	d := HelloWorld("Kitchen Chromebook")
	if d.Type != "stack" {
		t.Fatalf("Type = %q, want stack", d.Type)
	}
	if len(d.Children) != 2 {
		t.Fatalf("children = %d, want 2", len(d.Children))
	}
	if d.Children[0].Type != "text" {
		t.Fatalf("child type = %q, want text", d.Children[0].Type)
	}
	if d.Children[1].Type != "overlay" || d.Children[1].Props["id"] != GlobalOverlayComponentID {
		t.Fatalf("second child should be global overlay slot, got %+v", d.Children[1])
	}
}

func TestTerminalOutputPatch(t *testing.T) {
	d := TerminalOutputPatch("line1\nline2")
	if d.Type != "text" {
		t.Fatalf("Type = %q, want text", d.Type)
	}
	if d.Props["id"] != "terminal_output" {
		t.Fatalf("id = %q, want terminal_output", d.Props["id"])
	}
	if d.Props["value"] != "line1\nline2" {
		t.Fatalf("value = %q, want line1\\nline2", d.Props["value"])
	}
}

func TestTerminalViewIncludesRefreshButton(t *testing.T) {
	d := TerminalViewWithOutput("device-1", "hello")
	if d.Type != "stack" {
		t.Fatalf("Type = %q, want stack", d.Type)
	}
	if len(d.Children) < 4 {
		t.Fatalf("children = %d, want at least 4", len(d.Children))
	}

	if d.Children[1].Type != "expand" {
		t.Fatalf("child[1].Type = %q, want expand", d.Children[1].Type)
	}
	if len(d.Children[1].Children) != 1 {
		t.Fatalf("expand child count = %d, want 1", len(d.Children[1].Children))
	}
	if d.Children[1].Children[0].Type != "scroll" {
		t.Fatalf("expand child type = %q, want scroll", d.Children[1].Children[0].Type)
	}
	if d.Children[1].Children[0].Props["id"] != "terminal_output_scroll" {
		t.Fatalf("scroll id = %q, want terminal_output_scroll", d.Children[1].Children[0].Props["id"])
	}

	var found bool
	for _, child := range d.Children {
		if child.Type != "button" {
			continue
		}
		if child.Props["id"] == "terminal_refresh_button" &&
			child.Props["label"] == "Refresh" &&
			child.Props["action"] == "terminal_refresh" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected terminal refresh button descriptor")
	}
}

func TestPAReceiverOverlayPatch(t *testing.T) {
	d := PAReceiverOverlayPatch("PA from device-1")
	if d.Type != "overlay" {
		t.Fatalf("Type = %q, want overlay", d.Type)
	}
	if d.Props["id"] != GlobalOverlayComponentID {
		t.Fatalf("id = %q, want %q", d.Props["id"], GlobalOverlayComponentID)
	}
	if len(d.Children) == 0 || d.Children[0].Type != "stack" {
		t.Fatalf("expected stack child in PA overlay patch, got %+v", d.Children)
	}
	if len(d.Children[0].Children) == 0 || d.Children[0].Children[0].Type != "text" {
		t.Fatalf("expected text child in PA overlay patch, got %+v", d.Children[0].Children)
	}
	if d.Children[0].Children[0].Props["value"] != "PA from device-1" {
		t.Fatalf("overlay text value = %q, want PA from device-1", d.Children[0].Children[0].Props["value"])
	}
}

func TestVoiceAssistantResponseViewEmbedsPromptAndResponse(t *testing.T) {
	d := VoiceAssistantResponseView("device-1", "what is the weather", "It is sunny in Test City")
	if d.Type != "stack" {
		t.Fatalf("Type = %q, want stack", d.Type)
	}

	var promptValue string
	var responseValue string
	for _, child := range d.Children {
		if child.Props["id"] != "voice_assistant_response_card" {
			continue
		}
		for _, grandchild := range child.Children {
			switch grandchild.Props["id"] {
			case "voice_assistant_response_prompt":
				promptValue = grandchild.Props["value"]
			case "voice_assistant_response_text":
				responseValue = grandchild.Props["value"]
			}
		}
	}
	if promptValue != "You said: what is the weather" {
		t.Fatalf("prompt = %q, want user prompt", promptValue)
	}
	if responseValue != "It is sunny in Test City" {
		t.Fatalf("response = %q, want assistant response", responseValue)
	}
}

func TestVoiceAssistantResponsePatchEmbedsResponse(t *testing.T) {
	d := VoiceAssistantResponsePatch("It is sunny in Test City")
	if d.Type != "overlay" {
		t.Fatalf("Type = %q, want overlay", d.Type)
	}
	if d.Props["id"] != GlobalOverlayComponentID {
		t.Fatalf("id = %q, want %q", d.Props["id"], GlobalOverlayComponentID)
	}
	if len(d.Children) == 0 || len(d.Children[0].Children) < 2 {
		t.Fatalf("overlay shape unexpected: %+v", d)
	}
	if got := d.Children[0].Children[1].Props["value"]; got != "It is sunny in Test City" {
		t.Fatalf("overlay response = %q, want assistant response", got)
	}
}

func TestMultiWindowViewUsesAdaptiveGridAndFocusActions(t *testing.T) {
	d := MultiWindowView("viewer", []string{"cam-a", "cam-b", "cam-c"}, "cam-b")
	if d.Type != "stack" {
		t.Fatalf("Type = %q, want stack", d.Type)
	}
	if len(d.Children) < 3 {
		t.Fatalf("children = %d, want at least 3", len(d.Children))
	}
	grid := d.Children[1]
	if grid.Type != "grid" {
		t.Fatalf("grid type = %q, want grid", grid.Type)
	}
	if grid.Props["columns"] != "2" {
		t.Fatalf("grid columns = %q, want 2", grid.Props["columns"])
	}
	if len(grid.Children) != 3 {
		t.Fatalf("grid children = %d, want 3", len(grid.Children))
	}

	foundFocusAction := false
	foundFocusedLabel := false
	for _, tile := range grid.Children {
		if tile.Type != "stack" || len(tile.Children) < 3 {
			continue
		}
		button := tile.Children[2]
		if button.Type != "button" {
			continue
		}
		if button.Props["action"] == "multi_window_focus:cam-b" {
			foundFocusAction = true
			if button.Props["label"] == "Hearing cam-b" {
				foundFocusedLabel = true
			}
		}
	}
	if !foundFocusAction {
		t.Fatalf("expected focus button action multi_window_focus:cam-b")
	}
	if !foundFocusedLabel {
		t.Fatalf("expected focused peer button label Hearing cam-b")
	}
}

func TestPhotoFrameViewIncludesKeepAwakeAndFullscreenImage(t *testing.T) {
	d := PhotoFrameView("https://example.invalid/photo-1.jpg", "Kitchen memories", 1, 4)
	if d.Type != "stack" {
		t.Fatalf("Type = %q, want stack", d.Type)
	}
	if d.Props["id"] != "photo_frame_root" {
		t.Fatalf("id = %q, want photo_frame_root", d.Props["id"])
	}
	if len(d.Children) < 3 {
		t.Fatalf("children = %d, want at least 3", len(d.Children))
	}

	keepAwake := d.Children[0]
	if keepAwake.Type != "keep_awake" || keepAwake.Props["enabled"] != "true" {
		t.Fatalf("first child = %+v, want keep_awake enabled", keepAwake)
	}
	if len(keepAwake.Children) != 1 || keepAwake.Children[0].Type != "fullscreen" {
		t.Fatalf("keep_awake children = %+v, want fullscreen", keepAwake.Children)
	}
	fullscreen := keepAwake.Children[0]
	if len(fullscreen.Children) != 1 || fullscreen.Children[0].Type != "image" {
		t.Fatalf("fullscreen children = %+v, want image", fullscreen.Children)
	}
	if got := fullscreen.Children[0].Props["url"]; got != "https://example.invalid/photo-1.jpg" {
		t.Fatalf("image url = %q, want provided photo url", got)
	}

	overlay := d.Children[1]
	if overlay.Type != "stack" || overlay.Props["id"] != "photo_frame_overlay" {
		t.Fatalf("overlay = %+v, want photo_frame_overlay stack", overlay)
	}
	if len(overlay.Children) != 2 {
		t.Fatalf("overlay child count = %d, want 2", len(overlay.Children))
	}
	if got := overlay.Children[0].Props["value"]; got != "Kitchen memories" {
		t.Fatalf("caption = %q, want Kitchen memories", got)
	}
	if got := overlay.Children[1].Props["value"]; got != "2 / 4" {
		t.Fatalf("progress = %q, want 2 / 4", got)
	}

	if d.Children[2].Type != "overlay" || d.Children[2].Props["id"] != GlobalOverlayComponentID {
		t.Fatalf("third child should be global overlay slot, got %+v", d.Children[2])
	}
}
