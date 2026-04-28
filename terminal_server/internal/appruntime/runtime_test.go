package appruntime

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strconv"
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

func TestRuntimeRejectsInvalidAppID(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "bad_app_id")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"bad_app_id\"\napp_id = \"not-an-app-id\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}

	_, err := NewRuntime().LoadPackage(context.Background(), appDir)
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("LoadPackage() error = %v, want ErrInvalidManifest", err)
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

func TestRuntimeMigrationJournalPathUsesAppID(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_identity")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"migrate_identity\"\napp_id = \"app:sha256:1111111111111111111111111111111111111111111111111111111111111111\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
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

	status, err := runtime.RetryMigration("migrate_identity")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if !strings.Contains(status.JournalPath, "apps/app:sha256:1111111111111111111111111111111111111111111111111111111111111111/migrate/") {
		t.Fatalf("RetryMigration() journal_path = %q, want app_id-scoped migration path", status.JournalPath)
	}
}

func TestRuntimeMigrationStateReplaysFromJournal(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_replay")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"migrate_replay\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
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

	if _, err := runtime.RetryMigration("migrate_replay"); err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if _, err := runtime.AbortMigration("migrate_replay", MigrationAbortToCheckpoint); err != nil {
		t.Fatalf("AbortMigration() error = %v", err)
	}

	restarted := NewRuntime()
	if _, err := restarted.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() after restart error = %v", err)
	}

	status, err := restarted.GetMigrationStatus("migrate_replay")
	if err != nil {
		t.Fatalf("GetMigrationStatus() after restart error = %v", err)
	}
	if status.Verdict != "step_failed" {
		t.Fatalf("GetMigrationStatus() verdict = %q, want step_failed", status.Verdict)
	}
	if status.StepsCompleted != 1 {
		t.Fatalf("GetMigrationStatus() steps_completed = %d, want 1", status.StepsCompleted)
	}
	if status.LastStep != 2 {
		t.Fatalf("GetMigrationStatus() last_step = %d, want 2", status.LastStep)
	}
	if status.LastError != "step 2 aborted by operator" {
		t.Fatalf("GetMigrationStatus() last_error = %q, want %q", status.LastError, "step 2 aborted by operator")
	}
}

func TestRuntimeInterruptedMigrationReplaysAsStepFailedAndResumes(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_interrupted_replay")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"migrate_interrupted_replay\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
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
	initialStatus, err := runtime.GetMigrationStatus("migrate_interrupted_replay")
	if err != nil {
		t.Fatalf("GetMigrationStatus() error = %v", err)
	}
	journalPath := filepath.Join(appDir, filepath.FromSlash(initialStatus.JournalPath))
	if err := os.MkdirAll(filepath.Dir(journalPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(journal dir) error = %v", err)
	}

	entries := []map[string]any{
		{
			"event":           "retry_started",
			"step":            0,
			"steps_completed": 0,
			"steps_planned":   2,
			"verdict":         "running",
		},
		{
			"event":           "step_started",
			"step":            2,
			"steps_completed": 1,
			"steps_planned":   2,
			"verdict":         "running",
			"from_version":    "2",
			"to_version":      "3",
			"script":          "0002_2_to_3.tal",
		},
	}
	var payload []byte
	for _, entry := range entries {
		line, marshalErr := json.Marshal(entry)
		if marshalErr != nil {
			t.Fatalf("Marshal(journal entry) error = %v", marshalErr)
		}
		payload = append(payload, line...)
		payload = append(payload, '\n')
	}
	if err := os.WriteFile(journalPath, payload, 0o644); err != nil {
		t.Fatalf("WriteFile(journal) error = %v", err)
	}

	restarted := NewRuntime()
	if _, err := restarted.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() after restart error = %v", err)
	}

	status, err := restarted.GetMigrationStatus("migrate_interrupted_replay")
	if err != nil {
		t.Fatalf("GetMigrationStatus() after restart error = %v", err)
	}
	if status.Verdict != "step_failed" {
		t.Fatalf("GetMigrationStatus() verdict = %q, want step_failed", status.Verdict)
	}
	if status.StepsCompleted != 1 {
		t.Fatalf("GetMigrationStatus() steps_completed = %d, want 1", status.StepsCompleted)
	}
	if status.LastStep != 2 {
		t.Fatalf("GetMigrationStatus() last_step = %d, want 2", status.LastStep)
	}
	if status.LastError != "step 2 interrupted before commit" {
		t.Fatalf("GetMigrationStatus() last_error = %q, want %q", status.LastError, "step 2 interrupted before commit")
	}

	status, err = restarted.RetryMigration("migrate_interrupted_replay")
	if err != nil {
		t.Fatalf("RetryMigration() after interrupted replay error = %v", err)
	}
	if status.Verdict != "ok" {
		t.Fatalf("RetryMigration() verdict = %q, want ok", status.Verdict)
	}
	if status.StepsCompleted != 2 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 2", status.StepsCompleted)
	}

	journalBytes, readErr := os.ReadFile(journalPath)
	if readErr != nil {
		t.Fatalf("ReadFile(journal) error = %v", readErr)
	}
	journalEntries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalEventForStep(journalEntries, "step_committed", 2) {
		t.Fatalf("migration journal missing step 2 commit after interrupted replay retry: %+v", journalEntries)
	}
}

func TestRuntimeRetryMigrationCrashInjectionReplaysAtJournalBoundaries(t *testing.T) {
	testCases := []struct {
		name                string
		event               string
		step                int
		wantReplayCompleted int
		wantReplayLastStep  int
		wantReplayLastError string
	}{
		{
			name:                "retry started boundary",
			event:               "retry_started",
			step:                0,
			wantReplayCompleted: 0,
			wantReplayLastStep:  0,
			wantReplayLastError: ErrMigrationInterrupted.Error(),
		},
		{
			name:                "step started boundary",
			event:               "step_started",
			step:                1,
			wantReplayCompleted: 0,
			wantReplayLastStep:  1,
			wantReplayLastError: "step 1 interrupted before commit",
		},
		{
			name:                "step committed boundary",
			event:               "step_committed",
			step:                1,
			wantReplayCompleted: 1,
			wantReplayLastStep:  1,
			wantReplayLastError: "step 1 interrupted before commit",
		},
		{
			name:                "second step started boundary",
			event:               "step_started",
			step:                2,
			wantReplayCompleted: 1,
			wantReplayLastStep:  2,
			wantReplayLastError: "step 2 interrupted before commit",
		},
		{
			name:                "second step committed boundary",
			event:               "step_committed",
			step:                2,
			wantReplayCompleted: 2,
			wantReplayLastStep:  2,
			wantReplayLastError: "step 2 interrupted before commit",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			appDir := filepath.Join(tempDir, "migrate_crash_replay")
			if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}
			manifest := "name = \"migrate_crash_replay\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
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

			crashInjected := false
			runtime.migrationHook = func(event string, step int) error {
				if crashInjected {
					return nil
				}
				if event == tc.event && step == tc.step {
					crashInjected = true
					return errors.New("injected crash")
				}
				return nil
			}

			status, err := runtime.RetryMigration("migrate_crash_replay")
			if !errors.Is(err, ErrMigrationInterrupted) {
				t.Fatalf("RetryMigration() error = %v, want ErrMigrationInterrupted", err)
			}
			if status.Verdict != "running" {
				t.Fatalf("RetryMigration() verdict = %q, want running", status.Verdict)
			}

			restarted := NewRuntime()
			if _, err := restarted.LoadPackage(context.Background(), appDir); err != nil {
				t.Fatalf("LoadPackage() after restart error = %v", err)
			}

			replayed, err := restarted.GetMigrationStatus("migrate_crash_replay")
			if err != nil {
				t.Fatalf("GetMigrationStatus() after restart error = %v", err)
			}
			if replayed.Verdict != "step_failed" {
				t.Fatalf("GetMigrationStatus() verdict = %q, want step_failed", replayed.Verdict)
			}
			if replayed.StepsCompleted != tc.wantReplayCompleted {
				t.Fatalf("GetMigrationStatus() steps_completed = %d, want %d", replayed.StepsCompleted, tc.wantReplayCompleted)
			}
			if replayed.LastStep != tc.wantReplayLastStep {
				t.Fatalf("GetMigrationStatus() last_step = %d, want %d", replayed.LastStep, tc.wantReplayLastStep)
			}
			if replayed.LastError != tc.wantReplayLastError {
				t.Fatalf("GetMigrationStatus() last_error = %q, want %q", replayed.LastError, tc.wantReplayLastError)
			}

			replayed, err = restarted.RetryMigration("migrate_crash_replay")
			if err != nil {
				t.Fatalf("RetryMigration() after replay error = %v", err)
			}
			if replayed.Verdict != "ok" {
				t.Fatalf("RetryMigration() verdict = %q, want ok", replayed.Verdict)
			}
			if replayed.StepsCompleted != 2 {
				t.Fatalf("RetryMigration() steps_completed = %d, want 2", replayed.StepsCompleted)
			}
		})
	}
}

func TestRuntimeDryRunMigrationJournalReplayExercisesAllBoundaries(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_dryrun_harness")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"migrate_dryrun_harness\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
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

	results, err := runtime.DryRunMigrationJournalReplay("migrate_dryrun_harness")
	if err != nil {
		t.Fatalf("DryRunMigrationJournalReplay() error = %v", err)
	}

	wantBoundaries := []MigrationDryRunBoundary{
		{Event: "retry_started", Step: 0},
		{Event: "step_started", Step: 1},
		{Event: "step_committed", Step: 1},
		{Event: "step_started", Step: 2},
		{Event: "step_committed", Step: 2},
	}
	if len(results) != len(wantBoundaries) {
		t.Fatalf("DryRunMigrationJournalReplay() boundaries = %d, want %d", len(results), len(wantBoundaries))
	}

	for i, result := range results {
		if result.Boundary != wantBoundaries[i] {
			t.Fatalf("DryRunMigrationJournalReplay() boundary[%d] = %+v, want %+v", i, result.Boundary, wantBoundaries[i])
		}
		if result.Interrupted.Verdict != "running" {
			t.Fatalf("DryRunMigrationJournalReplay() interrupted[%d] verdict = %q, want running", i, result.Interrupted.Verdict)
		}
		if result.Replay.Verdict != "step_failed" {
			t.Fatalf("DryRunMigrationJournalReplay() replay[%d] verdict = %q, want step_failed", i, result.Replay.Verdict)
		}
		if result.Final.Verdict != "ok" {
			t.Fatalf("DryRunMigrationJournalReplay() final[%d] verdict = %q, want ok", i, result.Final.Verdict)
		}
		if result.Final.StepsCompleted != 2 {
			t.Fatalf("DryRunMigrationJournalReplay() final[%d] steps_completed = %d, want 2", i, result.Final.StepsCompleted)
		}
	}
}

func TestRuntimeDryRunMigrationJournalReplayReturnsEmptyWhenNoSteps(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_dryrun_empty")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"migrate_dryrun_empty\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
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

	results, err := runtime.DryRunMigrationJournalReplay("migrate_dryrun_empty")
	if err != nil {
		t.Fatalf("DryRunMigrationJournalReplay() error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("DryRunMigrationJournalReplay() result count = %d, want 0", len(results))
	}
}

func TestRuntimeLoadPackageRejectsMigrationWhenDryRunGateFails(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_dryrun_gate_fail")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"migrate_dryrun_gate_fail\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte("def not_migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}

	runtime := NewRuntime()
	runtime.SetMigrationDryRunGateEnabled(true)
	_, err := runtime.LoadPackage(context.Background(), appDir)
	if !errors.Is(err, ErrMigrationDryRunFailed) {
		t.Fatalf("LoadPackage() error = %v, want ErrMigrationDryRunFailed", err)
	}
	if _, ok := runtime.GetPackage("migrate_dryrun_gate_fail"); ok {
		t.Fatalf("GetPackage() ok = true, want false")
	}
}

func TestRuntimeLoadPackageRejectsDrainPolicyWithoutIncompatibleDuringDryRunGate(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_dryrun_drain_policy")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := `name = "migrate_dryrun_drain_policy"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
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
	runtime.SetMigrationDryRunGateEnabled(true)
	_, err := runtime.LoadPackage(context.Background(), appDir)
	if !errors.Is(err, ErrMigrationDryRunFailed) {
		t.Fatalf("LoadPackage() error = %v, want ErrMigrationDryRunFailed", err)
	}
	if !strings.Contains(err.Error(), "drain_policy=drain without compatibility=incompatible") {
		t.Fatalf("LoadPackage() error = %q, want drain compatibility detail", err.Error())
	}
	if _, ok := runtime.GetPackage("migrate_dryrun_drain_policy"); ok {
		t.Fatalf("GetPackage() ok = true, want false")
	}
}

func TestRuntimeLoadPackageRejectsMultiVersionWithoutReadAdapterDuringDryRunGate(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_dryrun_multi_version")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := `name = "migrate_dryrun_multi_version"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "multi_version"
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
	runtime.SetMigrationDryRunGateEnabled(true)
	_, err := runtime.LoadPackage(context.Background(), appDir)
	if !errors.Is(err, ErrMigrationDryRunFailed) {
		t.Fatalf("LoadPackage() error = %v, want ErrMigrationDryRunFailed", err)
	}
	if !strings.Contains(err.Error(), "read-adapter dry-run validation is not implemented") {
		t.Fatalf("LoadPackage() error = %q, want read-adapter validation detail", err.Error())
	}
	if _, ok := runtime.GetPackage("migrate_dryrun_multi_version"); ok {
		t.Fatalf("GetPackage() ok = true, want false")
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
drain_timeout_seconds = 2

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
	if !errors.Is(err, ErrMigrationDrainPending) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationDrainPending", err)
	}
	if status.Verdict != "drain_pending" {
		t.Fatalf("RetryMigration() verdict = %q, want drain_pending", status.Verdict)
	}
	if status.LastError != ErrMigrationDrainPending.Error() {
		t.Fatalf("RetryMigration() last_error = %q, want %q", status.LastError, ErrMigrationDrainPending.Error())
	}
	if status.StepsCompleted != 0 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 0", status.StepsCompleted)
	}

	runtime.mu.Lock()
	state := runtime.migrations["migrate_drain_guard"]
	state.DrainBlockedAt = time.Now().Add(-3 * time.Second)
	runtime.migrations["migrate_drain_guard"] = state
	runtime.mu.Unlock()

	status, err = runtime.RetryMigration("migrate_drain_guard")
	if !errors.Is(err, ErrMigrationDrainTimeout) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationDrainTimeout", err)
	}
	if status.Verdict != "aborted" {
		t.Fatalf("RetryMigration() verdict = %q, want aborted", status.Verdict)
	}
	if status.LastError != ErrMigrationDrainTimeout.Error() {
		t.Fatalf("RetryMigration() last_error = %q, want %q", status.LastError, ErrMigrationDrainTimeout.Error())
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

func TestRuntimeDrainPendingBlockedAtReplaysFromJournal(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_drain_replay")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := `name = "migrate_drain_replay"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1
drain_timeout_seconds = 60

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

	if _, err := runtime.RetryMigration("migrate_drain_replay"); !errors.Is(err, ErrMigrationDrainPending) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationDrainPending", err)
	}

	runtime.mu.RLock()
	firstBlocked := runtime.migrations["migrate_drain_replay"].DrainBlockedAt
	runtime.mu.RUnlock()
	if firstBlocked.IsZero() {
		t.Fatalf("initial drain blocked timestamp is zero")
	}

	restarted := NewRuntime()
	if _, err := restarted.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() after restart error = %v", err)
	}

	restarted.mu.RLock()
	replayedBlocked := restarted.migrations["migrate_drain_replay"].DrainBlockedAt
	restarted.mu.RUnlock()
	if replayedBlocked.IsZero() {
		t.Fatalf("replayed drain blocked timestamp is zero")
	}
	if !replayedBlocked.Equal(firstBlocked) {
		t.Fatalf("replayed drain blocked timestamp = %s, want %s", replayedBlocked.Format(time.RFC3339Nano), firstBlocked.Format(time.RFC3339Nano))
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

type migrationJournalEntry struct {
	Event           string `json:"event"`
	Step            int    `json:"step"`
	FromVersion     string `json:"from_version"`
	ToVersion       string `json:"to_version"`
	Script          string `json:"script"`
	Error           string `json:"error"`
	EffectSequence  int    `json:"effect_sequence"`
	CheckpointEvery int    `json:"checkpoint_every"`
}

func parseMigrationJournalEntries(t *testing.T, data []byte) []migrationJournalEntry {
	t.Helper()
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	entries := make([]migrationJournalEntry, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry migrationJournalEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("Unmarshal(journal line %q) error = %v", line, err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("Scan(journal) error = %v", err)
	}
	return entries
}

func buildRuntimeFixtureRows(count int) string {
	if count <= 0 {
		return ""
	}
	var b strings.Builder
	for i := 1; i <= count; i++ {
		b.WriteString("{\"key\":\"history/")
		b.WriteString(strconv.Itoa(i))
		b.WriteString("\",\"value\":{\"count\":")
		b.WriteString(strconv.Itoa(i))
		b.WriteString("}}\n")
	}
	return b.String()
}

func hasMigrationJournalEvent(entries []migrationJournalEntry, event string) bool {
	for _, entry := range entries {
		if entry.Event == event {
			return true
		}
	}
	return false
}

func hasMigrationJournalErrorContaining(entries []migrationJournalEntry, event string, want string) bool {
	for _, entry := range entries {
		if entry.Event == event && strings.Contains(entry.Error, want) {
			return true
		}
	}
	return false
}

func hasMigrationJournalEventForStep(entries []migrationJournalEntry, event string, step int) bool {
	for _, entry := range entries {
		if entry.Event == event && entry.Step == step {
			return true
		}
	}
	return false
}

func hasMigrationJournalEventSequence(entries []migrationJournalEntry, sequence []string) bool {
	if len(sequence) == 0 {
		return true
	}
	index := 0
	for _, entry := range entries {
		if entry.Event != sequence[index] {
			continue
		}
		index++
		if index == len(sequence) {
			return true
		}
	}
	return false
}

func hasMigrationStepMetadata(entries []migrationJournalEntry, event string, step int, fromVersion string, toVersion string, script string) bool {
	for _, entry := range entries {
		if entry.Event != event || entry.Step != step {
			continue
		}
		if entry.FromVersion == fromVersion && entry.ToVersion == toVersion && entry.Script == script {
			return true
		}
	}
	return false
}

func hasMigrationCheckpointMetadata(entries []migrationJournalEntry, step int, effectSequence int, checkpointEvery int) bool {
	for _, entry := range entries {
		if entry.Event != "checkpoint_committed" || entry.Step != step {
			continue
		}
		if entry.EffectSequence == effectSequence && entry.CheckpointEvery == checkpointEvery {
			return true
		}
	}
	return false
}
