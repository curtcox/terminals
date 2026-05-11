package appruntime

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
)

func TestRuntimeMigrationStatusAndActions(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_stub")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"migrate_stub\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.GetMigrationStatus("migrate_stub")
	if err != nil {
		t.Fatalf("GetMigrationStatus() error = %v", err)
	}
	if status.App != "migrate_stub" || status.Version != "1.0.0" {
		t.Fatalf("GetMigrationStatus() = %+v, want app/version populated", status)
	}
	if status.ExecutorReady {
		t.Fatalf("ExecutorReady = true, want false")
	}

	status, err = runtime.RetryMigration("migrate_stub")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "idle" {
		t.Fatalf("RetryMigration() verdict = %q, want idle", status.Verdict)
	}

	status, err = runtime.AbortMigration("migrate_stub", "")
	if err != nil {
		t.Fatalf("AbortMigration() error = %v", err)
	}
	if status.Verdict != "idle" {
		t.Fatalf("AbortMigration() verdict = %q, want idle", status.Verdict)
	}

	if _, err := runtime.ReconcileMigration("migrate_stub", "rec-1", "accept_current"); !errors.Is(err, ErrMigrationReconcilePending) {
		t.Fatalf("ReconcileMigration() error = %v, want ErrMigrationReconcilePending", err)
	}
}

func TestRuntimeMigrationLifecycleWithSteps(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_live")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"migrate_live\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate 1) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0002_2_to_3.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate 2) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.GetMigrationStatus("migrate_live")
	if err != nil {
		t.Fatalf("GetMigrationStatus() error = %v", err)
	}
	if !status.ExecutorReady {
		t.Fatalf("ExecutorReady = false, want true")
	}
	if status.StepsPlanned != 2 || status.StepsCompleted != 0 {
		t.Fatalf("status steps = %d/%d, want 0/2", status.StepsCompleted, status.StepsPlanned)
	}
	if status.LastStep != 0 {
		t.Fatalf("status last_step = %d, want 0", status.LastStep)
	}

	status, err = runtime.RetryMigration("migrate_live")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() verdict = %q, want ok", status.Verdict)
	}
	if status.StepsCompleted != 2 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 2", status.StepsCompleted)
	}
	if status.LastStep != 2 {
		t.Fatalf("RetryMigration() last_step = %d, want 2", status.LastStep)
	}
	if status.JournalPath == "" {
		t.Fatalf("RetryMigration() journal_path empty")
	}
	journalFile := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
	journalBytes, err := os.ReadFile(journalFile)
	if err != nil {
		t.Fatalf("ReadFile(journal) error = %v", err)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalEvent(entries, "retry_started") || !hasMigrationJournalEvent(entries, "retry_committed") {
		t.Fatalf("migration journal missing retry events: %+v", entries)
	}
	if !hasMigrationStepMetadata(entries, "step_started", 2, "2", "3", "0002_2_to_3.tal") {
		t.Fatalf("migration journal missing step metadata for step 2 start: %+v", entries)
	}
	if !hasMigrationStepMetadata(entries, "step_committed", 2, "2", "3", "0002_2_to_3.tal") {
		t.Fatalf("migration journal missing step metadata for step 2 commit: %+v", entries)
	}
	if !hasMigrationJournalEventForStep(entries, "step_started", 1) || !hasMigrationJournalEventForStep(entries, "step_committed", 1) {
		t.Fatalf("migration journal missing step 1 lifecycle events: %+v", entries)
	}
	if !hasMigrationJournalEventForStep(entries, "step_started", 2) || !hasMigrationJournalEventForStep(entries, "step_committed", 2) {
		t.Fatalf("migration journal missing step 2 lifecycle events: %+v", entries)
	}

	status, err = runtime.AbortMigration("migrate_live", MigrationAbortToCheckpoint)
	if err != nil {
		t.Fatalf("AbortMigration() error = %v", err)
	}
	if status.Verdict != "step_failed" {
		t.Fatalf("AbortMigration() verdict = %q, want step_failed", status.Verdict)
	}
	if status.StepsCompleted != 1 {
		t.Fatalf("AbortMigration() steps_completed = %d, want 1", status.StepsCompleted)
	}
	if status.LastStep != 2 {
		t.Fatalf("AbortMigration() last_step = %d, want 2", status.LastStep)
	}
	if status.LastError != "step 2 aborted by operator" {
		t.Fatalf("AbortMigration() last_error = %q, want %q", status.LastError, "step 2 aborted by operator")
	}

	runtime.mu.Lock()
	state := runtime.migrations["migrate_live"]
	state.Verdict = "running"
	state.StepsCompleted = 1
	state.LastStep = 2
	runtime.migrations["migrate_live"] = state
	runtime.mu.Unlock()

	status, err = runtime.AbortMigration("migrate_live", MigrationAbortToCheckpoint)
	if err != nil {
		t.Fatalf("AbortMigration(in-flight step) error = %v", err)
	}
	if status.Verdict != "step_failed" {
		t.Fatalf("AbortMigration(in-flight step) verdict = %q, want step_failed", status.Verdict)
	}
	if status.StepsCompleted != 1 {
		t.Fatalf("AbortMigration(in-flight step) steps_completed = %d, want 1", status.StepsCompleted)
	}
	if status.LastStep != 2 {
		t.Fatalf("AbortMigration(in-flight step) last_step = %d, want 2", status.LastStep)
	}
	if status.LastError != "step 2 aborted by operator" {
		t.Fatalf("AbortMigration(in-flight step) last_error = %q, want %q", status.LastError, "step 2 aborted by operator")
	}

	status, err = runtime.RetryMigration("migrate_live")
	if err != nil {
		t.Fatalf("RetryMigration() second run error = %v", err)
	}
	if status.StepsCompleted != 2 {
		t.Fatalf("RetryMigration() second run steps_completed = %d, want 2", status.StepsCompleted)
	}

	status, err = runtime.AbortMigration("migrate_live", MigrationAbortToBaseline)
	if err != nil {
		t.Fatalf("AbortMigration(to baseline) error = %v", err)
	}
	if status.StepsCompleted != 0 {
		t.Fatalf("AbortMigration(to baseline) steps_completed = %d, want 0", status.StepsCompleted)
	}
	if status.LastStep != 0 {
		t.Fatalf("AbortMigration(to baseline) last_step = %d, want 0", status.LastStep)
	}
	if status.LastError != "aborted to baseline by operator" {
		t.Fatalf("AbortMigration(to baseline) last_error = %q, want %q", status.LastError, "aborted to baseline by operator")
	}
	journalBytes, err = os.ReadFile(journalFile)
	if err != nil {
		t.Fatalf("ReadFile(journal after abort) error = %v", err)
	}
	journalText := string(journalBytes)
	if !strings.Contains(journalText, `"event":"aborted"`) || !strings.Contains(journalText, `"target":"baseline"`) {
		t.Fatalf("migration journal missing abort baseline entry: %q", journalText)
	}
	entries = parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalEventSequence(entries, []string{"retry_started", "step_started", "step_committed", "retry_committed"}) {
		t.Fatalf("missing second retry event sequence in journal: %+v", entries)
	}
	if !hasMigrationJournalEventForStep(entries, "step_started", 2) || !hasMigrationJournalEventForStep(entries, "step_committed", 2) {
		t.Fatalf("second retry did not resume step 2 lifecycle events: %+v", entries)
	}

	if _, err := runtime.AbortMigration("migrate_live", "invalid_target"); !errors.Is(err, ErrMigrationAbortTargetInvalid) {
		t.Fatalf("AbortMigration(invalid target) error = %v, want ErrMigrationAbortTargetInvalid", err)
	}
}

func TestRuntimeAbortBaselineEntersReconcilePendingWhenArtifactInverseFails(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_inverse_fail")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"migrate_inverse_fail\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
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
	if _, err := runtime.RetryMigration("migrate_inverse_fail"); err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}

	pkg, ok := runtime.GetPackage("migrate_inverse_fail")
	if !ok {
		t.Fatalf("GetPackage() ok = false, want true")
	}
	state := runtime.migrations["migrate_inverse_fail"]
	appendMigrationJournalEntry(pkg, state, "artifact_inverse_failed", map[string]any{
		"record_id":              "artifact:history-photo",
		"recommended_resolution": "manual",
		"error":                  "artifact current revision is not a descendant of journaled patch",
	})

	status, err := runtime.AbortMigration("migrate_inverse_fail", MigrationAbortToBaseline)
	if !errors.Is(err, ErrMigrationReconcilePending) {
		t.Fatalf("AbortMigration(to baseline) error = %v, want ErrMigrationReconcilePending", err)
	}
	if status.Verdict != "reconcile_pending" {
		t.Fatalf("AbortMigration(to baseline) verdict = %q, want reconcile_pending", status.Verdict)
	}
	if status.StepsCompleted != 0 || status.LastStep != 0 {
		t.Fatalf("AbortMigration(to baseline) step state = %d/%d, want 0/0", status.StepsCompleted, status.LastStep)
	}
	if status.ReconciliationPath == "" {
		t.Fatalf("AbortMigration(to baseline) reconciliation_path empty")
	}
	if len(status.PendingRecords) != 1 || status.PendingRecords[0].RecordID != "artifact:history-photo" || status.PendingRecords[0].RecommendedResolution != "manual" {
		t.Fatalf("AbortMigration(to baseline) pending_records = %+v, want artifact:history-photo/manual", status.PendingRecords)
	}

	restarted := NewRuntime()
	if _, err := restarted.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() after restart error = %v", err)
	}
	status, err = restarted.GetMigrationStatus("migrate_inverse_fail")
	if err != nil {
		t.Fatalf("GetMigrationStatus() after restart error = %v", err)
	}
	if status.Verdict != "reconcile_pending" || len(status.PendingRecords) != 1 {
		t.Fatalf("GetMigrationStatus() after restart = verdict %q pending %+v, want reconcile_pending with one record", status.Verdict, status.PendingRecords)
	}
}

func TestRuntimeReloadMigrationStateStartsFromInstalledVersion(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_reload")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifestV2 := "name = \"migrate_reload\"\nversion = \"2\"\nlanguage = \"tal/1\"\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifestV2), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v2) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate 1) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0002_2_to_3.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate 2) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	manifestV3 := "name = \"migrate_reload\"\nversion = \"3\"\nlanguage = \"tal/1\"\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifestV3), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v3) error = %v", err)
	}

	if _, changed, err := runtime.ReloadPackage(context.Background(), "migrate_reload"); err != nil || !changed {
		t.Fatalf("ReloadPackage() error=%v changed=%v, want changed reload", err, changed)
	}

	status, err := runtime.GetMigrationStatus("migrate_reload")
	if err != nil {
		t.Fatalf("GetMigrationStatus() error = %v", err)
	}
	if status.StepsPlanned != 2 || status.StepsCompleted != 1 {
		t.Fatalf("status steps = %d/%d, want 1/2", status.StepsCompleted, status.StepsPlanned)
	}

	status, err = runtime.RetryMigration("migrate_reload")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() verdict = %q, want ok", status.Verdict)
	}
	if status.StepsCompleted != 2 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 2", status.StepsCompleted)
	}

	journalFile := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
	journalBytes, err := os.ReadFile(journalFile)
	if err != nil {
		t.Fatalf("ReadFile(journal) error = %v", err)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalEventForStep(entries, "step_started", 2) || !hasMigrationJournalEventForStep(entries, "step_committed", 2) {
		t.Fatalf("migration journal missing step 2 lifecycle events: %+v", entries)
	}
	if hasMigrationJournalEventForStep(entries, "step_started", 1) || hasMigrationJournalEventForStep(entries, "step_committed", 1) {
		t.Fatalf("migration journal unexpectedly ran step 1 after reload baseline: %+v", entries)
	}
}

func TestRuntimeReloadMigrationAfterKeyRotationUsesAppIDAndPendingVersionWindow(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_key_rotation")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	appID := "app:sha256:1111111111111111111111111111111111111111111111111111111111111111"
	manifestV2 := fmt.Sprintf(`name = "migrate_key_rotation"
app_id = %q
version = "2"
language = "tal/1"
author_key_id = "author-key-v1"
`, appID)
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifestV2), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v2) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate 1) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0002_2_to_3.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate 2) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	manifestV3RotatedKey := fmt.Sprintf(`name = "migrate_key_rotation"
app_id = %q
version = "3"
language = "tal/1"
author_key_id = "author-key-v2"
`, appID)
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifestV3RotatedKey), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v3) error = %v", err)
	}

	if _, changed, err := runtime.ReloadPackage(context.Background(), "migrate_key_rotation"); err != nil || !changed {
		t.Fatalf("ReloadPackage() error=%v changed=%v, want changed reload", err, changed)
	}

	status, err := runtime.GetMigrationStatus("migrate_key_rotation")
	if err != nil {
		t.Fatalf("GetMigrationStatus() error = %v", err)
	}
	if status.StepsPlanned != 2 || status.StepsCompleted != 1 {
		t.Fatalf("status steps = %d/%d, want 1/2 after rotation reload", status.StepsCompleted, status.StepsPlanned)
	}
	if !strings.Contains(status.JournalPath, "apps/"+appID+"/migrate/") {
		t.Fatalf("GetMigrationStatus() journal_path = %q, want app_id-scoped migration path", status.JournalPath)
	}

	status, err = runtime.RetryMigration("migrate_key_rotation")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "ok" || status.StepsCompleted != 2 {
		t.Fatalf("RetryMigration() = verdict %q steps %d, want ok steps 2", status.Verdict, status.StepsCompleted)
	}

	journalFile := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
	journalBytes, err := os.ReadFile(journalFile)
	if err != nil {
		t.Fatalf("ReadFile(journal) error = %v", err)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalEventForStep(entries, "step_started", 2) || !hasMigrationJournalEventForStep(entries, "step_committed", 2) {
		t.Fatalf("migration journal missing step 2 lifecycle events after rotation: %+v", entries)
	}
	if hasMigrationJournalEventForStep(entries, "step_started", 1) || hasMigrationJournalEventForStep(entries, "step_committed", 1) {
		t.Fatalf("migration journal unexpectedly replayed installed step 1 after rotation: %+v", entries)
	}
}

func TestRuntimeRetryMigrationWithFixtureExpectedMatch(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_fixture_match")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_fixture_match"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	seed := "{\"key\":\"history/a\",\"value\":{\"a\":2,\"z\":1}}\n"
	expected := "{\"key\":\"history/a\",\"value\":{\"a\":2,\"z\":1}}\n"
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_seed.ndjson"), []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(seed fixture) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_fixture_match")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() verdict = %q, want ok", status.Verdict)
	}
	if status.StepsCompleted != 1 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 1", status.StepsCompleted)
	}
}

func TestRuntimeRetryMigrationAppliesFixtureTransforms(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_fixture_transform")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_fixture_transform"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	migrateScript := `def migrate(record):
  record["label_normalized"] = lower(record["label"])
  record["schema_version"] = 2
  del record["legacy"]
  return record
`
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(migrateScript), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	seed := "{\"key\":\"history/a\",\"value\":{\"label\":\"Dishwasher Done\",\"legacy\":true}}\n"
	expected := "{\"key\":\"history/a\",\"value\":{\"label\":\"Dishwasher Done\",\"label_normalized\":\"dishwasher done\",\"schema_version\":2}}\n"
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_seed.ndjson"), []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(seed fixture) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_fixture_transform")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() verdict = %q, want ok", status.Verdict)
	}
	if status.StepsCompleted != 1 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 1", status.StepsCompleted)
	}
}

func TestRuntimeRetryMigrationAppliesTrimFixtureTransforms(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_fixture_trim")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_fixture_trim"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	migrateScript := `def migrate(record):
  record["label_trimmed"] = trim(record["label"])
  record["label_normalized"] = lower(trim(record["label"]))
  return record
`
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(migrateScript), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	seed := "{\"key\":\"history/a\",\"value\":{\"label\":\"  Dishwasher Done  \"}}\n"
	expected := "{\"key\":\"history/a\",\"value\":{\"label\":\"  Dishwasher Done  \",\"label_normalized\":\"dishwasher done\",\"label_trimmed\":\"Dishwasher Done\"}}\n"
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_seed.ndjson"), []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(seed fixture) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_fixture_trim")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() verdict = %q, want ok", status.Verdict)
	}
	if status.StepsCompleted != 1 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 1", status.StepsCompleted)
	}
}

func TestRuntimeRetryMigrationAppliesRecordGetFixtureTransforms(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_fixture_record_get")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_fixture_record_get"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	migrateScript := `def migrate(record):
  record["label_normalized"] = lower(trim(record.get("label", "")))
  record["fallback_label"] = record.get("missing_label", "untitled")
  record["source"] = 'fixture#migration'
  record["retry_count"] = record.get("retry_count", 0)
  record["archived"] = record.get("archived", false)
  record["last_seen"] = record.get("last_seen", null)
  record["metadata"] = record.get("metadata", {"source":"migration","tags":["default"]})
  return record
`
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(migrateScript), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	seed := "{\"key\":\"history/a\",\"value\":{\"label\":\"  Dishwasher Done  \"}}\n"
	expected := "{\"key\":\"history/a\",\"value\":{\"archived\":false,\"fallback_label\":\"untitled\",\"label\":\"  Dishwasher Done  \",\"label_normalized\":\"dishwasher done\",\"last_seen\":null,\"metadata\":{\"source\":\"migration\",\"tags\":[\"default\"]},\"retry_count\":0,\"source\":\"fixture#migration\"}}\n"
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_seed.ndjson"), []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(seed fixture) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_fixture_record_get")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() verdict = %q, want ok", status.Verdict)
	}
	if status.StepsCompleted != 1 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 1", status.StepsCompleted)
	}
}

func TestRuntimeRetryMigrationAppliesIdempotentFixtureGuard(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_fixture_idempotent_guard")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_fixture_idempotent_guard"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1
checkpoint_every = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	migrateScript := `def migrate(record):
  if "label_normalized" in record:
    continue
  record["label_normalized"] = lower(trim(record.get("label", "")))
  return record
`
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(migrateScript), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	seed := "{\"key\":\"history/a\",\"value\":{\"label\":\"Already Done\",\"label_normalized\":\"already\"}}\n{\"key\":\"history/b\",\"value\":{\"label\":\"  Dishwasher Done  \"}}\n"
	expected := "{\"key\":\"history/a\",\"value\":{\"label\":\"Already Done\",\"label_normalized\":\"already\"}}\n{\"key\":\"history/b\",\"value\":{\"label\":\"  Dishwasher Done  \",\"label_normalized\":\"dishwasher done\"}}\n"
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_seed.ndjson"), []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(seed fixture) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_fixture_idempotent_guard")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() verdict = %q, want ok", status.Verdict)
	}
	if status.StepsCompleted != 1 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 1", status.StepsCompleted)
	}
	journalBytes, err := os.ReadFile(filepath.Join(appDir, filepath.FromSlash(status.JournalPath)))
	if err != nil {
		t.Fatalf("ReadFile(journal) error = %v", err)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationCheckpointMetadata(entries, 1, 1, 1) {
		t.Fatalf("migration journal missing checkpoint evidence for changed fixture row: %+v", entries)
	}
	if hasMigrationCheckpointMetadata(entries, 1, 2, 1) {
		t.Fatalf("migration journal counted idempotently skipped fixture row as a store effect: %+v", entries)
	}
}

func TestRuntimeRetryMigrationAppliesPagedStoreFixtureTransforms(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_fixture_store_loop")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_fixture_store_loop"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1
checkpoint_every = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	migrateScript := `load("store", list_keys = "list_keys", get = "get", put = "put")
load("migrate.env", checkpoint = "checkpoint")
load("log", info = "info")

def migrate():
    cursor = None
    count = 0
    while True:
        page = list_keys(prefix = "history/", after = cursor, limit = 500)
        if len(page) == 0:
            break
        for key in page:
            rec = get(key)
            if "label_normalized" in rec:
                continue
            rec["label_normalized"] = _normalize(rec.get("label", ""))
            put(key, rec)
            count += 1
        cursor = page[-1]
        checkpoint(cursor = cursor)
    info("history.migrated", records = count)

def _normalize(label):
    return label.strip().lower()
`
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(migrateScript), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	seed := strings.Join([]string{
		"{\"key\":\"history/a\",\"value\":{\"label\":\"Already Done\",\"label_normalized\":\"already\"}}",
		"{\"key\":\"history/b\",\"value\":{\"label\":\"  Dishwasher Done  \"}}",
		"{\"key\":\"settings/theme\",\"value\":{\"label\":\"Dark\"}}",
		"",
	}, "\n")
	expected := strings.Join([]string{
		"{\"key\":\"history/a\",\"value\":{\"label\":\"Already Done\",\"label_normalized\":\"already\"}}",
		"{\"key\":\"history/b\",\"value\":{\"label\":\"  Dishwasher Done  \",\"label_normalized\":\"dishwasher done\"}}",
		"{\"key\":\"settings/theme\",\"value\":{\"label\":\"Dark\"}}",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_seed.ndjson"), []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(seed fixture) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_fixture_store_loop")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() verdict = %q, want ok", status.Verdict)
	}
	journalBytes, err := os.ReadFile(filepath.Join(appDir, filepath.FromSlash(status.JournalPath)))
	if err != nil {
		t.Fatalf("ReadFile(journal) error = %v", err)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationCheckpointMetadata(entries, 1, 1, 1) {
		t.Fatalf("migration journal missing checkpoint_every evidence: %+v", entries)
	}
	if !hasMigrationLogEntry(entries, 1, "info", "history.migrated", `"history.migrated", records = count`) {
		t.Fatalf("migration journal missing fixture log evidence: %+v", entries)
	}
}

func TestRuntimeRetryMigrationAppliesStoreDeleteFixtureEffects(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_fixture_store_delete")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_fixture_store_delete"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1
checkpoint_every = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	migrateScript := `load("store", list_keys = "list_keys", delete = "delete")
load("migrate.env", checkpoint = "checkpoint")

def migrate():
    cursor = None
    while True:
        page = list_keys(prefix = "history/expired/", after = cursor, limit = 500)
        if len(page) == 0: break
        for key in page:
            delete(key)
        cursor = page[-1]
        checkpoint(cursor = cursor)
`
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(migrateScript), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	seed := strings.Join([]string{
		"{\"key\":\"history/active/a\",\"value\":{\"label\":\"Keep\"}}",
		"{\"key\":\"history/expired/a\",\"value\":{\"label\":\"Remove A\"}}",
		"{\"key\":\"history/expired/b\",\"value\":{\"label\":\"Remove B\"}}",
		"{\"key\":\"settings/theme\",\"value\":{\"label\":\"Dark\"}}",
		"",
	}, "\n")
	expected := strings.Join([]string{
		"{\"key\":\"history/active/a\",\"value\":{\"label\":\"Keep\"}}",
		"{\"key\":\"settings/theme\",\"value\":{\"label\":\"Dark\"}}",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_seed.ndjson"), []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(seed fixture) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_fixture_store_delete")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() verdict = %q, want ok", status.Verdict)
	}
	journalBytes, err := os.ReadFile(filepath.Join(appDir, filepath.FromSlash(status.JournalPath)))
	if err != nil {
		t.Fatalf("ReadFile(journal) error = %v", err)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationCheckpointMetadata(entries, 1, 1, 1) || !hasMigrationCheckpointMetadata(entries, 1, 2, 1) {
		t.Fatalf("migration journal missing delete checkpoint evidence: %+v", entries)
	}
}

func TestRuntimeRetryMigrationDoesNotCheckpointUnchangedStorePut(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_fixture_store_loop_unchanged")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_fixture_store_loop_unchanged"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1
checkpoint_every = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	migrateScript := `load("store", list_keys = "list_keys", get = "get", put = "put")
load("migrate.env", checkpoint = "checkpoint")

def migrate():
    cursor = None
    while True:
        page = list_keys(prefix = "history/", after = cursor, limit = 500)
        if len(page) == 0: break
        for key in page:
            rec = get(key)
            rec["label"] = rec.get("label", "")
            put(key, rec)
        cursor = page[-1]
        checkpoint(cursor = cursor)
`
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(migrateScript), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	fixture := "{\"key\":\"history/a\",\"value\":{\"label\":\"Already Canonical\"}}\n"
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_seed.ndjson"), []byte(fixture), 0o644); err != nil {
		t.Fatalf("WriteFile(seed fixture) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(fixture), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_fixture_store_loop_unchanged")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() verdict = %q, want ok", status.Verdict)
	}
	journalBytes, err := os.ReadFile(filepath.Join(appDir, filepath.FromSlash(status.JournalPath)))
	if err != nil {
		t.Fatalf("ReadFile(journal) error = %v", err)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if hasMigrationCheckpointMetadata(entries, 1, 1, 1) {
		t.Fatalf("migration journal counted unchanged store put as checkpoint evidence: %+v", entries)
	}
}

func TestRuntimeRetryMigrationAllowsLogCallsInFixtureTransforms(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_fixture_log")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_fixture_log"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	migrateScript := `load("log", info = "info")

def migrate(record):
  record["label_normalized"] = lower(trim(record["label"]))
  info("history.migrated", records = 1)
  return record
`
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(migrateScript), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	seed := "{\"key\":\"history/a\",\"value\":{\"label\":\"  Dishwasher Done  \"}}\n"
	expected := "{\"key\":\"history/a\",\"value\":{\"label\":\"  Dishwasher Done  \",\"label_normalized\":\"dishwasher done\"}}\n"
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_seed.ndjson"), []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(seed fixture) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_fixture_log")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() verdict = %q, want ok", status.Verdict)
	}
	if status.StepsCompleted != 1 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 1", status.StepsCompleted)
	}
}

func TestRuntimeRetryMigrationPreservesHashInSingleQuotedStrings(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_fixture_single_quote_hash")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_fixture_single_quote_hash"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	migrateScript := `load("log", info = "info")

def migrate(record):
  record["tag"] = record.get("tag", '#kitchen')
  info('history tagged #kitchen', records = 1)
  return record
`
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(migrateScript), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	seed := "{\"key\":\"history/a\",\"value\":{\"label\":\"Tea\"}}\n"
	expected := "{\"key\":\"history/a\",\"value\":{\"label\":\"Tea\",\"tag\":\"#kitchen\"}}\n"
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_seed.ndjson"), []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(seed fixture) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_fixture_single_quote_hash")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() verdict = %q, want ok", status.Verdict)
	}
	journalBytes, err := os.ReadFile(filepath.Join(appDir, filepath.FromSlash(status.JournalPath)))
	if err != nil {
		t.Fatalf("ReadFile(journal) error = %v", err)
	}
	if !strings.Contains(string(journalBytes), `"message":"history tagged #kitchen"`) {
		t.Fatalf("migration journal missing single-quoted log message with hash: %q", string(journalBytes))
	}
}

func TestRuntimeRetryMigrationEmitsCheckpointEveryForFixtureEffects(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_checkpoint_every")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_checkpoint_every"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1
checkpoint_every = 2

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	script := "def migrate(record):\n    record[\"migrated\"] = true\n"
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(script), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	seed := strings.Join([]string{
		"{\"key\":\"history/1\",\"value\":{\"count\":1}}",
		"{\"key\":\"history/2\",\"value\":{\"count\":2}}",
		"{\"key\":\"history/3\",\"value\":{\"count\":3}}",
		"",
	}, "\n")
	expected := strings.Join([]string{
		"{\"key\":\"history/1\",\"value\":{\"count\":1,\"migrated\":true}}",
		"{\"key\":\"history/2\",\"value\":{\"count\":2,\"migrated\":true}}",
		"{\"key\":\"history/3\",\"value\":{\"count\":3,\"migrated\":true}}",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_seed.ndjson"), []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(seed fixture) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_checkpoint_every")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() verdict = %q, want ok", status.Verdict)
	}

	journalBytes, err := os.ReadFile(filepath.Join(appDir, filepath.FromSlash(status.JournalPath)))
	if err != nil {
		t.Fatalf("ReadFile(journal) error = %v", err)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationCheckpointMetadata(entries, 1, 2, 2) {
		t.Fatalf("migration journal missing checkpoint_every evidence: %+v", entries)
	}
}

func TestRuntimeRetryMigrationAbortCallFailsCurrentStep(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_abort_call")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_abort_call"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	script := "load(\"migrate.env\", abort = \"halt\")\ndef migrate(record):\n    halt(\"unsafe history shape\")\n"
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(script), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	fixture := "{\"key\":\"history/1\",\"value\":{\"count\":1}}\n"
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_seed.ndjson"), []byte(fixture), 0o644); err != nil {
		t.Fatalf("WriteFile(seed fixture) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(fixture), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_abort_call")
	if !errors.Is(err, ErrMigrationAborted) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationAborted", err)
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
	if !strings.Contains(status.LastError, "unsafe history shape") {
		t.Fatalf("RetryMigration() last_error = %q, want abort reason", status.LastError)
	}

	journalBytes, err := os.ReadFile(filepath.Join(appDir, filepath.FromSlash(status.JournalPath)))
	if err != nil {
		t.Fatalf("ReadFile(journal) error = %v", err)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalErrorContaining(entries, "step_failed_aborted", "unsafe history shape") {
		t.Fatalf("migration journal missing step_failed_aborted entry: %+v", entries)
	}
}

func TestRuntimeMigrationFixtureParsesAbortAliases(t *testing.T) {
	directScript := []byte("def migrate(record):\n    abort('unsafe # record shape')\n")
	transforms, err := parseRuntimeMigrationFixtureTransforms(directScript)
	if err != nil {
		t.Fatalf("parseRuntimeMigrationFixtureTransforms(direct) error = %v", err)
	}
	if len(transforms) != 1 || transforms[0].Operation != "abort" || transforms[0].Reason != "unsafe # record shape" {
		t.Fatalf("parseRuntimeMigrationFixtureTransforms(direct) = %+v, want single-quoted abort transform", transforms)
	}

	recordScript := []byte("load(\"migrate.env\", abort = \"stop_now\")\ndef migrate(record):\n    stop_now(\"bad record\")\n")
	transforms, err = parseRuntimeMigrationFixtureTransforms(recordScript)
	if err != nil {
		t.Fatalf("parseRuntimeMigrationFixtureTransforms() error = %v", err)
	}
	if len(transforms) != 1 || transforms[0].Operation != "abort" || transforms[0].Reason != "bad record" {
		t.Fatalf("parseRuntimeMigrationFixtureTransforms() = %+v, want abort transform", transforms)
	}

	storeScript := []byte(`load("store", list_keys = "keys", get = "read", put = "write")
load("migrate.env", checkpoint = "save", abort = "fail")

def migrate():
    cursor = None
    while True:
        page = keys(prefix = "history/", after = cursor, limit = 500)
        if len(page) == 0: break
        for key in page:
            rec = read(key)
            fail("unsafe store shape")
            write(key, rec)
        cursor = page[-1]
        save(cursor = cursor)
`)
	plan, err := parseRuntimeMigrationStoreFixturePlan(storeScript)
	if err != nil {
		t.Fatalf("parseRuntimeMigrationStoreFixturePlan() error = %v", err)
	}
	if plan == nil || len(plan.Transforms) != 1 || plan.Transforms[0].Operation != "abort" || plan.Transforms[0].Reason != "unsafe store shape" {
		t.Fatalf("parseRuntimeMigrationStoreFixturePlan() = %+v, want aliased abort transform", plan)
	}
}

func TestRuntimeRetryMigrationFailsWhenFixtureExpectedMismatch(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_fixture_mismatch")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_fixture_mismatch"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	seed := "{\"key\":\"history/a\",\"value\":{\"a\":2,\"z\":1}}\n"
	expected := "{\"key\":\"history/a\",\"value\":{\"a\":3,\"z\":1}}\n"
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_seed.ndjson"), []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(seed fixture) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_fixture_mismatch")
	if !errors.Is(err, ErrMigrationFixtureMismatch) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationFixtureMismatch", err)
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
	if !strings.Contains(status.LastError, "fixture mismatch") {
		t.Fatalf("RetryMigration() last_error = %q, want fixture mismatch message", status.LastError)
	}

	journalPath := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
	journalBytes, readErr := os.ReadFile(journalPath)
	if readErr != nil {
		t.Fatalf("ReadFile(journal) error = %v", readErr)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalEvent(entries, "step_failed_fixture_mismatch") {
		t.Fatalf("migration journal missing step_failed_fixture_mismatch event: %+v", entries)
	}
	if !hasMigrationJournalErrorContaining(entries, "step_failed_fixture_mismatch", "expected={\"a\":3,\"z\":1} actual={\"a\":2,\"z\":1}") {
		t.Fatalf("migration journal missing canonical expected/actual mismatch evidence: %+v", entries)
	}
	if !hasMigrationJournalErrorContaining(entries, "step_failed_fixture_mismatch", "first_diff_byte=5 expected_byte=0x33 actual_byte=0x32") {
		t.Fatalf("migration journal missing byte-level mismatch evidence: %+v", entries)
	}
}

func TestRuntimeRetryMigrationFailsWhenFixtureRecordNotCanonical(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_fixture_noncanonical")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_fixture_noncanonical"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	seed := "{\"key\":\"history/a\",\"value\":{\"a\":2,\"z\":1}}\n"
	expected := "{\"value\":{\"a\":2,\"z\":1},\"key\":\"history/a\"}\n"
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_seed.ndjson"), []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(seed fixture) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_fixture_noncanonical")
	if !errors.Is(err, ErrMigrationFixtureMismatch) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationFixtureMismatch", err)
	}
	if !strings.Contains(err.Error(), "not canonical JSON") {
		t.Fatalf("RetryMigration() error = %q, want non-canonical detail", err.Error())
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

	journalPath := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
	journalBytes, readErr := os.ReadFile(journalPath)
	if readErr != nil {
		t.Fatalf("ReadFile(journal) error = %v", readErr)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalEvent(entries, "step_failed_fixture_mismatch") {
		t.Fatalf("migration journal missing step_failed_fixture_mismatch event: %+v", entries)
	}
}

func TestRuntimeRetryMigrationFailsWhenFixtureKeyInvalid(t *testing.T) {
	cases := []struct {
		name       string
		key        string
		wantDetail string
	}{
		{
			name:       "non_nfc",
			key:        "history/cafe\u0301",
			wantDetail: "fixture key must be NFC normalized",
		},
		{
			name:       "too_long",
			key:        "history/" + strings.Repeat("x", runtimeMigrationFixtureMaxKeyBytes),
			wantDetail: "fixture key byte length must be 1..256",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			appName := "migrate_fixture_key_" + tc.name
			appDir := filepath.Join(tempDir, appName)
			if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
				t.Fatalf("MkdirAll(migrate) error = %v", err)
			}
			if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
				t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
			}
			manifest := fmt.Sprintf(`name = %q
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`, appName)
			if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
				t.Fatalf("WriteFile(manifest) error = %v", err)
			}
			if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
				t.Fatalf("WriteFile(main) error = %v", err)
			}
			if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
				t.Fatalf("WriteFile(migrate) error = %v", err)
			}
			fixture := fmt.Sprintf("{\"key\":%q,\"value\":{}}\n", tc.key)
			if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_seed.ndjson"), []byte(fixture), 0o644); err != nil {
				t.Fatalf("WriteFile(seed fixture) error = %v", err)
			}
			if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(fixture), 0o644); err != nil {
				t.Fatalf("WriteFile(expected fixture) error = %v", err)
			}

			runtime := NewRuntime()
			if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
				t.Fatalf("LoadPackage() error = %v", err)
			}

			status, err := runtime.RetryMigration(appName)
			if !errors.Is(err, ErrMigrationFixtureMismatch) {
				t.Fatalf("RetryMigration() error = %v, want ErrMigrationFixtureMismatch", err)
			}
			if !strings.Contains(err.Error(), tc.wantDetail) {
				t.Fatalf("RetryMigration() error = %q, want %q", err.Error(), tc.wantDetail)
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
			if !strings.Contains(status.LastError, "fixture mismatch") {
				t.Fatalf("RetryMigration() last_error = %q, want fixture mismatch message", status.LastError)
			}

			journalPath := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
			journalBytes, readErr := os.ReadFile(journalPath)
			if readErr != nil {
				t.Fatalf("ReadFile(journal) error = %v", readErr)
			}
			entries := parseMigrationJournalEntries(t, journalBytes)
			if !hasMigrationJournalEvent(entries, "step_failed_fixture_mismatch") {
				t.Fatalf("migration journal missing step_failed_fixture_mismatch event: %+v", entries)
			}
		})
	}
}

func TestRuntimeRetryMigrationFailsWhenFixtureRecordLimitExceeded(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_fixture_limit")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_fixture_limit"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	seed := buildRuntimeFixtureRows(runtimeMigrationFixtureMaxRows + 1)
	expected := "{\"key\":\"history/1\",\"value\":{\"count\":1}}\n"
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_seed.ndjson"), []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(seed fixture) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_fixture_limit")
	if !errors.Is(err, ErrMigrationFixtureMismatch) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationFixtureMismatch", err)
	}
	if !strings.Contains(err.Error(), "exceeds maximum records") {
		t.Fatalf("RetryMigration() error = %q, want max records detail", err.Error())
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
	if !strings.Contains(status.LastError, "fixture mismatch") {
		t.Fatalf("RetryMigration() last_error = %q, want fixture mismatch message", status.LastError)
	}

	journalPath := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
	journalBytes, readErr := os.ReadFile(journalPath)
	if readErr != nil {
		t.Fatalf("ReadFile(journal) error = %v", readErr)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalEvent(entries, "step_failed_fixture_mismatch") {
		t.Fatalf("migration journal missing step_failed_fixture_mismatch event: %+v", entries)
	}
}

func TestRuntimeRetryMigrationFailsWhenFixturePathEscapesRoot(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_fixture_escape")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_fixture_escape"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "../outside_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "outside_seed.ndjson"), []byte("{\"key\":\"history/a\",\"value\":{}}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(outside seed fixture) error = %v", err)
	}
	expected := "{\"key\":\"history/a\",\"value\":{}}\n"
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_fixture_escape")
	if !errors.Is(err, ErrMigrationFixtureMismatch) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationFixtureMismatch", err)
	}
	if !strings.Contains(err.Error(), "must resolve within package root") {
		t.Fatalf("RetryMigration() error = %q, want root-path detail", err.Error())
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
	if !strings.Contains(status.LastError, "fixture mismatch") {
		t.Fatalf("RetryMigration() last_error = %q, want fixture mismatch message", status.LastError)
	}

	journalPath := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
	journalBytes, readErr := os.ReadFile(journalPath)
	if readErr != nil {
		t.Fatalf("ReadFile(journal) error = %v", readErr)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalEvent(entries, "step_failed_fixture_mismatch") {
		t.Fatalf("migration journal missing step_failed_fixture_mismatch event: %+v", entries)
	}
}

func TestRuntimeRetryMigrationFailsWhenFixturePathEscapesRootViaSymlink(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_fixture_symlink_escape")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_fixture_symlink_escape"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "none"

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	outsideSeedPath := filepath.Join(tempDir, "outside_seed.ndjson")
	if err := os.WriteFile(outsideSeedPath, []byte("{\"key\":\"history/a\",\"value\":{}}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(outside seed fixture) error = %v", err)
	}
	seedPath := filepath.Join(appDir, "tests", "migrate_fixtures", "history_seed.ndjson")
	if err := os.Symlink(outsideSeedPath, seedPath); err != nil {
		if goruntime.GOOS == "windows" || errors.Is(err, fs.ErrPermission) {
			t.Skipf("Symlink not permitted on this platform: %v", err)
		}
		t.Fatalf("Symlink(seed fixture) error = %v", err)
	}
	expected := "{\"key\":\"history/a\",\"value\":{}}\n"
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_fixture_symlink_escape")
	if !errors.Is(err, ErrMigrationFixtureMismatch) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationFixtureMismatch", err)
	}
	if !strings.Contains(err.Error(), "must resolve within package root") {
		t.Fatalf("RetryMigration() error = %q, want root-path detail", err.Error())
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
	if !strings.Contains(status.LastError, "fixture mismatch") {
		t.Fatalf("RetryMigration() last_error = %q, want fixture mismatch message", status.LastError)
	}

	journalPath := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
	journalBytes, readErr := os.ReadFile(journalPath)
	if readErr != nil {
		t.Fatalf("ReadFile(journal) error = %v", readErr)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalEvent(entries, "step_failed_fixture_mismatch") {
		t.Fatalf("migration journal missing step_failed_fixture_mismatch event: %+v", entries)
	}
}

func TestRuntimeRetryMigrationFailsWhenFixtureDeclarationMissingForPendingStep(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_fixture_missing_step")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "tests", "migrate_fixtures"), 0o755); err != nil {
		t.Fatalf("MkdirAll(tests/migrate_fixtures) error = %v", err)
	}
	manifest := `name = "migrate_fixture_missing_step"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 2

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

[[migrate.fixture]]
step = "0001"
prior_version = "1"
seed = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected = "tests/migrate_fixtures/history_v2_expected.ndjson"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate 1) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0002_2_to_3.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate 2) error = %v", err)
	}
	seed := "{\"key\":\"history/a\",\"value\":{\"a\":1}}\n"
	expected := "{\"key\":\"history/a\",\"value\":{\"a\":1}}\n"
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_v1_seed.ndjson"), []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(seed fixture) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "tests", "migrate_fixtures", "history_v2_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected fixture) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_fixture_missing_step")
	if !errors.Is(err, ErrMigrationFixtureUnavailable) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationFixtureUnavailable", err)
	}
	if status.Verdict != "step_failed" {
		t.Fatalf("RetryMigration() verdict = %q, want step_failed", status.Verdict)
	}
	if status.StepsCompleted != 1 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 1", status.StepsCompleted)
	}
	if status.LastStep != 2 {
		t.Fatalf("RetryMigration() last_step = %d, want 2", status.LastStep)
	}
	if !strings.Contains(status.LastError, "fixture unavailable") {
		t.Fatalf("RetryMigration() last_error = %q, want fixture unavailable message", status.LastError)
	}

	journalPath := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
	journalBytes, readErr := os.ReadFile(journalPath)
	if readErr != nil {
		t.Fatalf("ReadFile(journal) error = %v", readErr)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalEvent(entries, "step_failed_fixture_unavailable") {
		t.Fatalf("migration journal missing step_failed_fixture_unavailable event: %+v", entries)
	}
}

func TestRuntimeRetryMigrationFailsWhenPendingScriptUnavailable(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_missing_step")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"migrate_missing_step\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate 1) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0002_2_to_3.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate 2) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	if _, err := runtime.RetryMigration("migrate_missing_step"); err != nil {
		t.Fatalf("RetryMigration() initial run error = %v", err)
	}
	if _, err := runtime.AbortMigration("migrate_missing_step", MigrationAbortToCheckpoint); err != nil {
		t.Fatalf("AbortMigration() error = %v", err)
	}

	if err := os.Remove(filepath.Join(appDir, "migrate", "0002_2_to_3.tal")); err != nil {
		t.Fatalf("Remove(migrate 2) error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_missing_step")
	if !errors.Is(err, ErrMigrationStepUnavailable) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationStepUnavailable", err)
	}
	if status.Verdict != "step_failed" {
		t.Fatalf("RetryMigration() verdict = %q, want step_failed", status.Verdict)
	}
	if status.StepsCompleted != 1 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 1", status.StepsCompleted)
	}
	if status.LastStep != 2 {
		t.Fatalf("RetryMigration() last_step = %d, want 2", status.LastStep)
	}
	if !strings.Contains(status.LastError, "script unavailable") {
		t.Fatalf("RetryMigration() last_error = %q, want script unavailable message", status.LastError)
	}
	journalPath := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
	journalBytes, readErr := os.ReadFile(journalPath)
	if readErr != nil {
		t.Fatalf("ReadFile(journal) error = %v", readErr)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationStepMetadata(entries, "step_failed_unavailable", 2, "2", "3", "0002_2_to_3.tal") {
		t.Fatalf("migration journal missing step_failed_unavailable metadata for step 2: %+v", entries)
	}
}

func TestRuntimeRetryMigrationFailsWhenPendingScriptInvalid(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_invalid_step")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"migrate_invalid_step\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate 1) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0002_2_to_3.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate 2) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	if _, err := runtime.RetryMigration("migrate_invalid_step"); err != nil {
		t.Fatalf("RetryMigration() initial run error = %v", err)
	}
	if _, err := runtime.AbortMigration("migrate_invalid_step", MigrationAbortToCheckpoint); err != nil {
		t.Fatalf("AbortMigration() error = %v", err)
	}

	invalid := "load(\"bus\", emit = \"emit\")\n\ndef migrate():\n    pass\n"
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0002_2_to_3.tal"), []byte(invalid), 0o644); err != nil {
		t.Fatalf("WriteFile(invalid migrate 2) error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_invalid_step")
	if !errors.Is(err, ErrMigrationStepInvalid) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationStepInvalid", err)
	}
	if status.Verdict != "step_failed" {
		t.Fatalf("RetryMigration() verdict = %q, want step_failed", status.Verdict)
	}
	if status.StepsCompleted != 1 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 1", status.StepsCompleted)
	}
	if status.LastStep != 2 {
		t.Fatalf("RetryMigration() last_step = %d, want 2", status.LastStep)
	}
	if !strings.Contains(status.LastError, "script invalid") {
		t.Fatalf("RetryMigration() last_error = %q, want script invalid message", status.LastError)
	}

	journalPath := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
	journalBytes, readErr := os.ReadFile(journalPath)
	if readErr != nil {
		t.Fatalf("ReadFile(journal) error = %v", readErr)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationStepMetadata(entries, "step_failed_invalid_script", 2, "2", "3", "0002_2_to_3.tal") {
		t.Fatalf("migration journal missing step_failed_invalid_script metadata for step 2: %+v", entries)
	}
}

func TestRuntimeRetryMigrationRejectsArtifactPatchForDifferentLineage(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_artifact_owner")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"migrate_artifact_owner\"\napp_id = \"app:sha256:1111111111111111111111111111111111111111111111111111111111111111\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	script := `load("artifact.self", patch = "patch")

def migrate():
    patch("artifact-1", owner_app_id = "app:sha256:2222222222222222222222222222222222222222222222222222222222222222", owner_manifest_name = "migrate_artifact_owner")
`
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(script), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_artifact_owner")
	if !errors.Is(err, ErrMigrationArtifactOwnership) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationArtifactOwnership", err)
	}
	if status.Verdict != "step_failed" {
		t.Fatalf("RetryMigration() verdict = %q, want step_failed", status.Verdict)
	}
	if status.StepsCompleted != 0 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 0", status.StepsCompleted)
	}
	if !strings.Contains(status.LastError, "host effect rejected") {
		t.Fatalf("RetryMigration() last_error = %q, want host rejection message", status.LastError)
	}
	journalPath := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
	journalBytes, readErr := os.ReadFile(journalPath)
	if readErr != nil {
		t.Fatalf("ReadFile(journal) error = %v", readErr)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalErrorContaining(entries, "step_failed_host_rejected", "owner_app_id \"app:sha256:2222222222222222222222222222222222222222222222222222222222222222\" does not match app_id \"app:sha256:1111111111111111111111111111111111111111111111111111111111111111\"") {
		t.Fatalf("migration journal missing artifact owner mismatch evidence: %+v", entries)
	}
}

func TestRuntimeRetryMigrationRejectsArtifactPatchWithoutOwnerAppID(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_artifact_owner_required")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"migrate_artifact_owner_required\"\napp_id = \"app:sha256:1111111111111111111111111111111111111111111111111111111111111111\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	script := `load("artifact.self", patch = "patch")

def migrate():
    patch("artifact-1")
`
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(script), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_artifact_owner_required")
	if !errors.Is(err, ErrMigrationArtifactOwnership) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationArtifactOwnership", err)
	}
	if status.Verdict != "step_failed" {
		t.Fatalf("RetryMigration() verdict = %q, want step_failed", status.Verdict)
	}
	if status.StepsCompleted != 0 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 0", status.StepsCompleted)
	}
	if !strings.Contains(status.LastError, "host effect rejected") {
		t.Fatalf("RetryMigration() last_error = %q, want host rejection message", status.LastError)
	}
	journalPath := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
	journalBytes, readErr := os.ReadFile(journalPath)
	if readErr != nil {
		t.Fatalf("ReadFile(journal) error = %v", readErr)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalErrorContaining(entries, "step_failed_host_rejected", "patch missing owner_app_id") {
		t.Fatalf("migration journal missing owner_app_id evidence: %+v", entries)
	}
}

func TestRuntimeRetryMigrationRejectsArtifactPatchWithoutArtifactID(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_artifact_id_required")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	appID := "app:sha256:1111111111111111111111111111111111111111111111111111111111111111"
	manifest := fmt.Sprintf("name = \"migrate_artifact_id_required\"\napp_id = %q\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n", appID)
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	script := fmt.Sprintf(`load("artifact.self", patch = "patch")

def migrate():
    artifact_id = "artifact-1"
    patch(artifact_id, owner_app_id = %q)
`, appID)
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(script), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_artifact_id_required")
	if !errors.Is(err, ErrMigrationArtifactOwnership) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationArtifactOwnership", err)
	}
	if status.Verdict != "step_failed" {
		t.Fatalf("RetryMigration() verdict = %q, want step_failed", status.Verdict)
	}
	if status.StepsCompleted != 0 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 0", status.StepsCompleted)
	}
	if !strings.Contains(status.LastError, "host effect rejected") {
		t.Fatalf("RetryMigration() last_error = %q, want host rejection message", status.LastError)
	}
	journalPath := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
	journalBytes, readErr := os.ReadFile(journalPath)
	if readErr != nil {
		t.Fatalf("ReadFile(journal) error = %v", readErr)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalErrorContaining(entries, "step_failed_host_rejected", "patch missing artifact_id") {
		t.Fatalf("migration journal missing artifact_id evidence: %+v", entries)
	}
}

func TestRuntimeRetryMigrationJournalsAcceptedArtifactPatchDeclarations(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_artifact_patch_journal")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	appID := "app:sha256:1111111111111111111111111111111111111111111111111111111111111111"
	manifest := fmt.Sprintf("name = \"migrate_artifact_patch_journal\"\napp_id = %q\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n", appID)
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	script := fmt.Sprintf(`load("artifact.self", patch = "patch")

def migrate():
    patch("artifact-1", owner_app_id = %q)
    patch("artifact-2", owner_app_id = %q)
`, appID, appID)
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(script), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_artifact_patch_journal")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() verdict = %q, want ok", status.Verdict)
	}

	journalPath := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
	journalBytes, readErr := os.ReadFile(journalPath)
	if readErr != nil {
		t.Fatalf("ReadFile(journal) error = %v", readErr)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalArtifactPatch(entries, "artifact-1", appID, 1) {
		t.Fatalf("migration journal missing first artifact patch evidence: %+v", entries)
	}
	if !hasMigrationJournalArtifactPatch(entries, "artifact-2", appID, 2) {
		t.Fatalf("migration journal missing second artifact patch evidence: %+v", entries)
	}
}

func TestRuntimeRetryMigrationRejectsArtifactPatchHardCap(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_artifact_patch_limit")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	appID := "app:sha256:1111111111111111111111111111111111111111111111111111111111111111"
	manifest := fmt.Sprintf("name = \"migrate_artifact_patch_limit\"\napp_id = %q\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n", appID)
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}

	var script strings.Builder
	script.WriteString("load(\"artifact.self\", patch = \"patch\")\n\ndef migrate():\n")
	for i := 0; i <= migrationMaxArtifactPatches; i++ {
		fmt.Fprintf(&script, "    patch(\"artifact-%d\", owner_app_id = %q)\n", i, appID)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(script.String()), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_artifact_patch_limit")
	if !errors.Is(err, ErrMigrationResourceLimit) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationResourceLimit", err)
	}
	if status.Verdict != "step_failed" {
		t.Fatalf("RetryMigration() verdict = %q, want step_failed", status.Verdict)
	}
	if status.StepsCompleted != 0 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 0", status.StepsCompleted)
	}
	if !strings.Contains(status.LastError, "resource limit exceeded") {
		t.Fatalf("RetryMigration() last_error = %q, want resource limit message", status.LastError)
	}

	journalPath := filepath.Join(appDir, filepath.FromSlash(status.JournalPath))
	journalBytes, readErr := os.ReadFile(journalPath)
	if readErr != nil {
		t.Fatalf("ReadFile(journal) error = %v", readErr)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalErrorContaining(entries, "step_failed_resource_limit", "artifact.self.patch count exceeds hard cap") {
		t.Fatalf("migration journal missing artifact patch cap evidence: %+v", entries)
	}
}

func TestRuntimeMigrationResourceLimitValidation(t *testing.T) {
	limits := runtimeMigrationResourceLimits{
		MaxStoreOps:              2,
		MaxWriteVolumeBytes:      10,
		MaxArtifactPatchAttempts: 1,
	}
	cases := []struct {
		name  string
		stats runtimeMigrationResourceStats
		want  string
	}{
		{
			name:  "store op cap",
			stats: runtimeMigrationResourceStats{StoreOps: 3},
			want:  "store ops exceed hard cap",
		},
		{
			name:  "write volume cap",
			stats: runtimeMigrationResourceStats{WriteVolumeBytes: 11},
			want:  "write volume exceeds hard cap",
		},
		{
			name:  "artifact patch cap",
			stats: runtimeMigrationResourceStats{ArtifactPatchAttempts: 2},
			want:  "artifact patch attempts exceed hard cap",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRuntimeMigrationResourceLimits(tc.stats, limits)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validateRuntimeMigrationResourceLimits() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestRuntimeRetryMigrationIgnoresCommentedLoadStatements(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_comment_load")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"migrate_comment_load\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	script := "# load(\"bus\", emit = \"emit\")\nload(\"store\", get = \"get\")\n\ndef migrate():\n    pass\n"
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(script), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_comment_load")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() verdict = %q, want ok", status.Verdict)
	}
	if status.StepsCompleted != 1 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 1", status.StepsCompleted)
	}
}

