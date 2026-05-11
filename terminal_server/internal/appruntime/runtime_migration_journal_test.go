package appruntime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	payload := make([]byte, 0, len(entries)*128)
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

