// Package io manages logical stream-routing state.
package io //nolint:revive

import (
	"errors"
	"sort"
	"sync"
)

var (
	// ErrRouteExists indicates the same route is already active.
	ErrRouteExists = errors.New("route already exists")
	// ErrRouteNotFound indicates a disconnect requested for a missing route.
	ErrRouteNotFound = errors.New("route not found")
)

// Route describes one logical stream connection.
type Route struct {
	SourceID   string
	TargetID   string
	StreamKind string
}

// Router tracks active logical stream routes.
type Router struct {
	mu     sync.RWMutex
	routes map[string]Route

	claims  *ClaimManager
	planner *MediaPlanner
}

// NewRouter creates an empty route registry.
func NewRouter() *Router {
	router := &Router{
		routes: make(map[string]Route),
		claims: NewClaimManager(),
	}
	router.planner = NewMediaPlanner(router)
	return router
}

// Connect creates a new logical route.
func (r *Router) Connect(sourceID, targetID, streamKind string) error {
	key := routeKey(sourceID, targetID, streamKind)
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.routes[key]; exists {
		return ErrRouteExists
	}
	r.routes[key] = Route{
		SourceID:   sourceID,
		TargetID:   targetID,
		StreamKind: streamKind,
	}
	return nil
}

// Disconnect removes an existing route.
func (r *Router) Disconnect(sourceID, targetID, streamKind string) error {
	key := routeKey(sourceID, targetID, streamKind)
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.routes[key]; !exists {
		return ErrRouteNotFound
	}
	delete(r.routes, key)
	return nil
}

// Routes returns a copy of all active routes.
func (r *Router) Routes() []Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Route, 0, len(r.routes))
	for _, v := range r.routes {
		out = append(out, v)
	}
	return out
}

// RouteCount returns the number of active routes.
func (r *Router) RouteCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.routes)
}

// Claims returns the shared in-memory claim manager.
func (r *Router) Claims() *ClaimManager {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.claims
}

// MediaPlanner returns the declarative media planner.
func (r *Router) MediaPlanner() *MediaPlanner {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.planner
}

// ConnectFanout creates one source->target route per target, skipping existing routes.
// It returns how many new routes were added.
func (r *Router) ConnectFanout(sourceID string, targetIDs []string, streamKind string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	added := 0
	for _, targetID := range targetIDs {
		key := routeKey(sourceID, targetID, streamKind)
		if _, exists := r.routes[key]; exists {
			continue
		}
		r.routes[key] = Route{
			SourceID:   sourceID,
			TargetID:   targetID,
			StreamKind: streamKind,
		}
		added++
	}
	return added
}

// DisconnectDevice removes any route where the given device is source or target.
// It returns how many routes were removed.
func (r *Router) DisconnectDevice(deviceID string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	removed := 0
	for key, route := range r.routes {
		if route.SourceID == deviceID || route.TargetID == deviceID {
			delete(r.routes, key)
			removed++
		}
	}
	return removed
}

// RoutesForDevice returns routes where the device appears as source or target.
func (r *Router) RoutesForDevice(deviceID string) []Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Route, 0)
	for _, route := range r.routes {
		if route.SourceID == deviceID || route.TargetID == deviceID {
			out = append(out, route)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SourceID == out[j].SourceID {
			if out[i].TargetID == out[j].TargetID {
				return out[i].StreamKind < out[j].StreamKind
			}
			return out[i].TargetID < out[j].TargetID
		}
		return out[i].SourceID < out[j].SourceID
	})
	return out
}

func routeKey(sourceID, targetID, streamKind string) string {
	return sourceID + "|" + targetID + "|" + streamKind
}
