package capability

import (
	"sort"
	"strings"
)

// HandlerList returns all registered runtime handlers sorted by id.
func (s *Service) HandlerList() []HandlerRegistration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]HandlerRegistration, 0, len(s.handlers))
	for _, handler := range s.handlers {
		out = append(out, handler)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// HandlerOnRun registers a routing handler that executes a REPL command when matched.
func (s *Service) HandlerOnRun(selector, action, command string) HandlerRegistration {
	s.mu.Lock()
	defer s.mu.Unlock()
	handler := HandlerRegistration{
		ID:         s.nextIDLocked("handler"),
		Selector:   normalizeHandlerSelector(selector),
		Action:     normalizeHandlerAction(action),
		RunCommand: strings.TrimSpace(command),
		UpdatedAt:  s.now(),
	}
	s.handlers[handler.ID] = handler
	s.appendRecentLocked("handler", handler.ID+" on")
	return handler
}

// HandlerOnEmit registers a routing handler that emits a bus event or intent when matched.
func (s *Service) HandlerOnEmit(selector, action, emitKind, emitName, emitPayload string) HandlerRegistration {
	s.mu.Lock()
	defer s.mu.Unlock()
	handler := HandlerRegistration{
		ID:          s.nextIDLocked("handler"),
		Selector:    normalizeHandlerSelector(selector),
		Action:      normalizeHandlerAction(action),
		EmitKind:    defaultIfBlank(strings.ToLower(strings.TrimSpace(emitKind)), "intent"),
		EmitName:    strings.TrimSpace(emitName),
		EmitPayload: strings.TrimSpace(emitPayload),
		UpdatedAt:   s.now(),
	}
	s.handlers[handler.ID] = handler
	s.appendRecentLocked("handler", handler.ID+" on")
	return handler
}

// HandlerOff removes one registered handler by id.
func (s *Service) HandlerOff(handlerID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	handlerID = strings.TrimSpace(handlerID)
	if _, ok := s.handlers[handlerID]; !ok {
		return false
	}
	delete(s.handlers, handlerID)
	s.appendRecentLocked("handler", handlerID+" off")
	return true
}

func normalizeHandlerSelector(selector string) string {
	return strings.ToLower(strings.TrimSpace(selector))
}

func normalizeHandlerAction(action string) string {
	return strings.ToLower(strings.TrimSpace(action))
}
