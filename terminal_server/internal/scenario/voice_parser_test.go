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
	for _, spoken := range []string{"stand down", "stop red alert"} {
		got := ParseVoiceTrigger("device-1", spoken, now)
		if got.Intent != "red alert" {
			t.Fatalf("spoken=%q intent = %q, want red alert", spoken, got.Intent)
		}
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

func TestParseVoiceTriggerInternalVideoCallAlias(t *testing.T) {
	now := time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC)
	got := ParseVoiceTrigger("device-1", "video call", now)
	if got.Intent != "internal video call" {
		t.Fatalf("Intent = %q, want internal video call", got.Intent)
	}
}

func TestParseVoiceTriggerInternalVideoCallWithTarget(t *testing.T) {
	now := time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC)
	got := ParseVoiceTrigger("device-1", "video call d2", now)
	if got.Intent != "internal video call" {
		t.Fatalf("Intent = %q, want internal video call", got.Intent)
	}
	if got.Arguments["target_device_id"] != "d2" {
		t.Fatalf("target_device_id = %q, want d2", got.Arguments["target_device_id"])
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

func TestParseVoiceTriggerAllCamerasAlias(t *testing.T) {
	now := time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC)
	got := ParseVoiceTrigger("device-1", "all cameras", now)
	if got.Intent != "multi window" {
		t.Fatalf("Intent = %q, want multi window", got.Intent)
	}
}

func TestParseVoiceTriggerAllCamerasFocusAlias(t *testing.T) {
	now := time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC)
	got := ParseVoiceTrigger("device-1", "all cameras focus d2", now)
	if got.Intent != "multi window" {
		t.Fatalf("Intent = %q, want multi window", got.Intent)
	}
	if got.Arguments["audio_focus_device_id"] != "d2" {
		t.Fatalf("audio_focus_device_id = %q, want d2", got.Arguments["audio_focus_device_id"])
	}
}

func TestParseVoiceTriggerAudioMonitorTellMeWhenPhrasing(t *testing.T) {
	now := time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC)
	cases := []struct {
		spoken string
		target string
	}{
		{"tell me when the dishwasher stops", "dishwasher"},
		{"tell me when the dryer beeps", "dryer"},
		{"tell me when the microwave is done", "microwave"},
		{"tell me when the laundry finishes", "laundry"},
		{"notify me when the oven beeps", "oven"},
		{"Tell me when the Dishwasher Stops", "dishwasher"},
	}
	for _, tc := range cases {
		got := ParseVoiceTrigger("device-1", tc.spoken, now)
		if got.Intent != "audio monitor" {
			t.Fatalf("spoken=%q intent = %q, want audio monitor", tc.spoken, got.Intent)
		}
		if got.Arguments["target"] != tc.target {
			t.Fatalf("spoken=%q target = %q, want %q", tc.spoken, got.Arguments["target"], tc.target)
		}
	}
}

func TestParseVoiceTriggerAudioMonitorRejectsEmptyTarget(t *testing.T) {
	now := time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC)
	// Unrelated spoken text should not be coerced into audio_monitor, and
	// the normalized lowercased form should be used as a fallback intent.
	got := ParseVoiceTrigger("device-1", "hello world", now)
	if got.Intent == "audio monitor" {
		t.Fatalf("unrelated text coerced to audio monitor; got %+v", got)
	}
	if got.Intent != "hello world" {
		t.Fatalf("Intent = %q, want passthrough 'hello world'", got.Intent)
	}
	if _, ok := got.Arguments["target"]; ok {
		t.Fatalf("target should not be set for unrelated text; got %+v", got.Arguments)
	}

	// "tell me when the" with no subject following should also not arm the
	// monitor — parseAudioMonitorTarget must return false.
	got = ParseVoiceTrigger("device-1", "tell me when the ", now)
	if got.Intent == "audio monitor" {
		t.Fatalf("empty target coerced to audio monitor; got %+v", got)
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
