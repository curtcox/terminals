package transport

import (
	"sync"
	"testing"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

func sampleRoutes() []iorouter.Route {
	return []iorouter.Route{
		{SourceID: "src-1", TargetID: "tgt-1", StreamKind: "audio"},
		{SourceID: "src-2", TargetID: "tgt-2", StreamKind: "video"},
	}
}

// TestMessagesForDevicePrefersLiveOverCaptured pins that live routes win
// when both are present.
func TestMessagesForDevicePrefersLiveOverCaptured(t *testing.T) {
	store := NewRouteReplayStore()
	store.Capture("dev", []iorouter.Route{{SourceID: "captured", TargetID: "t", StreamKind: "audio"}})

	live := []iorouter.Route{{SourceID: "live", TargetID: "t", StreamKind: "audio"}}
	out := store.MessagesForDevice("dev", live, true)
	if len(out) != 2 {
		t.Fatalf("expected 2 messages (StartStream+RouteStream), got %d", len(out))
	}
	if out[0].StartStream == nil || out[0].StartStream.SourceDeviceID != "live" {
		t.Fatalf("expected live route to be used, got %#v", out[0].StartStream)
	}
}

// TestMessagesForDeviceFallbackUsesCapturedWhenLiveEmpty pins the snap
// branch's reconnect replay behavior.
func TestMessagesForDeviceFallbackUsesCapturedWhenLiveEmpty(t *testing.T) {
	store := NewRouteReplayStore()
	store.Capture("dev", sampleRoutes())

	out := store.MessagesForDevice("dev", nil, true)
	if len(out) != 4 {
		t.Fatalf("expected 4 messages for 2 routes, got %d", len(out))
	}
	if out[0].StartStream == nil || out[1].RouteStream == nil {
		t.Fatalf("expected StartStream then RouteStream, got %#v %#v", out[0], out[1])
	}
}

// TestMessagesForDeviceNoFallbackReturnsNilWhenLiveEmpty pins the register
// branch's behavior of NOT using captured replay.
func TestMessagesForDeviceNoFallbackReturnsNilWhenLiveEmpty(t *testing.T) {
	store := NewRouteReplayStore()
	store.Capture("dev", sampleRoutes())

	out := store.MessagesForDevice("dev", nil, false)
	if out != nil {
		t.Fatalf("expected nil without fallback, got %#v", out)
	}
}

// TestMessagesForDeviceMetadataMatchesRouteDelta pins the metadata that
// reconnect replays must emit so clients treat them like live route deltas.
func TestMessagesForDeviceMetadataMatchesRouteDelta(t *testing.T) {
	store := NewRouteReplayStore()
	out := store.MessagesForDevice("dev", sampleRoutes()[:1], false)
	if len(out) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(out))
	}
	meta := out[0].StartStream.Metadata
	if meta["origin"] != "route_delta" {
		t.Fatalf("expected origin=route_delta, got %q", meta["origin"])
	}
	if meta["webrtc_mode"] != "server_managed" {
		t.Fatalf("expected webrtc_mode=server_managed, got %q", meta["webrtc_mode"])
	}
	if out[1].RouteStream.StreamID != out[0].StartStream.StreamID {
		t.Fatalf("StartStream/RouteStream stream IDs must match: %q vs %q", out[0].StartStream.StreamID, out[1].RouteStream.StreamID)
	}
}

// TestCaptureIsolatedByDevice pins that one device's snapshot does not
// leak into another's replay.
func TestCaptureIsolatedByDevice(t *testing.T) {
	store := NewRouteReplayStore()
	store.Capture("a", []iorouter.Route{{SourceID: "a-src", TargetID: "t", StreamKind: "audio"}})
	store.Capture("b", []iorouter.Route{{SourceID: "b-src", TargetID: "t", StreamKind: "audio"}})

	outA := store.MessagesForDevice("a", nil, true)
	outB := store.MessagesForDevice("b", nil, true)
	if outA[0].StartStream.SourceDeviceID != "a-src" {
		t.Fatalf("device a got wrong source: %q", outA[0].StartStream.SourceDeviceID)
	}
	if outB[0].StartStream.SourceDeviceID != "b-src" {
		t.Fatalf("device b got wrong source: %q", outB[0].StartStream.SourceDeviceID)
	}
}

// TestClearRemovesOnlyOneDevice pins that Clear is scoped to one device.
func TestClearRemovesOnlyOneDevice(t *testing.T) {
	store := NewRouteReplayStore()
	store.Capture("a", sampleRoutes())
	store.Capture("b", sampleRoutes())

	store.Clear("a")

	if got := store.Snapshot("a"); got != nil {
		t.Fatalf("expected device a cleared, got %#v", got)
	}
	if got := store.Snapshot("b"); len(got) != 2 {
		t.Fatalf("expected device b untouched, got %#v", got)
	}
}

// TestSnapshotReturnsCopy pins that mutating the returned slice does not
// corrupt internal state.
func TestSnapshotReturnsCopy(t *testing.T) {
	store := NewRouteReplayStore()
	store.Capture("dev", sampleRoutes())
	snap := store.Snapshot("dev")
	snap[0].SourceID = "mutated"

	again := store.Snapshot("dev")
	if again[0].SourceID == "mutated" {
		t.Fatalf("internal state was mutated through returned slice")
	}
}

// TestCaptureCopiesInput pins that mutating the input slice after Capture
// does not corrupt internal state.
func TestCaptureCopiesInput(t *testing.T) {
	store := NewRouteReplayStore()
	routes := sampleRoutes()
	store.Capture("dev", routes)
	routes[0].SourceID = "mutated"

	snap := store.Snapshot("dev")
	if snap[0].SourceID == "mutated" {
		t.Fatalf("internal state was mutated through caller's input slice")
	}
}

// TestEmptyDeviceIDIsNoOp pins that whitespace/empty IDs are ignored.
func TestEmptyDeviceIDIsNoOp(t *testing.T) {
	store := NewRouteReplayStore()
	store.Capture("", sampleRoutes())
	store.Capture("   ", sampleRoutes())
	if store.Snapshot("") != nil {
		t.Fatalf("expected nil snapshot for empty id")
	}
}

// TestConcurrentCaptureAndRead exercises -race for concurrent writers and
// readers across multiple devices.
func TestConcurrentCaptureAndRead(t *testing.T) {
	store := NewRouteReplayStore()
	const goroutines = 16
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			device := "dev"
			if id%2 == 0 {
				device = "other"
			}
			for j := 0; j < iterations; j++ {
				store.Capture(device, sampleRoutes())
			}
		}(i)
		go func(id int) {
			defer wg.Done()
			device := "dev"
			if id%2 == 0 {
				device = "other"
			}
			for j := 0; j < iterations; j++ {
				_ = store.MessagesForDevice(device, nil, true)
				_ = store.Snapshot(device)
			}
		}(i)
	}
	wg.Wait()
}
