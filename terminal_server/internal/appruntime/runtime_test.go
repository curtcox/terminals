package appruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
