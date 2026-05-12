package appruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRuntimeReconcileMigrationPendingRecords(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_reconcile")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"migrate_reconcile\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	runtime.mu.Lock()
	state := runtime.migrations["migrate_reconcile"]
	state.Verdict = "reconcile_pending"
	state.LastError = ErrMigrationReconcilePending.Error()
	state.ReconciliationPath = "apps/migrate_reconcile/migrate/r1/reconcile.json"
	state.PendingRecords = map[string]string{"rec-1": "force_rewind"}
	runtime.migrations["migrate_reconcile"] = state
	runtime.mu.Unlock()

	if status, err := runtime.AbortMigration("migrate_reconcile", MigrationAbortToBaseline); !errors.Is(err, ErrMigrationReconcilePending) {
		t.Fatalf("AbortMigration() error = %v, want ErrMigrationReconcilePending", err)
	} else {
		if status.Verdict != "reconcile_pending" {
			t.Fatalf("AbortMigration() verdict = %q, want reconcile_pending", status.Verdict)
		}
		if len(status.PendingRecords) != 1 || status.PendingRecords[0].RecordID != "rec-1" {
			t.Fatalf("AbortMigration() pending_records = %+v, want rec-1 preserved", status.PendingRecords)
		}
	}

	if _, err := runtime.RetryMigration("migrate_reconcile"); !errors.Is(err, ErrMigrationReconcilePending) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationReconcilePending", err)
	}

	runtime.mu.Lock()
	state = runtime.migrations["migrate_reconcile"]
	state.PendingRecords = nil
	state.Verdict = "reconcile_pending"
	runtime.migrations["migrate_reconcile"] = state
	runtime.mu.Unlock()

	status, err := runtime.RetryMigration("migrate_reconcile")
	if !errors.Is(err, ErrMigrationReconcilePending) {
		t.Fatalf("RetryMigration() with reconcile_pending verdict only error = %v, want ErrMigrationReconcilePending", err)
	}
	if status.Verdict != "reconcile_pending" {
		t.Fatalf("RetryMigration() with reconcile_pending verdict only verdict = %q, want reconcile_pending", status.Verdict)
	}
	if status.LastError != ErrMigrationReconcilePending.Error() {
		t.Fatalf("RetryMigration() with reconcile_pending verdict only last_error = %q, want %q", status.LastError, ErrMigrationReconcilePending.Error())
	}

	if _, err := runtime.ReconcileMigration("migrate_reconcile", "rec-1", "bad_resolution"); !errors.Is(err, ErrMigrationResolutionInvalid) {
		t.Fatalf("ReconcileMigration() invalid resolution error = %v, want ErrMigrationResolutionInvalid", err)
	}

	runtime.mu.Lock()
	state = runtime.migrations["migrate_reconcile"]
	state.PendingRecords = map[string]string{"rec-1": "force_rewind"}
	state.Verdict = "reconcile_pending"
	runtime.migrations["migrate_reconcile"] = state
	pkg := runtime.packages["migrate_reconcile"]
	appendMigrationJournalEntry(pkg, state, "reconcile_pending", map[string]any{
		"pending_records": state.PendingRecords,
	})
	runtime.mu.Unlock()

	status, err = runtime.ReconcileMigration("migrate_reconcile", "rec-1", "force_rewind")
	if err != nil {
		t.Fatalf("ReconcileMigration() error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("ReconcileMigration() verdict = %q, want ok", status.Verdict)
	}
	if status.ReconciliationPath != "" {
		t.Fatalf("ReconcileMigration() reconciliation_path = %q, want empty", status.ReconciliationPath)
	}
	if len(status.PendingRecords) != 0 {
		t.Fatalf("ReconcileMigration() pending_records = %d, want 0", len(status.PendingRecords))
	}
	if strings.TrimSpace(status.JournalPath) == "" {
		t.Fatalf("ReconcileMigration() journal_path empty")
	}
	reconcileJournalPath := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
	reconcileJournalBytes, err := os.ReadFile(reconcileJournalPath)
	if err != nil {
		t.Fatalf("ReadFile(reconcile journal) error = %v", err)
	}
	reconcileJournalText := string(reconcileJournalBytes)
	if !strings.Contains(reconcileJournalText, `"event":"reconcile_record"`) || !strings.Contains(reconcileJournalText, `"record_id":"rec-1"`) {
		t.Fatalf("migration journal missing reconcile entry: %q", reconcileJournalText)
	}

	restarted := NewRuntime()
	if _, err := restarted.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() after reconcile journal replay error = %v", err)
	}
	status, err = restarted.GetMigrationStatus("migrate_reconcile")
	if err != nil {
		t.Fatalf("GetMigrationStatus() after reconcile journal replay error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("GetMigrationStatus() after reconcile journal replay verdict = %q, want ok", status.Verdict)
	}
	if status.LastError != "" {
		t.Fatalf("GetMigrationStatus() after reconcile journal replay last_error = %q, want empty", status.LastError)
	}
	if status.ReconciliationPath != "" {
		t.Fatalf("GetMigrationStatus() after reconcile journal replay reconciliation_path = %q, want empty", status.ReconciliationPath)
	}
	if len(status.PendingRecords) != 0 {
		t.Fatalf("GetMigrationStatus() after reconcile journal replay pending_records = %d, want 0", len(status.PendingRecords))
	}

	runtime.mu.Lock()
	state = runtime.migrations["migrate_reconcile"]
	state.PendingRecords = map[string]string{"rec-9": "manual", "rec-2": "force_rewind"}
	runtime.migrations["migrate_reconcile"] = state
	runtime.mu.Unlock()

	status, err = runtime.GetMigrationStatus("migrate_reconcile")
	if err != nil {
		t.Fatalf("GetMigrationStatus() error = %v", err)
	}
	if len(status.PendingRecords) != 2 {
		t.Fatalf("GetMigrationStatus() pending_records = %d, want 2", len(status.PendingRecords))
	}
	if status.PendingRecords[0].RecordID != "rec-2" || status.PendingRecords[0].RecommendedResolution != "force_rewind" {
		t.Fatalf("GetMigrationStatus() pending_records[0] = %+v, want rec-2/force_rewind", status.PendingRecords[0])
	}
	if status.PendingRecords[1].RecordID != "rec-9" || status.PendingRecords[1].RecommendedResolution != "manual" {
		t.Fatalf("GetMigrationStatus() pending_records[1] = %+v, want rec-9/manual", status.PendingRecords[1])
	}
}

func TestRuntimeMigrationInvalidStepPlanDisablesExecutor(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_invalid_plan")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"migrate_invalid_plan\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "not_a_step.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.GetMigrationStatus("migrate_invalid_plan")
	if err != nil {
		t.Fatalf("GetMigrationStatus() error = %v", err)
	}
	if status.ExecutorReady {
		t.Fatalf("ExecutorReady = true, want false")
	}
	if !strings.Contains(status.LastError, "must match <step>_<from>_to_<to>.tal") {
		t.Fatalf("LastError = %q, want migration step format message", status.LastError)
	}

	status, err = runtime.RetryMigration("migrate_invalid_plan")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.ExecutorReady {
		t.Fatalf("RetryMigration() ExecutorReady = true, want false")
	}
	if status.StepsCompleted != 0 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 0", status.StepsCompleted)
	}
	if status.Verdict != "idle" {
		t.Fatalf("RetryMigration() verdict = %q, want idle", status.Verdict)
	}
}

func TestRuntimeMigrationInvalidLimitsDisableExecutor(t *testing.T) {
	testCases := []struct {
		name          string
		manifestField string
		manifestValue string
		wantMessage   string
	}{
		{
			name:          "max runtime seconds",
			manifestField: "max_runtime_seconds",
			manifestValue: "0",
			wantMessage:   "migrate.max_runtime_seconds must be a positive integer",
		},
		{
			name:          "checkpoint every",
			manifestField: "checkpoint_every",
			manifestValue: "-1",
			wantMessage:   "migrate.checkpoint_every must be a positive integer",
		},
		{
			name:          "drain timeout seconds",
			manifestField: "drain_timeout_seconds",
			manifestValue: "0",
			wantMessage:   "migrate.drain_timeout_seconds must be a positive integer",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			appDir := filepath.Join(tempDir, "migrate_invalid_limits")
			if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}
			manifest := fmt.Sprintf("name = \"migrate_invalid_limits\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n\n[migrate]\ndeclared_steps = 1\n%s = %s\n", tc.manifestField, tc.manifestValue)
			if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
				t.Fatalf("WriteFile(manifest) error = %v", err)
			}
			if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
				t.Fatalf("WriteFile(main) error = %v", err)
			}
			if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
				t.Fatalf("WriteFile(migrate) error = %v", err)
			}

			runtime := NewRuntime()
			if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
				t.Fatalf("LoadPackage() error = %v", err)
			}

			status, err := runtime.GetMigrationStatus("migrate_invalid_limits")
			if err != nil {
				t.Fatalf("GetMigrationStatus() error = %v", err)
			}
			if status.ExecutorReady {
				t.Fatalf("ExecutorReady = true, want false")
			}
			if !strings.Contains(status.LastError, tc.wantMessage) {
				t.Fatalf("LastError = %q, want contains %q", status.LastError, tc.wantMessage)
			}

			status, err = runtime.RetryMigration("migrate_invalid_limits")
			if err != nil {
				t.Fatalf("RetryMigration() error = %v", err)
			}
			if status.ExecutorReady {
				t.Fatalf("RetryMigration() ExecutorReady = true, want false")
			}
			if status.StepsCompleted != 0 {
				t.Fatalf("RetryMigration() steps_completed = %d, want 0", status.StepsCompleted)
			}
		})
	}
}

func TestRuntimeMigrationInvalidManifestPolicyDisablesExecutor(t *testing.T) {
	testCases := []struct {
		name          string
		compatibility string
		drainPolicy   string
		wantMessage   string
	}{
		{
			name:          "unknown compatibility",
			compatibility: "sometimes",
			drainPolicy:   "none",
			wantMessage:   `migrate.step 0001 has invalid compatibility "sometimes"`,
		},
		{
			name:          "unknown drain policy",
			compatibility: "compatible",
			drainPolicy:   "eventually",
			wantMessage:   `migrate.step 0001 has invalid drain_policy "eventually"`,
		},
		{
			name:          "incompatible without drain",
			compatibility: "incompatible",
			drainPolicy:   "none",
			wantMessage:   "migrate.step 0001 declares compatibility=incompatible with drain_policy=none",
		},
		{
			name:          "missing compatibility",
			compatibility: "",
			drainPolicy:   "none",
			wantMessage:   "migrate.step 0001 must declare compatibility",
		},
		{
			name:          "missing drain policy",
			compatibility: "compatible",
			drainPolicy:   "",
			wantMessage:   "migrate.step 0001 must declare drain_policy",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			appDir := filepath.Join(tempDir, "migrate_invalid_manifest_policy")
			if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}
			manifest := fmt.Sprintf(`name = "migrate_invalid_manifest_policy"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = %q
drain_policy = %q
`, tc.compatibility, tc.drainPolicy)
			if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
				t.Fatalf("WriteFile(manifest) error = %v", err)
			}
			if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
				t.Fatalf("WriteFile(main) error = %v", err)
			}
			if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
				t.Fatalf("WriteFile(migrate) error = %v", err)
			}

			runtime := NewRuntime()
			if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
				t.Fatalf("LoadPackage() error = %v", err)
			}

			status, err := runtime.GetMigrationStatus("migrate_invalid_manifest_policy")
			if err != nil {
				t.Fatalf("GetMigrationStatus() error = %v", err)
			}
			if status.ExecutorReady {
				t.Fatalf("ExecutorReady = true, want false")
			}
			if !strings.Contains(status.LastError, tc.wantMessage) {
				t.Fatalf("LastError = %q, want contains %q", status.LastError, tc.wantMessage)
			}

			status, err = runtime.RetryMigration("migrate_invalid_manifest_policy")
			if err != nil {
				t.Fatalf("RetryMigration() error = %v", err)
			}
			if status.ExecutorReady {
				t.Fatalf("RetryMigration() ExecutorReady = true, want false")
			}
			if status.StepsCompleted != 0 {
				t.Fatalf("RetryMigration() steps_completed = %d, want 0", status.StepsCompleted)
			}
		})
	}
}

func TestRuntimeRetryMigrationFailsWhenMaxRuntimeExceeded(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_runtime_timeout")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"migrate_runtime_timeout\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n\n[migrate]\ndeclared_steps = 1\nmax_runtime_seconds = 1\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}
	runtime.migrationHook = func(event string, step int) error {
		if event == "step_started" && step == 1 {
			time.Sleep(1100 * time.Millisecond)
		}
		return nil
	}

	status, err := runtime.RetryMigration("migrate_runtime_timeout")
	if !errors.Is(err, ErrMigrationRuntimeTimeout) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationRuntimeTimeout", err)
	}
	if status.Verdict != "step_failed" {
		t.Fatalf("RetryMigration() verdict = %q, want step_failed", status.Verdict)
	}
	if status.StepsCompleted != 0 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 0", status.StepsCompleted)
	}
	if status.LastStep != 1 {
		t.Fatalf("RetryMigration() last_step = %d, want 1", status.LastStep)
	}
	if status.LastError != ErrMigrationRuntimeTimeout.Error() {
		t.Fatalf("RetryMigration() last_error = %q, want %q", status.LastError, ErrMigrationRuntimeTimeout.Error())
	}

	journalBytes, err := os.ReadFile(filepath.Join(appDir, filepath.FromSlash(status.JournalPath)))
	if err != nil {
		t.Fatalf("ReadFile(journal) error = %v", err)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalErrorContaining(entries, "step_failed_timeout", ErrMigrationRuntimeTimeout.Error()) {
		t.Fatalf("migration journal missing step_failed_timeout entry: %+v", entries)
	}
}

func TestRuntimeRetryMigrationAppliesMaxRuntimePerStep(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_runtime_per_step")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := `name = "migrate_runtime_per_step"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 2
max_runtime_seconds = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.step]]
from = "2"
to = "3"
compatibility = "compatible"
drain_policy = "none"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(step 1) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0002_2_to_3.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(step 2) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}
	runtime.migrationHook = func(event string, _ int) error {
		if event == "step_started" {
			time.Sleep(600 * time.Millisecond)
		}
		return nil
	}

	status, err := runtime.RetryMigration("migrate_runtime_per_step")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() verdict = %q, want ok", status.Verdict)
	}
	if status.StepsCompleted != 2 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 2", status.StepsCompleted)
	}
}
