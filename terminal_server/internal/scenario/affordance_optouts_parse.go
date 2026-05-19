package scenario

import (
	"fmt"
	"strings"
)

type affordanceOptOutParser struct {
	out     []AffordanceOptOutEntry
	current *AffordanceOptOutEntry
}

func parseAffordanceOptOutAllowlistLines(lines []string) ([]AffordanceOptOutEntry, error) {
	parser := &affordanceOptOutParser{out: make([]AffordanceOptOutEntry, 0, 4)}
	for _, line := range lines {
		if err := parser.feed(line); err != nil {
			return nil, err
		}
	}
	return parser.finish()
}

func (p *affordanceOptOutParser) finish() ([]AffordanceOptOutEntry, error) {
	if err := p.flush(); err != nil {
		return nil, err
	}
	return p.out, nil
}

func (p *affordanceOptOutParser) flush() error {
	if p.current == nil {
		return nil
	}
	if err := validateAffordanceOptOutEntry(p.current); err != nil {
		return err
	}
	p.out = append(p.out, *p.current)
	p.current = nil
	return nil
}

func (p *affordanceOptOutParser) feed(line string) error {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return nil
	}
	if strings.HasPrefix(trimmed, "- ") {
		return p.startEntry(strings.TrimPrefix(trimmed, "- "))
	}
	if p.current == nil {
		return fmt.Errorf("invalid affordance opt-out format: %q", line)
	}
	return applyAffordanceOptOutField(p.current, trimmed)
}

func (p *affordanceOptOutParser) startEntry(trimmed string) error {
	if err := p.flush(); err != nil {
		return err
	}
	p.current = &AffordanceOptOutEntry{}
	if trimmed == "" {
		return nil
	}
	return applyAffordanceOptOutField(p.current, trimmed)
}

func validateAffordanceOptOutEntry(entry *AffordanceOptOutEntry) error {
	if strings.TrimSpace(entry.ScenarioID) == "" ||
		strings.TrimSpace(entry.Reason) == "" ||
		strings.TrimSpace(entry.Approver) == "" ||
		strings.TrimSpace(entry.ExpiresAt) == "" {
		return fmt.Errorf("affordance opt-out entry missing required field: %+v", *entry)
	}
	return nil
}

func applyAffordanceOptOutField(current *AffordanceOptOutEntry, trimmed string) error {
	key, value, ok := strings.Cut(trimmed, ":")
	if !ok {
		return fmt.Errorf("invalid affordance opt-out entry line: %q", trimmed)
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
		return fmt.Errorf("unknown affordance opt-out field %q", key)
	}
	return nil
}
