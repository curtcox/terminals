// Package appruntime loads and hot-reloads TAR/TAL application packages.
package appruntime

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
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
	// ErrMigrationExecutorUnavailable indicates migration actions are not wired yet.
	ErrMigrationExecutorUnavailable = errors.New("migration executor unavailable")
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
	mu               sync.RWMutex
	kernelAPIVersion string
	nextRevision     uint64
	packages         map[string]Package
	history          map[string][]Package
	migrations       map[string]migrationState
}

type migrationState struct {
	StepsPlanned       int
	StepsCompleted     int
	LastStep           int
	Verdict            string
	LastError          string
	JournalPath        string
	ReconciliationPath string
	ExecutorReady      bool
	RequiresDrain      bool
	DrainReady         bool
	PendingRecords     map[string]string
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

// LoadPackage parses and registers one app package.
func (r *Runtime) LoadPackage(ctx context.Context, root string) (Package, error) {
	_ = ctx
	pkg, err := r.packageFromDisk(root)
	if err != nil {
		return Package{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	pkg.Revision = r.nextRevision
	r.nextRevision++
	r.packages[pkg.Manifest.Name] = pkg
	r.history[pkg.Manifest.Name] = append(r.history[pkg.Manifest.Name], pkg)
	r.migrations[pkg.Manifest.Name] = newMigrationState(pkg)
	return pkg, nil
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
	r.migrations[name] = newMigrationState(next)
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
	r.migrations[name] = newMigrationState(previous)
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
	if state.Verdict == "reconcile_pending" && len(state.PendingRecords) > 0 {
		state.LastError = ErrMigrationReconcilePending.Error()
		r.migrations[name] = state
		appendMigrationJournalEntry(pkg, state, "retry_blocked_reconcile_pending", nil)
		return statusFromState(pkg, state), ErrMigrationReconcilePending
	}
	if state.RequiresDrain && !state.DrainReady {
		state.Verdict = "aborted"
		state.LastError = ErrMigrationDrainTimeout.Error()
		r.migrations[name] = state
		appendMigrationJournalEntry(pkg, state, "retry_blocked_drain_timeout", nil)
		return statusFromState(pkg, state), ErrMigrationDrainTimeout
	}

	nextStep := state.StepsCompleted + 1
	if nextStep < 1 {
		nextStep = 1
	}

	state.Verdict = "running"
	state.LastError = ""
	appendMigrationJournalEntry(pkg, state, "retry_started", map[string]any{"from_step": nextStep})
	for step := nextStep; step <= state.StepsPlanned; step++ {
		state.LastStep = step
		appendMigrationJournalEntry(pkg, state, "step_started", map[string]any{"step_id": step})
		state.StepsCompleted = step
		state.LastStep = step
		appendMigrationJournalEntry(pkg, state, "step_committed", map[string]any{"step_id": step})
	}
	if state.StepsCompleted > state.StepsPlanned {
		state.StepsCompleted = state.StepsPlanned
	}
	state.LastStep = state.StepsCompleted
	state.Verdict = "ok"
	state.JournalPath = migrationJournalPath(pkg)
	r.migrations[name] = state
	appendMigrationJournalEntry(pkg, state, "retry_committed", map[string]any{"from_step": nextStep, "to_step": state.StepsCompleted})
	return statusFromState(pkg, state), nil
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
	if ready && state.Verdict == "aborted" && state.LastError == ErrMigrationDrainTimeout.Error() {
		state.Verdict = "idle"
		state.LastError = ""
	}
	r.migrations[name] = state
	return nil
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
		state.StepsCompleted = 0
		state.LastStep = 0
		state.LastError = "aborted to baseline by operator"
	} else {
		if state.StepsCompleted > 0 {
			state.StepsCompleted--
		}
		if state.StepsCompleted < 0 {
			state.StepsCompleted = 0
		}
		state.LastStep = state.StepsCompleted
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
	if !state.ExecutorReady {
		return statusFromState(pkg, state), ErrMigrationExecutorUnavailable
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
		state = newMigrationState(pkg)
		r.migrations[name] = state
	}
	return pkg, state, nil
}

func newMigrationState(pkg Package) migrationState {
	steps := countMigrationSteps(pkg.RootPath)
	requiresDrain := packageRequiresDrainMigration(pkg.RootPath)
	state := migrationState{
		StepsPlanned:   steps,
		StepsCompleted: 0,
		LastStep:       0,
		Verdict:        "idle",
		ExecutorReady:  steps > 0,
		RequiresDrain:  requiresDrain,
		DrainReady:     !requiresDrain,
	}
	if steps > 0 {
		state.JournalPath = migrationJournalPath(pkg)
	}
	return state
}

type runtimeMigrationManifest struct {
	Migrate struct {
		Step []struct {
			Compatibility string `toml:"compatibility"`
			DrainPolicy   string `toml:"drain_policy"`
		} `toml:"step"`
	} `toml:"migrate"`
}

func packageRequiresDrainMigration(root string) bool {
	manifestPath := filepath.Join(root, "manifest.toml")
	var manifest runtimeMigrationManifest
	if _, err := toml.DecodeFile(manifestPath, &manifest); err != nil {
		return false
	}
	for _, step := range manifest.Migrate.Step {
		compatibility := strings.ToLower(strings.TrimSpace(step.Compatibility))
		drainPolicy := strings.ToLower(strings.TrimSpace(step.DrainPolicy))
		if compatibility == "incompatible" && drainPolicy == "drain" {
			return true
		}
	}
	return false
}

func countMigrationSteps(root string) int {
	matches, err := filepath.Glob(filepath.Join(root, "migrate", "*.tal"))
	if err != nil {
		return 0
	}
	return len(matches)
}

func countDowngradeMigrationSteps(root string) int {
	matches, err := filepath.Glob(filepath.Join(root, "migrate", "downgrade", "*.tal"))
	if err != nil {
		return 0
	}
	return len(matches)
}

func migrationJournalPath(pkg Package) string {
	return filepath.ToSlash(filepath.Join("apps", pkg.Manifest.Name, "migrate", fmt.Sprintf("r%d", pkg.Revision), "journal.ndjson"))
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
