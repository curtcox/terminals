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

func TestParseVoiceTriggerStandDownMapsToRedAlert(t *testing.T) {
	now := time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC)
	got := ParseVoiceTrigger("device-1", "stand down", now)
	if got.Intent != "red alert" {
		t.Fatalf("Intent = %q, want red alert", got.Intent)
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

func TestParseVoiceTriggerPhoneCallWithTarget(t *testing.T) {
	now := time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC)
	got := ParseVoiceTrigger("device-1", "call 5551212", now)
	if got.Intent != "phone call" {
		t.Fatalf("Intent = %q, want phone call", got.Intent)
	}
	if got.Arguments["target"] != "5551212" {
		t.Fatalf("target = %q, want 5551212", got.Arguments["target"])
	}
}

func TestParseVoiceTriggerAssistantQuery(t *testing.T) {
	now := time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC)
	got := ParseVoiceTrigger("device-1", "assistant weather tomorrow", now)
	if got.Intent != "voice assistant" {
		t.Fatalf("Intent = %q, want voice assistant", got.Intent)
	}
	if got.Arguments["query"] != "weather tomorrow" {
		t.Fatalf("query = %q, want weather tomorrow", got.Arguments["query"])
	}
}

func TestParseVoiceTriggerPAModeAlias(t *testing.T) {
	now := time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC)
	got := ParseVoiceTrigger("device-1", "PA mode", now)
	if got.Intent != "pa system" {
		t.Fatalf("Intent = %q, want pa system", got.Intent)
	}
}

func TestParseVoiceTriggerShowAllCamerasAlias(t *testing.T) {
	now := time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC)
	got := ParseVoiceTrigger("device-1", "show all cameras", now)
	if got.Intent != "multi window" {
		t.Fatalf("Intent = %q, want multi window", got.Intent)
	}
}

func TestParseVoiceTriggerPAStopAliases(t *testing.T) {
	now := time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC)
	for _, spoken := range []string{"end pa", "stop pa"} {
		got := ParseVoiceTrigger("device-1", spoken, now)
		if got.Intent != "pa system" {
			t.Fatalf("spoken=%q intent = %q, want pa system", spoken, got.Intent)
		}
	}
}
