package scenario

import (
	"context"
	"testing"
	"time"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

type stubObserveStore struct {
	out []iorouter.Observation
}

func (s stubObserveStore) Recent(context.Context, string, string, time.Time) []iorouter.Observation {
	return append([]iorouter.Observation(nil), s.out...)
}

func (s stubObserveStore) Artifact(context.Context, string) (iorouter.ArtifactRef, bool) {
	return iorouter.ArtifactRef{}, false
}

type stubWorldModel struct {
	people []EntityRecord
}

func (s stubWorldModel) LocateEntity(context.Context, EntityQuery) (*iorouter.LocationEstimate, error) {
	return &iorouter.LocationEstimate{Zone: "kitchen", Confidence: 0.8}, nil
}

func (s stubWorldModel) WhoIsHome(context.Context) ([]EntityRecord, error) {
	return append([]EntityRecord(nil), s.people...), nil
}

func (s stubWorldModel) VerifyDevice(context.Context, string, string) error {
	return nil
}

func (s stubWorldModel) RecentObservations(context.Context, string, string, time.Time) ([]iorouter.Observation, error) {
	return nil, nil
}

type stubBroadcast struct {
	last string
}

func (s *stubBroadcast) Notify(_ context.Context, _ []string, message string) error {
	s.last = message
	return nil
}

func TestSoundIdentificationScenarioStart(t *testing.T) {
	sc := &SoundIdentificationScenario{trigger: Trigger{SourceID: "d1"}}
	broadcast := &stubBroadcast{}
	err := sc.Start(context.Background(), &Environment{
		Observe: stubObserveStore{out: []iorouter.Observation{{
			Kind:       "sound.classifier",
			Subject:    "dog bark",
			Confidence: 0.91,
			OccurredAt: time.Now().UTC(),
		}}},
		Broadcast: broadcast,
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if broadcast.last == "" {
		t.Fatalf("expected a broadcast message")
	}
}

func TestPresenceQueryScenarioStart(t *testing.T) {
	sc := &PresenceQueryScenario{trigger: Trigger{SourceID: "d1"}}
	broadcast := &stubBroadcast{}
	err := sc.Start(context.Background(), &Environment{
		World: stubWorldModel{people: []EntityRecord{{
			EntityID:    "alice",
			DisplayName: "Alice",
			LastKnown:   &iorouter.LocationEstimate{Zone: "kitchen"},
			Confidence:  0.95,
		}}},
		Broadcast: broadcast,
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if broadcast.last == "" {
		t.Fatalf("expected a broadcast message")
	}
}
