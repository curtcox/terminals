package appruntime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
