package capability

import (
	"strings"
)

// BusEmit emits a named event with an optional payload on the event bus.
func (s *Service) BusEmit(kind, name, payload string) BusEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	event := BusEvent{
		ID:        s.nextIDLocked("bus"),
		Kind:      defaultIfBlank(kind, "event"),
		Name:      defaultIfBlank(name, "unnamed"),
		Payload:   strings.TrimSpace(payload),
		CreatedAt: s.now(),
	}
	s.bus = append(s.bus, event)
	s.appendRecentLocked("bus", event.ID+" "+event.Name)
	return event
}

// BusTail returns events emitted on the event bus with optional filtering.
func (s *Service) BusTail(kind, name string, limit int) []BusEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return filterBusEvents(s.bus, kind, name, limit)
}

// BusReplay returns events within an inclusive ID window with optional filtering.
func (s *Service) BusReplay(fromID, toID, kind, name string, limit int) []BusEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	window := busWindowByID(s.bus, strings.TrimSpace(fromID), strings.TrimSpace(toID))
	return filterBusEvents(window, kind, name, limit)
}

func busWindowByID(events []BusEvent, fromID, toID string) []BusEvent {
	if len(events) == 0 {
		return nil
	}
	start := 0
	if fromID != "" {
		for i, event := range events {
			if event.ID == fromID {
				start = i
				break
			}
		}
	}
	end := len(events) - 1
	if toID != "" {
		for i := len(events) - 1; i >= 0; i-- {
			if events[i].ID == toID {
				end = i
				break
			}
		}
	}
	if start > end {
		return nil
	}
	return append([]BusEvent(nil), events[start:end+1]...)
}

func filterBusEvents(events []BusEvent, kind, name string, limit int) []BusEvent {
	kind = strings.TrimSpace(kind)
	name = strings.TrimSpace(name)
	filtered := make([]BusEvent, 0, len(events))
	for _, event := range events {
		if kind != "" && !strings.EqualFold(event.Kind, kind) {
			continue
		}
		if name != "" && !strings.EqualFold(event.Name, name) {
			continue
		}
		filtered = append(filtered, event)
	}
	if limit > 0 && len(filtered) > limit {
		return append([]BusEvent(nil), filtered[len(filtered)-limit:]...)
	}
	return filtered
}

