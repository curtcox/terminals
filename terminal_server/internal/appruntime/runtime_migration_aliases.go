package appruntime

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

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
		block := match[1]
		addMigrationAliasMatches(block, migrateStoreListKeysAliasPattern, aliases.ListKeys)
		addMigrationAliasMatches(block, migrateStoreGetAliasPattern, aliases.Get)
		addMigrationAliasMatches(block, migrateStorePutAliasPattern, aliases.Put)
		addMigrationAliasMatches(block, migrateStoreDeleteAliasPattern, aliases.Delete)
	}
	return aliases
}

func addMigrationAliasMatches(block []byte, pattern *regexp.Regexp, into map[string]struct{}) {
	for _, aliasMatch := range pattern.FindAllSubmatch(block, -1) {
		if len(aliasMatch) >= 2 {
			into[string(aliasMatch[1])] = struct{}{}
		}
	}
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
