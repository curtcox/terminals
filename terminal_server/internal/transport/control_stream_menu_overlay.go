package transport

import (
	"context"
	"errors"
	"sort"
	"strings"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func (h *StreamHandler) handleMenuOverlayInput(ctx context.Context, deviceID, componentID, action string) ([]ServerMessage, bool, error) {
	componentID = strings.TrimSpace(componentID)
	action = strings.ToLower(strings.TrimSpace(action))
	if action == "" {
		return nil, false, nil
	}

	overlayOpen := h.isMenuOverlayOpen(deviceID)
	if isCornerOpenInput(componentID, action) {
		if overlayOpen {
			return h.closeMenuOverlay(ctx, deviceID)
		}
		return h.openMenuOverlay(ctx, deviceID)
	}
	if isMenuCloseInput(componentID, action) {
		if !overlayOpen {
			return nil, true, nil
		}
		return h.closeMenuOverlay(ctx, deviceID)
	}
	return nil, false, nil
}

func isCornerOpenInput(componentID, action string) bool {
	if action == "corner.open" {
		return true
	}
	return action == "open" && strings.HasSuffix(strings.TrimSpace(componentID), "/"+cornerAffordanceLogicalID)
}

func isMenuCloseInput(componentID, action string) bool {
	return action == "close" && strings.HasSuffix(strings.TrimSpace(componentID), "/menu.close")
}

func (h *StreamHandler) isMenuOverlayOpen(deviceID string) bool {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return false
	}
	h.mu.Lock()
	_, ok := h.menuOverlayByDevice[deviceID]
	h.mu.Unlock()
	return ok
}

func defaultOverlayInputPolicy() overlayInputPolicyConfig {
	return overlayInputPolicyConfig{
		Mode: overlayInputPolicyMixed,
		Overrides: map[overlayInputStream]bool{
			overlayStreamAudio: true,
		},
	}
}

func normalizeOverlayInputPolicy(mode string) overlayInputPolicy {
	switch strings.ToUpper(strings.TrimSpace(mode)) {
	case string(overlayInputPolicyLive):
		return overlayInputPolicyLive
	case string(overlayInputPolicyPaused):
		return overlayInputPolicyPaused
	default:
		return overlayInputPolicyMixed
	}
}

func normalizeOverlayInputStream(raw string) overlayInputStream {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(overlayStreamPointer):
		return overlayStreamPointer
	case string(overlayStreamTouch):
		return overlayStreamTouch
	case string(overlayStreamKeyboard):
		return overlayStreamKeyboard
	case string(overlayStreamAudio):
		return overlayStreamAudio
	case string(overlayStreamCamera):
		return overlayStreamCamera
	default:
		return ""
	}
}

func mergeOverlayPolicy(base, override overlayInputPolicyConfig) overlayInputPolicyConfig {
	out := overlayInputPolicyConfig{
		Mode:      base.Mode,
		Overrides: map[overlayInputStream]bool{},
	}
	for key, value := range base.Overrides {
		out.Overrides[key] = value
	}
	if override.Mode != "" {
		out.Mode = override.Mode
	}
	for key, value := range override.Overrides {
		out.Overrides[key] = value
	}
	return out
}

func (h *StreamHandler) overlayPolicyForOpen() overlayInputPolicyConfig {
	h.mu.Lock()
	defer h.mu.Unlock()
	return mergeOverlayPolicy(defaultOverlayInputPolicy(), h.menuOverlayPolicy)
}

func (h *StreamHandler) overlayPolicyForDevice(deviceID string) (overlayInputPolicyConfig, bool) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return overlayInputPolicyConfig{}, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	state, ok := h.menuOverlayByDevice[deviceID]
	if !ok {
		return overlayInputPolicyConfig{}, false
	}
	return mergeOverlayPolicy(defaultOverlayInputPolicy(), state.Policy), true
}

func (h *StreamHandler) shouldDropMainStreamWhileOverlayOpen(deviceID string, stream overlayInputStream) bool {
	policy, ok := h.overlayPolicyForDevice(deviceID)
	if !ok {
		return false
	}
	return !policyAllowsMainStream(policy, stream)
}

func policyAllowsMainStream(policy overlayInputPolicyConfig, stream overlayInputStream) bool {
	switch policy.Mode {
	case overlayInputPolicyLive:
		return true
	case overlayInputPolicyPaused:
		if allowed, ok := policy.Overrides[stream]; ok {
			return allowed
		}
		return false
	case overlayInputPolicyMixed:
		if allowed, ok := policy.Overrides[stream]; ok {
			return allowed
		}
		return stream != overlayStreamPointer && stream != overlayStreamTouch
	default:
		return true
	}
}

func inferOverlayInputStream(in *InputRequest) overlayInputStream {
	if in == nil {
		return overlayStreamPointer
	}
	if strings.TrimSpace(in.KeyText) != "" {
		return overlayStreamKeyboard
	}
	componentID := strings.TrimSpace(in.ComponentID)
	action := strings.ToLower(strings.TrimSpace(in.Action))
	if componentID == "terminal_input" || action == "change" || action == "submit" {
		return overlayStreamKeyboard
	}
	return overlayStreamPointer
}

func (h *StreamHandler) inputTargetsOverlay(deviceID, componentID string) bool {
	deviceID = strings.TrimSpace(deviceID)
	componentID = strings.TrimSpace(componentID)
	if deviceID == "" || componentID == "" {
		return false
	}
	overlayActivationID := menuOverlayActivationID(deviceID)
	if _, activationID, _, ok := parseScopedComponentID(componentID); ok {
		return activationID == overlayActivationID
	}
	return false
}

func (h *StreamHandler) shouldDropMainInputWhileOverlayOpen(deviceID string, in *InputRequest) bool {
	if in == nil {
		return false
	}
	policy, ok := h.overlayPolicyForDevice(deviceID)
	if !ok {
		return false
	}
	if h.inputTargetsOverlay(deviceID, in.ComponentID) {
		return false
	}
	return !policyAllowsMainStream(policy, inferOverlayInputStream(in))
}

func shouldSuspendRouteForOverlay(route iorouter.Route) bool {
	switch strings.TrimSpace(route.StreamKind) {
	case "audio", "video", "audio_mix", "pa_audio", "announcement_audio":
		return true
	default:
		return false
	}
}

func (h *StreamHandler) pauseRoutesForOverlayPolicy(
	ctx context.Context,
	deviceID string,
	policy overlayInputPolicyConfig,
) ([]iorouter.Route, error) {
	_ = ctx
	if policy.Mode != overlayInputPolicyPaused {
		return nil, nil
	}
	if policyAllowsMainStream(policy, overlayStreamAudio) && policyAllowsMainStream(policy, overlayStreamCamera) {
		return nil, nil
	}
	if h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.IO == nil {
		return nil, nil
	}
	routeIO, ok := h.runtime.Env.IO.(interface {
		RoutesForDevice(deviceID string) []iorouter.Route
		Disconnect(sourceID, targetID, streamKind string) error
	})
	if !ok {
		return nil, nil
	}
	routes := routeIO.RoutesForDevice(deviceID)
	suspended := make([]iorouter.Route, 0, len(routes))
	for _, route := range routes {
		if !shouldSuspendRouteForOverlay(route) {
			continue
		}
		if err := routeIO.Disconnect(route.SourceID, route.TargetID, route.StreamKind); err != nil {
			return nil, err
		}
		suspended = append(suspended, route)
	}
	return suspended, nil
}

func (h *StreamHandler) resumeRoutesForOverlayPolicy(ctx context.Context, suspended []iorouter.Route) error {
	_ = ctx
	if len(suspended) == 0 || h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.IO == nil {
		return nil
	}
	routeIO, ok := h.runtime.Env.IO.(interface {
		Connect(sourceID, targetID, streamKind string) error
	})
	if !ok {
		return nil
	}
	for _, route := range suspended {
		if err := routeIO.Connect(route.SourceID, route.TargetID, route.StreamKind); err != nil && !errors.Is(err, iorouter.ErrRouteExists) {
			return err
		}
	}
	return nil
}

func (h *StreamHandler) openMenuOverlay(ctx context.Context, deviceID string) ([]ServerMessage, bool, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, true, nil
	}
	activationID := menuOverlayActivationID(deviceID)
	h.mu.Lock()
	_, alreadyOpen := h.menuOverlayByDevice[deviceID]
	h.mu.Unlock()
	if alreadyOpen {
		return []ServerMessage{{
			UpdateUI: &UIUpdate{
				ComponentID: ui.GlobalOverlayComponentID,
				Node:        h.menuOverlayDescriptor(deviceID),
			},
		}}, true, nil
	}
	policy := h.overlayPolicyForOpen()
	if err := h.requestMenuOverlayClaim(ctx, deviceID, activationID); err != nil {
		return nil, true, err
	}
	suspended, err := h.pauseRoutesForOverlayPolicy(ctx, deviceID, policy)
	if err != nil {
		_ = h.releaseMenuOverlayClaim(ctx, activationID)
		return nil, true, err
	}
	h.mu.Lock()
	h.menuOverlayByDevice[deviceID] = menuOverlayState{
		ActivationID: activationID,
		Policy:       policy,
		Suspended:    suspended,
	}
	h.mu.Unlock()
	return []ServerMessage{{
		UpdateUI: &UIUpdate{
			ComponentID: ui.GlobalOverlayComponentID,
			Node:        h.menuOverlayDescriptor(deviceID),
		},
	}}, true, nil
}

func (h *StreamHandler) closeMenuOverlay(ctx context.Context, deviceID string) ([]ServerMessage, bool, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, true, nil
	}
	h.mu.Lock()
	state, ok := h.menuOverlayByDevice[deviceID]
	if ok {
		delete(h.menuOverlayByDevice, deviceID)
	}
	h.mu.Unlock()
	if ok {
		if err := h.resumeRoutesForOverlayPolicy(ctx, state.Suspended); err != nil {
			return nil, true, err
		}
		if err := h.releaseMenuOverlayClaim(ctx, state.ActivationID); err != nil {
			return nil, true, err
		}
		h.uiOwners.ForgetActivation(deviceID, state.ActivationID)
	}
	return []ServerMessage{{
		UpdateUI: &UIUpdate{
			ComponentID: ui.GlobalOverlayComponentID,
			Node:        ui.GlobalOverlaySlot(),
		},
	}}, true, nil
}

func menuOverlayActivationID(deviceID string) string {
	return menuOverlayActivationPrefix + strings.TrimSpace(deviceID)
}

func menuScopedComponentID(activationID, logicalID string) string {
	return "act:" + strings.TrimSpace(activationID) + "/" + strings.TrimSpace(logicalID)
}

func (h *StreamHandler) requestMenuOverlayClaim(ctx context.Context, deviceID, activationID string) error {
	if h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.IO == nil {
		return nil
	}
	routeIO, ok := h.runtime.Env.IO.(interface{ Claims() *iorouter.ClaimManager })
	if !ok {
		return nil
	}
	claims := routeIO.Claims()
	if claims == nil {
		return nil
	}
	_, err := claims.Request(ctx, []iorouter.Claim{{
		ActivationID: activationID,
		DeviceID:     strings.TrimSpace(deviceID),
		Resource:     "screen.overlay",
		Mode:         iorouter.ClaimShared,
		Priority:     int(scenario.PriorityNormal),
	}})
	return err
}

func (h *StreamHandler) releaseMenuOverlayClaim(ctx context.Context, activationID string) error {
	if h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.IO == nil {
		return nil
	}
	routeIO, ok := h.runtime.Env.IO.(interface{ Claims() *iorouter.ClaimManager })
	if !ok {
		return nil
	}
	claims := routeIO.Claims()
	if claims == nil {
		return nil
	}
	return claims.Release(ctx, strings.TrimSpace(activationID))
}

func (h *StreamHandler) menuOverlayDescriptor(deviceID string) ui.Descriptor {
	activationID := menuOverlayActivationID(deviceID)
	children := make([]ui.Descriptor, 0, 4)
	for _, appName := range h.menuOverlayApps(deviceID) {
		children = append(children, ui.New("button", map[string]string{
			"id":     menuScopedComponentID(activationID, "menu.app."+appName),
			"label":  appName,
			"action": "start:" + appName,
		}))
	}
	children = append(children,
		ui.New("button", map[string]string{
			"id":     menuScopedComponentID(activationID, "menu.privacy_toggle"),
			"label":  "Privacy",
			"action": "privacy.toggle",
		}),
		ui.New("button", map[string]string{
			"id":     menuScopedComponentID(activationID, "menu.bug_report"),
			"label":  "Report Bug",
			"action": bugReportActionPrefix + ":" + strings.TrimSpace(deviceID),
		}),
		ui.New("button", map[string]string{
			"id":     menuScopedComponentID(activationID, "menu.close"),
			"label":  "Close",
			"action": "close",
		}),
	)

	return ui.New("overlay", map[string]string{
		"id": ui.GlobalOverlayComponentID,
	}, ui.New("stack", map[string]string{
		"id": menuScopedComponentID(activationID, "menu.root"),
	}, children...))
}

func (h *StreamHandler) menuOverlayApps(deviceID string) []string {
	if h.runtime == nil || h.runtime.Engine == nil {
		return nil
	}
	items := h.runtime.Engine.RegistrySnapshot()
	names := make([]string, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	actor := Actor{Kind: "device", ID: strings.TrimSpace(deviceID)}
	if h.identityService != nil {
		resolved := h.identityService.ResolveActor(deviceID)
		if kind := strings.TrimSpace(resolved.Kind); kind != "" {
			actor.Kind = kind
		}
		if id := strings.TrimSpace(resolved.ID); id != "" {
			actor.ID = id
		}
	}
	if h.menuAppPolicy == nil {
		return names
	}
	return h.menuAppPolicy.VisibleApps(actor, names)
}
