package appruntime

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var migrationStoreBoilerplateLines = map[string]struct{}{
	"cursor = None":                {},
	"count = 0":                    {},
	"while True:":                  {},
	"if len(page) == 0: break":     {},
	"for key in page:":             {},
	"count += 1":                   {},
	"cursor = page[-1]":            {},
	"return label.strip().lower()": {},
	"if len(page) == 0:":           {},
	"break":                        {},
	"continue":                     {},
}

func migrationFixtureSkippableLine(line string) bool {
	return line == "" ||
		strings.HasPrefix(line, "def ") ||
		line == "pass" ||
		line == "continue" ||
		strings.HasPrefix(line, "load(") ||
		strings.HasPrefix(line, "return ")
}

func migrationStoreFixtureSkippableLine(line string) bool {
	if migrationFixtureSkippableLine(line) {
		return true
	}
	_, ok := migrationStoreBoilerplateLines[line]
	return ok
}

func parseMigrationFixtureRecordTransform(
	line string,
	lineNumber int,
	abortAliases map[string]struct{},
) (runtimeMigrationFixtureTransform, bool, error) {
	if reason, ok, err := migrationAbortCall(line, abortAliases); ok {
		if err != nil {
			return runtimeMigrationFixtureTransform{}, false, fmt.Errorf("line %d: invalid abort reason: %w", lineNumber, err)
		}
		return runtimeMigrationFixtureTransform{Operation: "abort", Reason: reason}, true, nil
	}
	if match := migrateRecordDeletePattern.FindStringSubmatch(line); match != nil {
		return runtimeMigrationFixtureTransform{Destination: match[1], Operation: "delete"}, true, nil
	}
	if match := migrateRecordSkipIfPresentPattern.FindStringSubmatch(line); match != nil {
		return runtimeMigrationFixtureTransform{Source: match[1], Operation: "skip_if_present"}, true, nil
	}
	if match := migrateRecordSkipIfPresentBlockPattern.FindStringSubmatch(line); match != nil {
		return runtimeMigrationFixtureTransform{Source: match[1], Operation: "skip_if_present"}, true, nil
	}
	if match := migrateRecordAssignmentPattern.FindStringSubmatch(line); match != nil {
		transform, err := parseRuntimeMigrationFixtureAssignment(match[1], match[2])
		if err != nil {
			return runtimeMigrationFixtureTransform{}, false, fmt.Errorf("line %d: %w", lineNumber, err)
		}
		return transform, true, nil
	}
	return runtimeMigrationFixtureTransform{}, false, nil
}

func migrationFixtureIgnorableCall(line string, logAliases map[string]string, checkpointAliases map[string]struct{}) bool {
	match := migrateCallPattern.FindStringSubmatch(line)
	if match == nil {
		return false
	}
	if _, ok := logAliases[match[1]]; ok {
		return true
	}
	if checkpointAliases != nil {
		if _, ok := checkpointAliases[match[1]]; ok {
			return true
		}
	}
	return false
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

type migrationStoreFixtureLineAction int

const (
	migrationStoreLineSkip migrationStoreFixtureLineAction = iota
	migrationStoreLinePrefix
	migrationStoreLineGet
	migrationStoreLinePut
	migrationStoreLineDelete
	migrationStoreLineTransform
	migrationStoreLineIgnorable
	migrationStoreLineUnsupported
)

type migrationStoreFixtureLineResult struct {
	action    migrationStoreFixtureLineAction
	prefix    string
	transform runtimeMigrationFixtureTransform
}

func parseMigrationStoreFixtureLine(
	line string,
	lineNumber int,
	storeAliases runtimeMigrationStoreAliases,
	abortAliases map[string]struct{},
	logAliases map[string]string,
	checkpointAliases map[string]struct{},
) (migrationStoreFixtureLineResult, error) {
	if migrationStoreFixtureSkippableLine(line) {
		return migrationStoreFixtureLineResult{action: migrationStoreLineSkip}, nil
	}
	if parsedPrefix, ok := migrationStoreListKeysPrefix(line, storeAliases.ListKeys); ok {
		return migrationStoreFixtureLineResult{action: migrationStoreLinePrefix, prefix: parsedPrefix}, nil
	}
	if migrationStoreGetStatement(line, storeAliases.Get) {
		return migrationStoreFixtureLineResult{action: migrationStoreLineGet}, nil
	}
	if migrationStorePutStatement(line, storeAliases.Put) {
		return migrationStoreFixtureLineResult{action: migrationStoreLinePut}, nil
	}
	if migrationStoreDeleteStatement(line, storeAliases.Delete) {
		return migrationStoreFixtureLineResult{action: migrationStoreLineDelete}, nil
	}
	if transform, parsed, err := parseMigrationFixtureRecordTransform(line, lineNumber, abortAliases); parsed || err != nil {
		if err != nil {
			return migrationStoreFixtureLineResult{}, err
		}
		return migrationStoreFixtureLineResult{action: migrationStoreLineTransform, transform: transform}, nil
	}
	recordLine := migrationStoreRecordLine(line)
	if transform, parsed, err := parseMigrationFixtureRecordTransform(recordLine, lineNumber, abortAliases); parsed || err != nil {
		if err != nil {
			return migrationStoreFixtureLineResult{}, err
		}
		return migrationStoreFixtureLineResult{action: migrationStoreLineTransform, transform: transform}, nil
	}
	if migrationFixtureIgnorableCall(line, logAliases, checkpointAliases) {
		return migrationStoreFixtureLineResult{action: migrationStoreLineIgnorable}, nil
	}
	return migrationStoreFixtureLineResult{action: migrationStoreLineUnsupported}, nil
}

func validateMigrationStoreFixturePlan(prefix string, sawGet, sawPut, sawDelete bool) error {
	if prefix == "" {
		return errors.New("store fixture migration missing list_keys prefix")
	}
	if !sawGet && !sawDelete {
		return errors.New("store fixture migration must get records or delete keys")
	}
	if !sawPut && !sawDelete {
		return errors.New("store fixture migration must put records or delete keys")
	}
	return nil
}

func migrationStoreRecordLine(line string) string {
	replacer := strings.NewReplacer(
		`rec["`, `record["`,
		"rec.get(", "record.get(",
		"in rec:", "in record:",
	)
	return replacer.Replace(line)
}
