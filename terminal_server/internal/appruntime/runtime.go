// Package appruntime loads and hot-reloads TAR/TAL application packages.
package appruntime

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
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
	Verdict            string
	LastError          string
	JournalPath        string
	ReconciliationPath string
	ExecutorReady      bool
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
	return next, true, nil
}

// RollbackPackage restores the previous successfully loaded package version.
func (r *Runtime) RollbackPackage(name string) (Package, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	history := r.history[name]
	if len(history) == 0 {
		return Package{}, ErrPackageNotFound
	}
	if len(history) < 2 {
		return Package{}, ErrNoPriorVersion
	}
	history = history[:len(history)-1]
	r.history[name] = history
	previous := history[len(history)-1]
	r.packages[name] = previous
	return previous, nil
}

// GetMigrationStatus returns migration status for one app package.
func (r *Runtime) GetMigrationStatus(name string) (MigrationStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pkg, ok := r.packages[name]
	if !ok {
		return MigrationStatus{}, ErrPackageNotFound
	}
	return MigrationStatus{
		App:                pkg.Manifest.Name,
		Version:            pkg.Manifest.Version,
		Revision:           pkg.Revision,
		StepsPlanned:       0,
		StepsCompleted:     0,
		Verdict:            "idle",
		LastError:          "",
		JournalPath:        "",
		ReconciliationPath: "",
		ExecutorReady:      false,
	}, nil
}

// RetryMigration retries an app migration run.
func (r *Runtime) RetryMigration(name string) (MigrationStatus, error) {
	status, err := r.GetMigrationStatus(name)
	if err != nil {
		return MigrationStatus{}, err
	}
	return status, ErrMigrationExecutorUnavailable
}

// AbortMigration aborts an in-flight app migration run.
func (r *Runtime) AbortMigration(name string) (MigrationStatus, error) {
	status, err := r.GetMigrationStatus(name)
	if err != nil {
		return MigrationStatus{}, err
	}
	return status, ErrMigrationExecutorUnavailable
}

// ReconcileMigration attempts to reconcile one migration record.
func (r *Runtime) ReconcileMigration(name, recordID, resolution string) (MigrationStatus, error) {
	_ = recordID
	_ = resolution
	status, err := r.GetMigrationStatus(name)
	if err != nil {
		return MigrationStatus{}, err
	}
	return status, ErrMigrationExecutorUnavailable
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
