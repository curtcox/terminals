package scenario

import (
	"testing"
	"time"
)

func TestParseVoiceTriggerRedAlert(t *testing.T) {
	now := time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC)
	got := ParseVoiceTrigger("device-1", "Red Alert", now)
	if got.Intent != "red alert" {
		t.Fatalf("Intent = %q, want red alert", got.Intent)
	}
	if got.Kind != TriggerVoice {
		t.Fatalf("Kind = %q, want voice", got.Kind)
	}
}

func TestParseVoiceTriggerTimer(t *testing.T) {
	now := time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC)
	got := ParseVoiceTrigger("device-1", "set a timer for 10 minutes", now)
	if got.Intent != "set timer" {
		t.Fatalf("Intent = %q, want set timer", got.Intent)
	}
	if got.Arguments["minutes"] != "10" {
		t.Fatalf("minutes = %q, want 10", got.Arguments["minutes"])
	}
	if got.Arguments["fire_unix_ms"] == "" {
		t.Fatalf("fire_unix_ms should be populated")
	}
}

func TestParseVoiceTriggerTerminal(t *testing.T) {
	now := time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC)
	got := ParseVoiceTrigger("device-1", "open terminal", now)
	if got.Intent != "terminal" {
		t.Fatalf("Intent = %q, want terminal", got.Intent)
	}
}
