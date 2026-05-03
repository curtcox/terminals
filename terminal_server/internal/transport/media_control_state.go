package transport

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/curtcox/terminals/terminal_server/internal/recording"
)

// MediaControlState owns media stream lifecycle state, recording hooks, and
// server-managed WebRTC signaling configuration.
type MediaControlState struct {
	mu sync.Mutex

	streams   map[string]mediaStreamState
	recording recording.Manager
	webrtc    WebRTCSignalEngine
}

func NewMediaControlState() *MediaControlState {
	return &MediaControlState{
		streams:   map[string]mediaStreamState{},
		recording: recording.NoopManager{},
	}
}

func (m *MediaControlState) SetRecordingManager(mgr recording.Manager) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if mgr == nil {
		m.recording = recording.NoopManager{}
		return
	}
	m.recording = mgr
}

func (m *MediaControlState) SetWebRTCSignalEngine(engine WebRTCSignalEngine) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.webrtc = engine
}

func (m *MediaControlState) CurrentRecordingManager() recording.Manager {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.recording
}

func (m *MediaControlState) ServerManagedSignalEngine(streamID string) (WebRTCSignalEngine, bool) {
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		return nil, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.webrtc == nil {
		return nil, false
	}
	state, ok := m.streams[streamID]
	if !ok {
		return nil, false
	}
	mode := strings.ToLower(strings.TrimSpace(state.Metadata["webrtc_mode"]))
	return m.webrtc, mode == "server_managed"
}

func (m *MediaControlState) PeerDeviceForStream(streamID, sourceDeviceID string) string {
	streamID = strings.TrimSpace(streamID)
	sourceDeviceID = strings.TrimSpace(sourceDeviceID)

	m.mu.Lock()
	state, ok := m.streams[streamID]
	m.mu.Unlock()
	if !ok {
		return ""
	}
	if sourceDeviceID == state.SourceDeviceID {
		return state.TargetDeviceID
	}
	if sourceDeviceID == state.TargetDeviceID {
		return state.SourceDeviceID
	}
	return ""
}

func (m *MediaControlState) RegisterStream(start StartStreamResponse) {
	streamID := strings.TrimSpace(start.StreamID)
	if streamID == "" {
		return
	}
	metadata := map[string]string{}
	for k, v := range start.Metadata {
		metadata[k] = v
	}
	state := mediaStreamState{
		StreamID:       streamID,
		Kind:           start.Kind,
		SourceDeviceID: start.SourceDeviceID,
		TargetDeviceID: start.TargetDeviceID,
		Metadata:       metadata,
		Ready:          false,
	}

	m.mu.Lock()
	m.streams[streamID] = state
	recorder := m.recording
	m.mu.Unlock()

	if recorder != nil {
		_ = recorder.Start(context.Background(), recording.Stream{
			StreamID:       streamID,
			Kind:           start.Kind,
			SourceDeviceID: start.SourceDeviceID,
			TargetDeviceID: start.TargetDeviceID,
			Metadata:       metadata,
		})
	}
}

func (m *MediaControlState) UnregisterStream(streamID string) {
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		return
	}
	m.mu.Lock()
	delete(m.streams, streamID)
	recorder := m.recording
	engine := m.webrtc
	m.mu.Unlock()

	if recorder != nil {
		_ = recorder.Stop(context.Background(), streamID)
	}
	if engine != nil {
		engine.RemoveStream(streamID)
	}
}

func (m *MediaControlState) MarkStreamReady(streamID string) {
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		return
	}
	m.mu.Lock()
	state, ok := m.streams[streamID]
	if !ok {
		state = mediaStreamState{
			StreamID: streamID,
			Kind:     "unknown",
		}
	}
	state.Ready = true
	m.streams[streamID] = state
	m.mu.Unlock()
}

func (m *MediaControlState) MediaStreamStatusData() map[string]string {
	streams := m.streamSnapshot()

	sort.Slice(streams, func(i, j int) bool {
		return streams[i].StreamID < streams[j].StreamID
	})

	ready := 0
	details := make([]string, 0, len(streams))
	for _, state := range streams {
		if state.Ready {
			ready++
		}
		details = append(details, fmt.Sprintf(
			"%s|%s|%s->%s|ready=%t",
			state.StreamID,
			state.Kind,
			state.SourceDeviceID,
			state.TargetDeviceID,
			state.Ready,
		))
	}

	return map[string]string{
		"media_streams_active":  strconv.Itoa(len(streams)),
		"media_streams_ready":   strconv.Itoa(ready),
		"media_streams_pending": strconv.Itoa(len(streams) - ready),
		"media_streams":         strings.Join(details, ";"),
	}
}

func (m *MediaControlState) RecordingStatusData() map[string]string {
	recorder := m.CurrentRecordingManager()
	activeReader, ok := recorder.(interface {
		Active() map[string]recording.Stream
	})
	if !ok {
		return map[string]string{
			"recording_active_streams": "0",
			"recording_stream_ids":     "",
		}
	}
	active := activeReader.Active()
	streamIDs := make([]string, 0, len(active))
	for streamID := range active {
		streamIDs = append(streamIDs, streamID)
	}
	sort.Strings(streamIDs)
	return map[string]string{
		"recording_active_streams": strconv.Itoa(len(streamIDs)),
		"recording_stream_ids":     strings.Join(streamIDs, ","),
	}
}

func (m *MediaControlState) RecentRecordingEvents(limit int) []recording.Event {
	recorder := m.CurrentRecordingManager()
	eventReader, ok := recorder.(interface {
		RecentEvents(limit int) []recording.Event
	})
	if !ok {
		return nil
	}
	return eventReader.RecentEvents(limit)
}

func (m *MediaControlState) ListPlaybackArtifacts() []recording.Artifact {
	recorder := m.CurrentRecordingManager()
	lister, ok := recorder.(interface {
		ListPlayableArtifacts() []recording.Artifact
	})
	if !ok {
		return nil
	}
	artifacts := lister.ListPlayableArtifacts()
	out := make([]recording.Artifact, len(artifacts))
	copy(out, artifacts)
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedUnixMS == out[j].UpdatedUnixMS {
			return out[i].ArtifactID < out[j].ArtifactID
		}
		return out[i].UpdatedUnixMS > out[j].UpdatedUnixMS
	})
	return out
}

func (m *MediaControlState) PlaybackMetadataForTarget(artifactID, targetDeviceID string) (recording.PlaybackMetadata, bool) {
	recorder := m.CurrentRecordingManager()
	provider, ok := recorder.(interface {
		PlaybackMetadata(artifactID, targetDeviceID string) (recording.PlaybackMetadata, bool)
	})
	if !ok {
		return recording.PlaybackMetadata{}, false
	}
	return provider.PlaybackMetadata(artifactID, targetDeviceID)
}

func (m *MediaControlState) streamSnapshot() []mediaStreamState {
	m.mu.Lock()
	defer m.mu.Unlock()
	streams := make([]mediaStreamState, 0, len(m.streams))
	for _, state := range m.streams {
		state.Metadata = copyMediaStringMap(state.Metadata)
		streams = append(streams, state)
	}
	return streams
}

func copyMediaStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
