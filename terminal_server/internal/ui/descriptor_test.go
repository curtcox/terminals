package ui

import "testing"

func TestHelloWorld(t *testing.T) {
	d := HelloWorld("Kitchen Chromebook")
	if d.Type != "stack" {
		t.Fatalf("Type = %q, want stack", d.Type)
	}
	if len(d.Children) != 1 {
		t.Fatalf("children = %d, want 1", len(d.Children))
	}
	if d.Children[0].Type != "text" {
		t.Fatalf("child type = %q, want text", d.Children[0].Type)
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
