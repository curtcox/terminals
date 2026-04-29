// Package appruntime loads and hot-reloads TAR/TAL application packages.
package appruntime

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	"golang.org/x/text/unicode/norm"
)

const (
	// LanguageTALV1 identifies the only supported TAL language level.
	LanguageTALV1 = "tal/1"
	// DefaultKernelAPIVersion is the runtime kernel API compatibility tag.
	DefaultKernelAPIVersion = "kernel/1"
)

var (
	// ErrInvalidManifest indicates app manifest parse or validation failure.
	ErrInvalidManifest = errors.New("invalid app manifest")
	// ErrPermissionDenied indicates an app requested undeclared capabilities.
	ErrPermissionDenied = errors.New("permission denied")
	// ErrKernelAPIIncompatible indicates the app requested an unsupported kernel API.
	ErrKernelAPIIncompatible = errors.New("kernel api incompatible")
	// ErrPackageNotFound indicates a package name is unknown to this runtime.
	ErrPackageNotFound = errors.New("app package not found")
	// ErrNoPriorVersion indicates rollback was requested but no prior version exists.
	ErrNoPriorVersion = errors.New("no prior app package version")
	// ErrMigrationReconcilePending indicates migration records must be reconciled first.
	ErrMigrationReconcilePending = errors.New("migration reconciliation is pending")
	// ErrMigrationRecordNotFound indicates a reconciliation record ID is unknown.
	ErrMigrationRecordNotFound = errors.New("migration reconciliation record not found")
	// ErrMigrationResolutionInvalid indicates a reconciliation resolution is unsupported.
	ErrMigrationResolutionInvalid = errors.New("migration reconciliation resolution is invalid")
	// ErrMigrationAbortTargetInvalid indicates an abort target is unsupported.
	ErrMigrationAbortTargetInvalid = errors.New("migration abort target is invalid")
	// ErrRollbackModeInvalid indicates rollback data mode options are invalid.
	ErrRollbackModeInvalid = errors.New("rollback data mode is invalid")
	// ErrRollbackKeepDataRequiresDowngrade indicates keep-data rollback requires downgrade steps.
	ErrRollbackKeepDataRequiresDowngrade = errors.New("rollback with keep-data requires migrate/downgrade steps")
	// ErrMigrationDrainTimeout indicates incompatible migration drain prerequisites were not satisfied.
	ErrMigrationDrainTimeout = errors.New("migration drain timeout elapsed before executor run")
	// ErrMigrationDrainPending indicates incompatible migration drain prerequisites are still in progress.
	ErrMigrationDrainPending = errors.New("migration drain is pending")
	// ErrMigrationStepUnavailable indicates a migration step script could not be read at execution time.
	ErrMigrationStepUnavailable = errors.New("migration step script unavailable")
	// ErrMigrationStepInvalid indicates a migration step script is present but invalid for execution.
	ErrMigrationStepInvalid = errors.New("migration step script invalid")
	// ErrMigrationFixtureUnavailable indicates a migration fixture file could not be read at execution time.
	ErrMigrationFixtureUnavailable = errors.New("migration fixture file unavailable")
	// ErrMigrationFixtureMismatch indicates migration fixture expected output diverged from actual output.
	ErrMigrationFixtureMismatch = errors.New("migration fixture expected output mismatch")
	// ErrMigrationInterrupted indicates a migration run was interrupted before committing.
	ErrMigrationInterrupted = errors.New("migration execution interrupted before commit")
	// ErrMigrationDryRunFailed indicates Gate 4 replay checks failed while loading a package.
	ErrMigrationDryRunFailed = errors.New("migration dry-run gate failed")
	// ErrMigrationRuntimeTimeout indicates migration execution exceeded its configured runtime budget.
	ErrMigrationRuntimeTimeout = errors.New("migration runtime timeout elapsed")
	// ErrMigrationAborted indicates migration code requested an executor abort.
	ErrMigrationAborted = errors.New("migration aborted by script")
	// ErrMigrationArtifactOwnership indicates a migration attempted to patch an artifact outside its lineage.
	ErrMigrationArtifactOwnership = errors.New("migration artifact ownership mismatch")
	// ErrMigrationResourceLimit indicates a migration exceeded an executor hard cap.
	ErrMigrationResourceLimit = errors.New("migration resource limit exceeded")
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

var migrateStepFilePattern = regexp.MustCompile(`^(\d+)_([^/]+)_to_([^/]+)\.tal$`)

var migrateLoadPattern = regexp.MustCompile(`(?m)^\s*load\(\s*["']([^"']+)["']`)

var migrateEntryPointPattern = regexp.MustCompile(`(?m)^\s*def\s+migrate\s*\(`)

var migrateReadAdapterEntryPointPattern = regexp.MustCompile(`(?m)^\s*def\s+read\s*\(\s*record\s*\)`)

var migrateRecordAssignmentPattern = regexp.MustCompile(`^\s*record\["([^"]+)"\]\s*=\s*(.+?)\s*$`)

var migrateRecordDeletePattern = regexp.MustCompile(`^\s*del\s+record\["([^"]+)"\]\s*$`)

var migrateRecordSkipIfPresentPattern = regexp.MustCompile(`^\s*if\s+["']([^"']+)["']\s+in\s+record\s*:\s*continue\s*$`)

var migrateRecordValuePattern = regexp.MustCompile(`^record\["([^"]+)"\]$`)

var migrateRecordGetValuePattern = regexp.MustCompile(`^record\.get\(\s*"([^"]+)"\s*,\s*("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*')\s*\)$`)

var migrateRecordLowerPattern = regexp.MustCompile(`^lower\(\s*record\["([^"]+)"\]\s*\)$`)

var migrateRecordLowerGetPattern = regexp.MustCompile(`^lower\(\s*record\.get\(\s*"([^"]+)"\s*,\s*("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*')\s*\)\s*\)$`)

var migrateRecordTrimPattern = regexp.MustCompile(`^trim\(\s*record\["([^"]+)"\]\s*\)$`)

var migrateRecordTrimGetPattern = regexp.MustCompile(`^trim\(\s*record\.get\(\s*"([^"]+)"\s*,\s*("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*')\s*\)\s*\)$`)

var migrateRecordLowerTrimPattern = regexp.MustCompile(`^lower\(\s*trim\(\s*record\["([^"]+)"\]\s*\)\s*\)$`)

var migrateRecordLowerTrimGetPattern = regexp.MustCompile(`^lower\(\s*trim\(\s*record\.get\(\s*"([^"]+)"\s*,\s*("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*')\s*\)\s*\)\s*\)$`)

var migrateRecordNormalizeGetPattern = regexp.MustCompile(`^_normalize\(\s*record\.get\(\s*"([^"]+)"\s*,\s*("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*')\s*\)\s*\)$`)

var migrateRecordNormalizePattern = regexp.MustCompile(`^_normalize\(\s*record\["([^"]+)"\]\s*\)$`)

var migrateAbortPattern = regexp.MustCompile(`^abort\(\s*("(?:\\.|[^"\\])*")\s*\)$`)

var migrateArtifactSelfLoadPattern = regexp.MustCompile(`(?m)^\s*load\(\s*["']artifact\.self["']\s*,(?P<args>[^)]*)\)`)

var migrateLoadAliasPattern = regexp.MustCompile(`\bpatch\s*=\s*["']([A-Za-z_][A-Za-z0-9_]*)["']`)

var migrateLogLoadPattern = regexp.MustCompile(`(?m)^\s*load\(\s*["']log["']\s*,(?P<args>[^)]*)\)`)

var migrateLogAliasPattern = regexp.MustCompile(`\b(debug|info|warn|error)\s*=\s*["']([A-Za-z_][A-Za-z0-9_]*)["']`)

var migrateStoreLoadPattern = regexp.MustCompile(`(?m)^\s*load\(\s*["']store["']\s*,(?P<args>[^)]*)\)`)

var migrateStoreListKeysAliasPattern = regexp.MustCompile(`\blist_keys\s*=\s*["']([A-Za-z_][A-Za-z0-9_]*)["']`)

var migrateStoreGetAliasPattern = regexp.MustCompile(`\bget\s*=\s*["']([A-Za-z_][A-Za-z0-9_]*)["']`)

var migrateStorePutAliasPattern = regexp.MustCompile(`\bput\s*=\s*["']([A-Za-z_][A-Za-z0-9_]*)["']`)

var migrateStoreDeleteAliasPattern = regexp.MustCompile(`\bdelete\s*=\s*["']([A-Za-z_][A-Za-z0-9_]*)["']`)

var migrateEnvLoadPattern = regexp.MustCompile(`(?m)^\s*load\(\s*["']migrate\.env["']\s*,(?P<args>[^)]*)\)`)

var migrateEnvCheckpointAliasPattern = regexp.MustCompile(`\bcheckpoint\s*=\s*["']([A-Za-z_][A-Za-z0-9_]*)["']`)

var migrateEnvAbortAliasPattern = regexp.MustCompile(`\babort\s*=\s*["']([A-Za-z_][A-Za-z0-9_]*)["']`)

var migrateCallPattern = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*\((.*)\)\s*$`)

var migrateStringArgPattern = regexp.MustCompile(`^\s*("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*')`)

var migrateOwnerAppIDPattern = regexp.MustCompile(`\bowner_app_id\s*=\s*("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*')`)

var allowedMigrationModules = map[string]struct{}{
	"store":         {},
	"artifact.self": {},
	"log":           {},
	"migrate.env":   {},
}

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

var allowedPermissions = map[string]struct{}{
	"placement.read": {},
	"claims.request": {},
	"ui.set":         {},
	"ui.patch":       {},
	"ui.clear":       {},
	"ui.transition":  {},
	"flow.apply":     {},
	"flow.patch":     {},
	"flow.stop":      {},
	"recent.pull":    {},
	"presence.read":  {},
	"world.read":     {},
	"world.verify":   {},
	"store.kv":       {},
	"store.query":    {},
	"scheduler":      {},
	"pty":            {},
	"telephony":      {},
	"ai.stt":         {},
	"ai.tts":         {},
	"ai.llm":         {},
	"http.outbound":  {},
	"bus.emit":       {},
}

// Manifest is the TAR/TAL app package descriptor.
type Manifest struct {
	Name              string
	AppID             string
	Version           string
	Language          string
	RequiresKernelAPI string
	Description       string
	Permissions       []string
	Exports           []string
	Kernels           []string
	Models            []string
	Migrate           string
	DevMode           bool
}

// AppManifest is a plan-aligned alias for Manifest.
type AppManifest = Manifest

// Package bundles one parsed app package from disk.
type Package struct {
	RootPath   string
	Manifest   Manifest
	MainPath   string
	LoadedAt   time.Time
	Revision   uint64
	FileDigest map[string]time.Time
}

// ActivationRequest describes activation startup context for a definition.
type ActivationRequest struct {
	DeviceID string
	Intent   string
	Payload  map[string]string
}

// Environment provides host services to an app activation.
type Environment struct{}

// Trigger is a typed activation event passed to Handle.
type Trigger struct {
	Kind       string
	Subject    string
	Attributes map[string]string
	OccurredAt time.Time
}

// Op is one host operation emitted from TAL result commits.
type Op struct {
	Kind   string
	Target string
	Args   map[string]string
}

// Result is the deterministic TAL handler output.
type Result struct {
	State any
	Ops   []Op
	Emit  []Trigger
	Done  bool
}

// MigrationStatus exposes app migration control-plane status.
type MigrationStatus struct {
	App                string
	Version            string
	Revision           uint64
	StepsPlanned       int
	StepsCompleted     int
	LastStep           int
	Verdict            string
	LastError          string
	JournalPath        string
	ReconciliationPath string
	ExecutorReady      bool
	PendingRecords     []MigrationReconciliationRecord
}

// MigrationDryRunBoundary identifies one interruption point in migration replay validation.
type MigrationDryRunBoundary struct {
	Event string
	Step  int
}

// MigrationDryRunResult captures replay and resumed status for one crash boundary.
type MigrationDryRunResult struct {
	Boundary    MigrationDryRunBoundary
	Replay      MigrationStatus
	Final       MigrationStatus
	Interrupted MigrationStatus
}

// MigrationReconciliationRecord describes one unresolved migration reconciliation item.
type MigrationReconciliationRecord struct {
	RecordID              string
	RecommendedResolution string
}

// RollbackOptions controls rollback data handling behavior.
type RollbackOptions struct {
	DataMode string
}

// AppDefinition provides activation matching and activation construction.
type AppDefinition interface {
	Name() string
	Match(req ActivationRequest) bool
	NewActivation(req ActivationRequest) (AppActivation, error)
}

// AppActivation is a scenario engine-supervised app lifecycle instance.
type AppActivation interface {
	ID() string
	DefinitionName() string
	Start(ctx context.Context, env *Environment) error
	Handle(ctx context.Context, env *Environment, trigger Trigger) error
	Stop(ctx context.Context, env *Environment) error
	Suspend(ctx context.Context, env *Environment) error
	Resume(ctx context.Context, env *Environment) error
}

// Runtime loads and hot-reloads app packages from disk.
type Runtime struct {
	mu                         sync.RWMutex
	kernelAPIVersion           string
	nextRevision               uint64
	packages                   map[string]Package
	history                    map[string][]Package
	migrations                 map[string]migrationState
	migrationHook              func(event string, step int) error
	migrationDryRunGateEnabled bool
	skipDryRunGate             bool
}

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

// NewRuntime returns an empty application runtime.
func NewRuntime() *Runtime {
	return NewRuntimeWithKernelAPI(DefaultKernelAPIVersion)
}

// NewRuntimeWithKernelAPI returns an empty runtime pinned to one kernel API.
func NewRuntimeWithKernelAPI(kernelAPIVersion string) *Runtime {
	if strings.TrimSpace(kernelAPIVersion) == "" {
		kernelAPIVersion = DefaultKernelAPIVersion
	}
	return &Runtime{
		kernelAPIVersion: kernelAPIVersion,
		nextRevision:     1,
		packages:         make(map[string]Package),
		history:          make(map[string][]Package),
		migrations:       make(map[string]migrationState),
	}
}

// SetMigrationDryRunGateEnabled configures whether load-time migration replay checks run as a blocking gate.
func (r *Runtime) SetMigrationDryRunGateEnabled(enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.migrationDryRunGateEnabled = enabled
}

// LoadPackage parses and registers one app package.
func (r *Runtime) LoadPackage(ctx context.Context, root string) (Package, error) {
	_ = ctx
	pkg, err := r.packageFromDisk(root)
	if err != nil {
		return Package{}, err
	}
	if err := r.runMigrationDryRunGate(pkg, pkg.Manifest.Name); err != nil {
		return Package{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	pkg.Revision = r.nextRevision
	r.nextRevision++
	r.packages[pkg.Manifest.Name] = pkg
	r.history[pkg.Manifest.Name] = append(r.history[pkg.Manifest.Name], pkg)
	r.migrations[pkg.Manifest.Name] = newMigrationState(pkg, "")
	return pkg, nil
}

func (r *Runtime) runMigrationDryRunGate(pkg Package, name string) error {
	if !r.migrationDryRunGateEnabled || r.skipDryRunGate {
		return nil
	}

	results, err := r.dryRunMigrationJournalReplayForPackage(pkg, name)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMigrationDryRunFailed, err)
	}
	if len(results) == 0 {
		return nil
	}
	return nil
}

// ReloadPackage reloads one known package if sources changed.
func (r *Runtime) ReloadPackage(_ context.Context, name string) (Package, bool, error) {
	r.mu.RLock()
	current, ok := r.packages[name]
	r.mu.RUnlock()
	if !ok {
		return Package{}, false, ErrPackageNotFound
	}

	nextDigest, err := collectDigest(current.RootPath)
	if err != nil {
		return Package{}, false, err
	}
	changed := hasDigestDiff(current.FileDigest, nextDigest)
	if !changed {
		return current, false, nil
	}

	next, err := r.packageFromDisk(current.RootPath)
	if err != nil {
		// Preserve current package; failed reload must not replace last-good.
		return current, true, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	next.Revision = r.nextRevision
	r.nextRevision++
	r.packages[name] = next
	r.history[name] = append(r.history[name], next)
	r.migrations[name] = newMigrationState(next, current.Manifest.Version)
	return next, true, nil
}

// RollbackPackage restores the previous successfully loaded package version.
func (r *Runtime) RollbackPackage(name string, options ...RollbackOptions) (Package, error) {
	rollbackOpts, err := normalizeRollbackOptions(options...)
	if err != nil {
		return Package{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	history := r.history[name]
	if len(history) == 0 {
		return Package{}, ErrPackageNotFound
	}
	if len(history) < 2 {
		return Package{}, ErrNoPriorVersion
	}
	if state, ok := r.migrations[name]; ok {
		if state.Verdict == "reconcile_pending" || len(state.PendingRecords) > 0 {
			return Package{}, ErrMigrationReconcilePending
		}
	}
	current := history[len(history)-1]
	previous := history[len(history)-2]
	hasDowngradeSteps := countDowngradeMigrationSteps(current.RootPath) > 0 || countDowngradeMigrationSteps(previous.RootPath) > 0
	if rollbackOpts.DataMode == RollbackDataModeKeepData && !hasDowngradeSteps {
		return Package{}, ErrRollbackKeepDataRequiresDowngrade
	}
	history = history[:len(history)-1]
	r.history[name] = history
	previous = history[len(history)-1]
	r.packages[name] = previous
	r.migrations[name] = newMigrationState(previous, "")
	return previous, nil
}

func normalizeRollbackOptions(options ...RollbackOptions) (RollbackOptions, error) {
	if len(options) > 1 {
		return RollbackOptions{}, ErrRollbackModeInvalid
	}
	opts := RollbackOptions{DataMode: RollbackDataModeArchiveData}
	if len(options) == 1 {
		opts = options[0]
	}
	mode := strings.ToLower(strings.TrimSpace(opts.DataMode))
	mode = strings.ReplaceAll(mode, "-", "_")
	if mode == "" {
		mode = RollbackDataModeArchiveData
	}
	switch mode {
	case RollbackDataModeArchiveData, RollbackDataModeKeepData, RollbackDataModePurge:
		opts.DataMode = mode
		return opts, nil
	default:
		return RollbackOptions{}, fmt.Errorf("%w: %s", ErrRollbackModeInvalid, opts.DataMode)
	}
}

// GetMigrationStatus returns migration status for one app package.
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
	_, state, err := r.requireMigrationStateLocked(name)
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

func copyDirTree(src, dst string) error {
	cleanSrc := filepath.Clean(src)
	cleanDst := filepath.Clean(dst)
	return filepath.WalkDir(cleanSrc, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(cleanSrc, path)
		if err != nil {
			return err
		}
		target := filepath.Join(cleanDst, rel)

		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode().Perm())
		}

		if d.Type()&fs.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(linkTarget, target)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		in, err := os.Open(filepath.Clean(path))
		if err != nil {
			return err
		}
		defer func() {
			_ = in.Close()
		}()

		out, err := os.OpenFile(filepath.Clean(target), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
		if err != nil {
			return err
		}
		defer func() {
			_ = out.Close()
		}()

		if _, err := io.Copy(out, in); err != nil {
			return err
		}
		return nil
	})
}

// AbortMigration aborts an in-flight app migration run.
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

func replayMigrationStateFromJournal(pkg Package, state migrationState) migrationState {
	if strings.TrimSpace(state.JournalPath) == "" {
		return state
	}

	absolutePath := filepath.Join(pkg.RootPath, filepath.FromSlash(state.JournalPath))
	file, err := os.Open(filepath.Clean(absolutePath))
	if err != nil {
		return state
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	lastEvent := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if event := migrationJournalString(entry["event"]); event != "" {
			lastEvent = event
		}
		if stepsCompleted, ok := migrationJournalInt(entry["steps_completed"]); ok {
			state.StepsCompleted = stepsCompleted
		}
		if step, ok := migrationJournalInt(entry["step"]); ok {
			state.LastStep = step
		}
		if verdict := migrationJournalString(entry["verdict"]); verdict != "" {
			state.Verdict = verdict
		}
		if lastError := migrationJournalString(entry["last_error"]); lastError != "" {
			state.LastError = lastError
		} else if state.Verdict == "ok" || state.Verdict == "running" || state.Verdict == "idle" {
			state.LastError = ""
		}
		if blockedSince, ok := migrationJournalTime(entry["blocked_since"]); ok {
			state.DrainBlockedAt = blockedSince
		} else if state.Verdict == "ok" || state.Verdict == "running" || state.Verdict == "idle" {
			state.DrainBlockedAt = time.Time{}
		}
		switch lastEvent {
		case "artifact_inverse_failed":
			if state.PendingRecords == nil {
				state.PendingRecords = map[string]string{}
			}
			recordID := migrationJournalString(entry["record_id"])
			if recordID == "" {
				recordID = migrationJournalString(entry["artifact_id"])
			}
			if recordID != "" {
				resolution := migrationJournalString(entry["recommended_resolution"])
				if resolution == "" {
					resolution = "manual"
				}
				state.PendingRecords[recordID] = resolution
				state.Verdict = "reconcile_pending"
				state.LastError = ErrMigrationReconcilePending.Error()
				if state.ReconciliationPath == "" {
					state.ReconciliationPath = migrationReconciliationPath(pkg)
				}
			}
		case "reconcile_pending":
			pending := migrationPendingRecordsFromJournalValue(entry["pending_records"])
			if len(pending) > 0 {
				state.PendingRecords = pending
				state.Verdict = "reconcile_pending"
				state.LastError = ErrMigrationReconcilePending.Error()
				if state.ReconciliationPath == "" {
					state.ReconciliationPath = migrationReconciliationPath(pkg)
				}
			}
		case "reconcile_record":
			recordID := migrationJournalString(entry["record_id"])
			if recordID != "" && len(state.PendingRecords) > 0 {
				delete(state.PendingRecords, recordID)
				if len(state.PendingRecords) == 0 {
					state.ReconciliationPath = ""
				}
			}
		case "retry_committed", "aborted":
			state.PendingRecords = nil
			state.ReconciliationPath = ""
		}
	}

	if state.StepsCompleted < 0 {
		state.StepsCompleted = 0
	}
	if state.StepsCompleted > state.StepsPlanned {
		state.StepsCompleted = state.StepsPlanned
	}
	if state.LastStep < 0 {
		state.LastStep = 0
	}
	if state.LastStep > state.StepsPlanned {
		state.LastStep = state.StepsPlanned
	}
	if state.Verdict == "running" {
		state.Verdict = "step_failed"
		if state.LastError == "" {
			if state.LastStep > 0 {
				state.LastError = fmt.Sprintf("step %d interrupted before commit", state.LastStep)
			} else {
				state.LastError = ErrMigrationInterrupted.Error()
			}
		}
		if state.LastStep <= 0 && strings.HasPrefix(lastEvent, "step_") {
			state.LastStep = state.StepsCompleted + 1
			if state.LastStep > state.StepsPlanned {
				state.LastStep = state.StepsPlanned
			}
		}
	}

	return state
}

func migrationArtifactInverseFailuresFromJournal(pkg Package, state migrationState) map[string]string {
	if strings.TrimSpace(state.JournalPath) == "" {
		return nil
	}
	absolutePath := filepath.Join(pkg.RootPath, filepath.FromSlash(state.JournalPath))
	file, err := os.Open(filepath.Clean(absolutePath))
	if err != nil {
		return nil
	}
	defer func() {
		_ = file.Close()
	}()

	pending := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		switch migrationJournalString(entry["event"]) {
		case "artifact_inverse_failed":
			recordID := migrationJournalString(entry["record_id"])
			if recordID == "" {
				recordID = migrationJournalString(entry["artifact_id"])
			}
			if recordID == "" {
				continue
			}
			resolution := migrationJournalString(entry["recommended_resolution"])
			if resolution == "" {
				resolution = "manual"
			}
			pending[recordID] = resolution
		case "reconcile_record":
			if recordID := migrationJournalString(entry["record_id"]); recordID != "" {
				delete(pending, recordID)
			}
		case "retry_committed", "aborted":
			pending = map[string]string{}
		}
	}
	if len(pending) == 0 {
		return nil
	}
	return pending
}

func migrationPendingRecordsFromJournalValue(raw any) map[string]string {
	values, ok := raw.(map[string]any)
	if !ok || len(values) == 0 {
		return nil
	}
	pending := make(map[string]string, len(values))
	for recordID, resolutionRaw := range values {
		recordID = strings.TrimSpace(recordID)
		if recordID == "" {
			continue
		}
		resolution := migrationJournalString(resolutionRaw)
		if resolution == "" {
			resolution = "manual"
		}
		pending[recordID] = resolution
	}
	if len(pending) == 0 {
		return nil
	}
	return pending
}

func migrationJournalInt(raw any) (int, bool) {
	switch v := raw.(type) {
	case float64:
		return int(v), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func migrationJournalString(raw any) string {
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func migrationJournalTime(raw any) (time.Time, bool) {
	value := migrationJournalString(raw)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func validateRuntimeMigrationScript(payload []byte) error {
	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" {
		return errors.New("script is empty")
	}
	if !migrateEntryPointPattern.Match(payload) {
		return errors.New("missing migrate() entrypoint")
	}
	for _, match := range migrateLoadPattern.FindAllSubmatch(payload, -1) {
		if len(match) < 2 {
			continue
		}
		module := strings.TrimSpace(string(match[1]))
		if _, ok := allowedMigrationModules[module]; !ok {
			return fmt.Errorf("loads disallowed module %q", module)
		}
	}
	return nil
}

type runtimeMigrationHostEffects struct {
	ArtifactPatches []runtimeMigrationArtifactPatchEffect
}

type runtimeMigrationArtifactPatchEffect struct {
	ArtifactID string
	OwnerAppID string
	Sequence   int
}

func collectRuntimeMigrationHostEffects(pkg Package, scriptSource []byte) (runtimeMigrationHostEffects, error) {
	var effects runtimeMigrationHostEffects
	patchAliases := artifactSelfPatchAliases(scriptSource)
	if len(patchAliases) == 0 {
		return effects, nil
	}
	patchCount := 0
	lines := strings.Split(string(scriptSource), "\n")
	for lineNumber, line := range lines {
		line = strings.TrimSpace(stripTALLineComment(line))
		if line == "" || strings.HasPrefix(line, "def ") || strings.HasPrefix(line, "load(") || strings.HasPrefix(line, "return ") || line == "pass" {
			continue
		}
		match := migrateCallPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		if _, ok := patchAliases[match[1]]; !ok {
			continue
		}
		patchCount++
		if patchCount > migrationMaxArtifactPatches {
			return effects, fmt.Errorf("%w: artifact.self.patch count exceeds hard cap (%d > %d)", ErrMigrationResourceLimit, patchCount, migrationMaxArtifactPatches)
		}
		appID := strings.TrimSpace(pkg.Manifest.AppID)
		if appID == "" {
			return effects, fmt.Errorf("%w: artifact.self.patch requires manifest app_id at line %d", ErrMigrationArtifactOwnership, lineNumber+1)
		}
		artifactID := migrationStringArgument(match[2])
		ownerAppID := migrationKeywordStringArgument(migrateOwnerAppIDPattern, match[2])
		if ownerAppID == "" {
			return effects, fmt.Errorf("%w: artifact %q patch missing owner_app_id at line %d", ErrMigrationArtifactOwnership, artifactID, lineNumber+1)
		}
		if ownerAppID != appID {
			return effects, fmt.Errorf("%w: artifact %q owner_app_id %q does not match app_id %q at line %d", ErrMigrationArtifactOwnership, artifactID, ownerAppID, appID, lineNumber+1)
		}
		effects.ArtifactPatches = append(effects.ArtifactPatches, runtimeMigrationArtifactPatchEffect{
			ArtifactID: artifactID,
			OwnerAppID: ownerAppID,
			Sequence:   patchCount,
		})
	}
	return effects, nil
}

func artifactSelfPatchAliases(scriptSource []byte) map[string]struct{} {
	aliases := make(map[string]struct{})
	for _, match := range migrateArtifactSelfLoadPattern.FindAllSubmatch(scriptSource, -1) {
		if len(match) < 2 {
			continue
		}
		for _, aliasMatch := range migrateLoadAliasPattern.FindAllSubmatch(match[1], -1) {
			if len(aliasMatch) < 2 {
				continue
			}
			aliases[string(aliasMatch[1])] = struct{}{}
		}
	}
	return aliases
}

func migrationLogAliases(scriptSource []byte) map[string]string {
	aliases := make(map[string]string)
	for _, match := range migrateLogLoadPattern.FindAllSubmatch(scriptSource, -1) {
		if len(match) < 2 {
			continue
		}
		for _, aliasMatch := range migrateLogAliasPattern.FindAllSubmatch(match[1], -1) {
			if len(aliasMatch) < 3 {
				continue
			}
			aliases[string(aliasMatch[2])] = string(aliasMatch[1])
		}
	}
	return aliases
}

type runtimeMigrationStoreAliases struct {
	ListKeys map[string]struct{}
	Get      map[string]struct{}
	Put      map[string]struct{}
	Delete   map[string]struct{}
}

func migrationStoreAliases(scriptSource []byte) runtimeMigrationStoreAliases {
	aliases := runtimeMigrationStoreAliases{
		ListKeys: make(map[string]struct{}),
		Get:      make(map[string]struct{}),
		Put:      make(map[string]struct{}),
		Delete:   make(map[string]struct{}),
	}
	for _, match := range migrateStoreLoadPattern.FindAllSubmatch(scriptSource, -1) {
		if len(match) < 2 {
			continue
		}
		for _, aliasMatch := range migrateStoreListKeysAliasPattern.FindAllSubmatch(match[1], -1) {
			if len(aliasMatch) >= 2 {
				aliases.ListKeys[string(aliasMatch[1])] = struct{}{}
			}
		}
		for _, aliasMatch := range migrateStoreGetAliasPattern.FindAllSubmatch(match[1], -1) {
			if len(aliasMatch) >= 2 {
				aliases.Get[string(aliasMatch[1])] = struct{}{}
			}
		}
		for _, aliasMatch := range migrateStorePutAliasPattern.FindAllSubmatch(match[1], -1) {
			if len(aliasMatch) >= 2 {
				aliases.Put[string(aliasMatch[1])] = struct{}{}
			}
		}
		for _, aliasMatch := range migrateStoreDeleteAliasPattern.FindAllSubmatch(match[1], -1) {
			if len(aliasMatch) >= 2 {
				aliases.Delete[string(aliasMatch[1])] = struct{}{}
			}
		}
	}
	return aliases
}

func migrationCheckpointAliases(scriptSource []byte) map[string]struct{} {
	aliases := make(map[string]struct{})
	for _, match := range migrateEnvLoadPattern.FindAllSubmatch(scriptSource, -1) {
		if len(match) < 2 {
			continue
		}
		for _, aliasMatch := range migrateEnvCheckpointAliasPattern.FindAllSubmatch(match[1], -1) {
			if len(aliasMatch) >= 2 {
				aliases[string(aliasMatch[1])] = struct{}{}
			}
		}
	}
	return aliases
}

func migrationAbortAliases(scriptSource []byte) map[string]struct{} {
	aliases := make(map[string]struct{})
	for _, match := range migrateEnvLoadPattern.FindAllSubmatch(scriptSource, -1) {
		if len(match) < 2 {
			continue
		}
		for _, aliasMatch := range migrateEnvAbortAliasPattern.FindAllSubmatch(match[1], -1) {
			if len(aliasMatch) >= 2 {
				aliases[string(aliasMatch[1])] = struct{}{}
			}
		}
	}
	return aliases
}

func migrationAbortCall(line string, aliases map[string]struct{}) (string, bool, error) {
	if match := migrateAbortPattern.FindStringSubmatch(line); match != nil {
		var reason string
		if err := json.Unmarshal([]byte(match[1]), &reason); err != nil {
			return "", true, err
		}
		return reason, true, nil
	}
	match := migrateCallPattern.FindStringSubmatch(line)
	if match == nil {
		return "", false, nil
	}
	if _, ok := aliases[match[1]]; !ok {
		return "", false, nil
	}
	reasonLiteral := migrationStringArgument(match[2])
	if reasonLiteral == "" {
		return "", true, errors.New("missing abort reason")
	}
	return reasonLiteral, true, nil
}

func migrationStringArgument(args string) string {
	match := migrateStringArgPattern.FindStringSubmatch(args)
	if match == nil {
		return ""
	}
	return decodeTALStringLiteral(match[1])
}

func migrationKeywordStringArgument(pattern *regexp.Regexp, args string) string {
	match := pattern.FindStringSubmatch(args)
	if match == nil {
		return ""
	}
	return decodeTALStringLiteral(match[1])
}

func decodeTALStringLiteral(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "'") && strings.HasSuffix(raw, "'") {
		raw = `"` + strings.ReplaceAll(strings.Trim(raw, "'"), `"`, `\"`) + `"`
	}
	var value string
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return ""
	}
	return value
}

type runtimeMigrationFixture struct {
	Step            int
	PriorVersion    string
	SeedPath        string
	ExpectedPath    string
	ReadAdapterPath string
}

type runtimeMigrationResourceStats struct {
	StoreOps              int
	WriteVolumeBytes      int64
	ArtifactPatchAttempts int
	Logs                  []runtimeMigrationLogEntry
}

type runtimeMigrationLogEntry struct {
	Level     string
	Message   string
	Arguments string
}

type runtimeMigrationResourceLimits struct {
	MaxStoreOps              int
	MaxWriteVolumeBytes      int64
	MaxArtifactPatchAttempts int
}

type runtimeMigrationStoreFixturePlan struct {
	Prefix     string
	Transforms []runtimeMigrationFixtureTransform
}

func defaultRuntimeMigrationResourceLimits() runtimeMigrationResourceLimits {
	return runtimeMigrationResourceLimits{
		MaxStoreOps:              migrationMaxStoreOps,
		MaxWriteVolumeBytes:      migrationMaxWriteVolumeBytes,
		MaxArtifactPatchAttempts: migrationMaxArtifactPatches,
	}
}

func verifyMigrationFixtureStep(root string, step migrationPlanStep, scriptSource []byte) (runtimeMigrationResourceStats, error) {
	fixture, err := findRuntimeMigrationFixture(root, step)
	if err != nil {
		return runtimeMigrationResourceStats{}, err
	}
	if fixture == nil {
		return runtimeMigrationResourceStats{}, nil
	}

	seedRecords, err := readRuntimeFixtureRecords(root, fixture.SeedPath)
	if err != nil {
		return runtimeMigrationResourceStats{}, err
	}
	expectedRecords, err := readRuntimeFixtureRecords(root, fixture.ExpectedPath)
	if err != nil {
		return runtimeMigrationResourceStats{}, err
	}
	actualRecords, stats, err := executeRuntimeMigrationFixture(scriptSource, seedRecords)
	if err != nil {
		if errors.Is(err, ErrMigrationAborted) {
			return runtimeMigrationResourceStats{}, err
		}
		return runtimeMigrationResourceStats{}, fmt.Errorf("%w: step %04d fixture execution failed: %v", ErrMigrationFixtureMismatch, step.Number, err)
	}
	if err := validateRuntimeMigrationResourceLimits(stats, defaultRuntimeMigrationResourceLimits()); err != nil {
		return runtimeMigrationResourceStats{}, fmt.Errorf("%w: step %04d: %v", ErrMigrationResourceLimit, step.Number, err)
	}

	if len(actualRecords) != len(expectedRecords) {
		return runtimeMigrationResourceStats{}, fmt.Errorf("%w: step %04d key count mismatch (actual=%d expected=%d)", ErrMigrationFixtureMismatch, step.Number, len(actualRecords), len(expectedRecords))
	}
	for key, actualValue := range actualRecords {
		expectedValue, ok := expectedRecords[key]
		if !ok {
			return runtimeMigrationResourceStats{}, fmt.Errorf("%w: step %04d expected missing key %q", ErrMigrationFixtureMismatch, step.Number, key)
		}
		if expectedValue != actualValue {
			return runtimeMigrationResourceStats{}, fmt.Errorf("%w: step %04d value mismatch for key %q: expected=%s actual=%s", ErrMigrationFixtureMismatch, step.Number, key, expectedValue, actualValue)
		}
	}
	for key := range expectedRecords {
		if _, ok := actualRecords[key]; !ok {
			return runtimeMigrationResourceStats{}, fmt.Errorf("%w: step %04d expected contains extra key %q", ErrMigrationFixtureMismatch, step.Number, key)
		}
	}
	if strings.EqualFold(strings.TrimSpace(step.DrainPolicy), "multi_version") {
		if err := verifyMigrationReadAdapterStep(root, step, fixture, expectedRecords, seedRecords); err != nil {
			return runtimeMigrationResourceStats{}, err
		}
	}

	return stats, nil
}

func verifyMigrationReadAdapterStep(root string, step migrationPlanStep, fixture *runtimeMigrationFixture, migratedRecords map[string]string, priorRecords map[string]string) error {
	adapterPath := strings.TrimSpace(fixture.ReadAdapterPath)
	if adapterPath == "" {
		return fmt.Errorf("%w: step %04d multi_version fixture must declare read_adapter", ErrMigrationFixtureUnavailable, step.Number)
	}
	fullPath, resolveErr := resolveRuntimeFixturePath(root, adapterPath)
	if resolveErr != nil {
		return fmt.Errorf("step %04d read_adapter %s: %w", step.Number, adapterPath, resolveErr)
	}
	adapterSource, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrMigrationFixtureUnavailable, adapterPath, err)
	}
	if err := validateRuntimeMigrationReadAdapter(adapterSource); err != nil {
		return fmt.Errorf("%w: step %04d read_adapter %s invalid: %v", ErrMigrationFixtureMismatch, step.Number, adapterPath, err)
	}
	adapterRecords, _, err := executeRuntimeMigrationFixture(adapterSource, migratedRecords)
	if err != nil {
		return fmt.Errorf("%w: step %04d read_adapter %s execution failed: %v", ErrMigrationFixtureMismatch, step.Number, adapterPath, err)
	}
	if err := compareRuntimeFixtureRecords(adapterRecords, priorRecords, step.Number, "read_adapter"); err != nil {
		return err
	}
	return nil
}

func validateRuntimeMigrationReadAdapter(payload []byte) error {
	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" {
		return errors.New("script is empty")
	}
	if !migrateReadAdapterEntryPointPattern.Match(payload) {
		return errors.New("missing read(record) entrypoint")
	}
	for _, match := range migrateLoadPattern.FindAllSubmatch(payload, -1) {
		if len(match) < 2 {
			continue
		}
		module := strings.TrimSpace(string(match[1]))
		if _, ok := allowedMigrationModules[module]; !ok {
			return fmt.Errorf("loads disallowed module %q", module)
		}
	}
	return nil
}

func compareRuntimeFixtureRecords(actualRecords map[string]string, expectedRecords map[string]string, step int, label string) error {
	if len(actualRecords) != len(expectedRecords) {
		return fmt.Errorf("%w: step %04d %s key count mismatch (actual=%d expected=%d)", ErrMigrationFixtureMismatch, step, label, len(actualRecords), len(expectedRecords))
	}
	for key, actualValue := range actualRecords {
		expectedValue, ok := expectedRecords[key]
		if !ok {
			return fmt.Errorf("%w: step %04d %s expected missing key %q", ErrMigrationFixtureMismatch, step, label, key)
		}
		if expectedValue != actualValue {
			return fmt.Errorf("%w: step %04d %s value mismatch for key %q: expected=%s actual=%s", ErrMigrationFixtureMismatch, step, label, key, expectedValue, actualValue)
		}
	}
	for key := range expectedRecords {
		if _, ok := actualRecords[key]; !ok {
			return fmt.Errorf("%w: step %04d %s expected contains extra key %q", ErrMigrationFixtureMismatch, step, label, key)
		}
	}
	return nil
}

type runtimeMigrationFixtureTransform struct {
	Destination string
	Source      string
	Default     any
	HasDefault  bool
	Operation   string
	Value       any
	Reason      string
}

func executeRuntimeMigrationFixture(scriptSource []byte, seedRecords map[string]string) (map[string]string, runtimeMigrationResourceStats, error) {
	stats := runtimeMigrationResourceStats{Logs: collectRuntimeMigrationLogs(scriptSource)}
	transforms, err := parseRuntimeMigrationFixtureTransforms(scriptSource)
	if err != nil {
		return executeRuntimeMigrationStoreFixture(scriptSource, seedRecords, err, stats.Logs)
	}
	if len(transforms) == 0 {
		out := make(map[string]string, len(seedRecords))
		for key, value := range seedRecords {
			out[key] = value
		}
		return out, stats, nil
	}

	for _, transform := range transforms {
		if transform.Operation == "abort" {
			return nil, runtimeMigrationResourceStats{}, fmt.Errorf("%w: %s", ErrMigrationAborted, transform.Reason)
		}
	}

	out := make(map[string]string, len(seedRecords))
	var writeVolume int64
	for key, rawValue := range seedRecords {
		var record map[string]any
		if err := json.Unmarshal([]byte(rawValue), &record); err != nil {
			return nil, runtimeMigrationResourceStats{}, fmt.Errorf("seed key %q is not a JSON object: %w", key, err)
		}
		skipRemainingTransforms := false
		for _, transform := range transforms {
			if skipRemainingTransforms {
				break
			}
			switch transform.Operation {
			case "skip_if_present":
				if _, ok := record[transform.Source]; ok {
					skipRemainingTransforms = true
				}
			case "copy":
				record[transform.Destination] = runtimeMigrationFixtureValue(record, transform)
			case "lower":
				value, ok := runtimeMigrationFixtureStringValue(record, transform)
				if !ok {
					return nil, runtimeMigrationResourceStats{}, fmt.Errorf("record key %q field %q is not a string for lower()", key, transform.Source)
				}
				record[transform.Destination] = strings.ToLower(value)
			case "trim":
				value, ok := runtimeMigrationFixtureStringValue(record, transform)
				if !ok {
					return nil, runtimeMigrationResourceStats{}, fmt.Errorf("record key %q field %q is not a string for trim()", key, transform.Source)
				}
				record[transform.Destination] = strings.TrimSpace(value)
			case "lower_trim":
				value, ok := runtimeMigrationFixtureStringValue(record, transform)
				if !ok {
					return nil, runtimeMigrationResourceStats{}, fmt.Errorf("record key %q field %q is not a string for lower(trim())", key, transform.Source)
				}
				record[transform.Destination] = strings.ToLower(strings.TrimSpace(value))
			case "literal":
				record[transform.Destination] = transform.Value
			case "delete":
				delete(record, transform.Destination)
			case "abort":
				return nil, runtimeMigrationResourceStats{}, fmt.Errorf("%w: %s", ErrMigrationAborted, transform.Reason)
			default:
				return nil, runtimeMigrationResourceStats{}, fmt.Errorf("unsupported fixture transform %q", transform.Operation)
			}
		}
		canonical, err := json.Marshal(record)
		if err != nil {
			return nil, runtimeMigrationResourceStats{}, fmt.Errorf("canonicalize migrated record %q: %w", key, err)
		}
		out[key] = string(canonical)
		writeVolume += int64(len(canonical))
	}
	stats.StoreOps = len(seedRecords)
	stats.WriteVolumeBytes = writeVolume
	return out, stats, nil
}

func executeRuntimeMigrationStoreFixture(scriptSource []byte, seedRecords map[string]string, recordModeErr error, logs []runtimeMigrationLogEntry) (map[string]string, runtimeMigrationResourceStats, error) {
	plan, err := parseRuntimeMigrationStoreFixturePlan(scriptSource)
	if err != nil {
		if recordModeErr != nil {
			return nil, runtimeMigrationResourceStats{}, recordModeErr
		}
		return nil, runtimeMigrationResourceStats{}, err
	}
	if plan == nil {
		return nil, runtimeMigrationResourceStats{}, recordModeErr
	}

	out := make(map[string]string, len(seedRecords))
	for key, value := range seedRecords {
		out[key] = value
	}

	keys := make([]string, 0, len(seedRecords))
	for key := range seedRecords {
		if strings.HasPrefix(key, plan.Prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	stats := runtimeMigrationResourceStats{Logs: logs}
	for _, key := range keys {
		var record map[string]any
		if err := json.Unmarshal([]byte(out[key]), &record); err != nil {
			return nil, runtimeMigrationResourceStats{}, fmt.Errorf("seed key %q is not a JSON object: %w", key, err)
		}
		skipPut := false
		for _, transform := range plan.Transforms {
			if skipPut {
				break
			}
			switch transform.Operation {
			case "skip_if_present":
				if _, ok := record[transform.Source]; ok {
					skipPut = true
				}
			case "copy":
				record[transform.Destination] = runtimeMigrationFixtureValue(record, transform)
			case "lower":
				value, ok := runtimeMigrationFixtureStringValue(record, transform)
				if !ok {
					return nil, runtimeMigrationResourceStats{}, fmt.Errorf("record key %q field %q is not a string for lower()", key, transform.Source)
				}
				record[transform.Destination] = strings.ToLower(value)
			case "trim":
				value, ok := runtimeMigrationFixtureStringValue(record, transform)
				if !ok {
					return nil, runtimeMigrationResourceStats{}, fmt.Errorf("record key %q field %q is not a string for trim()", key, transform.Source)
				}
				record[transform.Destination] = strings.TrimSpace(value)
			case "lower_trim":
				value, ok := runtimeMigrationFixtureStringValue(record, transform)
				if !ok {
					return nil, runtimeMigrationResourceStats{}, fmt.Errorf("record key %q field %q is not a string for lower(trim())", key, transform.Source)
				}
				record[transform.Destination] = strings.ToLower(strings.TrimSpace(value))
			case "literal":
				record[transform.Destination] = transform.Value
			case "delete":
				delete(record, transform.Destination)
			case "delete_record":
				delete(out, key)
				stats.StoreOps++
				skipPut = true
			case "abort":
				return nil, runtimeMigrationResourceStats{}, fmt.Errorf("%w: %s", ErrMigrationAborted, transform.Reason)
			default:
				return nil, runtimeMigrationResourceStats{}, fmt.Errorf("unsupported fixture transform %q", transform.Operation)
			}
		}
		if skipPut {
			continue
		}
		canonical, err := json.Marshal(record)
		if err != nil {
			return nil, runtimeMigrationResourceStats{}, fmt.Errorf("canonicalize migrated record %q: %w", key, err)
		}
		out[key] = string(canonical)
		stats.StoreOps++
		stats.WriteVolumeBytes += int64(len(canonical))
	}
	return out, stats, nil
}

func collectRuntimeMigrationLogs(scriptSource []byte) []runtimeMigrationLogEntry {
	logAliases := migrationLogAliases(scriptSource)
	if len(logAliases) == 0 {
		return nil
	}
	lines := strings.Split(string(scriptSource), "\n")
	logs := make([]runtimeMigrationLogEntry, 0)
	for _, rawLine := range lines {
		line := strings.TrimSpace(stripTALLineComment(rawLine))
		match := migrateCallPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		level, ok := logAliases[match[1]]
		if !ok {
			continue
		}
		message := migrationStringArgument(match[2])
		if message == "" {
			continue
		}
		logs = append(logs, runtimeMigrationLogEntry{
			Level:     level,
			Message:   message,
			Arguments: strings.TrimSpace(match[2]),
		})
	}
	return logs
}

func validateRuntimeMigrationResourceLimits(stats runtimeMigrationResourceStats, limits runtimeMigrationResourceLimits) error {
	if limits.MaxStoreOps > 0 && stats.StoreOps > limits.MaxStoreOps {
		return fmt.Errorf("store ops exceed hard cap (%d > %d)", stats.StoreOps, limits.MaxStoreOps)
	}
	if limits.MaxWriteVolumeBytes > 0 && stats.WriteVolumeBytes > limits.MaxWriteVolumeBytes {
		return fmt.Errorf("write volume exceeds hard cap (%d > %d bytes)", stats.WriteVolumeBytes, limits.MaxWriteVolumeBytes)
	}
	if limits.MaxArtifactPatchAttempts > 0 && stats.ArtifactPatchAttempts > limits.MaxArtifactPatchAttempts {
		return fmt.Errorf("artifact patch attempts exceed hard cap (%d > %d)", stats.ArtifactPatchAttempts, limits.MaxArtifactPatchAttempts)
	}
	return nil
}

func runtimeMigrationFixtureValue(record map[string]any, transform runtimeMigrationFixtureTransform) any {
	value, ok := record[transform.Source]
	if !ok && transform.HasDefault {
		return transform.Default
	}
	return value
}

func runtimeMigrationFixtureStringValue(record map[string]any, transform runtimeMigrationFixtureTransform) (string, bool) {
	value := runtimeMigrationFixtureValue(record, transform)
	text, ok := value.(string)
	return text, ok
}

func parseRuntimeMigrationFixtureTransforms(scriptSource []byte) ([]runtimeMigrationFixtureTransform, error) {
	lines := strings.Split(string(scriptSource), "\n")
	transforms := make([]runtimeMigrationFixtureTransform, 0)
	abortAliases := migrationAbortAliases(scriptSource)
	logAliases := migrationLogAliases(scriptSource)
	for lineNumber, line := range lines {
		line = strings.TrimSpace(stripTALLineComment(line))
		if line == "" || strings.HasPrefix(line, "def ") || line == "pass" || strings.HasPrefix(line, "load(") || strings.HasPrefix(line, "return ") {
			continue
		}
		if reason, ok, err := migrationAbortCall(line, abortAliases); ok {
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid abort reason: %w", lineNumber+1, err)
			}
			transforms = append(transforms, runtimeMigrationFixtureTransform{
				Operation: "abort",
				Reason:    reason,
			})
			continue
		}
		if match := migrateRecordDeletePattern.FindStringSubmatch(line); match != nil {
			transforms = append(transforms, runtimeMigrationFixtureTransform{
				Destination: match[1],
				Operation:   "delete",
			})
			continue
		}
		if match := migrateRecordSkipIfPresentPattern.FindStringSubmatch(line); match != nil {
			transforms = append(transforms, runtimeMigrationFixtureTransform{
				Source:    match[1],
				Operation: "skip_if_present",
			})
			continue
		}
		if match := migrateRecordAssignmentPattern.FindStringSubmatch(line); match != nil {
			transform, err := parseRuntimeMigrationFixtureAssignment(match[1], match[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNumber+1, err)
			}
			transforms = append(transforms, transform)
			continue
		}
		if match := migrateCallPattern.FindStringSubmatch(line); match != nil {
			if _, ok := logAliases[match[1]]; ok {
				continue
			}
		}
		return nil, fmt.Errorf("line %d uses unsupported fixture migration statement %q", lineNumber+1, line)
	}
	return transforms, nil
}

func parseRuntimeMigrationStoreFixturePlan(scriptSource []byte) (*runtimeMigrationStoreFixturePlan, error) {
	storeAliases := migrationStoreAliases(scriptSource)
	if len(storeAliases.ListKeys) == 0 || (len(storeAliases.Get) == 0 && len(storeAliases.Delete) == 0) || (len(storeAliases.Put) == 0 && len(storeAliases.Delete) == 0) {
		return nil, nil
	}
	abortAliases := migrationAbortAliases(scriptSource)
	checkpointAliases := migrationCheckpointAliases(scriptSource)
	logAliases := migrationLogAliases(scriptSource)
	lines := strings.Split(string(scriptSource), "\n")
	transforms := make([]runtimeMigrationFixtureTransform, 0)
	prefix := ""
	sawGet := false
	sawPut := false
	sawDelete := false
	for lineNumber, rawLine := range lines {
		line := strings.TrimSpace(stripTALLineComment(rawLine))
		if line == "" || strings.HasPrefix(line, "load(") || strings.HasPrefix(line, "def ") || line == "pass" {
			continue
		}
		if line == "cursor = None" || line == "count = 0" || line == "while True:" ||
			line == "if len(page) == 0: break" || line == "for key in page:" ||
			line == "count += 1" || line == "cursor = page[-1]" ||
			line == "return label.strip().lower()" {
			continue
		}
		if parsedPrefix, ok := migrationStoreListKeysPrefix(line, storeAliases.ListKeys); ok {
			if prefix != "" && prefix != parsedPrefix {
				return nil, fmt.Errorf("line %d uses multiple list_keys prefixes", lineNumber+1)
			}
			prefix = parsedPrefix
			continue
		}
		if migrationStoreGetStatement(line, storeAliases.Get) {
			sawGet = true
			continue
		}
		if migrationStorePutStatement(line, storeAliases.Put) {
			sawPut = true
			continue
		}
		if migrationStoreDeleteStatement(line, storeAliases.Delete) {
			sawDelete = true
			transforms = append(transforms, runtimeMigrationFixtureTransform{
				Operation: "delete_record",
			})
			continue
		}
		if reason, ok, err := migrationAbortCall(line, abortAliases); ok {
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid abort reason: %w", lineNumber+1, err)
			}
			transforms = append(transforms, runtimeMigrationFixtureTransform{
				Operation: "abort",
				Reason:    reason,
			})
			continue
		}
		recordLine := migrationStoreRecordLine(line)
		if match := migrateRecordDeletePattern.FindStringSubmatch(recordLine); match != nil {
			transforms = append(transforms, runtimeMigrationFixtureTransform{
				Destination: match[1],
				Operation:   "delete",
			})
			continue
		}
		if match := migrateRecordSkipIfPresentPattern.FindStringSubmatch(recordLine); match != nil {
			transforms = append(transforms, runtimeMigrationFixtureTransform{
				Source:    match[1],
				Operation: "skip_if_present",
			})
			continue
		}
		if match := migrateRecordAssignmentPattern.FindStringSubmatch(recordLine); match != nil {
			transform, err := parseRuntimeMigrationFixtureAssignment(match[1], match[2])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNumber+1, err)
			}
			transforms = append(transforms, transform)
			continue
		}
		if match := migrateCallPattern.FindStringSubmatch(line); match != nil {
			if _, ok := logAliases[match[1]]; ok {
				continue
			}
			if _, ok := checkpointAliases[match[1]]; ok {
				continue
			}
		}
		return nil, fmt.Errorf("line %d uses unsupported store fixture migration statement %q", lineNumber+1, line)
	}
	if prefix == "" {
		return nil, errors.New("store fixture migration missing list_keys prefix")
	}
	if !sawGet && !sawDelete {
		return nil, errors.New("store fixture migration must get records or delete keys")
	}
	if !sawPut && !sawDelete {
		return nil, errors.New("store fixture migration must put records or delete keys")
	}
	return &runtimeMigrationStoreFixturePlan{
		Prefix:     prefix,
		Transforms: transforms,
	}, nil
}

func migrationStoreListKeysPrefix(line string, aliases map[string]struct{}) (string, bool) {
	match := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*\s*=\s*([A-Za-z_][A-Za-z0-9_]*)\s*\((.*)\)$`).FindStringSubmatch(line)
	if match == nil {
		return "", false
	}
	if _, ok := aliases[match[1]]; !ok {
		return "", false
	}
	prefix := migrationKeywordStringArgument(regexp.MustCompile(`\bprefix\s*=\s*("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*')`), match[2])
	return prefix, prefix != ""
}

func migrationStoreGetStatement(line string, aliases map[string]struct{}) bool {
	match := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*\s*=\s*([A-Za-z_][A-Za-z0-9_]*)\s*\(\s*key\s*\)$`).FindStringSubmatch(line)
	if match == nil {
		return false
	}
	_, ok := aliases[match[1]]
	return ok
}

func migrationStorePutStatement(line string, aliases map[string]struct{}) bool {
	match := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*\(\s*key\s*,\s*[A-Za-z_][A-Za-z0-9_]*\s*\)$`).FindStringSubmatch(line)
	if match == nil {
		return false
	}
	_, ok := aliases[match[1]]
	return ok
}

func migrationStoreDeleteStatement(line string, aliases map[string]struct{}) bool {
	match := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*\(\s*key\s*\)$`).FindStringSubmatch(line)
	if match == nil {
		return false
	}
	_, ok := aliases[match[1]]
	return ok
}

func migrationStoreRecordLine(line string) string {
	replacer := strings.NewReplacer(
		`rec["`, `record["`,
		"rec.get(", "record.get(",
		"in rec:", "in record:",
	)
	return replacer.Replace(line)
}

func parseRuntimeMigrationFixtureAssignment(destination string, expression string) (runtimeMigrationFixtureTransform, error) {
	expression = strings.TrimSpace(expression)
	if match := migrateRecordNormalizeGetPattern.FindStringSubmatch(expression); match != nil {
		defaultValue, err := decodeMigrationDefaultLiteral(match[2])
		if err != nil {
			return runtimeMigrationFixtureTransform{}, err
		}
		return runtimeMigrationFixtureTransform{
			Destination: destination,
			Source:      match[1],
			Default:     defaultValue,
			HasDefault:  true,
			Operation:   "lower_trim",
		}, nil
	}
	if match := migrateRecordNormalizePattern.FindStringSubmatch(expression); match != nil {
		return runtimeMigrationFixtureTransform{
			Destination: destination,
			Source:      match[1],
			Operation:   "lower_trim",
		}, nil
	}
	if match := migrateRecordLowerTrimGetPattern.FindStringSubmatch(expression); match != nil {
		defaultValue, err := decodeMigrationDefaultLiteral(match[2])
		if err != nil {
			return runtimeMigrationFixtureTransform{}, err
		}
		return runtimeMigrationFixtureTransform{
			Destination: destination,
			Source:      match[1],
			Default:     defaultValue,
			HasDefault:  true,
			Operation:   "lower_trim",
		}, nil
	}
	if match := migrateRecordLowerTrimPattern.FindStringSubmatch(expression); match != nil {
		return runtimeMigrationFixtureTransform{
			Destination: destination,
			Source:      match[1],
			Operation:   "lower_trim",
		}, nil
	}
	if match := migrateRecordLowerGetPattern.FindStringSubmatch(expression); match != nil {
		defaultValue, err := decodeMigrationDefaultLiteral(match[2])
		if err != nil {
			return runtimeMigrationFixtureTransform{}, err
		}
		return runtimeMigrationFixtureTransform{
			Destination: destination,
			Source:      match[1],
			Default:     defaultValue,
			HasDefault:  true,
			Operation:   "lower",
		}, nil
	}
	if match := migrateRecordLowerPattern.FindStringSubmatch(expression); match != nil {
		return runtimeMigrationFixtureTransform{
			Destination: destination,
			Source:      match[1],
			Operation:   "lower",
		}, nil
	}
	if match := migrateRecordTrimGetPattern.FindStringSubmatch(expression); match != nil {
		defaultValue, err := decodeMigrationDefaultLiteral(match[2])
		if err != nil {
			return runtimeMigrationFixtureTransform{}, err
		}
		return runtimeMigrationFixtureTransform{
			Destination: destination,
			Source:      match[1],
			Default:     defaultValue,
			HasDefault:  true,
			Operation:   "trim",
		}, nil
	}
	if match := migrateRecordTrimPattern.FindStringSubmatch(expression); match != nil {
		return runtimeMigrationFixtureTransform{
			Destination: destination,
			Source:      match[1],
			Operation:   "trim",
		}, nil
	}
	if match := migrateRecordGetValuePattern.FindStringSubmatch(expression); match != nil {
		defaultValue, err := decodeMigrationDefaultLiteral(match[2])
		if err != nil {
			return runtimeMigrationFixtureTransform{}, err
		}
		return runtimeMigrationFixtureTransform{
			Destination: destination,
			Source:      match[1],
			Default:     defaultValue,
			HasDefault:  true,
			Operation:   "copy",
		}, nil
	}
	if match := migrateRecordValuePattern.FindStringSubmatch(expression); match != nil {
		return runtimeMigrationFixtureTransform{
			Destination: destination,
			Source:      match[1],
			Operation:   "copy",
		}, nil
	}

	var value any
	if err := json.Unmarshal([]byte(expression), &value); err != nil {
		return runtimeMigrationFixtureTransform{}, fmt.Errorf("unsupported assignment expression %q", expression)
	}
	return runtimeMigrationFixtureTransform{
		Destination: destination,
		Operation:   "literal",
		Value:       value,
	}, nil
}

func decodeMigrationDefaultLiteral(raw string) (any, error) {
	value := decodeTALStringLiteral(raw)
	if value == "" && strings.TrimSpace(raw) != `""` && strings.TrimSpace(raw) != "''" {
		return nil, fmt.Errorf("invalid record.get default literal %q", raw)
	}
	return value, nil
}

func stripTALLineComment(line string) string {
	inString := false
	escaped := false
	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && inString {
			escaped = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if r == '#' && !inString {
			return line[:i]
		}
	}
	return line
}

func findRuntimeMigrationFixture(root string, step migrationPlanStep) (*runtimeMigrationFixture, error) {
	manifestPath := filepath.Join(root, "manifest.toml")
	var manifest runtimeMigrationManifest
	if _, err := toml.DecodeFile(manifestPath, &manifest); err != nil {
		return nil, nil
	}
	hasFixtureDeclarations := len(manifest.Migrate.Fixture) > 0

	var match *runtimeMigrationFixture
	for _, fixture := range manifest.Migrate.Fixture {
		stepIDRaw := strings.TrimSpace(fixture.Step)
		if stepIDRaw == "" {
			continue
		}
		stepID, ok := runtimeMigrationFixtureStepMatches(stepIDRaw, step)
		if !ok {
			continue
		}
		priorVersion := strings.TrimSpace(fixture.PriorVersion)
		if priorVersion != "" && priorVersion != step.FromVersion {
			return nil, fmt.Errorf("%w: migrate.fixture step %q prior_version %q does not match step from-version %q", ErrMigrationFixtureMismatch, fixture.Step, priorVersion, step.FromVersion)
		}
		if match != nil {
			return nil, fmt.Errorf("%w: duplicate migrate.fixture entries for step %04d", ErrMigrationFixtureMismatch, step.Number)
		}
		seedPath := strings.TrimSpace(fixture.Seed)
		expectedPath := strings.TrimSpace(fixture.Expected)
		if seedPath == "" || expectedPath == "" {
			return nil, fmt.Errorf("%w: migrate.fixture step %q must declare seed and expected files", ErrMigrationFixtureMismatch, fixture.Step)
		}
		match = &runtimeMigrationFixture{
			Step:            stepID,
			PriorVersion:    priorVersion,
			SeedPath:        seedPath,
			ExpectedPath:    expectedPath,
			ReadAdapterPath: strings.TrimSpace(fixture.ReadAdapter),
		}
	}
	if match == nil && hasFixtureDeclarations {
		return nil, fmt.Errorf("%w: step %04d missing migrate.fixture declaration", ErrMigrationFixtureUnavailable, step.Number)
	}

	return match, nil
}

func runtimeMigrationFixtureStepMatches(stepIDRaw string, step migrationPlanStep) (int, bool) {
	if stepID, err := strconv.Atoi(stepIDRaw); err == nil {
		return stepID, stepID == step.Number
	}
	stepName := strings.TrimSuffix(step.ScriptName, ".tal")
	if stepIDRaw == stepName {
		return step.Number, true
	}
	return 0, false
}

func readRuntimeFixtureRecords(root string, relPath string) (map[string]string, error) {
	fullPath, resolveErr := resolveRuntimeFixturePath(root, relPath)
	if resolveErr != nil {
		return nil, resolveErr
	}
	payload, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrMigrationFixtureUnavailable, relPath, err)
	}
	if bytes.Contains(payload, []byte{'\r'}) {
		return nil, fmt.Errorf("%w: %s must use LF line endings", ErrMigrationFixtureMismatch, relPath)
	}
	if len(payload) == 0 || payload[len(payload)-1] != '\n' {
		return nil, fmt.Errorf("%w: %s must end with trailing LF", ErrMigrationFixtureMismatch, relPath)
	}

	lines := bytes.Split(payload, []byte{'\n'})
	recordCount := len(lines) - 1
	if recordCount > runtimeMigrationFixtureMaxRows {
		return nil, fmt.Errorf("%w: %s exceeds maximum records (%d)", ErrMigrationFixtureMismatch, relPath, runtimeMigrationFixtureMaxRows)
	}

	records := make(map[string]string)
	previousKey := ""
	for i := 0; i < len(lines)-1; i++ {
		lineNumber := i + 1
		line := lines[i]
		if len(line) == 0 {
			return nil, fmt.Errorf("%w: %s line %d is blank", ErrMigrationFixtureMismatch, relPath, lineNumber)
		}

		key, canonicalEnvelope, canonicalValue, parseErr := parseRuntimeFixtureRecord(line)
		if parseErr != nil {
			return nil, fmt.Errorf("%w: %s line %d: %v", ErrMigrationFixtureMismatch, relPath, lineNumber, parseErr)
		}
		if !bytes.Equal(line, canonicalEnvelope) {
			return nil, fmt.Errorf("%w: %s line %d is not canonical JSON", ErrMigrationFixtureMismatch, relPath, lineNumber)
		}
		if previousKey != "" && strings.Compare(previousKey, key) >= 0 {
			return nil, fmt.Errorf("%w: %s line %d is out of key order", ErrMigrationFixtureMismatch, relPath, lineNumber)
		}
		if _, exists := records[key]; exists {
			return nil, fmt.Errorf("%w: %s duplicate key %q", ErrMigrationFixtureMismatch, relPath, key)
		}

		records[key] = canonicalValue
		previousKey = key
	}

	return records, nil
}

func resolveRuntimeFixturePath(root string, relPath string) (string, error) {
	cleanRoot := filepath.Clean(root)
	cleanRel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(relPath)))
	if cleanRel == "." || cleanRel == string(filepath.Separator) || filepath.IsAbs(cleanRel) {
		return "", fmt.Errorf("%w: %s must resolve within package root", ErrMigrationFixtureMismatch, relPath)
	}
	fullPath := filepath.Join(cleanRoot, cleanRel)
	relToRoot, err := filepath.Rel(cleanRoot, fullPath)
	if err != nil {
		return "", fmt.Errorf("%w: %s must resolve within package root", ErrMigrationFixtureMismatch, relPath)
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %s must resolve within package root", ErrMigrationFixtureMismatch, relPath)
	}
	resolvedRoot, err := filepath.EvalSymlinks(cleanRoot)
	if err == nil {
		resolvedPath, resolvedErr := filepath.EvalSymlinks(fullPath)
		if resolvedErr == nil {
			relResolved, relErr := filepath.Rel(resolvedRoot, resolvedPath)
			if relErr != nil || relResolved == ".." || strings.HasPrefix(relResolved, ".."+string(filepath.Separator)) {
				return "", fmt.Errorf("%w: %s must resolve within package root", ErrMigrationFixtureMismatch, relPath)
			}
		}
	}
	return fullPath, nil
}

func parseRuntimeFixtureRecord(line []byte) (key string, canonicalEnvelope []byte, canonicalValue string, err error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(line, &envelope); err != nil {
		return "", nil, "", errors.New("parse error")
	}
	if len(envelope) != 2 {
		return "", nil, "", errors.New("fixture record must contain exactly key and value fields")
	}

	rawKey, ok := envelope["key"]
	if !ok {
		return "", nil, "", errors.New("fixture record missing key field")
	}
	rawValue, ok := envelope["value"]
	if !ok {
		return "", nil, "", errors.New("fixture record missing value field")
	}

	if err := json.Unmarshal(rawKey, &key); err != nil {
		return "", nil, "", errors.New("fixture key must be a string")
	}
	if !utf8.ValidString(key) {
		return "", nil, "", errors.New("fixture key must be valid UTF-8")
	}
	if !norm.NFC.IsNormalString(key) {
		return "", nil, "", errors.New("fixture key must be NFC normalized")
	}
	if len([]byte(key)) == 0 || len([]byte(key)) > runtimeMigrationFixtureMaxKeyBytes {
		return "", nil, "", fmt.Errorf("fixture key byte length must be 1..%d", runtimeMigrationFixtureMaxKeyBytes)
	}

	var value map[string]any
	if err := json.Unmarshal(rawValue, &value); err != nil {
		return "", nil, "", errors.New("fixture value must be an object")
	}

	canonicalEnvelope, err = json.Marshal(map[string]any{
		"key":   key,
		"value": value,
	})
	if err != nil {
		return "", nil, "", errors.New("failed to canonicalize fixture record")
	}

	canonicalValue, err = canonicalJSONValue(rawValue)
	if err != nil {
		return "", nil, "", fmt.Errorf("invalid value: %w", err)
	}
	return key, canonicalEnvelope, canonicalValue, nil
}

func canonicalJSONValue(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", errors.New("empty json value")
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", err
	}
	canonical, err := json.Marshal(decoded)
	if err != nil {
		return "", err
	}
	return string(canonical), nil
}

type runtimeMigrationManifest struct {
	Migrate struct {
		DeclaredSteps       int  `toml:"declared_steps"`
		MaxRuntimeSeconds   *int `toml:"max_runtime_seconds"`
		CheckpointEvery     *int `toml:"checkpoint_every"`
		DrainTimeoutSeconds *int `toml:"drain_timeout_seconds"`
		Fixture             []struct {
			Step         string `toml:"step"`
			PriorVersion string `toml:"prior_version"`
			Seed         string `toml:"seed"`
			Expected     string `toml:"expected"`
			ReadAdapter  string `toml:"read_adapter"`
		} `toml:"fixture"`
		Step []struct {
			From          string `toml:"from"`
			To            string `toml:"to"`
			Compatibility string `toml:"compatibility"`
			DrainPolicy   string `toml:"drain_policy"`
		} `toml:"step"`
	} `toml:"migrate"`
}

func loadMigrationPlan(root string) (int, []migrationPlanStep, error) {
	matches, err := filepath.Glob(filepath.Join(root, "migrate", "*.tal"))
	if err != nil {
		return 0, nil, nil
	}
	if len(matches) == 0 {
		return 0, nil, nil
	}

	steps := make([]migrationPlanStep, 0, len(matches))
	for _, match := range matches {
		base := filepath.Base(match)
		parts := migrateStepFilePattern.FindStringSubmatch(base)
		if parts == nil {
			return len(matches), nil, fmt.Errorf("%w: migration script %s must match <step>_<from>_to_<to>.tal", ErrInvalidManifest, base)
		}
		stepNumber, err := strconv.Atoi(parts[1])
		if err != nil || stepNumber <= 0 {
			return len(matches), nil, fmt.Errorf("%w: migration script %s has invalid step number", ErrInvalidManifest, base)
		}
		steps = append(steps, migrationPlanStep{
			Number:      stepNumber,
			FromVersion: strings.TrimSpace(parts[2]),
			ToVersion:   strings.TrimSpace(parts[3]),
			ScriptName:  base,
		})
	}

	sort.Slice(steps, func(i, j int) bool {
		return steps[i].Number < steps[j].Number
	})
	for i := range steps {
		expected := i + 1
		if steps[i].Number != expected {
			return len(matches), nil, fmt.Errorf("%w: migration step numbering gap: expected step %04d, found %04d", ErrInvalidManifest, expected, steps[i].Number)
		}
	}

	manifestPath := filepath.Join(root, "manifest.toml")
	var manifest runtimeMigrationManifest
	if _, err := toml.DecodeFile(manifestPath, &manifest); err == nil {
		if manifest.Migrate.DrainTimeoutSeconds != nil && *manifest.Migrate.DrainTimeoutSeconds <= 0 {
			return len(matches), nil, fmt.Errorf("%w: migrate.drain_timeout_seconds must be a positive integer", ErrInvalidManifest)
		}
		if manifest.Migrate.MaxRuntimeSeconds != nil && *manifest.Migrate.MaxRuntimeSeconds <= 0 {
			return len(matches), nil, fmt.Errorf("%w: migrate.max_runtime_seconds must be a positive integer", ErrInvalidManifest)
		}
		if manifest.Migrate.CheckpointEvery != nil && *manifest.Migrate.CheckpointEvery <= 0 {
			return len(matches), nil, fmt.Errorf("%w: migrate.checkpoint_every must be a positive integer", ErrInvalidManifest)
		}
		if len(manifest.Migrate.Step) > 0 {
			if manifest.Migrate.DeclaredSteps > 0 && manifest.Migrate.DeclaredSteps != len(manifest.Migrate.Step) {
				return len(matches), nil, fmt.Errorf("%w: migrate.declared_steps (%d) does not match migrate.step entries (%d)", ErrInvalidManifest, manifest.Migrate.DeclaredSteps, len(manifest.Migrate.Step))
			}
			if len(manifest.Migrate.Step) != len(steps) {
				return len(matches), nil, fmt.Errorf("%w: migrate.step entries (%d) do not match migrate scripts (%d)", ErrInvalidManifest, len(manifest.Migrate.Step), len(steps))
			}
			for i := range steps {
				manifestStep := manifest.Migrate.Step[i]
				steps[i].Compatibility = strings.TrimSpace(manifestStep.Compatibility)
				steps[i].DrainPolicy = strings.TrimSpace(manifestStep.DrainPolicy)
				steps[i].RequiresDrain = strings.EqualFold(steps[i].Compatibility, "incompatible") && strings.EqualFold(steps[i].DrainPolicy, "drain")
				if strings.TrimSpace(manifestStep.From) != "" && strings.TrimSpace(manifestStep.To) != "" {
					if strings.TrimSpace(manifestStep.From) != steps[i].FromVersion || strings.TrimSpace(manifestStep.To) != steps[i].ToVersion {
						return len(matches), nil, fmt.Errorf("%w: migrate.step %04d from/to does not match script %s", ErrInvalidManifest, i+1, steps[i].ScriptName)
					}
				}
			}
		}
	}

	return len(steps), steps, nil
}

func migrationPlanPendingSteps(plan []migrationPlanStep, nextStep int) []migrationPlanStep {
	if nextStep < 1 {
		nextStep = 1
	}
	out := make([]migrationPlanStep, 0, len(plan))
	for _, step := range plan {
		if step.Number < nextStep {
			continue
		}
		out = append(out, step)
	}
	return out
}

func migrationPlanRequiresDrainFromStep(plan []migrationPlanStep, nextStep int) bool {
	for _, step := range migrationPlanPendingSteps(plan, nextStep) {
		if step.RequiresDrain {
			return true
		}
	}
	return false
}

func rootOrFallbackPath(pkg Package) string {
	if strings.TrimSpace(pkg.RootPath) != "" {
		return pkg.RootPath
	}
	return "."
}

func packageDrainTimeout(root string) time.Duration {
	manifestPath := filepath.Join(root, "manifest.toml")
	var manifest runtimeMigrationManifest
	if _, err := toml.DecodeFile(manifestPath, &manifest); err != nil {
		return defaultMigrationDrainTimeout
	}
	if manifest.Migrate.DrainTimeoutSeconds == nil {
		return defaultMigrationDrainTimeout
	}
	return time.Duration(*manifest.Migrate.DrainTimeoutSeconds) * time.Second
}

func packageMigrationMaxRuntime(root string) time.Duration {
	var manifest runtimeMigrationManifest
	if _, err := toml.DecodeFile(filepath.Join(root, "manifest.toml"), &manifest); err != nil {
		return 0
	}
	if manifest.Migrate.MaxRuntimeSeconds == nil || *manifest.Migrate.MaxRuntimeSeconds <= 0 {
		return 0
	}
	return time.Duration(*manifest.Migrate.MaxRuntimeSeconds) * time.Second
}

func packageMigrationCheckpointEvery(root string) int {
	var manifest runtimeMigrationManifest
	if _, err := toml.DecodeFile(filepath.Join(root, "manifest.toml"), &manifest); err != nil {
		return 0
	}
	if manifest.Migrate.CheckpointEvery == nil || *manifest.Migrate.CheckpointEvery <= 0 {
		return 0
	}
	return *manifest.Migrate.CheckpointEvery
}

func countDowngradeMigrationSteps(root string) int {
	matches, err := filepath.Glob(filepath.Join(root, "migrate", "downgrade", "*.tal"))
	if err != nil {
		return 0
	}
	valid := 0
	for _, match := range matches {
		if migrateStepFilePattern.MatchString(filepath.Base(match)) {
			valid++
		}
	}
	return valid
}

func migrationJournalPath(pkg Package) string {
	return filepath.ToSlash(filepath.Join("apps", migrationIdentity(pkg.Manifest), "migrate", fmt.Sprintf("r%d", pkg.Revision), "journal.ndjson"))
}

func migrationReconciliationPath(pkg Package) string {
	return filepath.ToSlash(filepath.Join("apps", migrationIdentity(pkg.Manifest), "migrate", fmt.Sprintf("r%d", pkg.Revision), "reconcile.json"))
}

func migrationIdentity(manifest Manifest) string {
	appID := strings.TrimSpace(manifest.AppID)
	if appID != "" {
		return appID
	}
	return strings.TrimSpace(manifest.Name)
}

func appendMigrationJournalEntry(pkg Package, state migrationState, event string, fields map[string]any) {
	if strings.TrimSpace(state.JournalPath) == "" {
		return
	}

	absolutePath := filepath.Join(pkg.RootPath, filepath.FromSlash(state.JournalPath))
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(filepath.Clean(absolutePath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer func() {
		_ = file.Close()
	}()

	entry := map[string]any{
		"ts":              time.Now().UTC().Format(time.RFC3339Nano),
		"event":           strings.TrimSpace(event),
		"step":            state.LastStep,
		"steps_completed": state.StepsCompleted,
		"steps_planned":   state.StepsPlanned,
		"verdict":         state.Verdict,
		"last_error":      state.LastError,
	}
	for key, value := range fields {
		entry[key] = value
	}

	payload, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_, _ = file.Write(append(payload, '\n'))
}

func statusFromState(pkg Package, state migrationState) MigrationStatus {
	recordIDs := make([]string, 0, len(state.PendingRecords))
	for recordID := range state.PendingRecords {
		recordIDs = append(recordIDs, recordID)
	}
	sort.Strings(recordIDs)
	records := make([]MigrationReconciliationRecord, 0, len(recordIDs))
	for _, recordID := range recordIDs {
		records = append(records, MigrationReconciliationRecord{
			RecordID:              recordID,
			RecommendedResolution: state.PendingRecords[recordID],
		})
	}

	return MigrationStatus{
		App:                pkg.Manifest.Name,
		Version:            pkg.Manifest.Version,
		Revision:           pkg.Revision,
		StepsPlanned:       state.StepsPlanned,
		StepsCompleted:     state.StepsCompleted,
		LastStep:           state.LastStep,
		Verdict:            state.Verdict,
		LastError:          state.LastError,
		JournalPath:        state.JournalPath,
		ReconciliationPath: state.ReconciliationPath,
		ExecutorReady:      state.ExecutorReady,
		PendingRecords:     records,
	}
}

func isAllowedMigrationResolution(resolution string) bool {
	resolution = strings.TrimSpace(resolution)
	switch resolution {
	case "accept_current", "force_rewind", "manual":
		return true
	default:
		return false
	}
}

// GetPackage returns the latest loaded package for one app name.
func (r *Runtime) GetPackage(name string) (Package, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pkg, ok := r.packages[name]
	return pkg, ok
}

// GetPackageByRevision returns a previously loaded package revision.
func (r *Runtime) GetPackageByRevision(name string, revision uint64) (Package, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, pkg := range r.history[name] {
		if pkg.Revision == revision {
			return pkg, true
		}
	}
	return Package{}, false
}

// ListPackages returns loaded package names.
func (r *Runtime) ListPackages() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.packages))
	for name := range r.packages {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// ListPackageHistory returns the loaded versions for one package, oldest first.
func (r *Runtime) ListPackageHistory(name string) []Package {
	r.mu.RLock()
	defer r.mu.RUnlock()
	history := r.history[name]
	out := make([]Package, len(history))
	copy(out, history)
	return out
}

func (r *Runtime) packageFromDisk(root string) (Package, error) {
	manifestPath := filepath.Join(root, "manifest.toml")
	manifest, err := parseManifest(manifestPath)
	if err != nil {
		return Package{}, err
	}
	if err := validateManifest(manifest, r.kernelAPIVersion); err != nil {
		return Package{}, err
	}

	mainPath := filepath.Join(root, "main.tal")
	if _, err := os.Stat(mainPath); err != nil {
		return Package{}, ErrInvalidManifest
	}
	if err := validatePathsExist(root, manifest.Kernels); err != nil {
		return Package{}, err
	}
	if err := validatePathsExist(root, manifest.Models); err != nil {
		return Package{}, err
	}

	digest, err := collectDigest(root)
	if err != nil {
		return Package{}, err
	}
	return Package{
		RootPath:   root,
		Manifest:   manifest,
		MainPath:   mainPath,
		LoadedAt:   time.Now().UTC(),
		FileDigest: digest,
	}, nil
}

func validateManifest(manifest Manifest, kernelAPIVersion string) error {
	if strings.TrimSpace(manifest.Name) == "" ||
		strings.TrimSpace(manifest.Version) == "" ||
		strings.TrimSpace(manifest.Language) == "" {
		return ErrInvalidManifest
	}
	if !isValidAppID(manifest.AppID) {
		return ErrInvalidManifest
	}
	if manifest.Language != LanguageTALV1 {
		return ErrInvalidManifest
	}
	requires := strings.TrimSpace(manifest.RequiresKernelAPI)
	if requires != "" && !strings.EqualFold(requires, strings.TrimSpace(kernelAPIVersion)) {
		return ErrKernelAPIIncompatible
	}
	for _, permission := range manifest.Permissions {
		if _, ok := allowedPermissions[permission]; !ok {
			return ErrPermissionDenied
		}
	}
	if strings.Contains(manifest.Name, " ") {
		return ErrInvalidManifest
	}
	return nil
}

func parseManifest(path string) (Manifest, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return Manifest{}, ErrInvalidManifest
	}
	defer func() {
		_ = file.Close()
	}()

	manifest := Manifest{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		raw := strings.TrimSpace(parts[1])

		switch key {
		case "name":
			manifest.Name = parseString(raw)
		case "app_id":
			manifest.AppID = parseString(raw)
		case "version":
			manifest.Version = parseString(raw)
		case "language":
			manifest.Language = parseString(raw)
		case "requires_kernel_api":
			manifest.RequiresKernelAPI = parseString(raw)
		case "description":
			manifest.Description = parseString(raw)
		case "permissions":
			manifest.Permissions = parseStringArray(raw)
		case "exports":
			manifest.Exports = parseStringArray(raw)
		case "kernels":
			manifest.Kernels = parseStringArray(raw)
		case "models":
			manifest.Models = parseStringArray(raw)
		case "migrate":
			manifest.Migrate = parseString(raw)
		case "dev_mode":
			manifest.DevMode = parseBool(raw)
		}
	}
	if err := scanner.Err(); err != nil {
		return Manifest{}, ErrInvalidManifest
	}

	manifest.Permissions = normalizeStringSet(manifest.Permissions)
	manifest.Exports = normalizeStringSet(manifest.Exports)
	manifest.Kernels = normalizeStringSet(manifest.Kernels)
	manifest.Models = normalizeStringSet(manifest.Models)
	return manifest, nil
}

func isValidAppID(appID string) bool {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return true
	}
	const prefix = "app:sha256:"
	if !strings.HasPrefix(appID, prefix) {
		return false
	}
	hexPart := strings.TrimPrefix(appID, prefix)
	if len(hexPart) != 64 {
		return false
	}
	for _, r := range hexPart {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func collectDigest(root string) (map[string]time.Time, error) {
	digest := make(map[string]time.Time)
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		digest[path] = info.ModTime().UTC()
		return nil
	})
	return digest, err
}

func hasDigestDiff(a, b map[string]time.Time) bool {
	if len(a) != len(b) {
		return true
	}
	for k, v := range a {
		if ov, ok := b[k]; !ok || !ov.Equal(v) {
			return true
		}
	}
	return false
}

func validatePathsExist(root string, relPaths []string) error {
	for _, rel := range relPaths {
		clean := filepath.Clean(strings.TrimSpace(rel))
		if clean == "." || clean == "" {
			continue
		}
		if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			return fmt.Errorf("%w: invalid relative path %q", ErrInvalidManifest, rel)
		}
		full := filepath.Join(root, clean)
		info, err := os.Stat(full)
		if err != nil || info.IsDir() {
			return fmt.Errorf("%w: missing path %q", ErrInvalidManifest, rel)
		}
	}
	return nil
}

func parseString(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "\"")
	trimmed = strings.TrimSuffix(trimmed, "\"")
	return strings.TrimSpace(trimmed)
}

func parseBool(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	val, err := strconv.ParseBool(trimmed)
	if err != nil {
		return strings.EqualFold(parseString(trimmed), "true")
	}
	return val
}

func parseStringArray(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
		if single := parseString(trimmed); single != "" {
			return []string{single}
		}
		return nil
	}
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]"))
	if inner == "" {
		return nil
	}
	parts := strings.Split(inner, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := parseString(part); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func normalizeStringSet(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, val := range in {
		trimmed := strings.TrimSpace(val)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}
