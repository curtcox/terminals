package scenario

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
		"recent_imu_anomaly",
		"sound_identification",
		"sound_localization",
		"presence_query",
		"bluetooth_inventory",
		"terminal_verification",
		"photo_frame",
		"multi_window",
		"timer_reminder",
		"terminal",
		"bluetooth_passthrough",
		"usb_passthrough",
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

func TestValidateBuiltinAffordanceCoverageRejectsConfiguredOptOutWithoutAllowlistEntry(t *testing.T) {
	dir := t.TempDir()
	allowlistPath := filepath.Join(dir, "affordance_optouts.yaml")
	if err := os.WriteFile(allowlistPath, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	registry := []RegistrationInfo{
		{Name: "photo_frame", Priority: PriorityLow},
	}
	configuredOptOuts := map[string]struct{}{
		"photo_frame": {},
	}

	err := validateBuiltinAffordanceCoverage(
		registry,
		configuredOptOuts,
		allowlistPath,
		time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatalf("expected missing allowlist entry error")
	}
	if !strings.Contains(err.Error(), "skip withCornerAffordance without allowlist entry") {
		t.Fatalf("expected allowlist coverage error, got %v", err)
	}
}

func TestRegisterBuiltinsPanicsWhenConfiguredOptOutMissingFromAllowlist(t *testing.T) {
	dir := t.TempDir()
	allowlistPath := filepath.Join(dir, "affordance_optouts.yaml")
	if err := os.WriteFile(allowlistPath, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	originalPathFn := builtinAffordanceOptOutAllowlistPath
	originalNowFn := builtinAffordanceCoverageNow
	originalOptOuts := builtinMainLayerAffordanceOptOuts
	builtinAffordanceOptOutAllowlistPath = func() string { return allowlistPath }
	builtinAffordanceCoverageNow = func() time.Time {
		return time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC)
	}
	builtinMainLayerAffordanceOptOuts = map[string]struct{}{
		"photo_frame": {},
	}
	defer func() {
		builtinAffordanceOptOutAllowlistPath = originalPathFn
		builtinAffordanceCoverageNow = originalNowFn
		builtinMainLayerAffordanceOptOuts = originalOptOuts
	}()

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected RegisterBuiltins to panic when opt-out is missing from allowlist")
		}
	}()

	RegisterBuiltins(NewEngine())
}
