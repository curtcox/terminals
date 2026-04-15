package observation

import (
	"context"
	"testing"
	"time"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

func TestStoreRecentAndArtifactLookup(t *testing.T) {
	store := NewStore(4)
	now := time.Now().UTC()
	store.AddObservation(context.Background(), iorouter.Observation{
		Kind:       "sound.detected",
		Subject:    "dishwasher",
		Zone:       "kitchen",
		OccurredAt: now,
		Evidence: []iorouter.ArtifactRef{
			{ID: "a1", Kind: "audio_clip", URI: "artifact://a1"},
		},
	})

	observations := store.Recent(context.Background(), "sound", "kitchen", now.Add(-time.Second))
	if len(observations) != 1 {
		t.Fatalf("len(observations) = %d, want 1", len(observations))
	}
	if observations[0].Subject != "dishwasher" {
		t.Fatalf("subject = %q, want dishwasher", observations[0].Subject)
	}

	artifact, ok := store.Artifact(context.Background(), "a1")
	if !ok {
		t.Fatalf("artifact a1 should be available")
	}
	if artifact.URI != "artifact://a1" {
		t.Fatalf("artifact URI = %q, want artifact://a1", artifact.URI)
	}
}
