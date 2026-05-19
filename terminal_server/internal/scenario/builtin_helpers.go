package scenario

import (
	"context"
	"errors"
	"strings"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

const (
	micCaptureFallbackResource    = "mic.capture"
	micAnalyzeFallbackResource    = "mic.analyze"
	speakerMainFallbackResource   = "speaker.main"
	screenOverlayFallbackResource = "screen.overlay"
)

func resolveAudioInputCaptureResource(env *Environment, deviceID string) string {
	return resolveEndpointResource(
		env,
		deviceID,
		"microphone",
		func(endpointID string) string { return "audio_in." + endpointID + ".capture" },
		micCaptureFallbackResource,
	)
}

func resolveAudioInputAnalyzeResource(env *Environment, deviceID string) string {
	return resolveEndpointResource(
		env,
		deviceID,
		"microphone",
		func(endpointID string) string { return "audio_in." + endpointID + ".analyze" },
		micAnalyzeFallbackResource,
	)
}

func resolveAudioOutResource(env *Environment, deviceID string) string {
	return resolveEndpointResource(
		env,
		deviceID,
		"speakers",
		func(endpointID string) string { return "audio_out." + endpointID },
		speakerMainFallbackResource,
	)
}

func resolveDisplayOverlayResource(env *Environment, deviceID string) string {
	return resolveEndpointResource(
		env,
		deviceID,
		"display",
		func(endpointID string) string { return "display." + endpointID + ".overlay" },
		screenOverlayFallbackResource,
	)
}

func resolveEndpointResource(
	env *Environment,
	deviceID string,
	family string,
	compose func(endpointID string) string,
	fallback string,
) string {
	for _, endpointID := range endpointResourceIDsForDevice(env, deviceID, family) {
		if endpointID == "" {
			continue
		}
		return compose(endpointID)
	}
	return fallback
}

func capabilitiesForDevice(env *Environment, deviceID string) map[string]string {
	if env == nil || env.Devices == nil {
		return nil
	}
	provider, ok := env.Devices.(interface {
		Get(deviceID string) (device.Device, bool)
	})
	if !ok {
		return nil
	}
	record, ok := provider.Get(strings.TrimSpace(deviceID))
	if !ok {
		return nil
	}
	if len(record.Capabilities) == 0 {
		return nil
	}
	out := make(map[string]string, len(record.Capabilities))
	for key, value := range record.Capabilities {
		out[key] = value
	}
	return out
}

func sanitizeResourceToken(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(raw))
	lastDash := false
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	token := strings.Trim(b.String(), "-")
	if token == "" {
		return ""
	}
	return token
}

func intentMatches(intent string, accepted ...string) bool {
	normalized := strings.TrimSpace(strings.ToLower(intent))
	for _, candidate := range accepted {
		if normalized == strings.TrimSpace(strings.ToLower(candidate)) {
			return true
		}
	}
	return false
}

func notifySource(ctx context.Context, env *Environment, sourceID, message string) error {
	if env == nil || env.Broadcast == nil {
		return nil
	}
	deviceIDs := []string{}
	if strings.TrimSpace(sourceID) != "" {
		deviceIDs = []string{strings.TrimSpace(sourceID)}
	}
	return env.Broadcast.Notify(ctx, deviceIDs, message)
}

type ioRoute struct {
	sourceID   string
	targetID   string
	streamKind string
}

func connectOwnedRoute(env *Environment, sourceID, targetID, streamKind string) (ioRoute, bool, error) {
	err := env.IO.Connect(sourceID, targetID, streamKind)
	if err != nil {
		if errors.Is(err, iorouter.ErrRouteExists) {
			return ioRoute{}, false, nil
		}
		return ioRoute{}, false, err
	}
	return ioRoute{
		sourceID:   sourceID,
		targetID:   targetID,
		streamKind: streamKind,
	}, true, nil
}

func connectBidirectionalSourceTargetsOwned(
	_ context.Context,
	env *Environment,
	sourceID string,
	targetIDs []string,
	streamKind string,
) ([]ioRoute, error) {
	if env == nil || env.IO == nil || env.Devices == nil {
		return nil, nil
	}
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return nil, nil
	}
	if targetIDs == nil {
		targetIDs = nonSourceDeviceIDs(env, sourceID)
	}
	routes := make([]ioRoute, 0)
	for _, peerID := range targetIDs {
		var err error
		routes, err = appendBidirectionalPeerRoutes(routes, env, sourceID, peerID, streamKind)
		if err != nil {
			return nil, err
		}
	}
	return routes, nil
}

func appendBidirectionalPeerRoutes(
	routes []ioRoute,
	env *Environment,
	sourceID, peerID, streamKind string,
) ([]ioRoute, error) {
	if peerID == "" || peerID == sourceID {
		return routes, nil
	}
	route, added, err := connectOwnedRoute(env, sourceID, peerID, streamKind)
	if err != nil {
		return nil, err
	}
	if added {
		routes = append(routes, route)
	}
	route, added, err = connectOwnedRoute(env, peerID, sourceID, streamKind)
	if err != nil {
		return nil, err
	}
	if added {
		routes = append(routes, route)
	}
	return routes, nil
}

func disconnectOwnedRoutes(env *Environment, routes []ioRoute) error {
	if env == nil || env.IO == nil || len(routes) == 0 {
		return nil
	}
	for _, route := range routes {
		err := env.IO.Disconnect(route.sourceID, route.targetID, route.streamKind)
		if err != nil && !errors.Is(err, iorouter.ErrRouteNotFound) {
			return err
		}
	}
	return nil
}

func reconnectOwnedRoutes(env *Environment, routes []ioRoute) error {
	if env == nil || env.IO == nil || len(routes) == 0 {
		return nil
	}
	for _, route := range routes {
		err := env.IO.Connect(route.sourceID, route.targetID, route.streamKind)
		if err != nil && !errors.Is(err, iorouter.ErrRouteExists) {
			return err
		}
	}
	return nil
}

func nonSourceDeviceIDs(env *Environment, sourceID string) []string {
	if env == nil || env.Devices == nil {
		return nil
	}
	sourceID = strings.TrimSpace(sourceID)
	peers := make([]string, 0)
	for _, deviceID := range env.Devices.ListDeviceIDs() {
		deviceID = strings.TrimSpace(deviceID)
		if deviceID == "" || deviceID == sourceID {
			continue
		}
		peers = append(peers, deviceID)
	}
	return peers
}

func peerTargetDeviceIDs(env *Environment, sourceID string, args map[string]string) []string {
	if env == nil || env.Devices == nil {
		return nil
	}
	sourceID = strings.TrimSpace(sourceID)
	raw := ""
	if args != nil {
		raw = strings.TrimSpace(args["device_ids"])
	}
	if raw == "" {
		return nonSourceDeviceIDs(env, sourceID)
	}
	return filterPeerDeviceIDs(sourceID, strings.Split(raw, ","), peerDeviceIDSet(env, sourceID))
}

func peerDeviceIDSet(env *Environment, sourceID string) map[string]struct{} {
	validSet := map[string]struct{}{}
	for _, deviceID := range env.Devices.ListDeviceIDs() {
		trimmed := strings.TrimSpace(deviceID)
		if trimmed == "" || trimmed == sourceID {
			continue
		}
		validSet[trimmed] = struct{}{}
	}
	return validSet
}

func filterPeerDeviceIDs(sourceID string, parts []string, validSet map[string]struct{}) []string {
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		deviceID := strings.TrimSpace(part)
		if deviceID == "" || deviceID == sourceID {
			continue
		}
		if _, ok := validSet[deviceID]; !ok {
			continue
		}
		if _, exists := seen[deviceID]; exists {
			continue
		}
		seen[deviceID] = struct{}{}
		out = append(out, deviceID)
	}
	return out
}
