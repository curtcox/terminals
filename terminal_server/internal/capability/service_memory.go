package capability

import (
	"strings"
)

// Remember stores a memory entry in the given scope.
func (s *Service) Remember(scope, text string) MemoryEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := MemoryEntry{
		ID:        s.nextIDLocked("mem"),
		Scope:     defaultIfBlank(scope, "general"),
		Text:      strings.TrimSpace(text),
		CreatedAt: s.now(),
	}
	s.memories = append(s.memories, item)
	s.appendRecentLocked("memory", item.ID+" "+item.Text)
	return item
}

// Recall returns memory entries whose text or scope matches the query.
func (s *Service) Recall(query string) []MemoryEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	needle := strings.ToLower(strings.TrimSpace(query))
	out := make([]MemoryEntry, 0, len(s.memories))
	for _, item := range s.memories {
		if needle == "" || strings.Contains(strings.ToLower(item.Text), needle) || strings.Contains(strings.ToLower(item.Scope), needle) {
			out = append(out, item)
		}
	}
	return out
}

// MemoryStream returns memory entries in insertion order with optional scope filtering.
func (s *Service) MemoryStream(scope string) []MemoryEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	needle := strings.ToLower(strings.TrimSpace(scope))
	out := make([]MemoryEntry, 0, len(s.memories))
	for _, item := range s.memories {
		if needle != "" && !strings.EqualFold(item.Scope, needle) && !strings.Contains(strings.ToLower(item.Text), needle) {
			continue
		}
		out = append(out, item)
	}
	return out
}
