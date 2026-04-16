// Package observation stores recent observations and evidence artifacts.
package observation

import (
	"context"
	"strings"
	"sync"
	"time"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

// Store keeps recent observations and artifacts for sensing scenarios.
type Store struct {
	mu           sync.RWMutex
	observations []iorouter.Observation
	artifacts    map[string]iorouter.ArtifactRef
	max          int
}

// NewStore returns an in-memory observation store.
func NewStore(capacity int) *Store {
	if capacity <= 0 {
		capacity = 2048
	}
	return &Store{
		artifacts: make(map[string]iorouter.ArtifactRef),
		max:       capacity,
	}
}

// AddObservation stores one observation and referenced artifacts.
func (s *Store) AddObservation(_ context.Context, observation iorouter.Observation) {
	if s == nil {
		return
	}
	if observation.OccurredAt.IsZero() {
		observation.OccurredAt = time.Now().UTC()
	}
	if observation.Attributes == nil {
		observation.Attributes = map[string]string{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.observations = append(s.observations, observation)
	if over := len(s.observations) - s.max; over > 0 {
		s.observations = append([]iorouter.Observation(nil), s.observations[over:]...)
	}
	for _, artifact := range observation.Evidence {
		if artifact.ID == "" {
			continue
		}
		s.artifacts[artifact.ID] = artifact
	}
}

// Recent returns observations filtered by kind/zone/since.
func (s *Store) Recent(_ context.Context, kind, zone string, since time.Time) []iorouter.Observation {
	if s == nil {
		return nil
	}
	kind = strings.TrimSpace(strings.ToLower(kind))
	zone = strings.TrimSpace(strings.ToLower(zone))

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]iorouter.Observation, 0, len(s.observations))
	for i := len(s.observations) - 1; i >= 0; i-- {
		ob := s.observations[i]
		if !since.IsZero() && ob.OccurredAt.Before(since) {
			continue
		}
		if kind != "" && !strings.Contains(strings.ToLower(ob.Kind), kind) {
			continue
		}
		if zone != "" && strings.ToLower(strings.TrimSpace(ob.Zone)) != zone {
			continue
		}
		out = append(out, ob)
	}
	return out
}

// Artifact looks up one artifact by ID.
func (s *Store) Artifact(_ context.Context, artifactID string) (iorouter.ArtifactRef, bool) {
	if s == nil {
		return iorouter.ArtifactRef{}, false
	}
	artifactID = strings.TrimSpace(artifactID)
	if artifactID == "" {
		return iorouter.ArtifactRef{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	artifact, ok := s.artifacts[artifactID]
	return artifact, ok
}
