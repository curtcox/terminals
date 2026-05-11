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
