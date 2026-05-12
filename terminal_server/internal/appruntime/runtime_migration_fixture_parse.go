package appruntime

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

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
