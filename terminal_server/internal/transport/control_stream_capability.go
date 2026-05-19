package transport

import (
	"context"
	"sort"
	"strings"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

func (h *StreamHandler) handleCapabilityChangeEffects(
	ctx context.Context,
	deviceID string,
	beforeCaps map[string]string,
	afterCaps map[string]string,
) []ServerMessage {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil
	}
	if !capabilityStateChanged(beforeCaps, afterCaps) {
		return nil
	}

	lostResources := lostCapabilityResources(beforeCaps, afterCaps)
	gainedResources := gainedCapabilityResources(beforeCaps, afterCaps)
	emitCapabilityEvents(ctx, h.runtime, deviceID, beforeCaps, afterCaps, lostResources, gainedResources)
	if len(lostResources) == 0 && len(gainedResources) == 0 {
		return nil
	}
	if h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.IO == nil {
		return nil
	}

	routeIO, ok := h.runtime.Env.IO.(interface {
		Claims() *iorouter.ClaimManager
		RoutesForDevice(deviceID string) []iorouter.Route
		Disconnect(sourceID, targetID, streamKind string) error
	})
	if !ok {
		return nil
	}

	h.suspendClaimsForLostResources(ctx, routeIO, deviceID, lostResources)
	if claims := routeIO.Claims(); claims != nil && len(gainedResources) > 0 {
		h.restoreSuspendedClaims(ctx, claims, deviceID, gainedResources)
	}
	return h.disconnectRoutesForLostResources(routeIO, deviceID, lostResources)
}

func (h *StreamHandler) rememberSuspendedClaims(deviceID string, claims []iorouter.Claim) {
	if len(claims) == 0 {
		return
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	existing := h.suspendedClaimsByDevice[deviceID]
	if len(existing) == 0 {
		h.suspendedClaimsByDevice[deviceID] = append([]iorouter.Claim(nil), claims...)
		return
	}
	seen := map[string]struct{}{}
	for _, claim := range existing {
		seen[suspendedClaimKey(claim)] = struct{}{}
	}
	for _, claim := range claims {
		key := suspendedClaimKey(claim)
		if _, ok := seen[key]; ok {
			continue
		}
		existing = append(existing, claim)
		seen[key] = struct{}{}
	}
	h.suspendedClaimsByDevice[deviceID] = existing
}

func (h *StreamHandler) restoreSuspendedClaims(ctx context.Context, claims *iorouter.ClaimManager, deviceID string, gainedResources map[string]struct{}) {
	if claims == nil {
		return
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return
	}

	h.mu.Lock()
	pending := append([]iorouter.Claim(nil), h.suspendedClaimsByDevice[deviceID]...)
	h.mu.Unlock()
	if len(pending) == 0 {
		return
	}

	activeByKey := map[string]struct{}{}
	for _, claim := range claims.Snapshot(deviceID) {
		activeByKey[suspendedClaimKey(claim)] = struct{}{}
	}

	remaining := make([]iorouter.Claim, 0, len(pending))
	for _, suspended := range pending {
		if _, gained := gainedResources[suspended.Resource]; !gained {
			remaining = append(remaining, suspended)
			continue
		}
		key := suspendedClaimKey(suspended)
		if _, alreadyActive := activeByKey[key]; alreadyActive {
			continue
		}
		if _, err := claims.Request(ctx, []iorouter.Claim{suspended}); err != nil {
			remaining = append(remaining, suspended)
			continue
		}
	}

	h.mu.Lock()
	if len(remaining) == 0 {
		delete(h.suspendedClaimsByDevice, deviceID)
	} else {
		h.suspendedClaimsByDevice[deviceID] = remaining
	}
	h.mu.Unlock()
}

func suspendedClaimKey(claim iorouter.Claim) string {
	return strings.TrimSpace(claim.DeviceID) + "/" + strings.TrimSpace(claim.Resource) + "/" + strings.TrimSpace(claim.ActivationID)
}

func lostCapabilityResources(beforeCaps, afterCaps map[string]string) map[string]struct{} {
	before := capabilityResources(beforeCaps)
	after := capabilityResources(afterCaps)
	lost := map[string]struct{}{}
	for resource := range before {
		if _, exists := after[resource]; exists {
			continue
		}
		lost[resource] = struct{}{}
	}
	return lost
}

func gainedCapabilityResources(beforeCaps, afterCaps map[string]string) map[string]struct{} {
	before := capabilityResources(beforeCaps)
	after := capabilityResources(afterCaps)
	gained := map[string]struct{}{}
	for resource := range after {
		if _, exists := before[resource]; exists {
			continue
		}
		gained[resource] = struct{}{}
	}
	return gained
}

func capabilityInvalidations(beforeCaps, afterCaps map[string]string) []CapabilityInvalidation {
	lost := lostCapabilityResources(beforeCaps, afterCaps)
	if len(lost) == 0 {
		return nil
	}
	names := make([]string, 0, len(lost))
	for resource := range lost {
		names = append(names, resource)
	}
	sort.Strings(names)
	out := make([]CapabilityInvalidation, 0, len(names))
	for _, resource := range names {
		out = append(out, CapabilityInvalidation{
			Resource: resource,
			Reason:   "capability_lost",
		})
	}
	return out
}

func capabilityResources(caps map[string]string) map[string]struct{} {
	resources := map[string]struct{}{}
	if len(caps) == 0 {
		return resources
	}
	capabilityDisplayResources(caps, resources)
	capabilityInputResources(caps, resources)
	capabilityAudioResources(caps, resources)
	capabilityCameraResources(caps, resources)
	capabilityEdgeResources(caps, resources)
	capabilityConnectivityResources(caps, resources)
	return resources
}

func endpointResourceIDs(caps map[string]string, prefix string) []string {
	if len(caps) == 0 || prefix == "" {
		return nil
	}
	return endpointIDsFromIndexState(parseEndpointIndexState(caps, prefix))
}

func sanitizeResourceID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(raw))
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-._")
	if out == "" {
		return "id"
	}
	return out
}

func truthyCapability(raw string) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" || raw == "0" || raw == "false" || raw == "no" || raw == "off" {
		return false
	}
	return true
}

func shouldDisconnectRouteForLostResources(route iorouter.Route, deviceID string, lostResources map[string]struct{}) bool {
	if len(lostResources) == 0 {
		return false
	}
	streamKind := strings.ToLower(strings.TrimSpace(route.StreamKind))
	sourceID := strings.TrimSpace(route.SourceID)
	targetID := strings.TrimSpace(route.TargetID)
	if sourceID != deviceID && targetID != deviceID {
		return false
	}

	_, lostMicCapture := lostResources["mic.capture"]
	_, lostMicAnalyze := lostResources["mic.analyze"]
	_, lostSpeaker := lostResources["speaker.main"]
	_, lostCameraCapture := lostResources["camera.capture"]
	_, lostCameraAnalyze := lostResources["camera.analyze"]
	_, lostScreenMain := lostResources["screen.main"]
	_, lostScreenOverlay := lostResources["screen.overlay"]
	lostAudioInEndpoint := hasLostResourcePrefix(lostResources, "audio_in.")
	lostAudioOutEndpoint := hasLostResourcePrefix(lostResources, "audio_out.")
	lostCameraEndpoint := hasLostResourcePrefix(lostResources, "camera.")
	lostDisplayEndpoint := hasLostResourcePrefix(lostResources, "display.")

	if (lostMicCapture || lostMicAnalyze || lostAudioInEndpoint) && sourceID == deviceID && strings.Contains(streamKind, "audio") {
		return true
	}
	if (lostSpeaker || lostAudioOutEndpoint) && targetID == deviceID && strings.Contains(streamKind, "audio") {
		return true
	}
	if (lostCameraCapture || lostCameraAnalyze || lostCameraEndpoint) && sourceID == deviceID && strings.Contains(streamKind, "video") {
		return true
	}
	if (lostScreenMain || lostScreenOverlay || lostDisplayEndpoint) && targetID == deviceID && strings.Contains(streamKind, "video") {
		return true
	}

	return false
}

func hasLostResourcePrefix(lostResources map[string]struct{}, prefix string) bool {
	if len(lostResources) == 0 || strings.TrimSpace(prefix) == "" {
		return false
	}
	for resource := range lostResources {
		if strings.HasPrefix(resource, prefix) {
			return true
		}
	}
	return false
}

func emitCapabilityEvents(
	ctx context.Context,
	runtime *scenario.Runtime,
	deviceID string,
	beforeCaps map[string]string,
	afterCaps map[string]string,
	lostResources map[string]struct{},
	gainedResources map[string]struct{},
) {
	if runtime == nil || runtime.Env == nil || runtime.Env.Broadcast == nil {
		return
	}
	targets := []string{deviceID}
	_ = runtime.Env.Broadcast.Notify(ctx, targets, "terminal.capability.updated")
	if len(gainedResources) > 0 {
		_ = runtime.Env.Broadcast.Notify(ctx, targets, "terminal.capability.added")
	}
	if len(lostResources) > 0 {
		_ = runtime.Env.Broadcast.Notify(ctx, targets, "terminal.capability.removed")
	}
	if displayGeometryCapabilitiesChanged(beforeCaps, afterCaps) {
		_ = runtime.Env.Broadcast.Notify(ctx, targets, "terminal.display.resized")
	}
	if audioRouteCapabilitiesChanged(beforeCaps, afterCaps) {
		_ = runtime.Env.Broadcast.Notify(ctx, targets, "terminal.audio_route.changed")
	}
	if len(lostResources) == 0 {
		return
	}
	_ = runtime.Env.Broadcast.Notify(ctx, targets, "terminal.resource.lost")
	names := make([]string, 0, len(lostResources))
	for resource := range lostResources {
		names = append(names, resource)
	}
	sort.Strings(names)
	for _, resource := range names {
		_ = runtime.Env.Broadcast.Notify(ctx, targets, "terminal.resource.lost:"+resource)
	}
}

func audioRouteCapabilitiesChanged(beforeCaps, afterCaps map[string]string) bool {
	before := audioRouteCapabilityState(beforeCaps)
	after := audioRouteCapabilityState(afterCaps)
	if len(before) != len(after) {
		return true
	}
	for key, value := range before {
		if after[key] != value {
			return true
		}
	}
	return false
}

func displayGeometryCapabilitiesChanged(beforeCaps, afterCaps map[string]string) bool {
	before := displayGeometryCapabilityState(beforeCaps)
	after := displayGeometryCapabilityState(afterCaps)
	if len(before) != len(after) {
		return true
	}
	for key, value := range before {
		if after[key] != value {
			return true
		}
	}
	return false
}

func displayGeometryCapabilityState(caps map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range caps {
		if !isDisplayGeometryCapabilityKey(key) {
			continue
		}
		out[key] = value
	}
	return out
}

func isDisplayGeometryCapabilityKey(key string) bool {
	switch {
	case key == "screen.width":
		return true
	case key == "screen.height":
		return true
	case key == "screen.density":
		return true
	case key == "screen.orientation":
		return true
	case strings.HasPrefix(key, "screen.safe."):
		return true
	case strings.HasPrefix(key, "display."):
		if strings.HasSuffix(key, ".width") || strings.HasSuffix(key, ".height") || strings.HasSuffix(key, ".density") || strings.HasSuffix(key, ".orientation") {
			return true
		}
		return strings.Contains(key, ".safe.")
	default:
		return false
	}
}

func audioRouteCapabilityState(caps map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range caps {
		if !isAudioRouteCapabilityKey(key) {
			continue
		}
		out[key] = value
	}
	return out
}

func isAudioRouteCapabilityKey(key string) bool {
	switch {
	case key == "microphone.present":
		return true
	case key == "microphone.endpoint_count":
		return true
	case strings.HasPrefix(key, "microphone.endpoint."):
		return true
	case key == "speakers.present":
		return true
	case key == "speakers.endpoint_count":
		return true
	case strings.HasPrefix(key, "speakers.endpoint."):
		return true
	default:
		return false
	}
}

func capabilityStateChanged(beforeCaps, afterCaps map[string]string) bool {
	if len(beforeCaps) != len(afterCaps) {
		return true
	}
	for key, beforeValue := range beforeCaps {
		if afterValue, ok := afterCaps[key]; !ok || afterValue != beforeValue {
			return true
		}
	}
	return false
}
