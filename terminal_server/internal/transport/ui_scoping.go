package transport

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

var (
	logicalComponentIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	scopedComponentIDPattern  = regexp.MustCompile(`^act:([A-Za-z0-9_:-]+)/([A-Za-z0-9_.-]+)$`)
)

func rewriteDescriptorIDsForActivation(root ui.Descriptor, activationID string) (ui.Descriptor, []string, error) {
	activationID = strings.TrimSpace(activationID)
	if activationID == "" {
		activationID = "main"
	}

	seen := map[string]struct{}{}
	ids := make([]string, 0, 16)
	var rewriteNode func(node ui.Descriptor, path string) (ui.Descriptor, error)
	rewriteNode = func(node ui.Descriptor, path string) (ui.Descriptor, error) {
		rawID := descriptorNodeID(node)
		if rawID != "" {
			scopedID, err := canonicalScopedComponentID(rawID, activationID)
			if err != nil {
				return ui.Descriptor{}, fmt.Errorf("%s.id: %w", path, err)
			}
			if _, exists := seen[scopedID]; exists {
				return ui.Descriptor{}, fmt.Errorf("%s.id: duplicate component id %q", path, scopedID)
			}
			seen[scopedID] = struct{}{}
			ids = append(ids, scopedID)
			node = setDescriptorNodeID(node, scopedID)
		}
		if len(node.Children) == 0 {
			return node, nil
		}
		children := make([]ui.Descriptor, 0, len(node.Children))
		for i := range node.Children {
			childPath := fmt.Sprintf("%s.children[%d]", path, i)
			rewritten, err := rewriteNode(node.Children[i], childPath)
			if err != nil {
				return ui.Descriptor{}, err
			}
			children = append(children, rewritten)
		}
		node.Children = children
		return node, nil
	}

	rewritten, err := rewriteNode(root, "root")
	if err != nil {
		return ui.Descriptor{}, nil, err
	}
	return rewritten, ids, nil
}

func canonicalScopedComponentID(rawID, defaultActivationID string) (string, error) {
	rawID = strings.TrimSpace(rawID)
	if rawID == "" {
		return "", nil
	}
	if scoped, _, _, ok := parseScopedComponentID(rawID); ok {
		return scoped, nil
	}
	if strings.HasPrefix(rawID, "__affordance.") {
		return "", fmt.Errorf("logical component id %q uses reserved wrapper prefix", rawID)
	}
	if !logicalComponentIDPattern.MatchString(rawID) {
		return "", fmt.Errorf("logical component id %q is invalid", rawID)
	}
	return scopeComponentID(defaultActivationID, rawID), nil
}

func parseScopedComponentID(componentID string) (scoped, activationID, logicalID string, ok bool) {
	componentID = strings.TrimSpace(componentID)
	if componentID == "" {
		return "", "", "", false
	}
	matches := scopedComponentIDPattern.FindStringSubmatch(componentID)
	if len(matches) != 3 {
		return "", "", "", false
	}
	return componentID, strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2]), true
}

func scopeComponentID(activationID, logicalID string) string {
	return "act:" + strings.TrimSpace(activationID) + "/" + strings.TrimSpace(logicalID)
}

func descriptorNodeID(node ui.Descriptor) string {
	if trimmed := strings.TrimSpace(node.ID); trimmed != "" {
		return trimmed
	}
	if node.Props == nil {
		return ""
	}
	return strings.TrimSpace(node.Props["id"])
}

func setDescriptorNodeID(node ui.Descriptor, id string) ui.Descriptor {
	node.ID = id
	if node.Props == nil {
		node.Props = map[string]string{}
	}
	node.Props["id"] = id
	return node
}

type uiActionOwnershipTracker struct {
	mu               sync.Mutex
	componentOwner   map[string]string
	knownActivations map[string]map[string]struct{}
}

func newUIActionOwnershipTracker() *uiActionOwnershipTracker {
	return &uiActionOwnershipTracker{
		componentOwner:   map[string]string{},
		knownActivations: map[string]map[string]struct{}{},
	}
}

func (t *uiActionOwnershipTracker) RecordSetUI(deviceID, activationID string, componentIDs []string) {
	if t == nil {
		return
	}
	deviceID = strings.TrimSpace(deviceID)
	activationID = strings.TrimSpace(activationID)
	if deviceID == "" || activationID == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.knownActivations[deviceID]; !exists {
		t.knownActivations[deviceID] = map[string]struct{}{}
	}
	prefix := ownershipActivationPrefix(deviceID, activationID)
	for key := range t.componentOwner {
		if strings.HasPrefix(key, prefix) {
			delete(t.componentOwner, key)
		}
	}
	for _, componentID := range componentIDs {
		componentID = strings.TrimSpace(componentID)
		if componentID == "" {
			continue
		}
		ownerActivationID := activationID
		if _, parsedActivationID, _, ok := parseScopedComponentID(componentID); ok {
			ownerActivationID = parsedActivationID
		}
		t.componentOwner[ownershipKey(deviceID, componentID)] = ownerActivationID
		if ownerActivationID != "" {
			t.knownActivations[deviceID][ownerActivationID] = struct{}{}
		}
	}
	if activationID != "" {
		if _, exists := t.knownActivations[deviceID]; !exists {
			t.knownActivations[deviceID] = map[string]struct{}{}
		}
		t.knownActivations[deviceID][activationID] = struct{}{}
	}
}

func (t *uiActionOwnershipTracker) RecordUpdate(deviceID, activationID string, componentIDs []string) {
	if t == nil {
		return
	}
	deviceID = strings.TrimSpace(deviceID)
	activationID = strings.TrimSpace(activationID)
	if deviceID == "" || activationID == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.knownActivations[deviceID]; !exists {
		t.knownActivations[deviceID] = map[string]struct{}{}
	}
	for _, componentID := range componentIDs {
		componentID = strings.TrimSpace(componentID)
		if componentID == "" {
			continue
		}
		ownerActivationID := activationID
		if _, parsedActivationID, _, ok := parseScopedComponentID(componentID); ok {
			ownerActivationID = parsedActivationID
		}
		t.componentOwner[ownershipKey(deviceID, componentID)] = ownerActivationID
		if ownerActivationID != "" {
			t.knownActivations[deviceID][ownerActivationID] = struct{}{}
		}
	}
	if activationID != "" {
		t.knownActivations[deviceID][activationID] = struct{}{}
	}
}

func (t *uiActionOwnershipTracker) Resolve(deviceID, componentID string) (string, string, bool) {
	if t == nil {
		return "", "unknown_activation", false
	}
	deviceID = strings.TrimSpace(deviceID)
	componentID = strings.TrimSpace(componentID)
	if _, activationID, _, ok := parseScopedComponentID(componentID); !ok {
		return "", "unscoped", false
	} else {
		t.mu.Lock()
		defer t.mu.Unlock()
		if owner, exists := t.componentOwner[ownershipKey(deviceID, componentID)]; exists {
			return owner, "", true
		}
		if activations, exists := t.knownActivations[deviceID]; exists {
			if _, known := activations[activationID]; known {
				return "", "stale_node", false
			}
		}
		return "", "unknown_activation", false
	}
}

func (t *uiActionOwnershipTracker) HasKnownActivation(deviceID string) bool {
	if t == nil {
		return false
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	activations, exists := t.knownActivations[deviceID]
	return exists && len(activations) > 0
}

func (t *uiActionOwnershipTracker) ForgetActivation(deviceID, activationID string) {
	if t == nil {
		return
	}
	deviceID = strings.TrimSpace(deviceID)
	activationID = strings.TrimSpace(activationID)
	if deviceID == "" || activationID == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	prefix := ownershipActivationPrefix(deviceID, activationID)
	for key := range t.componentOwner {
		if strings.HasPrefix(key, prefix) {
			delete(t.componentOwner, key)
		}
	}
	if activations, exists := t.knownActivations[deviceID]; exists {
		delete(activations, activationID)
		if len(activations) == 0 {
			delete(t.knownActivations, deviceID)
		}
	}
}

func (t *uiActionOwnershipTracker) ForgetDevice(deviceID string) {
	if t == nil {
		return
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	devicePrefix := strings.TrimSpace(deviceID) + "|"
	for key := range t.componentOwner {
		if strings.HasPrefix(key, devicePrefix) {
			delete(t.componentOwner, key)
		}
	}
	delete(t.knownActivations, deviceID)
}

func rewriteAndValidateUpdateUI(deviceID string, update *UIUpdate, tracker *uiActionOwnershipTracker) (*UIUpdate, error) {
	if update == nil {
		return nil, fmt.Errorf("update ui is nil")
	}
	componentID := strings.TrimSpace(update.ComponentID)
	if componentID == "" {
		return nil, fmt.Errorf("update ui component_id is empty")
	}
	if _, _, _, ok := parseScopedComponentID(componentID); !ok {
		if !logicalComponentIDPattern.MatchString(componentID) {
			return nil, fmt.Errorf("update ui component_id %q is unscoped", componentID)
		}
		componentID = scopeComponentID(strings.TrimSpace(deviceID), componentID)
	}
	scopedID, activationID, _, ok := parseScopedComponentID(componentID)
	if !ok {
		return nil, fmt.Errorf("update ui component_id %q is unscoped", componentID)
	}
	if tracker != nil {
		if _, reason, exists := tracker.Resolve(deviceID, scopedID); !exists {
			return nil, fmt.Errorf("update ui component_id %q unknown (%s)", scopedID, reason)
		}
	}
	rewrittenNode, componentIDs, err := rewriteDescriptorIDsForActivation(update.Node, activationID)
	if err != nil {
		return nil, err
	}
	if tracker != nil {
		tracker.RecordUpdate(deviceID, activationID, componentIDs)
	}
	out := *update
	out.ComponentID = scopedID
	out.Node = rewrittenNode
	return &out, nil
}

func ownershipKey(deviceID, componentID string) string {
	return strings.TrimSpace(deviceID) + "|" + strings.TrimSpace(componentID)
}

func ownershipActivationPrefix(deviceID, activationID string) string {
	return strings.TrimSpace(deviceID) + "|act:" + strings.TrimSpace(activationID) + "/"
}
