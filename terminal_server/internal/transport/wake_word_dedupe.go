package transport

import (
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

type wakeWordWinnerPolicy string

const (
	wakeWordWinnerPolicyFirstHeard        wakeWordWinnerPolicy = "first_heard"
	wakeWordWinnerPolicyHighestConfidence wakeWordWinnerPolicy = "highest_confidence"
	wakeWordWinnerPolicyClosestTerminal   wakeWordWinnerPolicy = "closest_terminal"
)

const defaultWakeWordDedupeWindow = 1200 * time.Millisecond

func defaultWakeWordWinnerPolicy() wakeWordWinnerPolicy {
	return wakeWordWinnerPolicyFirstHeard
}

type wakeWordCandidate struct {
	DeviceID       string
	Spoken         string
	HeardAt        time.Time
	Confidence     float64
	DistanceMeters float64
}

type wakeWordDedupeStage struct {
	mu              sync.Mutex
	window          time.Duration
	policy          wakeWordWinnerPolicy
	historyByPhrase map[string][]wakeWordCandidate
}

func newWakeWordDedupeStage(window time.Duration, policy wakeWordWinnerPolicy) *wakeWordDedupeStage {
	if window <= 0 {
		window = defaultWakeWordDedupeWindow
	}
	if policy == "" {
		policy = defaultWakeWordWinnerPolicy()
	}
	return &wakeWordDedupeStage{
		window:          window,
		policy:          policy,
		historyByPhrase: map[string][]wakeWordCandidate{},
	}
}

func (s *wakeWordDedupeStage) Allow(candidate wakeWordCandidate) bool {
	if s == nil {
		return true
	}
	phrase := normalizeWakeWordPhrase(candidate.Spoken)
	if phrase == "" {
		return true
	}
	if candidate.HeardAt.IsZero() {
		candidate.HeardAt = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	history := append([]wakeWordCandidate(nil), s.historyByPhrase[phrase]...)
	cutoff := candidate.HeardAt.Add(-s.window)
	filtered := history[:0]
	for _, prior := range history {
		if prior.HeardAt.Before(cutoff) {
			continue
		}
		filtered = append(filtered, prior)
	}
	filtered = append(filtered, candidate)
	s.historyByPhrase[phrase] = filtered

	winner, ok := selectWakeWordWinner(s.policy, filtered)
	if !ok {
		return true
	}
	return sameWakeWordCandidate(winner, candidate)
}

func selectWakeWordWinner(policy wakeWordWinnerPolicy, candidates []wakeWordCandidate) (wakeWordCandidate, bool) {
	if len(candidates) == 0 {
		return wakeWordCandidate{}, false
	}
	sorted := append([]wakeWordCandidate(nil), candidates...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left := sorted[i]
		right := sorted[j]
		switch policy {
		case wakeWordWinnerPolicyHighestConfidence:
			if left.Confidence != right.Confidence {
				return left.Confidence > right.Confidence
			}
		case wakeWordWinnerPolicyClosestTerminal:
			leftDistance := sanitizeDistance(left.DistanceMeters)
			rightDistance := sanitizeDistance(right.DistanceMeters)
			if leftDistance != rightDistance {
				return leftDistance < rightDistance
			}
		case "", wakeWordWinnerPolicyFirstHeard:
			// fall through to earliest-heard tie breaker.
		default:
			// Unknown policy degrades to first-heard.
		}
		if !left.HeardAt.Equal(right.HeardAt) {
			return left.HeardAt.Before(right.HeardAt)
		}
		return strings.TrimSpace(left.DeviceID) < strings.TrimSpace(right.DeviceID)
	})
	return sorted[0], true
}

func normalizeWakeWordPhrase(spoken string) string {
	return strings.ToLower(strings.TrimSpace(spoken))
}

func sameWakeWordCandidate(left, right wakeWordCandidate) bool {
	return normalizeWakeWordPhrase(left.Spoken) == normalizeWakeWordPhrase(right.Spoken) &&
		strings.TrimSpace(left.DeviceID) == strings.TrimSpace(right.DeviceID) &&
		left.HeardAt.Equal(right.HeardAt)
}

func sanitizeDistance(distance float64) float64 {
	if distance < 0 || math.IsNaN(distance) || math.IsInf(distance, 0) {
		return math.MaxFloat64
	}
	return distance
}
