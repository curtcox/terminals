package admin

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
)

func (h *Handler) writeJSONError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) withRequestLogging(next http.Handler) http.Handler {
	logger := eventlog.Component("admin.http")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx, end := eventlog.WithSpan(r.Context(), "admin:"+r.Method+":"+r.URL.Path)
		defer end()
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
		eventlog.Emit(ctx, "admin.http.request", slog.LevelInfo, "admin request",
			slog.String("component", "admin.http"),
			slog.Group("http",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			),
		)
		logger.Debug("admin request served", "event", "admin.http.request", "method", r.Method, "path", r.URL.Path)
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func truthyFormValue(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "t", "yes", "y", "on":
		return true
	default:
		return false
	}
}
