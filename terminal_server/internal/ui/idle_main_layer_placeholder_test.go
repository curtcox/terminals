package ui

import "testing"

func TestIdleMainLayerPlaceholderShape(t *testing.T) {
	root := IdleMainLayerPlaceholder()
	if root.Type != "stack" {
		t.Fatalf("root type = %q", root.Type)
	}
	if root.ID != "__runtime.main_placeholder.root" {
		t.Fatalf("root id = %q", root.ID)
	}
	if root.Props["client_chrome"] != "hidden" {
		t.Fatalf("client_chrome = %q", root.Props["client_chrome"])
	}
	if len(root.Children) != 1 {
		t.Fatalf("children len = %d", len(root.Children))
	}
	child := root.Children[0]
	if child.Type != "text" || child.Props["value"] != "Awaiting server UI" {
		t.Fatalf("child = %#v", child)
	}
}
