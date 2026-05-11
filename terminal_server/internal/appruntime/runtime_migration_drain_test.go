package appruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
	if !status.RequiresDrain || status.DrainReady {
		t.Fatalf("RetryMigration() drain status requires_drain=%v drain_ready=%v, want true/false", status.RequiresDrain, status.DrainReady)
	}
	if status.DrainTimeout != 2*time.Second {
		t.Fatalf("RetryMigration() drain_timeout = %s, want 2s", status.DrainTimeout)
	}
	if status.DrainBlockedAt.IsZero() {
		t.Fatalf("RetryMigration() drain_blocked_at is zero")
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
	if !status.RequiresDrain || status.DrainReady {
		t.Fatalf("RetryMigration() timeout drain status requires_drain=%v drain_ready=%v, want true/false", status.RequiresDrain, status.DrainReady)
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
	if status.RequiresDrain || !status.DrainReady {
		t.Fatalf("RetryMigration() after drain ready drain status requires_drain=%v drain_ready=%v, want false/true", status.RequiresDrain, status.DrainReady)
	}
}

func TestRuntimeRetryMigrationDrainTimeoutAbortsWithoutRunningStep(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_drain_timeout_evidence")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := `name = "migrate_drain_timeout_evidence"
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

	pendingStatus, err := runtime.RetryMigration("migrate_drain_timeout_evidence")
	if !errors.Is(err, ErrMigrationDrainPending) {
		t.Fatalf("first RetryMigration() error = %v, want ErrMigrationDrainPending", err)
	}

	runtime.mu.Lock()
	state := runtime.migrations["migrate_drain_timeout_evidence"]
	state.DrainBlockedAt = time.Now().Add(-3 * time.Second)
	runtime.migrations["migrate_drain_timeout_evidence"] = state
	runtime.mu.Unlock()

	timeoutStatus, err := runtime.RetryMigration("migrate_drain_timeout_evidence")
	if !errors.Is(err, ErrMigrationDrainTimeout) {
		t.Fatalf("second RetryMigration() error = %v, want ErrMigrationDrainTimeout", err)
	}
	if timeoutStatus.Verdict != "aborted" {
		t.Fatalf("RetryMigration() verdict = %q, want aborted", timeoutStatus.Verdict)
	}
	if timeoutStatus.StepsCompleted != 0 {
		t.Fatalf("RetryMigration() steps_completed = %d, want 0", timeoutStatus.StepsCompleted)
	}

	journalBytes, err := os.ReadFile(filepath.Join(appDir, filepath.FromSlash(pendingStatus.JournalPath)))
	if err != nil {
		t.Fatalf("ReadFile(journal) error = %v", err)
	}
	entries := parseMigrationJournalEntries(t, journalBytes)
	if !hasMigrationJournalEvent(entries, "retry_blocked_drain_pending") {
		t.Fatalf("journal missing retry_blocked_drain_pending entry: %+v", entries)
	}
	if !hasMigrationJournalEvent(entries, "retry_blocked_drain_timeout") {
		t.Fatalf("journal missing retry_blocked_drain_timeout entry: %+v", entries)
	}
	for _, forbidden := range []string{"retry_started", "step_started", "step_committed", "retry_committed"} {
		if hasMigrationJournalEvent(entries, forbidden) {
			t.Fatalf("journal contains %q after drain-timeout abort; migration body must not run while drain is unsatisfied: %+v", forbidden, entries)
		}
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

func TestRuntimeDrainReadyReplaysFromJournal(t *testing.T) {
	tempDir := t.TempDir()
	appDir := filepath.Join(tempDir, "migrate_drain_ready_replay")
	if err := os.MkdirAll(filepath.Join(appDir, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	manifest := `name = "migrate_drain_ready_replay"
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
	if _, err := runtime.RetryMigration("migrate_drain_ready_replay"); !errors.Is(err, ErrMigrationDrainPending) {
		t.Fatalf("RetryMigration() error = %v, want ErrMigrationDrainPending", err)
	}
	if err := runtime.SetMigrationDrainReady("migrate_drain_ready_replay", true); err != nil {
		t.Fatalf("SetMigrationDrainReady() error = %v", err)
	}

	restarted := NewRuntime()
	if _, err := restarted.LoadPackage(context.Background(), appDir); err != nil {
		t.Fatalf("LoadPackage() after restart error = %v", err)
	}

	restarted.mu.RLock()
	replayedReady := restarted.migrations["migrate_drain_ready_replay"].DrainReady
	restarted.mu.RUnlock()
	if !replayedReady {
		t.Fatalf("replayed drain readiness = false, want true")
	}

	status, err := restarted.RetryMigration("migrate_drain_ready_replay")
	if err != nil {
		t.Fatalf("RetryMigration() after restart error = %v", err)
	}
	if status.Verdict != "ok" || status.StepsCompleted != 1 {
		t.Fatalf("RetryMigration() after restart = verdict %q steps %d, want ok steps 1", status.Verdict, status.StepsCompleted)
	}
}

