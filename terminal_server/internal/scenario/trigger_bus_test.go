package scenario

import (
	"testing"
	"time"
)

func TestSourceFromKindMapping(t *testing.T) {
	cases := []struct {
		name string
		kind TriggerKind
		want TriggerSource
	}{
		{name: "voice", kind: TriggerVoice, want: SourceVoice},
		{name: "schedule", kind: TriggerSchedule, want: SourceSchedule},
		{name: "event", kind: TriggerEvent, want: SourceEvent},
		{name: "cascade", kind: TriggerCascade, want: SourceCascade},
		{name: "manual", kind: TriggerManual, want: SourceManual},
		{name: "unknown defaults to manual", kind: TriggerKind("unknown"), want: SourceManual},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sourceFromKind(tc.kind); got != tc.want {
				t.Fatalf("sourceFromKind(%q) = %q, want %q", tc.kind, got, tc.want)
			}
		})
	}
}

func TestNormalizeTriggerDefaultsIntentAndEventSources(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	got := normalizeTrigger(Trigger{
		Kind:      TriggerSchedule,
		SourceID:  "  d1  ",
		Intent:    "  reminder  ",
		Arguments: map[string]string{"k": "v"},
		EventV2: &EventRecord{
			Kind: "  timer.fired  ",
		},
	}, now)

	if got.SourceID != "d1" {
		t.Fatalf("SourceID = %q, want d1", got.SourceID)
	}
	if got.Intent != "reminder" {
		t.Fatalf("Intent = %q, want reminder", got.Intent)
	}
	if got.IntentV2 == nil {
		t.Fatalf("expected IntentV2 to be populated")
	}
	if got.IntentV2.Source != SourceSchedule {
		t.Fatalf("IntentV2.Source = %q, want %q", got.IntentV2.Source, SourceSchedule)
	}
	if got.EventV2 == nil {
		t.Fatalf("expected EventV2 to remain populated")
	}
	if got.EventV2.Kind != "timer.fired" {
		t.Fatalf("EventV2.Kind = %q, want timer.fired", got.EventV2.Kind)
	}
	if got.EventV2.Source != SourceSchedule {
		t.Fatalf("EventV2.Source = %q, want %q", got.EventV2.Source, SourceSchedule)
	}
	if !got.EventV2.OccurredAt.Equal(now) {
		t.Fatalf("EventV2.OccurredAt = %v, want %v", got.EventV2.OccurredAt, now)
	}
}

func TestIntentEventBusPublishNormalizesBeforeFanout(t *testing.T) {
	bus := NewIntentEventBus()
	ch, cancel := bus.Subscribe(1)
	defer cancel()

	bus.Publish(Trigger{
		Kind:     TriggerVoice,
		SourceID: " d1 ",
		Intent:   " terminal ",
		EventV2: &EventRecord{
			Kind: " voice.detected ",
		},
	})

	select {
	case got := <-ch:
		if got.SourceID != "d1" {
			t.Fatalf("SourceID = %q, want d1", got.SourceID)
		}
		if got.Intent != "terminal" {
			t.Fatalf("Intent = %q, want terminal", got.Intent)
		}
		if got.IntentV2 == nil || got.IntentV2.Source != SourceVoice {
			t.Fatalf("IntentV2 source = %+v, want %q", got.IntentV2, SourceVoice)
		}
		if got.EventV2 == nil || got.EventV2.Source != SourceVoice {
			t.Fatalf("EventV2 source = %+v, want %q", got.EventV2, SourceVoice)
		}
		if got.EventV2.OccurredAt.IsZero() {
			t.Fatalf("expected EventV2.OccurredAt to be set")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timed out waiting for published trigger")
	}
}
