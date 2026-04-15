package appruntime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRuntimeLoadAndReloadPackage(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "sound_watch")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := "name = \"sound_watch\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}

	runtime := NewRuntime()
	pkg, err := runtime.LoadPackage(context.Background(), appDir)
	if err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}
	if pkg.Manifest.Name != "sound_watch" {
		t.Fatalf("package name = %q, want sound_watch", pkg.Manifest.Name)
	}

	time.Sleep(5 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start():\n  pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main updated) error = %v", err)
	}
	_, changed, err := runtime.ReloadPackage(context.Background(), "sound_watch")
	if err != nil {
		t.Fatalf("ReloadPackage() error = %v", err)
	}
	if !changed {
		t.Fatalf("ReloadPackage() changed = false, want true")
	}
}
