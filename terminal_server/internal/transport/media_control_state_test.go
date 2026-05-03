package transport

import (
	"context"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/recording"
)

func TestMediaControlStateRegistersStartsAndStopsRecording(t *testing.T) {
	state := NewMediaControlState()
	recorder := recording.NewMemoryManager()
	state.SetRecordingManager(recorder)

	state.RegisterStream(StartStreamResponse{
		StreamID:       "route:d1|d2|audio",
		Kind:           "audio",
		SourceDeviceID: "d1",
		TargetDeviceID: "d2",
		Metadata:       map[string]string{"origin": "test"},
	})

	active := recorder.Active()
	if _, ok := active["route:d1|d2|audio"]; !ok {
		t.Fatalf("expected recorder to track registered stream, got %+v", active)
	}

	state.UnregisterStream("route:d1|d2|audio")
	if active := recorder.Active(); len(active) != 0 {
		t.Fatalf("expected recorder to stop unregistered stream, got %+v", active)
	}
}

func TestMediaControlStateMarksUnknownStreamReady(t *testing.T) {
	state := NewMediaControlState()

	state.MarkStreamReady("stream-1")

	data := state.MediaStreamStatusData()
	if data["media_streams_active"] != "1" {
		t.Fatalf("media_streams_active = %q, want 1", data["media_streams_active"])
	}
	if data["media_streams_ready"] != "1" {
		t.Fatalf("media_streams_ready = %q, want 1", data["media_streams_ready"])
	}
	if data["media_streams_pending"] != "0" {
		t.Fatalf("media_streams_pending = %q, want 0", data["media_streams_pending"])
	}
}

func TestMediaControlStateServerManagedEngineUsesCopiedMetadata(t *testing.T) {
	state := NewMediaControlState()
	engine := &mediaControlFakeWebRTCSignalEngine{}
	state.SetWebRTCSignalEngine(engine)
	metadata := map[string]string{"webrtc_mode": "server_managed"}

	state.RegisterStream(StartStreamResponse{
		StreamID:       "route:d1|d2|audio",
		Kind:           "audio",
		SourceDeviceID: "d1",
		TargetDeviceID: "d2",
		Metadata:       metadata,
	})
	metadata["webrtc_mode"] = "peer_managed"

	gotEngine, serverManaged := state.ServerManagedSignalEngine("route:d1|d2|audio")
	if gotEngine != engine {
		t.Fatalf("engine = %v, want configured engine", gotEngine)
	}
	if !serverManaged {
		t.Fatal("expected stream to remain server-managed after caller metadata mutation")
	}
}

func TestMediaControlStatePeerLookupAndEngineRemoval(t *testing.T) {
	state := NewMediaControlState()
	engine := &mediaControlFakeWebRTCSignalEngine{}
	state.SetWebRTCSignalEngine(engine)

	state.RegisterStream(StartStreamResponse{
		StreamID:       "custom-stream",
		Kind:           "video",
		SourceDeviceID: "camera",
		TargetDeviceID: "display",
		Metadata:       map[string]string{"webrtc_mode": "server_managed"},
	})

	if peer := state.PeerDeviceForStream("custom-stream", "camera"); peer != "display" {
		t.Fatalf("peer for source = %q, want display", peer)
	}
	if peer := state.PeerDeviceForStream("custom-stream", "display"); peer != "camera" {
		t.Fatalf("peer for target = %q, want camera", peer)
	}

	state.UnregisterStream("custom-stream")
	if len(engine.removeCalls) != 1 || engine.removeCalls[0] != "custom-stream" {
		t.Fatalf("removeCalls = %+v, want [custom-stream]", engine.removeCalls)
	}
	if _, serverManaged := state.ServerManagedSignalEngine("custom-stream"); serverManaged {
		t.Fatal("expected unregistered stream to no longer be server-managed")
	}
}

type mediaControlFakeWebRTCSignalEngine struct {
	removeCalls []string
}

func (f *mediaControlFakeWebRTCSignalEngine) HandleSignal(context.Context, WebRTCSignalEngineRequest) ([]WebRTCSignalEngineResponse, error) {
	return nil, nil
}

func (f *mediaControlFakeWebRTCSignalEngine) RemoveStream(streamID string) {
	f.removeCalls = append(f.removeCalls, streamID)
}
