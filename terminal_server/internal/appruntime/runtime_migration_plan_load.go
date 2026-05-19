package appruntime

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func migrationStepsFromScripts(matches []string) ([]migrationPlanStep, error) {
	steps := make([]migrationPlanStep, 0, len(matches))
	for _, match := range matches {
		base := filepath.Base(match)
		parts := migrateStepFilePattern.FindStringSubmatch(base)
		if parts == nil {
			return nil, fmt.Errorf("%w: migration script %s must match <step>_<from>_to_<to>.tal", ErrInvalidManifest, base)
		}
		stepNumber, err := strconv.Atoi(parts[1])
		if err != nil || stepNumber <= 0 {
			return nil, fmt.Errorf("%w: migration script %s has invalid step number", ErrInvalidManifest, base)
		}
		steps = append(steps, migrationPlanStep{
			Number:      stepNumber,
			FromVersion: strings.TrimSpace(parts[2]),
			ToVersion:   strings.TrimSpace(parts[3]),
			ScriptName:  base,
		})
	}
	return steps, nil
}

func validateMigrationStepNumbering(steps []migrationPlanStep) error {
	sort.Slice(steps, func(i, j int) bool {
		return steps[i].Number < steps[j].Number
	})
	for i := range steps {
		expected := i + 1
		if steps[i].Number != expected {
			return fmt.Errorf("%w: migration step numbering gap: expected step %04d, found %04d", ErrInvalidManifest, expected, steps[i].Number)
		}
	}
	return nil
}

func validateMigrationManifestNumericFields(manifest runtimeMigrationManifest) error {
	if manifest.Migrate.DrainTimeoutSeconds != nil && *manifest.Migrate.DrainTimeoutSeconds <= 0 {
		return fmt.Errorf("%w: migrate.drain_timeout_seconds must be a positive integer", ErrInvalidManifest)
	}
	if manifest.Migrate.MaxRuntimeSeconds != nil && *manifest.Migrate.MaxRuntimeSeconds <= 0 {
		return fmt.Errorf("%w: migrate.max_runtime_seconds must be a positive integer", ErrInvalidManifest)
	}
	if manifest.Migrate.CheckpointEvery != nil && *manifest.Migrate.CheckpointEvery <= 0 {
		return fmt.Errorf("%w: migrate.checkpoint_every must be a positive integer", ErrInvalidManifest)
	}
	return nil
}

type migrationManifestStepEntry struct {
	From          string
	To            string
	Compatibility string
	DrainPolicy   string
}

func validateMigrationManifestStepEntry(step migrationPlanStep, manifestStep migrationManifestStepEntry, index int) error {
	stepNum := index + 1
	if step.Compatibility == "" {
		return fmt.Errorf("%w: migrate.step %04d must declare compatibility", ErrInvalidManifest, stepNum)
	}
	if step.DrainPolicy == "" {
		return fmt.Errorf("%w: migrate.step %04d must declare drain_policy", ErrInvalidManifest, stepNum)
	}
	if step.Compatibility != "compatible" && step.Compatibility != "incompatible" {
		return fmt.Errorf("%w: migrate.step %04d has invalid compatibility %q", ErrInvalidManifest, stepNum, step.Compatibility)
	}
	if step.DrainPolicy != "none" && step.DrainPolicy != "drain" && step.DrainPolicy != "multi_version" {
		return fmt.Errorf("%w: migrate.step %04d has invalid drain_policy %q", ErrInvalidManifest, stepNum, step.DrainPolicy)
	}
	if step.Compatibility == "incompatible" && step.DrainPolicy == "none" {
		return fmt.Errorf("%w: migrate.step %04d declares compatibility=incompatible with drain_policy=none", ErrInvalidManifest, stepNum)
	}
	from := strings.TrimSpace(manifestStep.From)
	to := strings.TrimSpace(manifestStep.To)
	if from != "" && to != "" && (from != step.FromVersion || to != step.ToVersion) {
		return fmt.Errorf("%w: migrate.step %04d from/to does not match script %s", ErrInvalidManifest, stepNum, step.ScriptName)
	}
	return nil
}

func enrichMigrationStepsFromManifest(manifest runtimeMigrationManifest, steps []migrationPlanStep) error {
	if len(manifest.Migrate.Step) == 0 {
		return nil
	}
	if manifest.Migrate.DeclaredSteps > 0 && manifest.Migrate.DeclaredSteps != len(manifest.Migrate.Step) {
		return fmt.Errorf("%w: migrate.declared_steps (%d) does not match migrate.step entries (%d)", ErrInvalidManifest, manifest.Migrate.DeclaredSteps, len(manifest.Migrate.Step))
	}
	if len(manifest.Migrate.Step) != len(steps) {
		return fmt.Errorf("%w: migrate.step entries (%d) do not match migrate scripts (%d)", ErrInvalidManifest, len(manifest.Migrate.Step), len(steps))
	}
	for i := range steps {
		raw := manifest.Migrate.Step[i]
		steps[i].Compatibility = strings.TrimSpace(raw.Compatibility)
		steps[i].DrainPolicy = strings.TrimSpace(raw.DrainPolicy)
		manifestStep := migrationManifestStepEntry{
			From:          raw.From,
			To:            raw.To,
			Compatibility: raw.Compatibility,
			DrainPolicy:   raw.DrainPolicy,
		}
		if err := validateMigrationManifestStepEntry(steps[i], manifestStep, i); err != nil {
			return err
		}
		steps[i].RequiresDrain = strings.EqualFold(steps[i].Compatibility, "incompatible") && strings.EqualFold(steps[i].DrainPolicy, "drain")
	}
	return nil
}
