package transport

import (
	"context"
	"errors"
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

	claims := routeIO.Claims()
	if claims != nil {
		if len(lostResources) > 0 {
			suspendedClaims := make([]iorouter.Claim, 0)
			for _, claim := range claims.Snapshot(deviceID) {
				if _, exists := lostResources[claim.Resource]; !exists {
					continue
				}
				suspendedClaims = append(suspendedClaims, claim)
			}
			h.rememberSuspendedClaims(deviceID, suspendedClaims)
			if len(suspendedClaims) > 0 {
				_ = claims.ReleaseClaims(ctx, suspendedClaims)
			}
		}
		if len(gainedResources) > 0 {
			h.restoreSuspendedClaims(ctx, claims, deviceID, gainedResources)
		}
	}

	routes := routeIO.RoutesForDevice(deviceID)
	out := make([]ServerMessage, 0, len(routes))
	if len(lostResources) == 0 {
		return out
	}
	for _, route := range routes {
		if !shouldDisconnectRouteForLostResources(route, deviceID, lostResources) {
			continue
		}
		if err := routeIO.Disconnect(route.SourceID, route.TargetID, route.StreamKind); err != nil && !errors.Is(err, iorouter.ErrRouteNotFound) {
			continue
		}
		out = append(out, ServerMessage{
			StopStream: &StopStreamResponse{StreamID: routeStreamID(route)},
		})
	}
	return out
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

	if (caps["screen.width"] != "" && caps["screen.height"] != "") || truthyCapability(caps["display.count"]) {
		resources["screen.main"] = struct{}{}
		resources["screen.overlay"] = struct{}{}
	}
	for _, displayID := range endpointResourceIDs(caps, "display.") {
		resources["display."+displayID+".main"] = struct{}{}
		resources["display."+displayID+".overlay"] = struct{}{}
	}
	if truthyCapability(caps["keyboard.physical"]) || strings.TrimSpace(caps["keyboard.layout"]) != "" {
		resources["keyboard.primary"] = struct{}{}
	}
	if strings.TrimSpace(caps["pointer.type"]) != "" {
		resources["pointer.primary"] = struct{}{}
	}
	if truthyCapability(caps["touch.supported"]) {
		resources["touch.primary"] = struct{}{}
	}
	if truthyCapability(caps["speakers.present"]) || truthyCapability(caps["speakers.endpoint_count"]) {
		resources["speaker.main"] = struct{}{}
	}
	for _, endpointID := range endpointResourceIDs(caps, "speakers.endpoint.") {
		resources["audio_out."+endpointID] = struct{}{}
	}
	if truthyCapability(caps["microphone.present"]) || truthyCapability(caps["microphone.endpoint_count"]) {
		resources["mic.capture"] = struct{}{}
		resources["mic.analyze"] = struct{}{}
	}
	for _, endpointID := range endpointResourceIDs(caps, "microphone.endpoint.") {
		resources["audio_in."+endpointID+".capture"] = struct{}{}
		resources["audio_in."+endpointID+".analyze"] = struct{}{}
	}
	if truthyCapability(caps["camera.present"]) || truthyCapability(caps["camera.endpoint_count"]) {
		resources["camera.capture"] = struct{}{}
		resources["camera.analyze"] = struct{}{}
	}
	for _, endpointID := range endpointResourceIDs(caps, "camera.endpoint.") {
		resources["camera."+endpointID+".capture"] = struct{}{}
		resources["camera."+endpointID+".analyze"] = struct{}{}
	}
	if truthyCapability(caps["haptics.supported"]) {
		resources["haptic.primary"] = struct{}{}
	}
	if truthyCapability(caps["edge.compute.cpu_realtime"]) {
		resources[iorouter.ResourceComputeCPUShared] = struct{}{}
	}
	if truthyCapability(caps["edge.compute.gpu_realtime"]) {
		resources[iorouter.ResourceComputeGPUShared] = struct{}{}
	}
	if truthyCapability(caps["edge.compute.npu_realtime"]) {
		resources[iorouter.ResourceComputeNPUShared] = struct{}{}
	}
	if truthyCapability(caps["edge.retention.audio_sec"]) {
		resources[iorouter.ResourceBufferAudio] = struct{}{}
	}
	if truthyCapability(caps["edge.retention.video_sec"]) {
		resources[iorouter.ResourceBufferVideo] = struct{}{}
	}
	if truthyCapability(caps["edge.retention.sensor_sec"]) {
		resources[iorouter.ResourceBufferSensor] = struct{}{}
	}
	if truthyCapability(caps["edge.retention.radio_sec"]) {
		resources[iorouter.ResourceBufferRadio] = struct{}{}
	}
	if truthyCapability(caps["connectivity.bluetooth_version"]) {
		resources[iorouter.ResourceRadioBLEScan] = struct{}{}
	}
	if truthyCapability(caps["connectivity.wifi_signal_strength"]) {
		resources[iorouter.ResourceRadioWiFiScan] = struct{}{}
	}
	return resources
}

func endpointResourceIDs(caps map[string]string, prefix string) []string {
	if len(caps) == 0 || prefix == "" {
		return nil
	}

	indexToID := map[string]string{}
	indexes := map[string]struct{}{}
	indexHasAvailability := map[string]bool{}
	indexAvailable := map[string]bool{}
	for key, value := range caps {
		rest, ok := strings.CutPrefix(key, prefix)
		if !ok {
			continue
		}
		parts := strings.Split(rest, ".")
		if len(parts) < 2 {
			continue
		}
		index := strings.TrimSpace(parts[0])
		if index == "" {
			continue
		}
		indexes[index] = struct{}{}
		if parts[1] == "id" {
			if id := sanitizeResourceID(value); id != "" {
				indexToID[index] = id
			}
		}
		if parts[1] == "available" {
			indexHasAvailability[index] = true
			indexAvailable[index] = truthyCapability(value)
		}
	}

	if len(indexes) == 0 {
		return nil
	}

	sortedIndexes := make([]string, 0, len(indexes))
	for index := range indexes {
		sortedIndexes = append(sortedIndexes, index)
	}
	sort.Strings(sortedIndexes)

	ids := make([]string, 0, len(sortedIndexes))
	for _, index := range sortedIndexes {
		if indexHasAvailability[index] && !indexAvailable[index] {
			continue
		}
		if id := indexToID[index]; id != "" {
			ids = append(ids, id)
			continue
		}
		ids = append(ids, "endpoint-"+sanitizeResourceID(index))
	}
	return ids
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
