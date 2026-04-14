package recording

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiskManagerPersistsActiveIndexAndEvents(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewDiskManager(dir)
	if err != nil {
		t.Fatalf("NewDiskManager() error = %v", err)
	}

	stream := Stream{
		StreamID:       "route:d1|d2|audio",
		Kind:           "audio",
		SourceDeviceID: "d1",
		TargetDeviceID: "d2",
		Metadata: map[string]string{
			"origin": "route_delta",
		},
	}
	if err := mgr.Start(context.Background(), stream); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if len(mgr.Active()) != 1 {
		t.Fatalf("len(Active()) after start = %d, want 1", len(mgr.Active()))
	}

	activeJSON, err := os.ReadFile(filepath.Join(dir, "active.json"))
	if err != nil {
		t.Fatalf("ReadFile(active.json) error = %v", err)
	}
	if !strings.Contains(string(activeJSON), "route:d1|d2|audio") {
		t.Fatalf("active.json missing stream id: %s", string(activeJSON))
	}

	if err := mgr.Stop(context.Background(), "route:d1|d2|audio"); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if len(mgr.Active()) != 0 {
		t.Fatalf("len(Active()) after stop = %d, want 0", len(mgr.Active()))
	}

	eventLog, err := os.ReadFile(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatalf("ReadFile(events.jsonl) error = %v", err)
	}
	logText := string(eventLog)
	if !strings.Contains(logText, "\"action\":\"start\"") {
		t.Fatalf("events.jsonl missing start action: %s", logText)
	}
	if !strings.Contains(logText, "\"action\":\"stop\"") {
		t.Fatalf("events.jsonl missing stop action: %s", logText)
	}
	recent := mgr.RecentEvents(10)
	if len(recent) != 2 {
		t.Fatalf("len(RecentEvents(10)) = %d, want 2", len(recent))
	}
	if recent[0].Action != "start" || recent[1].Action != "stop" {
		t.Fatalf("RecentEvents actions = %q then %q, want start then stop", recent[0].Action, recent[1].Action)
	}
}

func TestDiskManagerLoadsExistingActiveIndex(t *testing.T) {
	dir := t.TempDir()
	existing := `{
  "route:d1|d3|video": {
    "StreamID": "route:d1|d3|video",
    "Kind": "video",
    "SourceDeviceID": "d1",
    "TargetDeviceID": "d3",
    "Metadata": {
      "origin": "restore"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "active.json"), []byte(existing), 0o644); err != nil {
		t.Fatalf("WriteFile(active.json) error = %v", err)
	}

	mgr, err := NewDiskManager(dir)
	if err != nil {
		t.Fatalf("NewDiskManager() error = %v", err)
	}
	active := mgr.Active()
	if len(active) != 1 {
		t.Fatalf("len(Active()) = %d, want 1", len(active))
	}
	if active["route:d1|d3|video"].Kind != "video" {
		t.Fatalf("kind = %q, want video", active["route:d1|d3|video"].Kind)
	}
}

func TestDiskManagerWriteDeviceAudio(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewDiskManager(dir)
	if err != nil {
		t.Fatalf("NewDiskManager() error = %v", err)
	}
	if err := mgr.Start(context.Background(), Stream{
		StreamID:       "route:d1|d2|audio",
		Kind:           "audio",
		SourceDeviceID: "d1",
		TargetDeviceID: "d2",
	}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	chunk := []byte{0x01, 0x02, 0x03, 0x04}
	if err := mgr.WriteDeviceAudio("d1", chunk); err != nil {
		t.Fatalf("WriteDeviceAudio() error = %v", err)
	}

	audioPath := filepath.Join(dir, "streams", "route_d1_d2_audio", "audio.raw")
	got, err := os.ReadFile(audioPath)
	if err != nil {
		t.Fatalf("ReadFile(audio.raw) error = %v", err)
	}
	if string(got) != string(chunk) {
		t.Fatalf("audio bytes = %v, want %v", got, chunk)
	}
}

func TestDiskManagerListPlayableArtifactsAndPlaybackMetadata(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewDiskManager(dir)
	if err != nil {
		t.Fatalf("NewDiskManager() error = %v", err)
	}
	if err := mgr.Start(context.Background(), Stream{
		StreamID:       "route:d3|d4|audio",
		Kind:           "audio",
		SourceDeviceID: "d3",
		TargetDeviceID: "d4",
	}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := mgr.WriteDeviceAudio("d3", []byte{0x10, 0x20, 0x30}); err != nil {
		t.Fatalf("WriteDeviceAudio() error = %v", err)
	}
	if err := mgr.Stop(context.Background(), "route:d3|d4|audio"); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	artifacts := mgr.ListPlayableArtifacts()
	if len(artifacts) != 1 {
		t.Fatalf("len(ListPlayableArtifacts()) = %d, want 1", len(artifacts))
	}
	if artifacts[0].ArtifactID != "route:d3|d4|audio" {
		t.Fatalf("ArtifactID = %q, want route:d3|d4|audio", artifacts[0].ArtifactID)
	}
	if artifacts[0].SizeBytes != 3 {
		t.Fatalf("SizeBytes = %d, want 3", artifacts[0].SizeBytes)
	}
	if artifacts[0].AudioPath == "" {
		t.Fatalf("AudioPath should not be empty")
	}

	metadata, ok := mgr.PlaybackMetadata("route:d3|d4|audio", "kitchen-display")
	if !ok {
		t.Fatalf("PlaybackMetadata() should resolve known artifact")
	}
	if metadata.TargetDeviceID != "kitchen-display" {
		t.Fatalf("TargetDeviceID = %q, want kitchen-display", metadata.TargetDeviceID)
	}
	if metadata.Artifact.StreamID != "route:d3|d4|audio" {
		t.Fatalf("StreamID = %q, want route:d3|d4|audio", metadata.Artifact.StreamID)
	}
	if _, ok := mgr.PlaybackMetadata("missing-artifact", "kitchen-display"); ok {
		t.Fatalf("PlaybackMetadata() should not resolve missing artifact")
	}
}
