package io

import (
	"errors"
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
}

// NewRouter creates an empty route registry.
func NewRouter() *Router {
	return &Router{routes: make(map[string]Route)}
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

func routeKey(sourceID, targetID, streamKind string) string {
	return sourceID + "|" + targetID + "|" + streamKind
}
