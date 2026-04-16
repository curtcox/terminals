package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/appruntime"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

func TestRegisterAppScenarioDefinitions(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "kitchen_watch")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(app) error = %v", err)
	}
	manifest := "name = \"kitchen_watch\"\nversion = \"1.0.0\"\nlanguage = \"tal/1\"\nexports = [\"watch\"]\n"
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}

	runtime := appruntime.NewRuntime()
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	engine := scenario.NewEngine()
	registerAppScenarioDefinitions(engine, runtime)

	snapshot := engine.RegistrySnapshot()
	if len(snapshot) != 1 {
		t.Fatalf("len(RegistrySnapshot()) = %d, want 1", len(snapshot))
	}
	if snapshot[0].Name != "app.kitchen_watch.watch" {
		t.Fatalf("registered name = %q, want app.kitchen_watch.watch", snapshot[0].Name)
	}

	match, ok := engine.MatchActivation(scenario.ActivationRequest{
		Trigger: scenario.Trigger{
			Kind:     scenario.TriggerManual,
			SourceID: "kitchen-1",
			Intent:   "watch",
		},
		RequestedAt: time.Now().UTC(),
	})
	if !ok {
		t.Fatalf("MatchActivation() ok = false, want true")
	}
	if match.Registration.Definition == nil || match.Registration.Definition.Name() != "app.kitchen_watch.watch" {
		t.Fatalf("matched registration definition = %+v, want app.kitchen_watch.watch", match.Registration.Definition)
	}

	activation, ok := match.Activation.(*appScenarioActivation)
	if !ok {
		t.Fatalf("activation type = %T, want *appScenarioActivation", match.Activation)
	}
	if got := activation.activation.ID(); got != "app:kitchen_watch:watch:kitchen-1:r1" {
		t.Fatalf("activation id = %q, want app:kitchen_watch:watch:kitchen-1:r1", got)
	}
}
