package appruntime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func (r *Runtime) retryMigrationBlockedByReconcile(name string, pkg Package, state migrationState) (migrationState, MigrationStatus, bool, error) {
	if state.Verdict != "reconcile_pending" && len(state.PendingRecords) == 0 {
		return state, MigrationStatus{}, false, nil
	}
	state.Verdict = "reconcile_pending"
	state.LastError = ErrMigrationReconcilePending.Error()
	r.migrations[name] = state
	appendMigrationJournalEntry(pkg, state, "retry_blocked_reconcile_pending", nil)
	return state, statusFromState(pkg, state), true, ErrMigrationReconcilePending
}

func (r *Runtime) retryMigrationDrainGate(name string, pkg Package, state migrationState) (migrationState, MigrationStatus, bool, error) {
	if !state.RequiresDrain || state.DrainReady {
		return state, MigrationStatus{}, false, nil
	}
	now := time.Now().UTC()
	if state.DrainBlockedAt.IsZero() {
		state.DrainBlockedAt = now
	}
	timeout := state.DrainTimeout
	if timeout <= 0 {
		timeout = defaultMigrationDrainTimeout
	}
	journalFields := map[string]any{
		"timeout_seconds": int(timeout.Seconds()),
		"blocked_since":   state.DrainBlockedAt.Format(time.RFC3339Nano),
	}
	if now.Sub(state.DrainBlockedAt) < timeout {
		state.Verdict = "drain_pending"
		state.LastError = ErrMigrationDrainPending.Error()
		r.migrations[name] = state
		appendMigrationJournalEntry(pkg, state, "retry_blocked_drain_pending", journalFields)
		return state, statusFromState(pkg, state), true, ErrMigrationDrainPending
	}
	state.Verdict = "aborted"
	state.LastError = ErrMigrationDrainTimeout.Error()
	r.migrations[name] = state
	appendMigrationJournalEntry(pkg, state, "retry_blocked_drain_timeout", journalFields)
	return state, statusFromState(pkg, state), true, ErrMigrationDrainTimeout
}

func (r *Runtime) retryMigrationPrepareState(name string, state migrationState) (migrationState, int) {
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
	r.migrations[name] = state
	return state, nextStep
}

func (r *Runtime) retryMigrationFailStepUnavailable(
	name string,
	pkg Package,
	state migrationState,
	step migrationPlanStep,
	scriptErr error,
) (migrationState, MigrationStatus, error) {
	state.Verdict = "step_failed"
	state.LastStep = step.Number
	state.LastError = fmt.Sprintf("step %d script unavailable: %s", step.Number, step.ScriptName)
	r.migrations[name] = state
	appendMigrationJournalEntry(pkg, state, "step_failed_unavailable", map[string]any{
		"step_id":      step.Number,
		"from_version": step.FromVersion,
		"to_version":   step.ToVersion,
		"script":       step.ScriptName,
		"error":        scriptErr.Error(),
	})
	return state, statusFromState(pkg, state), fmt.Errorf("%w: step %d script %s: %v", ErrMigrationStepUnavailable, step.Number, step.ScriptName, scriptErr)
}

func (r *Runtime) retryMigrationFailStepInvalid(
	name string,
	pkg Package,
	state migrationState,
	step migrationPlanStep,
	scriptErr error,
) (migrationState, MigrationStatus, error) {
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
	return state, statusFromState(pkg, state), fmt.Errorf("%w: step %d script %s: %v", ErrMigrationStepInvalid, step.Number, step.ScriptName, scriptErr)
}

func (r *Runtime) retryMigrationFailHostEffects(
	name string,
	pkg Package,
	state migrationState,
	step migrationPlanStep,
	hostErr error,
) (migrationState, MigrationStatus, error) {
	state.Verdict = "step_failed"
	state.LastStep = step.Number
	if errors.Is(hostErr, ErrMigrationResourceLimit) {
		state.LastError = fmt.Sprintf("step %d resource limit exceeded: %s", step.Number, step.ScriptName)
		r.migrations[name] = state
		appendMigrationJournalEntry(pkg, state, "step_failed_resource_limit", map[string]any{
			"step_id": step.Number, "from_version": step.FromVersion, "to_version": step.ToVersion,
			"script": step.ScriptName, "error": hostErr.Error(),
		})
		return state, statusFromState(pkg, state), hostErr
	}
	state.LastError = fmt.Sprintf("step %d host effect rejected: %s", step.Number, step.ScriptName)
	r.migrations[name] = state
	appendMigrationJournalEntry(pkg, state, "step_failed_host_rejected", map[string]any{
		"step_id": step.Number, "from_version": step.FromVersion, "to_version": step.ToVersion,
		"script": step.ScriptName, "error": hostErr.Error(),
	})
	return state, statusFromState(pkg, state), hostErr
}

func (r *Runtime) retryMigrationHandleFixtureError(
	name string,
	pkg Package,
	state migrationState,
	step migrationPlanStep,
	fixtureErr error,
) (migrationState, MigrationStatus, bool, error) {
	state.Verdict = "step_failed"
	state.LastStep = step.Number
	baseFields := map[string]any{
		"step_id": step.Number, "from_version": step.FromVersion,
		"to_version": step.ToVersion, "script": step.ScriptName, "error": fixtureErr.Error(),
	}
	switch {
	case errors.Is(fixtureErr, ErrMigrationAborted):
		state.LastError = fixtureErr.Error()
		r.migrations[name] = state
		appendMigrationJournalEntry(pkg, state, "step_failed_aborted", baseFields)
		return state, statusFromState(pkg, state), true, fixtureErr
	case errors.Is(fixtureErr, ErrMigrationResourceLimit):
		state.LastError = fmt.Sprintf("step %d resource limit exceeded: %s", step.Number, step.ScriptName)
		r.migrations[name] = state
		appendMigrationJournalEntry(pkg, state, "step_failed_resource_limit", baseFields)
		return state, statusFromState(pkg, state), true, fixtureErr
	case errors.Is(fixtureErr, ErrMigrationFixtureUnavailable):
		state.LastError = fmt.Sprintf("step %d fixture unavailable: %s", step.Number, step.ScriptName)
		r.migrations[name] = state
		appendMigrationJournalEntry(pkg, state, "step_failed_fixture_unavailable", baseFields)
		return state, statusFromState(pkg, state), true, fixtureErr
	default:
		state.LastError = fmt.Sprintf("step %d fixture mismatch: %s", step.Number, step.ScriptName)
		r.migrations[name] = state
		appendMigrationJournalEntry(pkg, state, "step_failed_fixture_mismatch", baseFields)
		return state, statusFromState(pkg, state), true, fixtureErr
	}
}

func (r *Runtime) retryMigrationRunStep(
	name string,
	pkg Package,
	state migrationState,
	step migrationPlanStep,
) (migrationState, MigrationStatus, bool, error) {
	stepStartedAt := time.Now()
	stepPath := filepath.Join(pkg.RootPath, "migrate", step.ScriptName)
	if _, statErr := os.Stat(stepPath); statErr != nil {
		state, status, err := r.retryMigrationFailStepUnavailable(name, pkg, state, step, statErr)
		return state, status, true, err
	}
	scriptSource, readErr := os.ReadFile(stepPath)
	if readErr != nil {
		state, status, err := r.retryMigrationFailStepUnavailable(name, pkg, state, step, readErr)
		return state, status, true, err
	}
	if scriptErr := validateRuntimeMigrationScript(scriptSource); scriptErr != nil {
		state, status, err := r.retryMigrationFailStepInvalid(name, pkg, state, step, scriptErr)
		return state, status, true, err
	}
	hostEffects, hostErr := collectRuntimeMigrationHostEffects(pkg, scriptSource)
	if hostErr != nil {
		state, status, err := r.retryMigrationFailHostEffects(name, pkg, state, step, hostErr)
		return state, status, true, err
	}
	state.LastStep = step.Number
	appendMigrationJournalEntry(pkg, state, "step_started", map[string]any{
		"step_id": step.Number, "from_version": step.FromVersion,
		"to_version": step.ToVersion, "script": step.ScriptName,
	})
	var err error
	state, err = r.maybeInterruptMigrationLocked(name, state, "step_started")
	if err != nil {
		return state, statusFromState(pkg, state), true, err
	}
	for _, effect := range hostEffects.ArtifactPatches {
		appendMigrationJournalEntry(pkg, state, "artifact_patch_planned", map[string]any{
			"step_id": step.Number, "from_version": step.FromVersion, "to_version": step.ToVersion,
			"script": step.ScriptName, "artifact_id": effect.ArtifactID,
			"owner_app_id": effect.OwnerAppID, "effect_sequence": effect.Sequence,
		})
	}
	if timedOut, timeoutStatus := r.maybeFailMigrationRuntimeTimeoutLocked(name, pkg, state, step.Number, stepStartedAt); timedOut {
		return state, timeoutStatus, true, ErrMigrationRuntimeTimeout
	}
	stats, fixtureErr := verifyMigrationFixtureStep(pkg.RootPath, step, scriptSource)
	if fixtureErr != nil {
		state, status, handled, err := r.retryMigrationHandleFixtureError(name, pkg, state, step, fixtureErr)
		return state, status, handled, err
	}
	for _, logEntry := range stats.Logs {
		appendMigrationJournalEntry(pkg, state, "migration_log", map[string]any{
			"step_id": step.Number, "from_version": step.FromVersion, "to_version": step.ToVersion,
			"script": step.ScriptName, "level": logEntry.Level, "message": logEntry.Message,
			"arguments": logEntry.Arguments,
		})
	}
	state, err = r.appendMigrationCheckpointEntriesLocked(name, pkg, state, step, stats.StoreOps)
	if err != nil {
		return state, statusFromState(pkg, state), true, err
	}
	state.StepsCompleted = step.Number
	state.LastStep = step.Number
	appendMigrationJournalEntry(pkg, state, "step_committed", map[string]any{
		"step_id": step.Number, "from_version": step.FromVersion,
		"to_version": step.ToVersion, "script": step.ScriptName,
	})
	state, err = r.maybeInterruptMigrationLocked(name, state, "step_committed")
	if err != nil {
		return state, statusFromState(pkg, state), true, err
	}
	if timedOut, timeoutStatus := r.maybeFailMigrationRuntimeTimeoutLocked(name, pkg, state, step.Number, stepStartedAt); timedOut {
		return state, timeoutStatus, true, ErrMigrationRuntimeTimeout
	}
	return state, MigrationStatus{}, false, nil
}
