package recording

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
)

// Event represents one persisted recording lifecycle event.
type Event struct {
	AtUnixMS int64  `json:"at_unix_ms"`
	Action   string `json:"action"`
	StreamID string `json:"stream_id"`
	Kind     string `json:"kind,omitempty"`
	SourceID string `json:"source_id,omitempty"`
	TargetID string `json:"target_id,omitempty"`
}

// DiskManager persists recording lifecycle metadata to disk.
//
// This is scaffolding for Phase-7 recording/playback: it indexes active
// stream recordings and appends lifecycle events so playback-oriented flows
// can later resolve recorded streams from durable metadata.
type DiskManager struct {
	mu     sync.Mutex
	dir    string
	active map[string]Stream
}

var nonPathSafe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// NewDiskManager creates or opens a metadata directory.
func NewDiskManager(dir string) (*DiskManager, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	mgr := &DiskManager{
		dir:    dir,
		active: map[string]Stream{},
	}
	if err := mgr.loadActiveIndex(); err != nil {
		return nil, err
	}
	return mgr, nil
}

// Start upserts active recording metadata and appends a "start" event.
func (m *DiskManager) Start(_ context.Context, stream Stream) error {
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
	m.active[streamID] = stream
	err := m.persistLocked()
	if err == nil {
		err = m.appendEventLocked(Event{
			AtUnixMS: time.Now().UnixMilli(),
			Action:   "start",
			StreamID: streamID,
			Kind:     stream.Kind,
			SourceID: stream.SourceDeviceID,
			TargetID: stream.TargetDeviceID,
		})
	}
	m.mu.Unlock()
	return err
}

// Stop removes active recording metadata and appends a "stop" event.
func (m *DiskManager) Stop(_ context.Context, streamID string) error {
	if streamID == "" {
		return nil
	}
	m.mu.Lock()
	stream, hadStream := m.active[streamID]
	delete(m.active, streamID)
	err := m.persistLocked()
	if err == nil {
		event := Event{
			AtUnixMS: time.Now().UnixMilli(),
			Action:   "stop",
			StreamID: streamID,
		}
		if hadStream {
			event.Kind = stream.Kind
			event.SourceID = stream.SourceDeviceID
			event.TargetID = stream.TargetDeviceID
		}
		err = m.appendEventLocked(event)
	}
	m.mu.Unlock()
	return err
}

// Active returns a copy of currently active recording metadata.
func (m *DiskManager) Active() map[string]Stream {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make(map[string]Stream, len(m.active))
	for streamID, stream := range m.active {
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

// WriteDeviceAudio appends raw audio bytes for active streams sourced by the
// provided device. Files are written under:
//
//	<recording-dir>/streams/<stream-id-sanitized>/audio.raw
func (m *DiskManager) WriteDeviceAudio(deviceID string, chunk []byte) error {
	if deviceID == "" || len(chunk) == 0 {
		return nil
	}
	m.mu.Lock()
	streamIDs := make([]string, 0, len(m.active))
	for streamID, stream := range m.active {
		if stream.SourceDeviceID == deviceID {
			streamIDs = append(streamIDs, streamID)
		}
	}
	dir := m.dir
	m.mu.Unlock()

	for _, streamID := range streamIDs {
		streamDir := filepath.Join(dir, "streams", sanitizePathComponent(streamID))
		if err := os.MkdirAll(streamDir, 0o755); err != nil {
			return err
		}
		audioPath := filepath.Join(streamDir, "audio.raw")
		f, err := os.OpenFile(audioPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		if _, err := f.Write(chunk); err != nil {
			_ = f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (m *DiskManager) activeIndexPath() string {
	return filepath.Join(m.dir, "active.json")
}

func (m *DiskManager) eventLogPath() string {
	return filepath.Join(m.dir, "events.jsonl")
}

// RecentEvents returns the most recent persisted lifecycle events, newest last.
// If limit <= 0, all events are returned.
func (m *DiskManager) RecentEvents(limit int) []Event {
	path := m.eventLogPath()
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	reader := bufio.NewReader(f)
	events := make([]Event, 0)
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			var event Event
			if err := json.Unmarshal(line, &event); err == nil {
				events = append(events, event)
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return events
		}
	}

	if limit <= 0 || len(events) <= limit {
		return events
	}
	return events[len(events)-limit:]
}

func (m *DiskManager) loadActiveIndex() error {
	path := m.activeIndexPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(raw) == 0 {
		return nil
	}
	var decoded map[string]Stream
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	m.active = map[string]Stream{}
	for streamID, stream := range decoded {
		if streamID == "" {
			continue
		}
		metadata := map[string]string{}
		for k, v := range stream.Metadata {
			metadata[k] = v
		}
		stream.Metadata = metadata
		m.active[streamID] = stream
	}
	return nil
}

func (m *DiskManager) persistLocked() error {
	path := m.activeIndexPath()
	encoded, err := json.MarshalIndent(m.active, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0o644)
}

func (m *DiskManager) appendEventLocked(event Event) error {
	path := m.eventLogPath()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	encoded, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := f.Write(encoded); err != nil {
		return err
	}
	if _, err := f.Write([]byte{'\n'}); err != nil {
		return err
	}
	eventlog.Emit(context.Background(), "recording.segment_flushed", slog.LevelInfo, "recording event flushed",
		slog.String("component", "recording.disk"),
		slog.String("action", event.Action),
		slog.String("stream_id", event.StreamID),
		slog.String("kind", event.Kind),
		slog.String("source_id", event.SourceID),
		slog.String("target_id", event.TargetID),
	)
	return nil
}

func sanitizePathComponent(value string) string {
	sanitized := nonPathSafe.ReplaceAllString(value, "_")
	if sanitized == "" {
		return "stream"
	}
	return sanitized
}

func (m *DiskManager) streamAudioPath(streamID string) string {
	return filepath.Join(m.dir, "streams", sanitizePathComponent(streamID), "audio.raw")
}

// ListPlayableArtifacts returns playable audio artifacts discovered in the
// recording directory. Artifacts are ordered by newest first.
func (m *DiskManager) ListPlayableArtifacts() []Artifact {
	candidates := map[string]Stream{}
	for streamID, stream := range m.Active() {
		candidates[streamID] = stream
	}
	for _, event := range m.RecentEvents(0) {
		if event.Action != "start" || strings.TrimSpace(event.StreamID) == "" {
			continue
		}
		stream := candidates[event.StreamID]
		stream.StreamID = event.StreamID
		if stream.Kind == "" {
			stream.Kind = event.Kind
		}
		if stream.SourceDeviceID == "" {
			stream.SourceDeviceID = event.SourceID
		}
		if stream.TargetDeviceID == "" {
			stream.TargetDeviceID = event.TargetID
		}
		candidates[event.StreamID] = stream
	}

	artifacts := make([]Artifact, 0, len(candidates))
	for streamID, stream := range candidates {
		audioPath := m.streamAudioPath(streamID)
		stat, err := os.Stat(audioPath)
		if err != nil || stat.Size() <= 0 {
			continue
		}
		artifacts = append(artifacts, Artifact{
			ArtifactID:     streamID,
			StreamID:       streamID,
			Kind:           stream.Kind,
			SourceDeviceID: stream.SourceDeviceID,
			TargetDeviceID: stream.TargetDeviceID,
			AudioPath:      audioPath,
			SizeBytes:      stat.Size(),
			UpdatedUnixMS:  stat.ModTime().UnixMilli(),
		})
	}

	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].UpdatedUnixMS == artifacts[j].UpdatedUnixMS {
			return artifacts[i].ArtifactID < artifacts[j].ArtifactID
		}
		return artifacts[i].UpdatedUnixMS > artifacts[j].UpdatedUnixMS
	})
	return artifacts
}

// PlaybackMetadata resolves one playback artifact for a target device.
func (m *DiskManager) PlaybackMetadata(artifactID, targetDeviceID string) (PlaybackMetadata, bool) {
	artifactID = strings.TrimSpace(artifactID)
	targetDeviceID = strings.TrimSpace(targetDeviceID)
	if artifactID == "" || targetDeviceID == "" {
		return PlaybackMetadata{}, false
	}
	for _, artifact := range m.ListPlayableArtifacts() {
		if artifact.ArtifactID == artifactID {
			return PlaybackMetadata{
				Artifact:       artifact,
				TargetDeviceID: targetDeviceID,
			}, true
		}
	}
	return PlaybackMetadata{}, false
}
