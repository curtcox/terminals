package transport

import (
	"strings"
	"sync"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

// RouteReplayStore captures the route set last observed for a device at
// disconnect time so that a subsequent reconnect can replay the
// StartStream/RouteStream messages even after the live router state was torn
// down. It owns its own mutex; StreamHandler does not share locks with it.
type RouteReplayStore struct {
	mu       sync.Mutex
	byDevice map[string][]iorouter.Route
}

// NewRouteReplayStore returns an empty replay store.
func NewRouteReplayStore() *RouteReplayStore {
	return &RouteReplayStore{byDevice: map[string][]iorouter.Route{}}
}

// Capture stores a copy of routes under deviceID. An empty deviceID is a
// no-op. Callers may freely mutate the slice they passed in afterward.
func (s *RouteReplayStore) Capture(deviceID string, routes []iorouter.Route) {
	key := strings.TrimSpace(deviceID)
	if key == "" || len(routes) == 0 {
		return
	}
	snapshot := make([]iorouter.Route, len(routes))
	copy(snapshot, routes)
	s.mu.Lock()
	s.byDevice[key] = snapshot
	s.mu.Unlock()
}

// Snapshot returns a copy of the routes captured for deviceID, or nil if none
// were captured. The returned slice is owned by the caller.
func (s *RouteReplayStore) Snapshot(deviceID string) []iorouter.Route {
	key := strings.TrimSpace(deviceID)
	if key == "" {
		return nil
	}
	s.mu.Lock()
	stored := s.byDevice[key]
	s.mu.Unlock()
	if len(stored) == 0 {
		return nil
	}
	out := make([]iorouter.Route, len(stored))
	copy(out, stored)
	return out
}

// Clear removes any captured snapshot for deviceID.
func (s *RouteReplayStore) Clear(deviceID string) {
	key := strings.TrimSpace(deviceID)
	if key == "" {
		return
	}
	s.mu.Lock()
	delete(s.byDevice, key)
	s.mu.Unlock()
}

// MessagesForDevice builds the StartStream + RouteStream pair for each route
// to replay on reconnect. When liveRoutes is non-empty it is used directly.
// Otherwise, if useCapturedFallback is true, the most recent captured
// snapshot is used instead. Both StartStream and RouteStream are emitted for
// each route, with origin=route_delta and webrtc_mode=server_managed metadata
// to match the live route-delta path.
func (s *RouteReplayStore) MessagesForDevice(deviceID string, liveRoutes []iorouter.Route, useCapturedFallback bool) []ServerMessage {
	routes := liveRoutes
	if len(routes) == 0 && useCapturedFallback {
		routes = s.Snapshot(deviceID)
	}
	if len(routes) == 0 {
		return nil
	}
	out := make([]ServerMessage, 0, 2*len(routes))
	for _, route := range routes {
		routeID := routeStreamID(route)
		routing := routeDeltaStreamRouting()
		out = append(out, ServerMessage{
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
		}, ServerMessage{
			RouteStream: &RouteStreamResponse{
				StreamID:       routeID,
				SourceDeviceID: route.SourceID,
				TargetDeviceID: route.TargetID,
				Kind:           route.StreamKind,
				Routing:        routing,
			},
		})
	}
	return out
}
