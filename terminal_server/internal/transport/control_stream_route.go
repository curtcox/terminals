package transport

import (
	"context"
	"strings"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

func (h *StreamHandler) appendRouteMessageForPeers(
	out []ServerMessage,
	sessionDeviceID string,
	sourceDeviceID string,
	targetDeviceID string,
	msg ServerMessage,
) []ServerMessage {
	peers := []string{}
	seen := map[string]struct{}{}
	for _, deviceID := range []string{sourceDeviceID, targetDeviceID} {
		deviceID = strings.TrimSpace(deviceID)
		if deviceID == "" {
			continue
		}
		if _, exists := seen[deviceID]; exists {
			continue
		}
		seen[deviceID] = struct{}{}
		peers = append(peers, deviceID)
	}
	sessionDeviceID = strings.TrimSpace(sessionDeviceID)
	for _, peerDeviceID := range peers {
		next := msg
		if peerDeviceID != sessionDeviceID {
			next.RelayToDeviceID = peerDeviceID
		}
		out = append(out, next)
	}
	return out
}

func (h *StreamHandler) routeSnapshotForDevice(deviceID string) []iorouter.Route {
	if h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.IO == nil {
		return nil
	}
	routeProvider, ok := h.runtime.Env.IO.(interface {
		RoutesForDevice(deviceID string) []iorouter.Route
	})
	if !ok {
		return nil
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil
	}
	return routeProvider.RoutesForDevice(deviceID)
}

func routeStreamID(route iorouter.Route) string {
	return "route:" + route.SourceID + "|" + route.TargetID + "|" + route.StreamKind
}

func (h *StreamHandler) relayWebRTCSignal(signal *WebRTCSignalRequest, sourceDeviceID string) []ServerMessage {
	if signal == nil {
		return nil
	}
	streamID := strings.TrimSpace(signal.StreamID)
	signalType := strings.TrimSpace(signal.SignalType)
	if streamID == "" || signalType == "" {
		return nil
	}
	peerDeviceID := h.peerDeviceForStream(streamID, strings.TrimSpace(sourceDeviceID))
	if peerDeviceID == "" {
		return nil
	}
	return []ServerMessage{
		{
			WebRTCSignal: &WebRTCSignalResponse{
				StreamID:   streamID,
				SignalType: signalType,
				Payload:    signal.Payload,
			},
			RelayToDeviceID: peerDeviceID,
		},
	}
}

func (h *StreamHandler) handleWebRTCSignal(
	ctx context.Context,
	signal *WebRTCSignalRequest,
	sourceDeviceID string,
) []ServerMessage {
	if signal == nil {
		return nil
	}
	sourceDeviceID = strings.TrimSpace(sourceDeviceID)
	streamID := strings.TrimSpace(signal.StreamID)
	if streamID == "" || sourceDeviceID == "" {
		return h.relayWebRTCSignal(signal, sourceDeviceID)
	}

	engine, serverManaged := h.serverManagedSignalEngine(streamID)
	if serverManaged && engine != nil {
		if out, ok := serverManagedWebRTCMessages(ctx, engine, streamID, sourceDeviceID, *signal); ok {
			return out
		}
	}
	return h.relayWebRTCSignal(signal, sourceDeviceID)
}

func serverManagedWebRTCMessages(
	ctx context.Context,
	engine WebRTCSignalEngine,
	streamID, sourceDeviceID string,
	signal WebRTCSignalRequest,
) ([]ServerMessage, bool) {
	responses, err := engine.HandleSignal(ctx, WebRTCSignalEngineRequest{
		StreamID: streamID,
		DeviceID: sourceDeviceID,
		Signal:   signal,
	})
	if err != nil {
		return nil, false
	}
	out := make([]ServerMessage, 0, len(responses))
	for _, response := range responses {
		msg, ok := serverMessageFromWebRTCEngineResponse(response, sourceDeviceID)
		if !ok {
			continue
		}
		out = append(out, msg)
	}
	return out, true
}

func serverMessageFromWebRTCEngineResponse(response WebRTCSignalEngineResponse, sourceDeviceID string) (ServerMessage, bool) {
	msg := ServerMessage{
		WebRTCSignal: &WebRTCSignalResponse{
			StreamID:   strings.TrimSpace(response.Signal.StreamID),
			SignalType: strings.TrimSpace(response.Signal.SignalType),
			Payload:    response.Signal.Payload,
		},
	}
	target := strings.TrimSpace(response.TargetDeviceID)
	if msg.WebRTCSignal.StreamID == "" || msg.WebRTCSignal.SignalType == "" || target == "" {
		return ServerMessage{}, false
	}
	if target != sourceDeviceID {
		msg.RelayToDeviceID = target
	}
	return msg, true
}

func (h *StreamHandler) disconnectScenarioRoutes(deviceID, scenarioName string) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return
	}
	if h.runtime == nil || h.runtime.Env == nil || h.runtime.Env.IO == nil {
		return
	}
	routeProvider, ok := h.runtime.Env.IO.(interface {
		RoutesForDevice(deviceID string) []iorouter.Route
		Disconnect(sourceID, targetID, streamKind string) error
	})
	if !ok {
		return
	}

	for _, route := range routeProvider.RoutesForDevice(deviceID) {
		if !isScenarioOwnedRoute(deviceID, scenarioName, route) {
			continue
		}
		_ = routeProvider.Disconnect(route.SourceID, route.TargetID, route.StreamKind)
	}
}

func isScenarioOwnedRoute(deviceID, scenarioName string, route iorouter.Route) bool {
	switch scenarioName {
	case "intercom":
		return route.StreamKind == "audio" && (route.SourceID == deviceID || route.TargetID == deviceID)
	case "internal_video_call":
		if route.StreamKind != "audio" && route.StreamKind != "video" {
			return false
		}
		return route.SourceID == deviceID || route.TargetID == deviceID
	case "pa_system":
		return route.SourceID == deviceID && route.StreamKind == "pa_audio"
	case "announcement":
		return route.SourceID == deviceID && route.StreamKind == "announcement_audio"
	case "multi_window":
		if route.TargetID != deviceID {
			return false
		}
		return route.StreamKind == "video" || route.StreamKind == "audio_mix" || route.StreamKind == "audio"
	default:
		return false
	}
}
