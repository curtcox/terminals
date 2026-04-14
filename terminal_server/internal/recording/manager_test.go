package recording

import (
	"context"
	"testing"
)

func TestMemoryManagerStartStop(t *testing.T) {
	mgr := NewMemoryManager()

	if err := mgr.Start(context.Background(), Stream{
		StreamID:       "route:d1|d2|audio",
		Kind:           "audio",
		SourceDeviceID: "d1",
		TargetDeviceID: "d2",
		Metadata: map[string]string{
			"origin": "route_delta",
		},
	}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	active := mgr.Active()
	if len(active) != 1 {
		t.Fatalf("len(Active()) = %d, want 1", len(active))
	}
	if active["route:d1|d2|audio"].Kind != "audio" {
		t.Fatalf("active kind = %q, want audio", active["route:d1|d2|audio"].Kind)
	}

	if err := mgr.Stop(context.Background(), "route:d1|d2|audio"); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if len(mgr.Active()) != 0 {
		t.Fatalf("len(Active()) = %d, want 0", len(mgr.Active()))
	}
}
