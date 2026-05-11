package capability

import (
	"strings"
)

// AnnotateCanvas appends an annotation to the named canvas.
func (s *Service) AnnotateCanvas(canvas, text string) Annotation {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := Annotation{
		ID:        s.nextIDLocked("ann"),
		Canvas:    defaultIfBlank(canvas, "default"),
		Text:      strings.TrimSpace(text),
		CreatedAt: s.now(),
	}
	s.annotations = append(s.annotations, item)
	s.appendRecentLocked("canvas", item.ID+" "+item.Text)
	return item
}

// ListCanvas returns annotations on the given canvas.
func (s *Service) ListCanvas(canvas string) []Annotation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	canvas = strings.TrimSpace(canvas)
	out := make([]Annotation, 0, len(s.annotations))
	for _, item := range s.annotations {
		if canvas == "" || item.Canvas == canvas {
			out = append(out, item)
		}
	}
	return out
}
