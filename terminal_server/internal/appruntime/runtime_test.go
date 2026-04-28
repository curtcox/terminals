package appruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRuntimeLoadAndReloadPackage(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "sound_watch")
	if err := os.MkdirAll(filepath.Join(appDir, "kernels"), 0o755); err != nil {
		t.Fatalf("MkdirAll(kernels) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "models"), 0o755); err != nil {
		t.Fatalf("MkdirAll(models) error = %v", err)
	}
	manifest := `name = "sound_watch"
version = "1.0.0"
language = "tal/1"
requires_kernel_api = "kernel/1"
permissions = ["ui.set", "store.kv", "ui.set"]
exports = ["sound_watch"]
kernels = ["kernels/sound_loc.wasm"]
models = ["models/home_sounds.onnx"]
migrate = "migrate"
dev_mode = true
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "kernels", "sound_loc.wasm"), []byte("wasm"), 0o644); err != nil {
		t.Fatalf("WriteFile(kernel) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "models", "home_sounds.onnx"), []byte("onnx"), 0o644); err != nil {
		t.Fatalf("WriteFile(model) error = %v", err)
	}

	runtime := NewRuntime()
	pkg, err := runtime.LoadPackage(context.Background(), appDir)
	if err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}
	if pkg.Manifest.Name != "sound_watch" {
		t.Fatalf("package name = %q, want sound_watch", pkg.Manifest.Name)
	}
	if pkg.Revision != 1 {
		t.Fatalf("revision = %d, want 1", pkg.Revision)
	}
	if got := pkg.Manifest.Permissions; len(got) != 2 || got[0] != "store.kv" || got[1] != "ui.set" {
		t.Fatalf("permissions = %+v, want sorted unique [store.kv ui.set]", got)
	}
	if !pkg.Manifest.DevMode {
		t.Fatalf("DevMode = false, want true")
	}
	if pkg.Manifest.Migrate != "migrate" {
		t.Fatalf("Migrate = %q, want migrate", pkg.Manifest.Migrate)
	}

	time.Sleep(5 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start():\n  pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main updated) error = %v", err)
	}
	reloaded, changed, err := runtime.ReloadPackage(context.Background(), "sound_watch")
	if err != nil {
		t.Fatalf("ReloadPackage() error = %v", err)
	}
	if !changed {
		t.Fatalf("ReloadPackage() changed = false, want true")
	}
	if reloaded.Revision != 2 {
		t.Fatalf("reloaded revision = %d, want 2", reloaded.Revision)
	}

	history := runtime.ListPackageHistory("sound_watch")
	if len(history) != 2 {
		t.Fatalf("len(history) = %d, want 2", len(history))
	}
	if history[0].Revision != 1 || history[1].Revision != 2 {
		t.Fatalf("history revisions = [%d, %d], want [1, 2]", history[0].Revision, history[1].Revision)
	}

	gotByRev, ok := runtime.GetPackageByRevision("sound_watch", 1)
	if !ok || gotByRev.Manifest.Version != "1.0.0" {
		t.Fatalf("GetPackageByRevision(1) = (%+v, %v), want manifest version 1.0.0", gotByRev, ok)
	}
}

func TestRuntimeRejectsUndeclaredPermission(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "bad_app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"bad_app\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\npermissions = [\"root.shell\"]\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}

	_, err := NewRuntime().LoadPackage(context.Background(), appDir)
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("LoadPackage() error = %v, want ErrPermissionDenied", err)
	}
}

func TestRuntimeRejectsIncompatibleKernelAPI(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "kernel_mismatch")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"kernel_mismatch\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\nrequires_kernel_api = \"kernel/9\"\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}

	_, err := NewRuntime().LoadPackage(context.Background(), appDir)
	if !errors.Is(err, ErrKernelAPIIncompatible) {
		t.Fatalf("LoadPackage() error = %v, want ErrKernelAPIIncompatible", err)
	}
}

func TestRuntimeReloadFailureKeepsLastGood(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "sticky")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifestPath := filepath.Join(appDir, "manifest.toml")
	if err := os.WriteFile(manifestPath, []byte("name = \"sticky\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}

	runtime := NewRuntime()
	original, err := runtime.LoadPackage(context.Background(), appDir)
	if err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	time.Sleep(5 * time.Millisecond)
	if err := os.WriteFile(manifestPath, []byte("name = \"sticky\"\nversion = \"1.0.1\"\nlanguage = \"tal/9\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest bad) error = %v", err)
	}

	pkg, changed, err := runtime.ReloadPackage(context.Background(), "sticky")
	if !changed {
		t.Fatalf("ReloadPackage() changed = false, want true")
	}
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("ReloadPackage() error = %v, want ErrInvalidManifest", err)
	}
	if pkg.Revision != original.Revision {
		t.Fatalf("ReloadPackage() revision = %d, want original %d", pkg.Revision, original.Revision)
	}

	current, ok := runtime.GetPackage("sticky")
	if !ok {
		t.Fatalf("GetPackage(sticky) ok = false, want true")
	}
	if current.Revision != original.Revision {
		t.Fatalf("current revision = %d, want original %d", current.Revision, original.Revision)
	}
	if len(runtime.ListPackageHistory("sticky")) != 1 {
		t.Fatalf("history length changed after failed reload")
	}
}

func TestRuntimeRollbackPackage(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "rollback")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifestPath := filepath.Join(appDir, "manifest.toml")
	if err := os.WriteFile(manifestPath, []byte("name = \"rollback\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v1) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage(v1) error = %v", err)
	}

	time.Sleep(5 * time.Millisecond)
	if err := os.WriteFile(manifestPath, []byte("name = \"rollback\"\nversion = \"1.1.0\"\nlanguage = \"tal/1\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v2) error = %v", err)
	}
	if _, changed, err := runtime.ReloadPackage(context.Background(), "rollback"); err != nil || !changed {
		t.Fatalf("ReloadPackage(v2) = (%v, %v), want changed true no error", err, changed)
	}

	rolledBack, err := runtime.RollbackPackage("rollback")
	if err != nil {
		t.Fatalf("RollbackPackage() error = %v", err)
	}
	if rolledBack.Manifest.Version != "1.0.0" {
		t.Fatalf("rolled back version = %q, want 1.0.0", rolledBack.Manifest.Version)
	}
	if len(runtime.ListPackageHistory("rollback")) != 1 {
		t.Fatalf("history length after rollback = %d, want 1", len(runtime.ListPackageHistory("rollback")))
	}
}

func TestRuntimeRollbackRequiresPriorVersion(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "rollback_single")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"rollback_single\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
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
	if _, err := runtime.RollbackPackage("rollback_single"); !errors.Is(err, ErrNoPriorVersion) {
		t.Fatalf("RollbackPackage() error = %v, want ErrNoPriorVersion", err)
	}
}

func TestRuntimeRollbackBlockedWhenMigrationReconcilePending(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "rollback_pending")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifestPath := filepath.Join(appDir, "manifest.toml")
	if err := os.WriteFile(manifestPath, []byte("name = \"rollback_pending\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v1) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage(v1) error = %v", err)
	}

	time.Sleep(5 * time.Millisecond)
	if err := os.WriteFile(manifestPath, []byte("name = \"rollback_pending\"\nversion = \"1.1.0\"\nlanguage = \"tal/1\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v2) error = %v", err)
	}
	if _, changed, err := runtime.ReloadPackage(context.Background(), "rollback_pending"); err != nil || !changed {
		t.Fatalf("ReloadPackage(v2) = (%v, %v), want changed true no error", err, changed)
	}

	runtime.mu.Lock()
	state := runtime.migrations["rollback_pending"]
	state.Verdict = "reconcile_pending"
	state.LastError = ErrMigrationReconcilePending.Error()
	state.ReconciliationPath = "apps/rollback_pending/migrate/r2/reconcile.json"
	state.PendingRecords = map[string]string{"rec-1": "manual"}
	runtime.migrations["rollback_pending"] = state
	runtime.mu.Unlock()

	if _, err := runtime.RollbackPackage("rollback_pending"); !errors.Is(err, ErrMigrationReconcilePending) {
		t.Fatalf("RollbackPackage() error = %v, want ErrMigrationReconcilePending", err)
	}

	current, ok := runtime.GetPackage("rollback_pending")
	if !ok {
		t.Fatalf("GetPackage(rollback_pending) ok = false, want true")
	}
	if current.Manifest.Version != "1.1.0" {
		t.Fatalf("current version = %q, want 1.1.0", current.Manifest.Version)
	}
	if len(runtime.ListPackageHistory("rollback_pending")) != 2 {
		t.Fatalf("history length after blocked rollback = %d, want 2", len(runtime.ListPackageHistory("rollback_pending")))
	}
}

func TestRuntimeRollbackKeepDataRequiresDowngradeSteps(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "rollback_keep_data")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifestPath := filepath.Join(appDir, "manifest.toml")
	if err := os.WriteFile(manifestPath, []byte("name = \"rollback_keep_data\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v1) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage(v1) error = %v", err)
	}

	time.Sleep(5 * time.Millisecond)
	if err := os.WriteFile(manifestPath, []byte("name = \"rollback_keep_data\"\nversion = \"1.1.0\"\nlanguage = \"tal/1\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v2) error = %v", err)
	}
	if _, changed, err := runtime.ReloadPackage(context.Background(), "rollback_keep_data"); err != nil || !changed {
		t.Fatalf("ReloadPackage(v2) = (%v, %v), want changed true no error", err, changed)
	}

	if _, err := runtime.RollbackPackage("rollback_keep_data", RollbackOptions{DataMode: RollbackDataModeKeepData}); !errors.Is(err, ErrRollbackKeepDataRequiresDowngrade) {
		t.Fatalf("RollbackPackage(keep_data) error = %v, want ErrRollbackKeepDataRequiresDowngrade", err)
	}
}

func TestRuntimeRollbackKeepDataAllowedWithDowngradeSteps(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "rollback_keep_data_downgrade")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifestPath := filepath.Join(appDir, "manifest.toml")
	if err := os.WriteFile(manifestPath, []byte("name = \"rollback_keep_data_downgrade\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v1) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage(v1) error = %v", err)
	}

	time.Sleep(5 * time.Millisecond)
	if err := os.WriteFile(manifestPath, []byte("name = \"rollback_keep_data_downgrade\"\nversion = \"1.1.0\"\nlanguage = \"tal/1\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v2) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "migrate", "downgrade"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate/downgrade) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "downgrade", "0001_2_to_1.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(downgrade step) error = %v", err)
	}
	if _, changed, err := runtime.ReloadPackage(context.Background(), "rollback_keep_data_downgrade"); err != nil || !changed {
		t.Fatalf("ReloadPackage(v2) = (%v, %v), want changed true no error", err, changed)
	}

	rolledBack, err := runtime.RollbackPackage("rollback_keep_data_downgrade", RollbackOptions{DataMode: RollbackDataModeKeepData})
	if err != nil {
		t.Fatalf("RollbackPackage(keep_data) error = %v", err)
	}
	if rolledBack.Manifest.Version != "1.0.0" {
		t.Fatalf("rolled back version = %q, want 1.0.0", rolledBack.Manifest.Version)
	}
}

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

	if _, err := runtime.ReconcileMigration("migrate_stub", "rec-1", "accept_current"); !errors.Is(err, ErrMigrationExecutorUnavailable) {
		t.Fatalf("ReconcileMigration() error = %v, want ErrMigrationExecutorUnavailable", err)
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
	journalText := string(journalBytes)
	if !strings.Contains(journalText, `"event":"retry_started"`) || !strings.Contains(journalText, `"event":"retry_committed"`) {
		t.Fatalf("migration journal missing retry events: %q", journalText)
	}

	status, err = runtime.AbortMigration("migrate_live", MigrationAbortToCheckpoint)
	if err != nil {
		t.Fatalf("AbortMigration() error = %v", err)
	}
	if status.Verdict != "aborted" {
		t.Fatalf("AbortMigration() verdict = %q, want aborted", status.Verdict)
	}
	if status.StepsCompleted != 1 {
		t.Fatalf("AbortMigration() steps_completed = %d, want 1", status.StepsCompleted)
	}
	if status.LastStep != 1 {
		t.Fatalf("AbortMigration() last_step = %d, want 1", status.LastStep)
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
	journalText = string(journalBytes)
	if !strings.Contains(journalText, `"event":"aborted"`) || !strings.Contains(journalText, `"target":"baseline"`) {
		t.Fatalf("migration journal missing abort baseline entry: %q", journalText)
	}

	if _, err := runtime.AbortMigration("migrate_live", "invalid_target"); !errors.Is(err, ErrMigrationAbortTargetInvalid) {
		t.Fatalf("AbortMigration(invalid target) error = %v, want ErrMigrationAbortTargetInvalid", err)
	}
}

func TestRuntimeRetryMigrationRequiresDrainReadiness(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_drain_guard")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := `name = "migrate_drain_guard"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "incompatible"
drain_policy = "drain"
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

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	status, err := runtime.RetryMigration("migrate_drain_guard")
	if !errors.Is(err, ErrMigrationDrainTimeout) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationDrainTimeout", err)
	}
	if status.Verdict != "aborted" {
		t.Fatalf("RetryMigration() verdict = %q, want aborted", status.Verdict)
	}
	if status.LastError != ErrMigrationDrainTimeout.Error() {
		t.Fatalf("RetryMigration() last_error = %q, want %q", status.LastError, ErrMigrationDrainTimeout.Error())
	}
	if status.StepsCompleted != 0 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 0", status.StepsCompleted)
	}

	if err := runtime.SetMigrationDrainReady("migrate_drain_guard", true); err != nil {
		t.Fatalf("SetMigrationDrainReady() error = %v", err)
	}

	status, err = runtime.RetryMigration("migrate_drain_guard")
	if err != nil {
		t.Fatalf("RetryMigration() after drain ready error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() after drain ready verdict = %q, want ok", status.Verdict)
	}
	if status.StepsCompleted != 1 {
		t.Fatalf("RetryMigration() after drain ready steps_completed = %d, want 1", status.StepsCompleted)
	}
}

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
	if _, err := runtime.ReconcileMigration("migrate_reconcile", "rec-1", "bad_resolution"); !errors.Is(err, ErrMigrationResolutionInvalid) {
		t.Fatalf("ReconcileMigration() invalid resolution error = %v, want ErrMigrationResolutionInvalid", err)
	}
	status, err := runtime.ReconcileMigration("migrate_reconcile", "rec-1", "force_rewind")
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

func TestRuntimeDefinitionsUsesExportsAndNameFallback(t *testing.T) {
	tempDir := t.TempDir()

	withExports := filepath.Join(tempDir, "with_exports")
	if err := os.MkdirAll(withExports, 0o755); err != nil {
		t.Fatalf("MkdirAll(with_exports) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(withExports, "manifest.toml"), []byte(
		"name = \"with_exports\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\nexports = [\"watch\", \"alert\"]\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(with_exports manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(withExports, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(with_exports main) error = %v", err)
	}

	noExports := filepath.Join(tempDir, "no_exports")
	if err := os.MkdirAll(noExports, 0o755); err != nil {
		t.Fatalf("MkdirAll(no_exports) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(noExports, "manifest.toml"), []byte(
		"name = \"no_exports\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(no_exports manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(noExports, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(no_exports main) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), withExports); err != nil {
		t.Fatalf("LoadPackage(with_exports) error = %v", err)
	}
	if _, err := runtime.LoadPackage(context.Background(), noExports); err != nil {
		t.Fatalf("LoadPackage(no_exports) error = %v", err)
	}

	defs := runtime.Definitions()
	if len(defs) != 3 {
		t.Fatalf("len(Definitions()) = %d, want 3", len(defs))
	}

	names := make(map[string]bool, len(defs))
	for _, def := range defs {
		names[def.Name()] = true
	}
	if !names["app.with_exports.watch"] || !names["app.with_exports.alert"] || !names["app.no_exports"] {
		t.Fatalf("definition names = %+v, want app.with_exports.watch, app.with_exports.alert, app.no_exports", names)
	}
}

func TestRuntimeDefinitionActivationPinnedToRevision(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "pinning")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifestPath := filepath.Join(appDir, "manifest.toml")
	if err := os.WriteFile(manifestPath, []byte(
		"name = \"pinning\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\nexports = [\"watch\"]\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v1) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}

	runtime := NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage(v1) error = %v", err)
	}

	defs := runtime.Definitions()
	if len(defs) != 1 {
		t.Fatalf("len(Definitions()) = %d, want 1", len(defs))
	}
	first, err := defs[0].NewActivation(ActivationRequest{Intent: "watch", DeviceID: "d1"})
	if err != nil {
		t.Fatalf("NewActivation(v1) error = %v", err)
	}
	if first == nil {
		t.Fatalf("NewActivation(v1) = nil")
	}
	if got := first.ID(); got != "app:pinning:watch:d1:r1" {
		t.Fatalf("first activation id = %q, want app:pinning:watch:d1:r1", got)
	}

	time.Sleep(5 * time.Millisecond)
	if err := os.WriteFile(manifestPath, []byte(
		"name = \"pinning\"\nversion = \"1.1.0\"\nlanguage = \"tal/1\"\nexports = [\"watch\"]\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v2) error = %v", err)
	}
	if _, changed, err := runtime.ReloadPackage(context.Background(), "pinning"); err != nil || !changed {
		t.Fatalf("ReloadPackage(v2) = (%v, %v), want changed true no error", err, changed)
	}

	second, err := defs[0].NewActivation(ActivationRequest{Intent: "watch", DeviceID: "d1"})
	if err != nil {
		t.Fatalf("NewActivation(v2) error = %v", err)
	}
	if second == nil {
		t.Fatalf("NewActivation(v2) = nil")
	}
	if got := second.ID(); got != "app:pinning:watch:d1:r2" {
		t.Fatalf("second activation id = %q, want app:pinning:watch:d1:r2", got)
	}
}
