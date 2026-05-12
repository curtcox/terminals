package appruntime

// TAL migration script validation, fixture types, verification, and execution.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

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

type runtimeMigrationFixtureTransform struct {
	Destination string
	Source      string
	Default     any
	HasDefault  bool
	Operation   string
	Value       any
	Reason      string
}

type runtimeFixtureValueDiff struct {
	Offset       int
	ExpectedByte string
	ActualByte   string
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
			return runtimeMigrationResourceStats{}, runtimeFixtureValueMismatchError(step.Number, "", key, expectedValue, actualValue)
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
	for lineNumber, rawLine := range strings.Split(string(payload), "\n") {
		line := strings.TrimSpace(stripTALLineComment(rawLine))
		if !strings.HasPrefix(line, "return") {
			continue
		}
		if !migrateReadAdapterIdentityReturnPattern.MatchString(line) {
			return fmt.Errorf("line %d uses unsupported read_adapter return expression %q", lineNumber+1, line)
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
			return runtimeFixtureValueMismatchError(step, label, key, expectedValue, actualValue)
		}
	}
	for key := range expectedRecords {
		if _, ok := actualRecords[key]; !ok {
			return fmt.Errorf("%w: step %04d %s expected contains extra key %q", ErrMigrationFixtureMismatch, step, label, key)
		}
	}
	return nil
}

func runtimeFixtureValueMismatchError(step int, label string, key string, expectedValue string, actualValue string) error {
	prefix := fmt.Sprintf("step %04d", step)
	if label != "" {
		prefix += " " + label
	}
	diff := firstFixtureValueDiff(expectedValue, actualValue)
	return fmt.Errorf("%w: %s value mismatch for key %q: expected=%s actual=%s first_diff_byte=%d expected_byte=%s actual_byte=%s", ErrMigrationFixtureMismatch, prefix, key, expectedValue, actualValue, diff.Offset, diff.ExpectedByte, diff.ActualByte)
}

func firstFixtureValueDiff(expectedValue string, actualValue string) runtimeFixtureValueDiff {
	minLen := len(expectedValue)
	if len(actualValue) < minLen {
		minLen = len(actualValue)
	}
	for i := 0; i < minLen; i++ {
		if expectedValue[i] != actualValue[i] {
			return runtimeFixtureValueDiff{
				Offset:       i,
				ExpectedByte: fmt.Sprintf("0x%02x", expectedValue[i]),
				ActualByte:   fmt.Sprintf("0x%02x", actualValue[i]),
			}
		}
	}
	if len(expectedValue) != len(actualValue) {
		diff := runtimeFixtureValueDiff{Offset: minLen}
		if len(expectedValue) > minLen {
			diff.ExpectedByte = fmt.Sprintf("0x%02x", expectedValue[minLen])
		} else {
			diff.ExpectedByte = "<eof>"
		}
		if len(actualValue) > minLen {
			diff.ActualByte = fmt.Sprintf("0x%02x", actualValue[minLen])
		} else {
			diff.ActualByte = "<eof>"
		}
		return diff
	}
	return runtimeFixtureValueDiff{Offset: -1, ExpectedByte: "<none>", ActualByte: "<none>"}
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
	storeOps := 0
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
		canonicalValue := string(canonical)
		out[key] = canonicalValue
		if canonicalValue != rawValue {
			storeOps++
			writeVolume += int64(len(canonical))
		}
	}
	stats.StoreOps = storeOps
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
		canonicalValue := string(canonical)
		if canonicalValue != out[key] {
			out[key] = canonicalValue
			stats.StoreOps++
			stats.WriteVolumeBytes += int64(len(canonical))
		}
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
