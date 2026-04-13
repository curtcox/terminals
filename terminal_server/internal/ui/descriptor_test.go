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
