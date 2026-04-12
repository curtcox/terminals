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
