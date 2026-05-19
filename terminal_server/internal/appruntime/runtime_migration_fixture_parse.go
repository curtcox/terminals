package appruntime

import (
	"encoding/json"
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

type migrationAssignmentRule struct {
	pattern     *regexp.Regexp
	operation   string
	withDefault bool
}

var migrationAssignmentRules = []migrationAssignmentRule{
	{pattern: migrateRecordNormalizeGetPattern, operation: "lower_trim", withDefault: true},
	{pattern: migrateRecordNormalizePattern, operation: "lower_trim"},
	{pattern: migrateRecordLowerTrimGetPattern, operation: "lower_trim", withDefault: true},
	{pattern: migrateRecordLowerTrimPattern, operation: "lower_trim"},
	{pattern: migrateRecordLowerGetPattern, operation: "lower", withDefault: true},
	{pattern: migrateRecordLowerPattern, operation: "lower"},
	{pattern: migrateRecordTrimGetPattern, operation: "trim", withDefault: true},
	{pattern: migrateRecordTrimPattern, operation: "trim"},
	{pattern: migrateRecordGetValuePattern, operation: "copy", withDefault: true},
	{pattern: migrateRecordValuePattern, operation: "copy"},
}

func parseRuntimeMigrationFixtureTransforms(scriptSource []byte) ([]runtimeMigrationFixtureTransform, error) {
	lines := strings.Split(string(scriptSource), "\n")
	transforms := make([]runtimeMigrationFixtureTransform, 0)
	abortAliases := migrationAbortAliases(scriptSource)
	logAliases := migrationLogAliases(scriptSource)
	for lineNumber, line := range lines {
		line = strings.TrimSpace(stripTALLineComment(line))
		if migrationFixtureSkippableLine(line) {
			continue
		}
		transform, parsed, err := parseMigrationFixtureRecordTransform(line, lineNumber+1, abortAliases)
		if err != nil {
			return nil, err
		}
		if parsed {
			transforms = append(transforms, transform)
			continue
		}
		if migrationFixtureIgnorableCall(line, logAliases, nil) {
			continue
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
		result, err := parseMigrationStoreFixtureLine(line, lineNumber+1, storeAliases, abortAliases, logAliases, checkpointAliases)
		if err != nil {
			return nil, err
		}
		switch result.action {
		case migrationStoreLineSkip, migrationStoreLineIgnorable, migrationStoreLineUnsupported:
			continue
		case migrationStoreLinePrefix:
			if prefix != "" && prefix != result.prefix {
				return nil, fmt.Errorf("line %d uses multiple list_keys prefixes", lineNumber+1)
			}
			prefix = result.prefix
		case migrationStoreLineGet:
			sawGet = true
		case migrationStoreLinePut:
			sawPut = true
		case migrationStoreLineDelete:
			sawDelete = true
			transforms = append(transforms, runtimeMigrationFixtureTransform{Operation: "delete_record"})
		case migrationStoreLineTransform:
			transforms = append(transforms, result.transform)
		default:
			return nil, fmt.Errorf("line %d uses unsupported store fixture migration statement %q", lineNumber+1, line)
		}
	}
	if err := validateMigrationStoreFixturePlan(prefix, sawGet, sawPut, sawDelete); err != nil {
		return nil, err
	}
	return &runtimeMigrationStoreFixturePlan{
		Prefix:     prefix,
		Transforms: transforms,
	}, nil
}

func parseRuntimeMigrationFixtureAssignment(destination string, expression string) (runtimeMigrationFixtureTransform, error) {
	expression = strings.TrimSpace(expression)
	for _, rule := range migrationAssignmentRules {
		if match := rule.pattern.FindStringSubmatch(expression); match != nil {
			transform := runtimeMigrationFixtureTransform{
				Destination: destination,
				Source:      match[1],
				Operation:   rule.operation,
			}
			if !rule.withDefault {
				return transform, nil
			}
			defaultValue, err := decodeMigrationDefaultLiteral(match[2])
			if err != nil {
				return runtimeMigrationFixtureTransform{}, err
			}
			transform.Default = defaultValue
			transform.HasDefault = true
			return transform, nil
		}
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
