package appruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeDryRunMigrationJournalReplayExercisesAllBoundaries(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_dryrun_harness")
	fixtureDir := filepath.Join(appDir, "tests", "migrate_fixtures")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(fixtures) error = %v", err)
	}
	manifest := strings.Join([]string{
		"name = \"migrate_dryrun_harness\"",
		"version = \"1.0.0\"",
		"language = \"tal/1\"",
		"",
		"[migrate]",
		"checkpoint_every = 2",
		"",
		"[[migrate.fixture]]",
		"step = \"1\"",
		"prior_version = \"1\"",
		"seed = \"tests/migrate_fixtures/step1_seed.ndjson\"",
		"expected = \"tests/migrate_fixtures/step1_expected.ndjson\"",
		"",
		"[[migrate.fixture]]",
		"step = \"2\"",
		"prior_version = \"2\"",
		"seed = \"tests/migrate_fixtures/step2_seed.ndjson\"",
		"expected = \"tests/migrate_fixtures/step2_expected.ndjson\"",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	step1Script := "def migrate(record):\n    record[\"label_normalized\"] = lower(trim(record[\"label\"]))\n"
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(step1Script), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate 1) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0002_2_to_3.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate 2) error = %v", err)
	}
	step1Seed := "{\"key\":\"history/1\",\"value\":{\"label\":\"  Foo  \"}}\n{\"key\":\"history/2\",\"value\":{\"label\":\"Bar\"}}\n"
	step1Expected := "{\"key\":\"history/1\",\"value\":{\"label\":\"  Foo  \",\"label_normalized\":\"foo\"}}\n{\"key\":\"history/2\",\"value\":{\"label\":\"Bar\",\"label_normalized\":\"bar\"}}\n"
	step2Rows := "{\"key\":\"history/3\",\"value\":{\"label\":\"Baz\"}}\n"
	if err := os.WriteFile(filepath.Join(fixtureDir, "step1_seed.ndjson"), []byte(step1Seed), 0o644); err != nil {
		t.Fatalf("WriteFile(step1 seed) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "step1_expected.ndjson"), []byte(step1Expected), 0o644); err != nil {
		t.Fatalf("WriteFile(step1 expected) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "step2_seed.ndjson"), []byte(step2Rows), 0o644); err != nil {
		t.Fatalf("WriteFile(step2 seed) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "step2_expected.ndjson"), []byte(step2Rows), 0o644); err != nil {
		t.Fatalf("WriteFile(step2 expected) error = %v", err)
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
		{Event: "checkpoint_committed", Step: 1},
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
	if !strings.Contains(err.Error(), "without migrate.fixture read_adapter") {
		t.Fatalf("LoadPackage() error = %q, want read-adapter validation detail", err.Error())
	}
	if _, ok := runtime.GetPackage("migrate_dryrun_multi_version"); ok {
		t.Fatalf("GetPackage() ok = true, want false")
	}
}

func TestRuntimeLoadPackageValidatesMultiVersionReadAdapterDuringDryRunGate(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_dryrun_multi_version_adapter")
	fixtureDir := filepath.Join(appDir, "tests", "migrate_fixtures")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(fixtures) error = %v", err)
	}
	manifest := `name = "migrate_dryrun_multi_version_adapter"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "multi_version"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
read_adapter = "tests/migrate_fixtures/read_v2_as_v1.tal"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	migrateScript := "def migrate(record):\n    record[\"label_normalized\"] = lower(trim(record[\"label\"]))\n"
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(migrateScript), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	seed := "{\"key\":\"history/1\",\"value\":{\"label\":\" Tea \"}}\n"
	expected := "{\"key\":\"history/1\",\"value\":{\"label\":\" Tea \",\"label_normalized\":\"tea\"}}\n"
	if err := os.WriteFile(filepath.Join(fixtureDir, "history_seed.ndjson"), []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(seed) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "history_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected) error = %v", err)
	}
	readAdapter := "def read(record):\n    del record[\"label_normalized\"]\n"
	if err := os.WriteFile(filepath.Join(fixtureDir, "read_v2_as_v1.tal"), []byte(readAdapter), 0o644); err != nil {
		t.Fatalf("WriteFile(read_adapter) error = %v", err)
	}

	runtime := NewRuntime()
	runtime.SetMigrationDryRunGateEnabled(true)
	if _, err := runtime.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}
}

func TestRuntimeLoadPackageRejectsUnsupportedReadAdapterReturnDuringDryRunGate(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_dryrun_multi_version_adapter_return")
	fixtureDir := filepath.Join(appDir, "tests", "migrate_fixtures")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(fixtures) error = %v", err)
	}
	manifest := `name = "migrate_dryrun_multi_version_adapter_return"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "multi_version"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
read_adapter = "tests/migrate_fixtures/read_v2_as_v1.tal"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	migrateScript := "def migrate(record):\n    return record\n"
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(migrateScript), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	fixture := "{\"key\":\"history/1\",\"value\":{\"label\":\"Tea\"}}\n"
	if err := os.WriteFile(filepath.Join(fixtureDir, "history_seed.ndjson"), []byte(fixture), 0o644); err != nil {
		t.Fatalf("WriteFile(seed) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "history_expected.ndjson"), []byte(fixture), 0o644); err != nil {
		t.Fatalf("WriteFile(expected) error = %v", err)
	}
	readAdapter := "def read(record):\n    return {}\n"
	if err := os.WriteFile(filepath.Join(fixtureDir, "read_v2_as_v1.tal"), []byte(readAdapter), 0o644); err != nil {
		t.Fatalf("WriteFile(read_adapter) error = %v", err)
	}

	runtime := NewRuntime()
	runtime.SetMigrationDryRunGateEnabled(true)
	if _, err := runtime.LoadPackage(context.Background(), appDir); err == nil {
		t.Fatalf("expected LoadPackage() to reject unsupported read_adapter return")
	} else if !strings.Contains(err.Error(), "unsupported read_adapter return expression") {
		t.Fatalf("LoadPackage() error = %q, want unsupported read_adapter return expression", err.Error())
	}
}

func TestRuntimeLoadPackageRejectsMultiVersionReadAdapterEscapingRootDuringDryRunGate(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_dryrun_multi_version_adapter_escape")
	fixtureDir := filepath.Join(appDir, "tests", "migrate_fixtures")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(fixtures) error = %v", err)
	}
	manifest := `name = "migrate_dryrun_multi_version_adapter_escape"
version = "1.0.0"
language = "tal/1"

[migrate]
declared_steps = 1

[[migrate.step]]
from = "1"
to = "2"
compatibility = "compatible"
drain_policy = "multi_version"

[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
seed = "tests/migrate_fixtures/history_seed.ndjson"
expected = "tests/migrate_fixtures/history_expected.ndjson"
read_adapter = "../outside_read_adapter.tal"
`
	if err := os.WriteFile(filepath.Join(appDir, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main) error = %v", err)
	}
	migrateScript := "def migrate(record):\n    record[\"label_normalized\"] = lower(trim(record[\"label\"]))\n"
	if err := os.WriteFile(filepath.Join(appDir, "migrate", "0001_1_to_2.tal"), []byte(migrateScript), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate) error = %v", err)
	}
	seed := "{\"key\":\"history/1\",\"value\":{\"label\":\" Tea \"}}\n"
	expected := "{\"key\":\"history/1\",\"value\":{\"label\":\" Tea \",\"label_normalized\":\"tea\"}}\n"
	if err := os.WriteFile(filepath.Join(fixtureDir, "history_seed.ndjson"), []byte(seed), 0o644); err != nil {
		t.Fatalf("WriteFile(seed) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "history_expected.ndjson"), []byte(expected), 0o644); err != nil {
		t.Fatalf("WriteFile(expected) error = %v", err)
	}
	outsideAdapterPath := filepath.Join(tempDir, "outside_read_adapter.tal")
	if err := os.WriteFile(outsideAdapterPath, []byte("def read(record):\n    return record\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(outside read_adapter) error = %v", err)
	}

	runtime := NewRuntime()
	runtime.SetMigrationDryRunGateEnabled(true)
	if _, err := runtime.LoadPackage(context.Background(), appDir); !errors.Is(err, ErrMigrationDryRunFailed) {
		t.Fatalf("LoadPackage() error = %v, want ErrMigrationDryRunFailed", err)
	} else if !strings.Contains(err.Error(), "read_adapter") || !strings.Contains(err.Error(), "must resolve within package root") {
		t.Fatalf("LoadPackage() error = %q, want read_adapter root escape detail", err.Error())
	}
}

