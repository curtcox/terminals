package appruntime

// manifest.toml migration plan loading and migration path helpers.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type migrationManifestFixture struct {
	Step         string `toml:"step"`
	PriorVersion string `toml:"prior_version"`
	Seed         string `toml:"seed"`
	Expected     string `toml:"expected"`
	ReadAdapter  string `toml:"read_adapter"`
}

type runtimeMigrationManifest struct {
	Migrate struct {
		DeclaredSteps       int                        `toml:"declared_steps"`
		MaxRuntimeSeconds   *int                       `toml:"max_runtime_seconds"`
		CheckpointEvery     *int                       `toml:"checkpoint_every"`
		DrainTimeoutSeconds *int                       `toml:"drain_timeout_seconds"`
		Fixture             []migrationManifestFixture `toml:"fixture"`
		Step                []struct {
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

	steps, err := migrationStepsFromScripts(matches)
	if err != nil {
		return len(matches), nil, err
	}
	if err := validateMigrationStepNumbering(steps); err != nil {
		return len(matches), nil, err
	}

	manifestPath := filepath.Join(root, "manifest.toml")
	var manifest runtimeMigrationManifest
	if _, err := toml.DecodeFile(manifestPath, &manifest); err == nil {
		if err := validateMigrationManifestNumericFields(manifest); err != nil {
			return len(matches), nil, err
		}
		if err := enrichMigrationStepsFromManifest(manifest, steps); err != nil {
			return len(matches), nil, err
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
		RequiresDrain:      state.RequiresDrain,
		DrainReady:         state.DrainReady,
		DrainTimeout:       state.DrainTimeout,
		DrainBlockedAt:     state.DrainBlockedAt,
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
