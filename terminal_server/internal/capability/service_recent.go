package capability

import (
	"strings"
)

// ListRecent returns the most recent activity entries in insertion order.
func (s *Service) ListRecent() []RecentItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]RecentItem(nil), s.recent...)
}
func (s *Service) appendRecentLocked(kind, text string) {
	item := RecentItem{
		ID:        s.nextIDLocked("recent"),
		Kind:      kind,
		Text:      strings.TrimSpace(text),
		CreatedAt: s.now(),
	}
	s.recent = append(s.recent, item)
	if len(s.recent) > 200 {
		s.recent = append([]RecentItem(nil), s.recent[len(s.recent)-200:]...)
	}
}
