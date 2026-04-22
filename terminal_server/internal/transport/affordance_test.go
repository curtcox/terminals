package transport

import (
	"strings"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestWithCornerAffordanceAddsDefaultScopedButton(t *testing.T) {
	root := ui.New("stack", map[string]string{"id": "root"}, ui.New("text", map[string]string{
		"id":    "existing",
		"value": "hello",
	}))

	got := withCornerAffordance(root, "device-1")

	corner := findNodeByID(&got, "act:device-1/__affordance.corner__")
	if corner == nil {
		t.Fatalf("expected scoped corner affordance node")
	}
	if corner.Type != "button" {
		t.Fatalf("corner type = %q, want button", corner.Type)
	}
	if corner.Props["action"] != "corner.open" {
		t.Fatalf("corner action = %q, want corner.open", corner.Props["action"])
	}
	if corner.Props["corner"] != "bottom-right" {
		t.Fatalf("corner placement = %q, want bottom-right", corner.Props["corner"])
	}
	if corner.Props["visible"] != "true" {
		t.Fatalf("corner visible = %q, want true", corner.Props["visible"])
	}
	if corner.Props["min_hit_dp"] != "44" {
		t.Fatalf("corner min_hit_dp = %q, want 44", corner.Props["min_hit_dp"])
	}
	if findNodeByID(&got, "existing") == nil {
		t.Fatalf("existing descriptor subtree should be preserved")
	}
}

func TestWithCornerAffordanceNoopWhenAlreadyPresent(t *testing.T) {
	root := ui.New("stack", map[string]string{"id": "root"}, ui.New("button", map[string]string{
		"id":     "act:device-1/__affordance.corner__",
		"action": "corner.open",
	}))

	got := withCornerAffordance(root, "device-1")
	count := countNodesWithIDPrefix(&got, "act:device-1/__affordance.corner__")
	if count != 1 {
		t.Fatalf("corner affordance node count = %d, want 1", count)
	}
}

func findNodeByID(node *ui.Descriptor, id string) *ui.Descriptor {
	if node == nil {
		return nil
	}
	if strings.TrimSpace(node.ID) == id || strings.TrimSpace(node.Props["id"]) == id {
		return node
	}
	for i := range node.Children {
		if found := findNodeByID(&node.Children[i], id); found != nil {
			return found
		}
	}
	return nil
}

func countNodesWithIDPrefix(node *ui.Descriptor, prefix string) int {
	if node == nil {
		return 0
	}
	count := 0
	nodeID := strings.TrimSpace(node.ID)
	if nodeID == "" {
		nodeID = strings.TrimSpace(node.Props["id"])
	}
	if strings.HasPrefix(nodeID, prefix) {
		count++
	}
	for i := range node.Children {
		count += countNodesWithIDPrefix(&node.Children[i], prefix)
	}
	return count
}
