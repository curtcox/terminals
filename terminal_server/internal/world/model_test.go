package world

import (
	"context"
	"testing"
	"time"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

func TestModelLocateAndVerifyDevice(t *testing.T) {
	model := NewModel()
	model.UpsertGeometry(context.Background(), DeviceGeometry{
		DeviceID:          "kitchen-tablet",
		Zone:              "kitchen",
		VerificationState: VerificationManual,
	})
	model.UpsertEntity(context.Background(), EntityRecord{
		EntityID:    "alice",
		Kind:        EntityPerson,
		DisplayName: "Alice",
		Confidence:  0.92,
		LastSeenAt:  time.Now().UTC(),
		LastKnown: &iorouter.LocationEstimate{
			Zone:       "kitchen",
			Confidence: 0.92,
		},
	})

	location, err := model.LocateEntity(context.Background(), EntityQuery{Person: "alice", MinConfidence: 0.5})
	if err != nil {
		t.Fatalf("LocateEntity() error = %v", err)
	}
	if location.Zone != "kitchen" {
		t.Fatalf("zone = %q, want kitchen", location.Zone)
	}

	if err := model.VerifyDevice(context.Background(), "kitchen-tablet", "marker"); err != nil {
		t.Fatalf("VerifyDevice() error = %v", err)
	}
	geometry, ok := model.Geometry(context.Background(), "kitchen-tablet")
	if !ok {
		t.Fatalf("expected geometry for kitchen-tablet")
	}
	if geometry.VerificationState != VerificationMarker {
		t.Fatalf("verification state = %q, want marker", geometry.VerificationState)
	}
}

func TestModelRecentObservations(t *testing.T) {
	now := time.Now().UTC()
	model := NewModel()

	model.AddObservation(context.Background(), iorouter.Observation{
		Kind:       "sound.classifier",
		Zone:       "kitchen",
		Subject:    "dishwasher",
		OccurredAt: now,
	})
	model.AddObservation(context.Background(), iorouter.Observation{
		Kind:       "bluetooth",
		Zone:       "garage",
		Subject:    "headphones",
		OccurredAt: now.Add(-2 * time.Minute),
	})

	observations, err := model.RecentObservations(context.Background(), "kitchen", "sound", now.Add(-time.Minute))
	if err != nil {
		t.Fatalf("RecentObservations() error = %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("len(observations) = %d, want 1", len(observations))
	}
	if observations[0].Subject != "dishwasher" {
		t.Fatalf("subject = %q, want dishwasher", observations[0].Subject)
	}
}
