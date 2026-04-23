package scenario

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateAffordanceOptOutAllowlistRejectsExpiredEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "affordance_optouts.yaml")
	content := strings.Join([]string{
		"- scenario_id: kiosk_demo",
		"  reason: locked demo flow",
		"  approver: @reviewer",
		"  expires_at: 2025-01-01",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	allowlist, err := LoadAffordanceOptOutAllowlist(path)
	if err != nil {
		t.Fatalf("LoadAffordanceOptOutAllowlist() error = %v", err)
	}
	err = ValidateAffordanceOptOutAllowlist(allowlist, map[string]struct{}{"kiosk_demo": {}}, time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatalf("expected expired opt-out error")
	}
}

func TestValidateAffordanceOptOutAllowlistRejectsMissingScenario(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "affordance_optouts.yaml")
	content := strings.Join([]string{
		"- scenario_id: missing_scenario",
		"  reason: locked demo flow",
		"  approver: @reviewer",
		"  expires_at: 2027-01-01",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	allowlist, err := LoadAffordanceOptOutAllowlist(path)
	if err != nil {
		t.Fatalf("LoadAffordanceOptOutAllowlist() error = %v", err)
	}
	err = ValidateAffordanceOptOutAllowlist(allowlist, map[string]struct{}{"photo_frame": {}}, time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatalf("expected missing scenario error")
	}
}

func TestLoadAffordanceOptOutAllowlistRejectsMissingRequiredField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "affordance_optouts.yaml")
	content := strings.Join([]string{
		"- scenario_id: kiosk_demo",
		"  reason: locked demo flow",
		"  expires_at: 2027-01-01",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := LoadAffordanceOptOutAllowlist(path); err == nil {
		t.Fatalf("expected missing required field error")
	}
}

func TestValidateAffordanceOptOutAllowlistRejectsInvalidReplacementAffordance(t *testing.T) {
	entries := []AffordanceOptOutEntry{
		{
			ScenarioID:            "kiosk_demo",
			Reason:                "locked demo flow",
			Approver:              "@reviewer",
			ExpiresAt:             "2027-01-01",
			ReplacementAffordance: "menu_button",
		},
	}

	err := ValidateAffordanceOptOutAllowlist(
		entries,
		map[string]struct{}{"kiosk_demo": {}},
		time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatalf("expected invalid replacement_affordance error")
	}
}

func TestCODEOWNERSHasAffordanceOptOutEntry(t *testing.T) {
	ok, err := CodeownersHasPathOwner("../../../.github/CODEOWNERS", "terminal_server/internal/scenario/affordance_optouts.yaml", "@scenario-engine-maintainers")
	if err != nil {
		t.Fatalf("CodeownersHasPathOwner() error = %v", err)
	}
	if !ok {
		t.Fatalf("expected scenario-engine maintainers CODEOWNERS owner for affordance opt-out allowlist")
	}
}

func TestValidateMainLayerAffordanceCoverageRejectsUnallowlistedOptOut(t *testing.T) {
	registry := []RegistrationInfo{
		{Name: "photo_frame", Priority: PriorityLow},
	}
	skipsAffordance := map[string]struct{}{
		"photo_frame": {},
	}

	err := ValidateMainLayerAffordanceCoverage(
		registry,
		skipsAffordance,
		nil,
		time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatalf("expected unallowlisted opt-out error")
	}
}

func TestValidateMainLayerAffordanceCoverageAllowsAllowlistedOptOut(t *testing.T) {
	registry := []RegistrationInfo{
		{Name: "photo_frame", Priority: PriorityLow},
	}
	skipsAffordance := map[string]struct{}{
		"photo_frame": {},
	}
	allowlist := []AffordanceOptOutEntry{
		{
			ScenarioID: "photo_frame",
			Reason:     "temporary locked kiosk flow",
			Approver:   "@scenario-engine-maintainers",
			ExpiresAt:  "2027-01-01",
		},
	}

	err := ValidateMainLayerAffordanceCoverage(
		registry,
		skipsAffordance,
		allowlist,
		time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("ValidateMainLayerAffordanceCoverage() error = %v", err)
	}
}
