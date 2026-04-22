package transport

import (
	"strings"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestRewriteDescriptorIDsForActivationScopesLogicalIDs(t *testing.T) {
	root := ui.New("stack", map[string]string{"id": "terminal_root"},
		ui.New("text", map[string]string{"id": "terminal_output", "value": "hello"}),
		ui.New("button", map[string]string{"id": "terminal_refresh", "action": "terminal_refresh"}),
	)

	got, ids, err := rewriteDescriptorIDsForActivation(root, "device-1")
	if err != nil {
		t.Fatalf("rewriteDescriptorIDsForActivation() error = %v", err)
	}

	if got.Props["id"] != "act:device-1/terminal_root" {
		t.Fatalf("root id = %q, want scoped", got.Props["id"])
	}
	if findNodeByID(&got, "act:device-1/terminal_output") == nil {
		t.Fatalf("expected scoped output node id")
	}
	if findNodeByID(&got, "act:device-1/terminal_refresh") == nil {
		t.Fatalf("expected scoped refresh node id")
	}
	if len(ids) != 3 {
		t.Fatalf("len(ids) = %d, want 3", len(ids))
	}
}

func TestRewriteDescriptorIDsForActivationRejectsInvalidAndDuplicateIDs(t *testing.T) {
	t.Run("rejects invalid logical id with delimiter", func(t *testing.T) {
		root := ui.New("stack", map[string]string{"id": "terminal:root"})
		_, _, err := rewriteDescriptorIDsForActivation(root, "device-1")
		if err == nil || !strings.Contains(err.Error(), "logical") {
			t.Fatalf("expected logical-id validation error, got %v", err)
		}
	})

	t.Run("rejects duplicate scoped ids", func(t *testing.T) {
		root := ui.New("stack", map[string]string{"id": "root"},
			ui.New("text", map[string]string{"id": "same"}),
			ui.New("button", map[string]string{"id": "same", "action": "noop"}),
		)
		_, _, err := rewriteDescriptorIDsForActivation(root, "device-1")
		if err == nil || !strings.Contains(err.Error(), "duplicate") {
			t.Fatalf("expected duplicate-id error, got %v", err)
		}
	})
}

func TestRewriteAndValidateUpdateUIRejectsUnknownTargets(t *testing.T) {
	tracker := newUIActionOwnershipTracker()
	root := ui.New("stack", map[string]string{"id": "root"}, ui.New("text", map[string]string{"id": "terminal_output"}))
	rewritten, ids, err := rewriteDescriptorIDsForActivation(root, "device-1")
	if err != nil {
		t.Fatalf("rewriteDescriptorIDsForActivation(set) error = %v", err)
	}
	tracker.RecordSetUI("device-1", "device-1", ids)

	_, err = rewriteAndValidateUpdateUI("device-1", &UIUpdate{
		ComponentID: "missing_component",
		Node:        ui.New("text", map[string]string{"id": "terminal_output", "value": "patched"}),
	}, tracker)
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("expected unknown component error, got %v", err)
	}

	got, err := rewriteAndValidateUpdateUI("device-1", &UIUpdate{
		ComponentID: "terminal_output",
		Node:        ui.New("text", map[string]string{"id": "terminal_output", "value": "patched"}),
	}, tracker)
	if err != nil {
		t.Fatalf("rewriteAndValidateUpdateUI() error = %v", err)
	}
	if got.ComponentID != "act:device-1/terminal_output" {
		t.Fatalf("update component_id = %q, want scoped", got.ComponentID)
	}
	if got.Node.Props["id"] != "act:device-1/terminal_output" {
		t.Fatalf("update node id = %q, want scoped", got.Node.Props["id"])
	}

	_ = rewritten
}

func TestUIActionOwnershipTrackerResolveClassifiesStaleNodeWithinActivation(t *testing.T) {
	tracker := newUIActionOwnershipTracker()
	tracker.RecordSetUI("device-1", "activation-a", []string{
		"act:activation-a/old_button",
	})
	tracker.RecordSetUI("device-1", "activation-a", []string{
		"act:activation-a/new_button",
	})

	_, reason, ok := tracker.Resolve("device-1", "act:activation-a/old_button")
	if ok {
		t.Fatalf("Resolve(old_button) ok = true, want false")
	}
	if reason != "stale_node" {
		t.Fatalf("Resolve(old_button) reason = %q, want stale_node", reason)
	}
}

func TestUIActionOwnershipTrackerResolveClassifiesUnknownActivationAfterSwap(t *testing.T) {
	tracker := newUIActionOwnershipTracker()
	tracker.RecordSetUI("device-1", "activation-a", []string{
		"act:activation-a/menu.open",
	})
	tracker.RecordSetUI("device-1", "activation-b", []string{
		"act:activation-b/menu.open",
	})
	tracker.ForgetActivation("device-1", "activation-a")

	owner, reason, ok := tracker.Resolve("device-1", "act:activation-b/menu.open")
	if !ok {
		t.Fatalf("Resolve(new activation) ok = false, reason=%q", reason)
	}
	if owner != "activation-b" {
		t.Fatalf("Resolve(new activation) owner = %q, want activation-b", owner)
	}

	_, reason, ok = tracker.Resolve("device-1", "act:activation-a/menu.open")
	if ok {
		t.Fatalf("Resolve(swapped-out activation) ok = true, want false")
	}
	if reason != "unknown_activation" {
		t.Fatalf("Resolve(swapped-out activation) reason = %q, want unknown_activation", reason)
	}
}

func TestMetricsSnapshotIncludesUnknownComponentReasons(t *testing.T) {
	m := &Metrics{}
	m.IncUnknownUIActionComponent("unscoped")
	m.IncUnknownUIActionComponent("unknown_activation")
	m.IncUnknownUIActionComponent("stale_node")

	snapshot := m.Snapshot()
	if snapshot[`ui_action_unknown_component_total{reason="unscoped"}`] != "1" {
		t.Fatalf("unscoped metric = %q, want 1", snapshot[`ui_action_unknown_component_total{reason="unscoped"}`])
	}
	if snapshot[`ui_action_unknown_component_total{reason="unknown_activation"}`] != "1" {
		t.Fatalf("unknown_activation metric = %q, want 1", snapshot[`ui_action_unknown_component_total{reason="unknown_activation"}`])
	}
	if snapshot[`ui_action_unknown_component_total{reason="stale_node"}`] != "1" {
		t.Fatalf("stale_node metric = %q, want 1", snapshot[`ui_action_unknown_component_total{reason="stale_node"}`])
	}
}
