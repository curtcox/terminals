// Package appruntime loads and hot-reloads TAR/TAL application packages.
package appruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultMigrationDrainTimeout = 90 * time.Second

const (
	runtimeMigrationFixtureMaxRows     = 4096
	runtimeMigrationFixtureMaxKeyBytes = 256
)

const (
	migrationMaxWriteVolumeBytes = 100 * 1024 * 1024
	migrationMaxStoreOps         = 1_000_000
	migrationMaxArtifactPatches  = 10_000
)

const (
	// MigrationAbortToCheckpoint aborts the current step and rewinds to the last checkpoint.
	MigrationAbortToCheckpoint = "checkpoint"
	// MigrationAbortToBaseline aborts the migration run and rewinds progress to pre-upgrade baseline.
	MigrationAbortToBaseline = "baseline"
	// RollbackDataModeArchiveData archives migration-owned data when reverse steps are unavailable.
	RollbackDataModeArchiveData = "archive_data"
	// RollbackDataModeKeepData preserves migration-owned data and requires reverse steps.
	RollbackDataModeKeepData = "keep_data"
	// RollbackDataModePurge purges migration-owned data when reverse steps are unavailable.
	RollbackDataModePurge = "purge"
)

type migrationState struct {
	StepsPlanned       int
	StepsCompleted     int
	StepPlan           []migrationPlanStep
	LastStep           int
	Verdict            string
	LastError          string
	JournalPath        string
	ReconciliationPath string
	ExecutorReady      bool
	RequiresDrain      bool
	DrainReady         bool
	DrainTimeout       time.Duration
	DrainBlockedAt     time.Time
	MaxRuntime         time.Duration
	CheckpointEvery    int
	PendingRecords     map[string]string
}

type migrationPlanStep struct {
	Number        int
	FromVersion   string
	ToVersion     string
	ScriptName    string
	Compatibility string
	DrainPolicy   string
	RequiresDrain bool
}

func (r *Runtime) SetMigrationDryRunGateEnabled(enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.migrationDryRunGateEnabled = enabled
}
func (r *Runtime) GetMigrationStatus(name string) (MigrationStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pkg, ok := r.packages[name]
	if !ok {
		return MigrationStatus{}, ErrPackageNotFound
	}
	state := r.migrations[name]
	return statusFromState(pkg, state), nil
}

// RetryMigration retries an app migration run.
func (r *Runtime) RetryMigration(name string) (MigrationStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pkg, state, err := r.requireMigrationStateLocked(name)
	if err != nil {
		return MigrationStatus{}, err
	}
	if !state.ExecutorReady {
		return statusFromState(pkg, state), nil
	}
	if state.Verdict == "reconcile_pending" || len(state.PendingRecords) > 0 {
		state.Verdict = "reconcile_pending"
		state.LastError = ErrMigrationReconcilePending.Error()
		r.migrations[name] = state
		appendMigrationJournalEntry(pkg, state, "retry_blocked_reconcile_pending", nil)
		return statusFromState(pkg, state), ErrMigrationReconcilePending
	}
	if state.RequiresDrain && !state.DrainReady {
		now := time.Now().UTC()
		if state.DrainBlockedAt.IsZero() {
			state.DrainBlockedAt = now
		}
		timeout := state.DrainTimeout
		if timeout <= 0 {
			timeout = defaultMigrationDrainTimeout
		}
		if now.Sub(state.DrainBlockedAt) < timeout {
			state.Verdict = "drain_pending"
			state.LastError = ErrMigrationDrainPending.Error()
			r.migrations[name] = state
			appendMigrationJournalEntry(pkg, state, "retry_blocked_drain_pending", map[string]any{
				"timeout_seconds": int(timeout.Seconds()),
				"blocked_since":   state.DrainBlockedAt.Format(time.RFC3339Nano),
			})
			return statusFromState(pkg, state), ErrMigrationDrainPending
		}
		state.Verdict = "aborted"
		state.LastError = ErrMigrationDrainTimeout.Error()
		r.migrations[name] = state
		appendMigrationJournalEntry(pkg, state, "retry_blocked_drain_timeout", map[string]any{
			"timeout_seconds": int(timeout.Seconds()),
			"blocked_since":   state.DrainBlockedAt.Format(time.RFC3339Nano),
		})
		return statusFromState(pkg, state), ErrMigrationDrainTimeout
	}

	nextStep := state.StepsCompleted + 1
	if nextStep < 1 {
		nextStep = 1
	}
	state.RequiresDrain = migrationPlanRequiresDrainFromStep(state.StepPlan, nextStep)
	if !state.RequiresDrain {
		state.DrainReady = true
	}

	state.Verdict = "running"
	state.LastError = ""
	state.DrainBlockedAt = time.Time{}
	retryStartedAt := time.Now()
	appendMigrationJournalEntry(pkg, state, "retry_started", map[string]any{"from_step": nextStep})
	state, err = r.maybeInterruptMigrationLocked(name, state, "retry_started")
	if err != nil {
		return statusFromState(pkg, state), err
	}
	if timedOut, timeoutStatus := r.maybeFailMigrationRuntimeTimeoutLocked(name, pkg, state, 0, retryStartedAt); timedOut {
		return timeoutStatus, ErrMigrationRuntimeTimeout
	}
	for _, step := range migrationPlanPendingSteps(state.StepPlan, nextStep) {
		stepStartedAt := time.Now()
		stepPath := filepath.Join(pkg.RootPath, "migrate", step.ScriptName)
		if _, statErr := os.Stat(stepPath); statErr != nil {
			state.Verdict = "step_failed"
			state.LastStep = step.Number
			state.LastError = fmt.Sprintf("step %d script unavailable: %s", step.Number, step.ScriptName)
			r.migrations[name] = state
			appendMigrationJournalEntry(pkg, state, "step_failed_unavailable", map[string]any{
				"step_id":      step.Number,
				"from_version": step.FromVersion,
				"to_version":   step.ToVersion,
				"script":       step.ScriptName,
				"error":        statErr.Error(),
			})
			return statusFromState(pkg, state), fmt.Errorf("%w: step %d script %s: %v", ErrMigrationStepUnavailable, step.Number, step.ScriptName, statErr)
		}
		scriptSource, readErr := os.ReadFile(stepPath)
		if readErr != nil {
			state.Verdict = "step_failed"
			state.LastStep = step.Number
			state.LastError = fmt.Sprintf("step %d script unavailable: %s", step.Number, step.ScriptName)
			r.migrations[name] = state
			appendMigrationJournalEntry(pkg, state, "step_failed_unavailable", map[string]any{
				"step_id":      step.Number,
				"from_version": step.FromVersion,
				"to_version":   step.ToVersion,
				"script":       step.ScriptName,
				"error":        readErr.Error(),
			})
			return statusFromState(pkg, state), fmt.Errorf("%w: step %d script %s: %v", ErrMigrationStepUnavailable, step.Number, step.ScriptName, readErr)
		}
		if scriptErr := validateRuntimeMigrationScript(scriptSource); scriptErr != nil {
			state.Verdict = "step_failed"
			state.LastStep = step.Number
			state.LastError = fmt.Sprintf("step %d script invalid: %s", step.Number, step.ScriptName)
			r.migrations[name] = state
			appendMigrationJournalEntry(pkg, state, "step_failed_invalid_script", map[string]any{
				"step_id":      step.Number,
				"from_version": step.FromVersion,
				"to_version":   step.ToVersion,
				"script":       step.ScriptName,
				"error":        scriptErr.Error(),
			})
			return statusFromState(pkg, state), fmt.Errorf("%w: step %d script %s: %v", ErrMigrationStepInvalid, step.Number, step.ScriptName, scriptErr)
		}
		hostEffects, hostErr := collectRuntimeMigrationHostEffects(pkg, scriptSource)
		if hostErr != nil {
			state.Verdict = "step_failed"
			state.LastStep = step.Number
			r.migrations[name] = state
			if errors.Is(hostErr, ErrMigrationResourceLimit) {
				state.LastError = fmt.Sprintf("step %d resource limit exceeded: %s", step.Number, step.ScriptName)
				r.migrations[name] = state
				appendMigrationJournalEntry(pkg, state, "step_failed_resource_limit", map[string]any{
					"step_id":      step.Number,
					"from_version": step.FromVersion,
					"to_version":   step.ToVersion,
					"script":       step.ScriptName,
					"error":        hostErr.Error(),
				})
				return statusFromState(pkg, state), hostErr
			}
			state.LastError = fmt.Sprintf("step %d host effect rejected: %s", step.Number, step.ScriptName)
			r.migrations[name] = state
			appendMigrationJournalEntry(pkg, state, "step_failed_host_rejected", map[string]any{
				"step_id":      step.Number,
				"from_version": step.FromVersion,
				"to_version":   step.ToVersion,
				"script":       step.ScriptName,
				"error":        hostErr.Error(),
			})
			return statusFromState(pkg, state), hostErr
		}
		state.LastStep = step.Number
		appendMigrationJournalEntry(pkg, state, "step_started", map[string]any{
			"step_id":      step.Number,
			"from_version": step.FromVersion,
			"to_version":   step.ToVersion,
			"script":       step.ScriptName,
		})
		state, err = r.maybeInterruptMigrationLocked(name, state, "step_started")
		if err != nil {
			return statusFromState(pkg, state), err
		}
		for _, effect := range hostEffects.ArtifactPatches {
			appendMigrationJournalEntry(pkg, state, "artifact_patch_planned", map[string]any{
				"step_id":         step.Number,
				"from_version":    step.FromVersion,
				"to_version":      step.ToVersion,
				"script":          step.ScriptName,
				"artifact_id":     effect.ArtifactID,
				"owner_app_id":    effect.OwnerAppID,
				"effect_sequence": effect.Sequence,
			})
		}
		if timedOut, timeoutStatus := r.maybeFailMigrationRuntimeTimeoutLocked(name, pkg, state, step.Number, stepStartedAt); timedOut {
			return timeoutStatus, ErrMigrationRuntimeTimeout
		}
		stats, fixtureErr := verifyMigrationFixtureStep(pkg.RootPath, step, scriptSource)
		if fixtureErr != nil {
			state.Verdict = "step_failed"
			state.LastStep = step.Number
			switch {
			case errors.Is(fixtureErr, ErrMigrationAborted):
				state.LastError = fixtureErr.Error()
				r.migrations[name] = state
				appendMigrationJournalEntry(pkg, state, "step_failed_aborted", map[string]any{
					"step_id":      step.Number,
					"from_version": step.FromVersion,
					"to_version":   step.ToVersion,
					"script":       step.ScriptName,
					"error":        fixtureErr.Error(),
				})
				return statusFromState(pkg, state), fixtureErr
			case errors.Is(fixtureErr, ErrMigrationResourceLimit):
				state.LastError = fmt.Sprintf("step %d resource limit exceeded: %s", step.Number, step.ScriptName)
				r.migrations[name] = state
				appendMigrationJournalEntry(pkg, state, "step_failed_resource_limit", map[string]any{
					"step_id":      step.Number,
					"from_version": step.FromVersion,
					"to_version":   step.ToVersion,
					"script":       step.ScriptName,
					"error":        fixtureErr.Error(),
				})
				return statusFromState(pkg, state), fixtureErr
			case errors.Is(fixtureErr, ErrMigrationFixtureUnavailable):
				state.LastError = fmt.Sprintf("step %d fixture unavailable: %s", step.Number, step.ScriptName)
				r.migrations[name] = state
				appendMigrationJournalEntry(pkg, state, "step_failed_fixture_unavailable", map[string]any{
					"step_id":      step.Number,
					"from_version": step.FromVersion,
					"to_version":   step.ToVersion,
					"script":       step.ScriptName,
					"error":        fixtureErr.Error(),
				})
				return statusFromState(pkg, state), fixtureErr
			default:
				state.LastError = fmt.Sprintf("step %d fixture mismatch: %s", step.Number, step.ScriptName)
				r.migrations[name] = state
				appendMigrationJournalEntry(pkg, state, "step_failed_fixture_mismatch", map[string]any{
					"step_id":      step.Number,
					"from_version": step.FromVersion,
					"to_version":   step.ToVersion,
					"script":       step.ScriptName,
					"error":        fixtureErr.Error(),
				})
				return statusFromState(pkg, state), fixtureErr
			}
		}
		for _, logEntry := range stats.Logs {
			appendMigrationJournalEntry(pkg, state, "migration_log", map[string]any{
				"step_id":      step.Number,
				"from_version": step.FromVersion,
				"to_version":   step.ToVersion,
				"script":       step.ScriptName,
				"level":        logEntry.Level,
				"message":      logEntry.Message,
				"arguments":    logEntry.Arguments,
			})
		}
		state, err = r.appendMigrationCheckpointEntriesLocked(name, pkg, state, step, stats.StoreOps)
		if err != nil {
			return statusFromState(pkg, state), err
		}
		state.StepsCompleted = step.Number
		state.LastStep = step.Number
		appendMigrationJournalEntry(pkg, state, "step_committed", map[string]any{
			"step_id":      step.Number,
			"from_version": step.FromVersion,
			"to_version":   step.ToVersion,
			"script":       step.ScriptName,
		})
		state, err = r.maybeInterruptMigrationLocked(name, state, "step_committed")
		if err != nil {
			return statusFromState(pkg, state), err
		}
		if timedOut, timeoutStatus := r.maybeFailMigrationRuntimeTimeoutLocked(name, pkg, state, step.Number, stepStartedAt); timedOut {
			return timeoutStatus, ErrMigrationRuntimeTimeout
		}
	}
	if state.StepsCompleted > state.StepsPlanned {
		state.StepsCompleted = state.StepsPlanned
	}
	state.LastStep = state.StepsCompleted
	state.Verdict = "ok"
	state.RequiresDrain = migrationPlanRequiresDrainFromStep(state.StepPlan, state.StepsCompleted+1)
	if !state.RequiresDrain {
		state.DrainReady = true
	}
	state.JournalPath = migrationJournalPath(pkg)
	r.migrations[name] = state
	appendMigrationJournalEntry(pkg, state, "retry_committed", map[string]any{"from_step": nextStep, "to_step": state.StepsCompleted})
	return statusFromState(pkg, state), nil
}

func (r *Runtime) maybeFailMigrationRuntimeTimeoutLocked(name string, pkg Package, state migrationState, step int, startedAt time.Time) (bool, MigrationStatus) {
	if state.MaxRuntime <= 0 || time.Since(startedAt) <= state.MaxRuntime {
		return false, MigrationStatus{}
	}
	if step <= 0 {
		step = state.LastStep
	}
	state.Verdict = "step_failed"
	state.LastStep = step
	state.LastError = ErrMigrationRuntimeTimeout.Error()
	r.migrations[name] = state
	appendMigrationJournalEntry(pkg, state, "step_failed_timeout", map[string]any{
		"step_id":             step,
		"max_runtime_seconds": int(state.MaxRuntime.Seconds()),
		"error":               ErrMigrationRuntimeTimeout.Error(),
	})
	return true, statusFromState(pkg, state)
}

func (r *Runtime) appendMigrationCheckpointEntriesLocked(name string, pkg Package, state migrationState, step migrationPlanStep, effectCount int) (migrationState, error) {
	if state.CheckpointEvery <= 0 || effectCount <= 0 {
		return state, nil
	}
	for effectSequence := state.CheckpointEvery; effectSequence <= effectCount; effectSequence += state.CheckpointEvery {
		appendMigrationJournalEntry(pkg, state, "checkpoint_committed", map[string]any{
			"step_id":          step.Number,
			"from_version":     step.FromVersion,
			"to_version":       step.ToVersion,
			"script":           step.ScriptName,
			"effect_sequence":  effectSequence,
			"checkpoint_every": state.CheckpointEvery,
		})
		var err error
		state, err = r.maybeInterruptMigrationLocked(name, state, "checkpoint_committed")
		if err != nil {
			return state, err
		}
	}
	return state, nil
}

func (r *Runtime) maybeInterruptMigrationLocked(name string, state migrationState, event string) (migrationState, error) {
	if r.migrationHook == nil {
		return state, nil
	}
	if err := r.migrationHook(event, state.LastStep); err != nil {
		state.Verdict = "running"
		r.migrations[name] = state
		return state, fmt.Errorf("%w: %v", ErrMigrationInterrupted, err)
	}
	return state, nil
}

// SetMigrationDrainReady updates whether incompatible migration steps are safe to execute.
func (r *Runtime) SetMigrationDrainReady(name string, ready bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	pkg, state, err := r.requireMigrationStateLocked(name)
	if err != nil {
		return err
	}
	state.DrainReady = ready
	if ready {
		state.DrainBlockedAt = time.Time{}
		if state.Verdict == "aborted" && state.LastError == ErrMigrationDrainTimeout.Error() {
			state.Verdict = "idle"
			state.LastError = ""
		}
		if state.Verdict == "drain_pending" && state.LastError == ErrMigrationDrainPending.Error() {
			state.Verdict = "idle"
			state.LastError = ""
		}
	} else {
		state.DrainBlockedAt = time.Time{}
	}
	r.migrations[name] = state
	appendMigrationJournalEntry(pkg, state, "drain_ready_changed", map[string]any{"ready": ready})
	return nil
}

// DryRunMigrationJournalReplay runs crash-injection replay checks at each migration journal boundary.
//
// The harness executes each boundary in isolation by copying the package tree,
// injecting one interruption (`retry_started`, `step_started`,
// `checkpoint_committed`, `step_committed`), reloading runtime state from
// journal, and asserting retry reaches `verdict = ok`.
func (r *Runtime) DryRunMigrationJournalReplay(name string) ([]MigrationDryRunResult, error) {
	r.mu.RLock()
	pkg, ok := r.packages[name]
	r.mu.RUnlock()
	if !ok {
		return nil, ErrPackageNotFound
	}

	return r.dryRunMigrationJournalReplayForPackage(pkg, name)
}

func (r *Runtime) dryRunMigrationJournalReplayForPackage(pkg Package, name string) ([]MigrationDryRunResult, error) {
	r.mu.RLock()
	kernelAPIVersion := r.kernelAPIVersion
	r.mu.RUnlock()

	_, plan, err := loadMigrationPlan(pkg.RootPath)
	if err != nil {
		return nil, err
	}
	if len(plan) == 0 {
		return []MigrationDryRunResult{}, nil
	}
	if err := validateMigrationDryRunPlan(pkg.RootPath, plan); err != nil {
		return nil, err
	}

	boundaries, err := migrationJournalDryRunBoundaries(pkg.RootPath, plan)
	if err != nil {
		return nil, err
	}
	results := make([]MigrationDryRunResult, 0, len(boundaries))
	for _, boundary := range boundaries {
		result, runErr := dryRunMigrationBoundary(pkg, kernelAPIVersion, name, boundary)
		if runErr != nil {
			return nil, runErr
		}
		results = append(results, result)
	}

	return results, nil
}

func validateMigrationDryRunPlan(root string, plan []migrationPlanStep) error {
	for _, step := range plan {
		if strings.EqualFold(strings.TrimSpace(step.DrainPolicy), "drain") &&
			!strings.EqualFold(strings.TrimSpace(step.Compatibility), "incompatible") {
			return fmt.Errorf("%w: migrate.step %04d declares drain_policy=drain without compatibility=incompatible", ErrInvalidManifest, step.Number)
		}
		if strings.EqualFold(strings.TrimSpace(step.DrainPolicy), "multi_version") {
			fixture, err := findRuntimeMigrationFixture(root, step)
			if err != nil {
				return err
			}
			if fixture == nil || strings.TrimSpace(fixture.ReadAdapterPath) == "" {
				return fmt.Errorf("%w: migrate.step %04d declares drain_policy=multi_version without migrate.fixture read_adapter", ErrInvalidManifest, step.Number)
			}
		}
	}
	return nil
}

func dryRunMigrationBoundary(pkg Package, kernelAPIVersion string, name string, boundary MigrationDryRunBoundary) (MigrationDryRunResult, error) {
	workRoot, err := os.MkdirTemp("", "migration-dryrun-*")
	if err != nil {
		return MigrationDryRunResult{}, fmt.Errorf("create migration dry-run root: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(workRoot)
	}()

	copyRoot := filepath.Join(workRoot, "pkg")
	if err := copyDirTree(pkg.RootPath, copyRoot); err != nil {
		return MigrationDryRunResult{}, fmt.Errorf("copy migration dry-run package: %w", err)
	}

	isolated := NewRuntimeWithKernelAPI(kernelAPIVersion)
	isolated.skipDryRunGate = true
	loadedPkg, err := isolated.LoadPackage(context.Background(), copyRoot)
	if err != nil {
		return MigrationDryRunResult{}, fmt.Errorf("load dry-run package: %w", err)
	}
	resolvedName := loadedPkg.Manifest.Name
	if strings.TrimSpace(name) != "" {
		resolvedName = name
	}
	if err := isolated.SetMigrationDrainReady(resolvedName, true); err != nil {
		return MigrationDryRunResult{}, fmt.Errorf("set dry-run drain readiness: %w", err)
	}

	crashInjected := false
	isolated.migrationHook = func(event string, step int) error {
		if crashInjected {
			return nil
		}
		if event == boundary.Event && step == boundary.Step {
			crashInjected = true
			return errors.New("injected dry-run crash")
		}
		return nil
	}

	interrupted, err := isolated.RetryMigration(resolvedName)
	if !errors.Is(err, ErrMigrationInterrupted) {
		return MigrationDryRunResult{}, fmt.Errorf("dry-run boundary %s/%d did not interrupt: %w", boundary.Event, boundary.Step, err)
	}

	restarted := NewRuntimeWithKernelAPI(kernelAPIVersion)
	restarted.skipDryRunGate = true
	if _, err := restarted.LoadPackage(context.Background(), copyRoot); err != nil {
		return MigrationDryRunResult{}, fmt.Errorf("reload dry-run package: %w", err)
	}
	if err := restarted.SetMigrationDrainReady(resolvedName, true); err != nil {
		return MigrationDryRunResult{}, fmt.Errorf("set replay drain readiness: %w", err)
	}

	replay, err := restarted.GetMigrationStatus(resolvedName)
	if err != nil {
		return MigrationDryRunResult{}, fmt.Errorf("read replay status: %w", err)
	}
	if replay.Verdict != "step_failed" {
		return MigrationDryRunResult{}, fmt.Errorf("dry-run replay verdict = %q, want step_failed", replay.Verdict)
	}

	final, err := restarted.RetryMigration(resolvedName)
	if err != nil {
		return MigrationDryRunResult{}, fmt.Errorf("resume replayed migration: %w", err)
	}
	if final.Verdict != "ok" {
		return MigrationDryRunResult{}, fmt.Errorf("dry-run final verdict = %q, want ok", final.Verdict)
	}

	return MigrationDryRunResult{
		Boundary:    boundary,
		Replay:      replay,
		Final:       final,
		Interrupted: interrupted,
	}, nil
}

func migrationJournalDryRunBoundaries(root string, plan []migrationPlanStep) ([]MigrationDryRunBoundary, error) {
	boundaries := make([]MigrationDryRunBoundary, 0, 1+(len(plan)*2))
	boundaries = append(boundaries, MigrationDryRunBoundary{Event: "retry_started", Step: 0})
	checkpointEvery := packageMigrationCheckpointEvery(root)
	for _, step := range plan {
		boundaries = append(boundaries, MigrationDryRunBoundary{Event: "step_started", Step: step.Number})
		if checkpointEvery > 0 {
			emitsCheckpoint, err := migrationStepEmitsCheckpoint(root, step, checkpointEvery)
			if err != nil {
				return nil, err
			}
			if emitsCheckpoint {
				boundaries = append(boundaries, MigrationDryRunBoundary{Event: "checkpoint_committed", Step: step.Number})
			}
		}
		boundaries = append(boundaries, MigrationDryRunBoundary{Event: "step_committed", Step: step.Number})
	}
	return boundaries, nil
}

func migrationStepEmitsCheckpoint(root string, step migrationPlanStep, checkpointEvery int) (bool, error) {
	fixture, err := findRuntimeMigrationFixture(root, step)
	if err != nil {
		return false, err
	}
	if fixture == nil {
		return false, nil
	}
	scriptSource, err := os.ReadFile(filepath.Join(root, "migrate", step.ScriptName))
	if err != nil {
		return false, fmt.Errorf("%w: step %d script %s: %v", ErrMigrationStepUnavailable, step.Number, step.ScriptName, err)
	}
	seedRecords, err := readRuntimeFixtureRecords(root, fixture.SeedPath)
	if err != nil {
		return false, err
	}
	_, stats, err := executeRuntimeMigrationFixture(scriptSource, seedRecords)
	if err != nil {
		return false, err
	}
	return stats.StoreOps >= checkpointEvery, nil
}

func (r *Runtime) AbortMigration(name, target string) (MigrationStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pkg, state, err := r.requireMigrationStateLocked(name)
	if err != nil {
		return MigrationStatus{}, err
	}
	target = strings.TrimSpace(target)
	if target == "" {
		target = MigrationAbortToCheckpoint
	}
	if target != MigrationAbortToCheckpoint && target != MigrationAbortToBaseline {
		return statusFromState(pkg, state), fmt.Errorf("%w: %s", ErrMigrationAbortTargetInvalid, target)
	}
	if state.Verdict == "reconcile_pending" || len(state.PendingRecords) > 0 {
		state.Verdict = "reconcile_pending"
		state.LastError = ErrMigrationReconcilePending.Error()
		r.migrations[name] = state
		appendMigrationJournalEntry(pkg, state, "abort_blocked_reconcile_pending", map[string]any{"target": target})
		return statusFromState(pkg, state), ErrMigrationReconcilePending
	}
	if !state.ExecutorReady {
		return statusFromState(pkg, state), nil
	}

	state.Verdict = "aborted"
	state.LastError = "aborted by operator"
	if target == MigrationAbortToBaseline {
		pending := migrationArtifactInverseFailuresFromJournal(pkg, state)
		state.StepsCompleted = 0
		state.LastStep = 0
		if len(pending) > 0 {
			state.Verdict = "reconcile_pending"
			state.LastError = ErrMigrationReconcilePending.Error()
			state.PendingRecords = pending
			state.ReconciliationPath = migrationReconciliationPath(pkg)
			r.migrations[name] = state
			appendMigrationJournalEntry(pkg, state, "reconcile_pending", map[string]any{
				"target":          target,
				"pending_records": pending,
			})
			return statusFromState(pkg, state), ErrMigrationReconcilePending
		}
		state.LastError = "aborted to baseline by operator"
	} else {
		failedStep := state.LastStep
		if failedStep < 1 {
			failedStep = state.StepsCompleted + 1
		}
		if failedStep <= state.StepsCompleted && state.StepsCompleted > 0 {
			state.StepsCompleted--
		}
		if state.StepsCompleted < 0 {
			state.StepsCompleted = 0
		}
		state.LastStep = failedStep
		state.Verdict = "step_failed"
		state.LastError = fmt.Sprintf("step %d aborted by operator", failedStep)
	}
	state.PendingRecords = nil
	state.ReconciliationPath = ""
	r.migrations[name] = state
	appendMigrationJournalEntry(pkg, state, "aborted", map[string]any{"target": target})
	return statusFromState(pkg, state), nil
}

// ReconcileMigration attempts to reconcile one migration record.
func (r *Runtime) ReconcileMigration(name, recordID, resolution string) (MigrationStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pkg, state, err := r.requireMigrationStateLocked(name)
	if err != nil {
		return MigrationStatus{}, err
	}
	if !isAllowedMigrationResolution(resolution) {
		return statusFromState(pkg, state), fmt.Errorf("%w: %s", ErrMigrationResolutionInvalid, resolution)
	}
	if len(state.PendingRecords) == 0 {
		return statusFromState(pkg, state), ErrMigrationReconcilePending
	}
	if _, ok := state.PendingRecords[recordID]; !ok {
		return statusFromState(pkg, state), fmt.Errorf("%w: %s", ErrMigrationRecordNotFound, recordID)
	}

	delete(state.PendingRecords, recordID)
	if len(state.PendingRecords) == 0 {
		state.Verdict = "ok"
		state.LastError = ""
		state.ReconciliationPath = ""
	} else {
		state.Verdict = "reconcile_pending"
		state.LastError = ErrMigrationReconcilePending.Error()
	}
	r.migrations[name] = state
	appendMigrationJournalEntry(pkg, state, "reconcile_record", map[string]any{
		"record_id":  recordID,
		"resolution": resolution,
	})
	return statusFromState(pkg, state), nil
}

func (r *Runtime) requireMigrationStateLocked(name string) (Package, migrationState, error) {
	pkg, ok := r.packages[name]
	if !ok {
		return Package{}, migrationState{}, ErrPackageNotFound
	}
	state, ok := r.migrations[name]
	if !ok {
		state = newMigrationState(pkg, "")
		r.migrations[name] = state
	}
	return pkg, state, nil
}

func newMigrationState(pkg Package, installedVersion string) migrationState {
	steps, plan, planErr := loadMigrationPlan(pkg.RootPath)
	stepsCompleted := 0
	if planErr == nil {
		if completed, completedErr := migrationStepsCompletedForInstalledVersion(plan, installedVersion, pkg.Manifest.Version); completedErr != nil {
			planErr = completedErr
		} else {
			stepsCompleted = completed
		}
	}
	nextStep := stepsCompleted + 1
	requiresDrain := migrationPlanRequiresDrainFromStep(plan, nextStep)
	verdict := "idle"
	lastStep := stepsCompleted
	if steps > 0 && stepsCompleted >= steps {
		verdict = "ok"
	}
	state := migrationState{
		StepsPlanned:    steps,
		StepsCompleted:  stepsCompleted,
		StepPlan:        plan,
		LastStep:        lastStep,
		Verdict:         verdict,
		ExecutorReady:   steps > 0 && planErr == nil,
		RequiresDrain:   requiresDrain,
		DrainReady:      !requiresDrain,
		DrainTimeout:    packageDrainTimeout(rootOrFallbackPath(pkg)),
		MaxRuntime:      packageMigrationMaxRuntime(rootOrFallbackPath(pkg)),
		CheckpointEvery: packageMigrationCheckpointEvery(rootOrFallbackPath(pkg)),
	}
	if planErr != nil {
		state.LastError = planErr.Error()
	}
	if steps > 0 {
		state.JournalPath = migrationJournalPath(pkg)
		state = replayMigrationStateFromJournal(pkg, state)
		state.RequiresDrain = migrationPlanRequiresDrainFromStep(state.StepPlan, state.StepsCompleted+1)
		if !state.RequiresDrain {
			state.DrainReady = true
		}
	}
	return state
}

func migrationStepsCompletedForInstalledVersion(plan []migrationPlanStep, installedVersion string, targetVersion string) (int, error) {
	fromVersion := strings.TrimSpace(installedVersion)
	if fromVersion == "" || len(plan) == 0 {
		return 0, nil
	}
	if fromVersion == strings.TrimSpace(targetVersion) {
		return len(plan), nil
	}
	for _, step := range plan {
		if step.FromVersion == fromVersion {
			if step.Number <= 1 {
				return 0, nil
			}
			return step.Number - 1, nil
		}
	}
	for _, step := range plan {
		if step.ToVersion == fromVersion {
			return step.Number, nil
		}
	}
	return 0, fmt.Errorf("migration plan does not include installed version %q", fromVersion)
}
