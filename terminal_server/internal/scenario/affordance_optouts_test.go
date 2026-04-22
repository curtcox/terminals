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

func TestCODEOWNERSHasAffordanceOptOutEntry(t *testing.T) {
	ok, err := CodeownersHasPathOwner("../../../.github/CODEOWNERS", "terminal_server/internal/scenario/affordance_optouts.yaml", "@curtcox")
	if err != nil {
		t.Fatalf("CodeownersHasPathOwner() error = %v", err)
	}
	if !ok {
		t.Fatalf("expected CODEOWNERS entry for affordance opt-out allowlist")
	}
}
