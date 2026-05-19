package transport

import (
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

func routeSetFromRoutes(routes []iorouter.Route) map[string]struct{} {
	out := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		out[routeStreamID(route)] = struct{}{}
	}
	return out
}

func routeMapFromRoutes(routes []iorouter.Route) map[string]iorouter.Route {
	out := make(map[string]iorouter.Route, len(routes))
	for _, route := range routes {
		out[routeStreamID(route)] = route
	}
	return out
}

func (h *StreamHandler) routeStartUpdatesForCommand(
	cmd *CommandRequest,
	before []iorouter.Route,
	after []iorouter.Route,
) []ServerMessage {
	beforeSet := routeSetFromRoutes(before)
	afterSet := routeMapFromRoutes(after)
	out := make([]ServerMessage, 0, len(after))
	for _, route := range after {
		routeID := routeStreamID(route)
		if _, exists := beforeSet[routeID]; exists {
			continue
		}
		out = h.appendRouteStartMessages(out, cmd, route, routeID)
	}
	for _, route := range before {
		routeID := routeStreamID(route)
		if _, exists := afterSet[routeID]; exists {
			continue
		}
		stopMsg := ServerMessage{
			StopStream: &StopStreamResponse{StreamID: routeID},
		}
		out = h.appendRouteMessageForPeers(out, cmd.DeviceID, route.SourceID, route.TargetID, stopMsg)
		h.unregisterMediaStream(routeID)
	}
	return out
}

func (h *StreamHandler) appendRouteStartMessages(
	out []ServerMessage,
	cmd *CommandRequest,
	route iorouter.Route,
	routeID string,
) []ServerMessage {
	routing := routeDeltaStreamRouting()
	startMsg := ServerMessage{
		StartStream: &StartStreamResponse{
			StreamID:       routeID,
			Kind:           route.StreamKind,
			SourceDeviceID: route.SourceID,
			TargetDeviceID: route.TargetID,
			Metadata: map[string]string{
				"origin":      "route_delta",
				"webrtc_mode": "server_managed",
			},
			Routing: routing,
		},
	}
	out = h.appendRouteMessageForPeers(out, cmd.DeviceID, route.SourceID, route.TargetID, startMsg)
	h.registerMediaStream(StartStreamResponse{
		StreamID:       routeID,
		Kind:           route.StreamKind,
		SourceDeviceID: route.SourceID,
		TargetDeviceID: route.TargetID,
		Metadata: map[string]string{
			"origin":      "route_delta",
			"webrtc_mode": "server_managed",
		},
		Routing: routing,
	})
	routeMsg := ServerMessage{
		RouteStream: &RouteStreamResponse{
			StreamID:       routeID,
			SourceDeviceID: route.SourceID,
			TargetDeviceID: route.TargetID,
			Kind:           route.StreamKind,
			Routing:        routing,
		},
	}
	return h.appendRouteMessageForPeers(out, cmd.DeviceID, route.SourceID, route.TargetID, routeMsg)
}

func (h *StreamHandler) routeStopUpdatesForCommand(
	cmd *CommandRequest,
	before []iorouter.Route,
	after []iorouter.Route,
) []ServerMessage {
	afterSet := routeMapFromRoutes(after)
	out := make([]ServerMessage, 0, len(before))
	for _, route := range before {
		routeID := routeStreamID(route)
		if _, exists := afterSet[routeID]; exists {
			continue
		}
		stopMsg := ServerMessage{
			StopStream: &StopStreamResponse{StreamID: routeID},
		}
		out = h.appendRouteMessageForPeers(out, cmd.DeviceID, route.SourceID, route.TargetID, stopMsg)
		h.unregisterMediaStream(routeID)
	}
	return out
}
