package capability

import (
	"sort"
	"strings"
)

// CohortUpsert creates or updates one named device cohort.
func (s *Service) CohortUpsert(name string, selectors []string) DeviceCohort {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.ToLower(strings.TrimSpace(name))
	cohort := DeviceCohort{
		Name:      name,
		Selectors: normalizeSelectors(selectors),
		UpdatedAt: s.now(),
	}
	s.cohorts[name] = cohort
	s.appendRecentLocked("cohort", name+" upsert")
	return cohort
}

// CohortGet returns one cohort by name.
func (s *Service) CohortGet(name string) (DeviceCohort, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cohort, ok := s.cohorts[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return DeviceCohort{}, false
	}
	cohort.Selectors = append([]string(nil), cohort.Selectors...)
	return cohort, true
}

// CohortList returns all cohorts sorted by name.
func (s *Service) CohortList() []DeviceCohort {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cohorts := make([]DeviceCohort, 0, len(s.cohorts))
	for _, cohort := range s.cohorts {
		copyCohort := cohort
		copyCohort.Selectors = append([]string(nil), cohort.Selectors...)
		cohorts = append(cohorts, copyCohort)
	}
	sort.Slice(cohorts, func(i, j int) bool { return cohorts[i].Name < cohorts[j].Name })
	return cohorts
}

// CohortDelete removes one cohort by name.
func (s *Service) CohortDelete(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	name = strings.ToLower(strings.TrimSpace(name))
	if _, ok := s.cohorts[name]; !ok {
		return false
	}
	delete(s.cohorts, name)
	s.appendRecentLocked("cohort", name+" deleted")
	return true
}
func normalizeSelectors(selectors []string) []string {
	if len(selectors) == 0 {
		return nil
	}
	out := make([]string, 0, len(selectors))
	seen := make(map[string]struct{}, len(selectors))
	for _, selector := range selectors {
		normalized := strings.ToLower(strings.TrimSpace(selector))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	sort.Strings(out)
	return out
}
