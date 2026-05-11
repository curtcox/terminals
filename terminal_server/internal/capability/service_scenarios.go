package capability

import (
	"sort"
	"strings"
)

// ScenarioDefine creates or updates one inline scenario definition.
func (s *Service) ScenarioDefine(def InlineScenarioDefinition) InlineScenarioDefinition {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.ToLower(strings.TrimSpace(def.Name))
	out := InlineScenarioDefinition{
		Name:         name,
		MatchIntents: normalizeScenarioTokens(def.MatchIntents),
		MatchEvents:  normalizeScenarioTokens(def.MatchEvents),
		Priority:     normalizeScenarioPriority(def.Priority),
		OnStart:      strings.TrimSpace(def.OnStart),
		OnInput:      strings.TrimSpace(def.OnInput),
		OnSuspend:    strings.TrimSpace(def.OnSuspend),
		OnResume:     strings.TrimSpace(def.OnResume),
		OnStop:       strings.TrimSpace(def.OnStop),
		UpdatedAt:    s.now(),
	}
	out.OnEvents = make([]InlineScenarioEventHook, 0, len(def.OnEvents))
	for _, hook := range def.OnEvents {
		kind := strings.TrimSpace(hook.Kind)
		command := strings.TrimSpace(hook.Command)
		if kind == "" || command == "" {
			continue
		}
		out.OnEvents = append(out.OnEvents, InlineScenarioEventHook{Kind: kind, Command: command})
	}
	s.scenarios[name] = out
	s.appendRecentLocked("scenario", name+" define")
	return out
}

// ScenarioGet returns one inline scenario definition by name.
func (s *Service) ScenarioGet(name string) (InlineScenarioDefinition, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	def, ok := s.scenarios[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return InlineScenarioDefinition{}, false
	}
	return cloneScenarioDefinition(def), true
}

// ScenarioList returns all inline scenario definitions sorted by name.
func (s *Service) ScenarioList() []InlineScenarioDefinition {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]InlineScenarioDefinition, 0, len(s.scenarios))
	for _, def := range s.scenarios {
		out = append(out, cloneScenarioDefinition(def))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ScenarioUndefine removes one inline scenario definition by name.
func (s *Service) ScenarioUndefine(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.ToLower(strings.TrimSpace(name))
	if _, ok := s.scenarios[name]; !ok {
		return false
	}
	delete(s.scenarios, name)
	s.appendRecentLocked("scenario", name+" undefine")
	return true
}
func normalizeScenarioTokens(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if !sliceContainsFold(out, trimmed) {
			out = append(out, trimmed)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func normalizeScenarioPriority(value string) string {
	priority := strings.ToLower(strings.TrimSpace(value))
	if priority == "" {
		return "normal"
	}
	switch priority {
	case "low", "normal", "high", "critical":
		return priority
	default:
		return "normal"
	}
}
func cloneScenarioDefinition(def InlineScenarioDefinition) InlineScenarioDefinition {
	clone := def
	clone.MatchIntents = append([]string(nil), def.MatchIntents...)
	clone.MatchEvents = append([]string(nil), def.MatchEvents...)
	clone.OnEvents = append([]InlineScenarioEventHook(nil), def.OnEvents...)
	return clone
}
