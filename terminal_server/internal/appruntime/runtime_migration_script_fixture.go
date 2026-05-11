// TAL migration script validation, host-effect collection, and fixture execution.
package appruntime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	"golang.org/x/text/unicode/norm"
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
		if artifactID == "" {
			return effects, fmt.Errorf("%w: artifact.self.patch missing artifact_id at line %d", ErrMigrationArtifactOwnership, lineNumber+1)
		}
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
		reason := decodeTALStringLiteral(match[1])
		if reason == "" && strings.TrimSpace(match[1]) != `""` && strings.TrimSpace(match[1]) != "''" {
			return "", true, fmt.Errorf("invalid abort reason literal %q", match[1])
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

type runtimeFixtureValueDiff struct {
	Offset       int
	ExpectedByte string
	ActualByte   string
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
		if line == "" || strings.HasPrefix(line, "def ") || line == "pass" || line == "continue" || strings.HasPrefix(line, "load(") || strings.HasPrefix(line, "return ") {
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
		if match := migrateRecordSkipIfPresentBlockPattern.FindStringSubmatch(line); match != nil {
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
			line == "return label.strip().lower()" ||
			line == "if len(page) == 0:" || line == "break" || line == "continue" {
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
		if match := migrateRecordSkipIfPresentBlockPattern.FindStringSubmatch(recordLine); match != nil {
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

	if strings.HasPrefix(expression, "'") {
		value := decodeTALStringLiteral(expression)
		if value == "" && expression != "''" {
			return runtimeMigrationFixtureTransform{}, fmt.Errorf("unsupported assignment expression %q", expression)
		}
		return runtimeMigrationFixtureTransform{
			Destination: destination,
			Operation:   "literal",
			Value:       value,
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
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, `"`) || strings.HasPrefix(raw, `'`) {
		value := decodeTALStringLiteral(raw)
		if value == "" && raw != `""` && raw != "''" {
			return nil, fmt.Errorf("invalid record.get default literal %q", raw)
		}
		return value, nil
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, fmt.Errorf("invalid record.get default literal %q", raw)
	}
	return value, nil
}

func stripTALLineComment(line string) string {
	var quote rune
	escaped := false
	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && quote != 0 {
			escaped = true
			continue
		}
		if r == '"' || r == '\'' {
			switch quote {
			case 0:
				quote = r
			case r:
				quote = 0
			}
			continue
		}
		if r == '#' && quote == 0 {
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
