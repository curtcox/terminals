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
