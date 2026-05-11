package capability

import (
	"sort"
	"strings"
	"time"
)

// Search returns items whose text matches the query across messages, board, artifacts and memories.
func (s *Service) Search(query string) []SearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return nil
	}
	out := make([]SearchResult, 0)
	for _, item := range s.searchCorpusLocked() {
		if strings.Contains(strings.ToLower(item.Text), needle) {
			out = append(out, SearchResult{ID: item.ID, Kind: item.Kind, Text: item.Text})
		}
	}
	return out
}

// SearchTimeline returns activity records in timeline order optionally filtered by scope.
func (s *Service) SearchTimeline(scope string) []RecentItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	needle := strings.ToLower(strings.TrimSpace(scope))
	out := make([]RecentItem, 0, len(s.recent))
	for _, item := range s.recent {
		if needle != "" && !strings.EqualFold(item.Kind, needle) && !strings.Contains(strings.ToLower(item.Text), needle) {
			continue
		}
		out = append(out, item)
	}
	return out
}

// SearchRelated returns indexed items related to the given subject reference or phrase.
func (s *Service) SearchRelated(subjectRef string) []SearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	subjectRef = strings.TrimSpace(subjectRef)
	if subjectRef == "" {
		return nil
	}
	tokens := normalizedTokens(subjectRef)
	if len(tokens) == 0 {
		return nil
	}
	type scored struct {
		result    SearchResult
		score     int
		createdAt time.Time
	}
	matches := make([]scored, 0)
	subjectLower := strings.ToLower(subjectRef)
	for _, item := range s.searchCorpusLocked() {
		score := 0
		idLower := strings.ToLower(item.ID)
		textLower := strings.ToLower(item.Text)
		if strings.EqualFold(item.ID, subjectRef) {
			score += 3
		}
		for _, token := range tokens {
			if strings.Contains(textLower, token) || strings.Contains(idLower, token) || strings.Contains(subjectLower, idLower) {
				score++
			}
		}
		if score == 0 {
			continue
		}
		matches = append(matches, scored{
			result:    SearchResult{ID: item.ID, Kind: item.Kind, Text: item.Text},
			score:     score,
			createdAt: item.CreatedAt,
		})
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		if !matches[i].createdAt.Equal(matches[j].createdAt) {
			return matches[i].createdAt.After(matches[j].createdAt)
		}
		return matches[i].result.ID < matches[j].result.ID
	})
	out := make([]SearchResult, 0, len(matches))
	for _, item := range matches {
		out = append(out, item.result)
	}
	return out
}

// SearchRecent returns the newest timeline entries for a scope.
func (s *Service) SearchRecent(scope string, limit int) []RecentItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	needle := strings.ToLower(strings.TrimSpace(scope))
	if limit <= 0 {
		limit = 20
	}
	buffer := make([]RecentItem, 0, limit)
	for _, item := range s.recent {
		if needle != "" && !strings.EqualFold(item.Kind, needle) && !strings.Contains(strings.ToLower(item.Text), needle) {
			continue
		}
		buffer = append(buffer, item)
		if len(buffer) > limit {
			buffer = append([]RecentItem(nil), buffer[len(buffer)-limit:]...)
		}
	}
	return buffer
}
func normalizedTokens(value string) []string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(value)))
	out := make([]string, 0, len(fields))
	seen := map[string]struct{}{}
	for _, field := range fields {
		field = strings.Trim(field, " .,;:!?()[]{}\"'")
		if len(field) < 2 {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		out = append(out, field)
	}
	return out
}
func (s *Service) searchCorpusLocked() []searchableItem {
	out := make([]searchableItem, 0, len(s.messages)+len(s.boardItems)+len(s.artifacts)+len(s.memories))
	for _, item := range s.messages {
		out = append(out, searchableItem{ID: item.ID, Kind: "message", Text: item.Text, CreatedAt: item.CreatedAt})
	}
	for _, item := range s.boardItems {
		out = append(out, searchableItem{ID: item.ID, Kind: "board", Text: item.Text, CreatedAt: item.CreatedAt})
	}
	for _, item := range s.artifacts {
		out = append(out, searchableItem{ID: item.ID, Kind: "artifact", Text: item.Title, CreatedAt: item.CreatedAt})
	}
	for _, item := range s.memories {
		out = append(out, searchableItem{ID: item.ID, Kind: "memory", Text: item.Text, CreatedAt: item.CreatedAt})
	}
	return out
}
