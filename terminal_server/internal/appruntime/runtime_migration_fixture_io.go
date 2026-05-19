package appruntime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	"golang.org/x/text/unicode/norm"
)

func findRuntimeMigrationFixture(root string, step migrationPlanStep) (*runtimeMigrationFixture, error) {
	manifestPath := filepath.Join(root, "manifest.toml")
	var manifest runtimeMigrationManifest
	if _, err := toml.DecodeFile(manifestPath, &manifest); err != nil {
		return nil, nil
	}
	hasFixtureDeclarations := len(manifest.Migrate.Fixture) > 0

	match, err := selectRuntimeMigrationFixture(manifest.Migrate.Fixture, step)
	if err != nil {
		return nil, err
	}
	if match == nil && hasFixtureDeclarations {
		return nil, fmt.Errorf("%w: step %04d missing migrate.fixture declaration", ErrMigrationFixtureUnavailable, step.Number)
	}

	return match, nil
}

func selectRuntimeMigrationFixture(declarations []migrationManifestFixture, step migrationPlanStep) (*runtimeMigrationFixture, error) {
	var match *runtimeMigrationFixture
	for _, fixture := range declarations {
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
