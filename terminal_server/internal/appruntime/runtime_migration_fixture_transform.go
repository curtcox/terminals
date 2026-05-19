package appruntime

import (
	"encoding/json"
	"fmt"
	"strings"
)

// applyRuntimeMigrationFixtureTransform mutates record for one transform.
// skipRemaining is true when skip_if_present matched.
func applyRuntimeMigrationFixtureTransform(
	record map[string]any,
	transform runtimeMigrationFixtureTransform,
	recordKey string,
) (skipRemaining bool, err error) {
	switch transform.Operation {
	case "skip_if_present":
		if _, ok := record[transform.Source]; ok {
			return true, nil
		}
	case "copy":
		record[transform.Destination] = runtimeMigrationFixtureValue(record, transform)
	case "lower":
		value, ok := runtimeMigrationFixtureStringValue(record, transform)
		if !ok {
			return false, fmt.Errorf("record key %q field %q is not a string for lower()", recordKey, transform.Source)
		}
		record[transform.Destination] = strings.ToLower(value)
	case "trim":
		value, ok := runtimeMigrationFixtureStringValue(record, transform)
		if !ok {
			return false, fmt.Errorf("record key %q field %q is not a string for trim()", recordKey, transform.Source)
		}
		record[transform.Destination] = strings.TrimSpace(value)
	case "lower_trim":
		value, ok := runtimeMigrationFixtureStringValue(record, transform)
		if !ok {
			return false, fmt.Errorf("record key %q field %q is not a string for lower(trim())", recordKey, transform.Source)
		}
		record[transform.Destination] = strings.ToLower(strings.TrimSpace(value))
	case "literal":
		record[transform.Destination] = transform.Value
	case "delete":
		delete(record, transform.Destination)
	case "abort":
		return false, fmt.Errorf("%w: %s", ErrMigrationAborted, transform.Reason)
	default:
		return false, fmt.Errorf("unsupported fixture transform %q", transform.Operation)
	}
	return false, nil
}

func applyRuntimeMigrationFixtureTransforms(
	record map[string]any,
	transforms []runtimeMigrationFixtureTransform,
	recordKey string,
) (bool, error) {
	skipRemaining := false
	for _, transform := range transforms {
		if skipRemaining {
			break
		}
		if transform.Operation == "abort" {
			return false, fmt.Errorf("%w: %s", ErrMigrationAborted, transform.Reason)
		}
		nextSkip, err := applyRuntimeMigrationFixtureTransform(record, transform, recordKey)
		if err != nil {
			return false, err
		}
		skipRemaining = skipRemaining || nextSkip
	}
	return skipRemaining, nil
}

func migrateFixtureSeedRecord(
	key string,
	rawValue string,
	transforms []runtimeMigrationFixtureTransform,
) (canonicalValue string, storeOps int, writeBytes int64, err error) {
	var record map[string]any
	if err := json.Unmarshal([]byte(rawValue), &record); err != nil {
		return "", 0, 0, fmt.Errorf("seed key %q is not a JSON object: %w", key, err)
	}
	if _, err := applyRuntimeMigrationFixtureTransforms(record, transforms, key); err != nil {
		return "", 0, 0, err
	}
	canonicalValue, err = canonicalizeMigrationFixtureRecord(record)
	if err != nil {
		return "", 0, 0, fmt.Errorf("canonicalize migrated record %q: %w", key, err)
	}
	if canonicalValue == rawValue {
		return canonicalValue, 0, 0, nil
	}
	return canonicalValue, 1, int64(len(canonicalValue)), nil
}

func applyStoreFixtureTransforms(
	out map[string]string,
	key string,
	record map[string]any,
	transforms []runtimeMigrationFixtureTransform,
) (deleted bool, skipPut bool, err error) {
	for _, transform := range transforms {
		if skipPut {
			break
		}
		if transform.Operation == "delete_record" {
			delete(out, key)
			return true, true, nil
		}
		nextSkip, err := applyRuntimeMigrationFixtureTransform(record, transform, key)
		if err != nil {
			return false, false, err
		}
		skipPut = skipPut || nextSkip
	}
	return false, skipPut, nil
}

func canonicalizeMigrationFixtureRecord(record map[string]any) (string, error) {
	canonical, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	return string(canonical), nil
}
