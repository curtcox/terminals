// Package recording defines stream recording lifecycle hooks.
package recording

import (
	"context"
	"sync"
)

// Artifact describes one playable recording artifact discovered on disk.
type Artifact struct {
	ArtifactID     string
	StreamID       string
	Kind           string
	SourceDeviceID string
	TargetDeviceID string
	AudioPath      string
	SizeBytes      int64
	UpdatedUnixMS  int64
}

// PlaybackMetadata contains transport-ready metadata for requesting playback.
type PlaybackMetadata struct {
	Artifact       Artifact
	TargetDeviceID string
}

// Stream describes one active media route eligible for recording.
type Stream struct {
	StreamID       string
	Kind           string
	SourceDeviceID string
	TargetDeviceID string
	Metadata       map[string]string
}

// Manager handles start/stop recording lifecycle events for streams.
type Manager interface {
	Start(ctx context.Context, stream Stream) error
	Stop(ctx context.Context, streamID string) error
}

// NoopManager ignores recording lifecycle events.
type NoopManager struct{}

// Start is a no-op for NoopManager.
func (NoopManager) Start(context.Context, Stream) error { return nil }

// Stop is a no-op for NoopManager.
func (NoopManager) Stop(context.Context, string) error { return nil }

// MemoryManager stores stream recording state in memory for tests/scaffolding.
type MemoryManager struct {
	mu      sync.Mutex
	streams map[string]Stream
}

// NewMemoryManager creates an in-memory recording manager.
func NewMemoryManager() *MemoryManager {
	return &MemoryManager{
		streams: map[string]Stream{},
	}
}

// Start records or replaces an active recording stream by ID.
func (m *MemoryManager) Start(_ context.Context, stream Stream) error {
	streamID := stream.StreamID
	if streamID == "" {
		return nil
	}
	metadata := map[string]string{}
	for k, v := range stream.Metadata {
		metadata[k] = v
	}
	stream.Metadata = metadata

	m.mu.Lock()
	m.streams[streamID] = stream
	m.mu.Unlock()
	return nil
}

// Stop removes an active recording stream by ID.
func (m *MemoryManager) Stop(_ context.Context, streamID string) error {
	if streamID == "" {
		return nil
	}
	m.mu.Lock()
	delete(m.streams, streamID)
	m.mu.Unlock()
	return nil
}

// Active returns a copy of currently active recording streams.
func (m *MemoryManager) Active() map[string]Stream {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make(map[string]Stream, len(m.streams))
	for streamID, stream := range m.streams {
		metadata := map[string]string{}
		for k, v := range stream.Metadata {
			metadata[k] = v
		}
		copyStream := stream
		copyStream.Metadata = metadata
		out[streamID] = copyStream
	}
	return out
}

// ListPlayableArtifacts returns no artifacts for the in-memory manager.
func (m *MemoryManager) ListPlayableArtifacts() []Artifact {
	return nil
}

// PlaybackMetadata reports no playback metadata for the in-memory manager.
func (m *MemoryManager) PlaybackMetadata(string, string) (PlaybackMetadata, bool) {
	return PlaybackMetadata{}, false
}
