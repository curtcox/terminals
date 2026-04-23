package scenario

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// AffordanceOptOutEntry captures one reviewed exception to corner-affordance injection.
type AffordanceOptOutEntry struct {
	ScenarioID            string
	Reason                string
	Approver              string
	ExpiresAt             string
	ReplacementAffordance string
}

var logicalIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.\-]+$`)

// LoadAffordanceOptOutAllowlist loads the checked-in affordance opt-out file.
func LoadAffordanceOptOutAllowlist(path string) ([]AffordanceOptOutEntry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(raw), "\n")
	out := make([]AffordanceOptOutEntry, 0, 4)
	var current *AffordanceOptOutEntry
	flush := func() error {
		if current == nil {
			return nil
		}
		if strings.TrimSpace(current.ScenarioID) == "" ||
			strings.TrimSpace(current.Reason) == "" ||
			strings.TrimSpace(current.Approver) == "" ||
			strings.TrimSpace(current.ExpiresAt) == "" {
			return fmt.Errorf("affordance opt-out entry missing required field: %+v", *current)
		}
		out = append(out, *current)
		current = nil
		return nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			if err := flush(); err != nil {
				return nil, err
			}
			current = &AffordanceOptOutEntry{}
			trimmed = strings.TrimPrefix(trimmed, "- ")
			if trimmed == "" {
				continue
			}
		}
		if current == nil {
			return nil, fmt.Errorf("invalid affordance opt-out format: %q", line)
		}
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			return nil, fmt.Errorf("invalid affordance opt-out entry line: %q", line)
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), "\"'")
		switch key {
		case "scenario_id":
			current.ScenarioID = value
		case "reason":
			current.Reason = value
		case "approver":
			current.Approver = value
		case "expires_at":
			current.ExpiresAt = value
		case "replacement_affordance":
			current.ReplacementAffordance = value
		default:
			return nil, fmt.Errorf("unknown affordance opt-out field %q", key)
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return out, nil
}

// ValidateAffordanceOptOutAllowlist validates expiry and scenario references.
func ValidateAffordanceOptOutAllowlist(entries []AffordanceOptOutEntry, scenarioIDs map[string]struct{}, now time.Time) error {
	for _, entry := range entries {
		if _, ok := scenarioIDs[strings.TrimSpace(entry.ScenarioID)]; !ok {
			return fmt.Errorf("affordance opt-out references unknown scenario_id %q", entry.ScenarioID)
		}
		if err := validateReplacementAffordance(entry.ReplacementAffordance); err != nil {
			return fmt.Errorf("invalid replacement_affordance for scenario_id %q: %w", entry.ScenarioID, err)
		}
		expiresAt, err := time.Parse("2006-01-02", strings.TrimSpace(entry.ExpiresAt))
		if err != nil {
			return fmt.Errorf("invalid expires_at for scenario_id %q: %w", entry.ScenarioID, err)
		}
		if !expiresAt.After(now.UTC()) {
			return fmt.Errorf("expired affordance opt-out for scenario_id %q at %s", entry.ScenarioID, entry.ExpiresAt)
		}
	}
	return nil
}

func validateReplacementAffordance(raw string) error {
	replacement := strings.TrimSpace(raw)
	if replacement == "" {
		return nil
	}
	if !strings.HasPrefix(replacement, "__affordance.") {
		return fmt.Errorf("must use reserved __affordance.* namespace")
	}
	if replacement == "__affordance." {
		return fmt.Errorf("must include affordance id suffix")
	}
	if !logicalIDPattern.MatchString(replacement) {
		return fmt.Errorf("must match logical id grammar [A-Za-z0-9_.-]+")
	}
	return nil
}

// CodeownersHasPathOwner reports whether CODEOWNERS contains a path rule with the requested owner.
func CodeownersHasPathOwner(path, matchPath, owner string) (bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		if strings.TrimSpace(fields[0]) != strings.TrimSpace(matchPath) {
			continue
		}
		for _, declaredOwner := range fields[1:] {
			if strings.TrimSpace(declaredOwner) == strings.TrimSpace(owner) {
				return true, nil
			}
		}
	}
	return false, nil
}
