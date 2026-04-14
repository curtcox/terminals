package scenario

import "testing"

func TestRegisterBuiltinsIncludesMasterplanScenarios(t *testing.T) {
	engine := NewEngine()
	RegisterBuiltins(engine)

	registry := engine.RegistrySnapshot()
	got := map[string]bool{}
	for _, item := range registry {
		got[item.Name] = true
	}

	want := []string{
		"intercom",
		"internal_video_call",
		"phone_call",
		"voice_assistant",
		"audio_monitor",
		"schedule_monitor",
		"photo_frame",
		"multi_window",
		"timer_reminder",
		"terminal",
		"pa_system",
		"announcement",
		"red_alert",
	}
	for _, name := range want {
		if !got[name] {
			t.Fatalf("missing builtin scenario registration: %s", name)
		}
	}
}
