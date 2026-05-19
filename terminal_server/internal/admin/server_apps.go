package admin

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/appruntime"
	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
)

func (h *Handler) handleApps(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.appRuntime == nil {
		h.writeJSON(w, http.StatusOK, map[string]any{"apps": []map[string]any{}})
		return
	}
	names := h.appRuntime.ListPackages()
	views := make([]map[string]any, 0, len(names))
	for _, name := range names {
		pkg, ok := h.appRuntime.GetPackage(name)
		if !ok {
			continue
		}
		history := h.appRuntime.ListPackageHistory(name)
		versions := make([]string, 0, len(history))
		for _, version := range history {
			versions = append(versions, strings.TrimSpace(version.Manifest.Version))
		}
		views = append(views, map[string]any{
			"name":             pkg.Manifest.Name,
			"version":          pkg.Manifest.Version,
			"revision":         pkg.Revision,
			"loaded_at_unixms": pkg.LoadedAt.UTC().UnixMilli(),
			"permissions":      pkg.Manifest.Permissions,
			"exports":          pkg.Manifest.Exports,
			"dev_mode":         pkg.Manifest.DevMode,
			"history_versions": versions,
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"apps": views})
}

func (h *Handler) handleReloadApp(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.appRuntime == nil {
		h.writeJSONError(w, http.StatusBadRequest, "app runtime not configured")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form")
		return
	}
	name := strings.TrimSpace(req.FormValue("app"))
	if name == "" {
		name = strings.TrimSpace(req.FormValue("name"))
	}
	if name == "" {
		h.writeJSONError(w, http.StatusBadRequest, "app is required")
		return
	}
	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()
	pkg, changed, err := h.appRuntime.ReloadPackage(ctx, name)
	if err != nil {
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if changed && h.syncAppDefs != nil {
		h.syncAppDefs()
	}
	eventlog.Emit(req.Context(), "admin.action.applied", slog.LevelInfo, "admin app reload applied",
		slog.String("component", "admin.http"),
		slog.String("action", "app.reload"),
		slog.String("app", pkg.Manifest.Name),
		slog.Bool("changed", changed),
		slog.String("version", pkg.Manifest.Version),
	)
	h.writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"action":   "reload",
		"app":      pkg.Manifest.Name,
		"changed":  changed,
		"version":  pkg.Manifest.Version,
		"revision": pkg.Revision,
	})
}

func (h *Handler) handleRollbackApp(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.appRuntime == nil {
		h.writeJSONError(w, http.StatusBadRequest, "app runtime not configured")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form")
		return
	}
	name := strings.TrimSpace(req.FormValue("app"))
	if name == "" {
		name = strings.TrimSpace(req.FormValue("name"))
	}
	if name == "" {
		h.writeJSONError(w, http.StatusBadRequest, "app is required")
		return
	}
	mode, err := rollbackDataModeFromForm(req.Form)
	if err != nil {
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	pkg, err := h.appRuntime.RollbackPackage(name, appruntime.RollbackOptions{DataMode: mode})
	if err != nil {
		if errors.Is(err, appruntime.ErrMigrationReconcilePending) {
			status, statusErr := h.appRuntime.GetMigrationStatus(name)
			if statusErr == nil {
				h.writeJSON(w, http.StatusConflict, map[string]any{
					"status":    "blocked",
					"action":    "rollback",
					"app":       name,
					"error":     err.Error(),
					"data_mode": normalizeRollbackDataMode(mode),
					"migration": mapMigrationStatus(status),
				})
				return
			}
		}
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if h.syncAppDefs != nil {
		h.syncAppDefs()
	}
	eventlog.Emit(req.Context(), "admin.action.applied", slog.LevelInfo, "admin app rollback applied",
		slog.String("component", "admin.http"),
		slog.String("action", "app.rollback"),
		slog.String("app", pkg.Manifest.Name),
		slog.String("version", pkg.Manifest.Version),
	)
	h.writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"action":    "rollback",
		"app":       pkg.Manifest.Name,
		"version":   pkg.Manifest.Version,
		"data_mode": normalizeRollbackDataMode(mode),
		"revision":  pkg.Revision,
	})
}

func (h *Handler) handleAppMigrationStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.appRuntime == nil {
		h.writeJSONError(w, http.StatusBadRequest, "app runtime not configured")
		return
	}
	name := strings.TrimSpace(req.URL.Query().Get("app"))
	if name == "" {
		name = strings.TrimSpace(req.URL.Query().Get("name"))
	}
	if name == "" {
		h.writeJSONError(w, http.StatusBadRequest, "app is required")
		return
	}
	status, err := h.appRuntime.GetMigrationStatus(name)
	if err != nil {
		h.writeMigrationError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"migration": mapMigrationStatus(status),
	})
}

func (h *Handler) handleAppMigrationLogs(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.appRuntime == nil {
		h.writeJSONError(w, http.StatusBadRequest, "app runtime not configured")
		return
	}
	name := strings.TrimSpace(req.URL.Query().Get("app"))
	if name == "" {
		name = strings.TrimSpace(req.URL.Query().Get("name"))
	}
	if name == "" {
		h.writeJSONError(w, http.StatusBadRequest, "app is required")
		return
	}

	step, err := parsePositiveOptionalInt(req.URL.Query().Get("step"))
	if err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "step must be a positive integer")
		return
	}
	limit, err := parsePositiveOptionalInt(req.URL.Query().Get("limit"))
	if err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "limit must be a positive integer")
		return
	}
	if limit == 0 {
		limit = 50
	}

	status, err := h.appRuntime.GetMigrationStatus(name)
	if err != nil {
		h.writeMigrationError(w, err)
		return
	}
	pkg, ok := h.appRuntime.GetPackage(name)
	if !ok {
		h.writeMigrationError(w, appruntime.ErrPackageNotFound)
		return
	}

	journalPath := status.JournalPath
	entries := []string{}
	exists := false
	if strings.TrimSpace(journalPath) != "" {
		absolutePath := filepath.Join(pkg.RootPath, filepath.FromSlash(journalPath))
		entries, exists, err = readMigrationJournalTail(absolutePath, step, limit)
		if err != nil {
			h.writeJSONError(w, http.StatusInternalServerError, "failed to read migration logs")
			return
		}
	}

	response := map[string]any{
		"status":         "ok",
		"app":            name,
		"journal_path":   journalPath,
		"journal_exists": exists,
		"line_count":     len(entries),
		"lines":          entries,
	}
	if step > 0 {
		response["step"] = step
	}
	h.writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleAppMigrationRetry(w http.ResponseWriter, req *http.Request) {
	h.handleAppMigrationAction(w, req, "retry")
}

func (h *Handler) handleAppMigrationAbort(w http.ResponseWriter, req *http.Request) {
	h.handleAppMigrationAction(w, req, "abort")
}

func (h *Handler) handleAppMigrationDrainReady(w http.ResponseWriter, req *http.Request) {
	h.handleAppMigrationAction(w, req, "drain-ready")
}

func (h *Handler) handleAppMigrationReconcile(w http.ResponseWriter, req *http.Request) {
	h.handleAppMigrationAction(w, req, "reconcile")
}

func (h *Handler) handleAppMigrationAction(w http.ResponseWriter, req *http.Request, action string) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.appRuntime == nil {
		h.writeJSONError(w, http.StatusBadRequest, "app runtime not configured")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form")
		return
	}
	name := strings.TrimSpace(req.FormValue("app"))
	if name == "" {
		name = strings.TrimSpace(req.FormValue("name"))
	}
	if name == "" {
		h.writeJSONError(w, http.StatusBadRequest, "app is required")
		return
	}

	status, target, ready, err := h.runAppMigrationAction(req, action, name)
	if err != nil {
		var reqErr migrationActionRequestError
		switch {
		case errors.As(err, &reqErr):
			h.writeJSONError(w, http.StatusBadRequest, reqErr.Error())
		case errors.Is(err, errMigrationActionUnknown):
			h.writeJSONError(w, http.StatusBadRequest, err.Error())
		default:
			h.writeMigrationError(w, err)
		}
		return
	}

	response := map[string]any{
		"status":    "ok",
		"action":    action,
		"app":       name,
		"migration": mapMigrationStatus(status),
	}
	if action == "abort" {
		if strings.TrimSpace(target) == "" {
			target = appruntime.MigrationAbortToCheckpoint
		}
		response["to"] = target
	}
	if action == "drain-ready" {
		response["ready"] = ready
	}
	h.writeJSON(w, http.StatusOK, response)
}

var errMigrationActionUnknown = errors.New("unknown migration action")

type migrationActionRequestError struct {
	msg string
}

func (e migrationActionRequestError) Error() string {
	return e.msg
}

func (h *Handler) runAppMigrationAction(
	req *http.Request,
	action, name string,
) (status appruntime.MigrationStatus, target string, ready bool, err error) {
	switch action {
	case "retry":
		status, err = h.appRuntime.RetryMigration(name)
	case "abort":
		target = strings.TrimSpace(req.FormValue("to"))
		status, err = h.appRuntime.AbortMigration(name, target)
	case "drain-ready":
		readyRaw := strings.TrimSpace(req.FormValue("ready"))
		if readyRaw == "" {
			return status, target, ready, migrationActionRequestError{msg: "ready is required"}
		}
		ready, err = strconv.ParseBool(readyRaw)
		if err != nil {
			return status, target, ready, migrationActionRequestError{msg: "ready must be true or false"}
		}
		err = h.appRuntime.SetMigrationDrainReady(name, ready)
		if err == nil {
			status, err = h.appRuntime.GetMigrationStatus(name)
		}
	case "reconcile":
		recordID := strings.TrimSpace(req.FormValue("record_id"))
		resolution := strings.TrimSpace(req.FormValue("resolution"))
		if recordID == "" || resolution == "" {
			return status, target, ready, migrationActionRequestError{msg: "record_id and resolution are required"}
		}
		status, err = h.appRuntime.ReconcileMigration(name, recordID, resolution)
	default:
		err = errMigrationActionUnknown
	}
	return status, target, ready, err
}

func (h *Handler) writeMigrationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, appruntime.ErrPackageNotFound):
		h.writeJSONError(w, http.StatusNotFound, err.Error())
	default:
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
	}
}

func mapMigrationStatus(status appruntime.MigrationStatus) map[string]any {
	pending := make([]map[string]any, 0, len(status.PendingRecords))
	for _, record := range status.PendingRecords {
		pending = append(pending, map[string]any{
			"record_id":              record.RecordID,
			"recommended_resolution": record.RecommendedResolution,
		})
	}
	drainBlockedSince := ""
	if !status.DrainBlockedAt.IsZero() {
		drainBlockedSince = status.DrainBlockedAt.UTC().Format(time.RFC3339Nano)
	}

	return map[string]any{
		"app":                   status.App,
		"version":               status.Version,
		"revision":              status.Revision,
		"steps_planned":         status.StepsPlanned,
		"steps_completed":       status.StepsCompleted,
		"last_step":             status.LastStep,
		"verdict":               status.Verdict,
		"last_error":            status.LastError,
		"journal_path":          status.JournalPath,
		"reconciliation_path":   status.ReconciliationPath,
		"executor_ready":        status.ExecutorReady,
		"requires_drain":        status.RequiresDrain,
		"drain_ready":           status.DrainReady,
		"drain_timeout_seconds": int(status.DrainTimeout.Seconds()),
		"drain_blocked_since":   drainBlockedSince,
		"pending_records":       pending,
	}
}

func rollbackDataModeFromForm(form url.Values) (string, error) {
	mode := strings.TrimSpace(firstNonEmpty(form.Get("mode"), form.Get("rollback_mode"), form.Get("data_mode")))
	keepData := truthyFormValue(form.Get("keep_data"))
	archiveData := truthyFormValue(form.Get("archive_data"))
	purge := truthyFormValue(form.Get("purge"))
	selected := 0
	if mode != "" {
		selected++
	}
	if keepData {
		selected++
		mode = appruntime.RollbackDataModeKeepData
	}
	if archiveData {
		selected++
		mode = appruntime.RollbackDataModeArchiveData
	}
	if purge {
		selected++
		mode = appruntime.RollbackDataModePurge
	}
	if selected > 1 {
		return "", fmt.Errorf("%w: choose exactly one of keep_data, archive_data, purge, or mode", appruntime.ErrRollbackModeInvalid)
	}
	return normalizeRollbackDataMode(mode), nil
}

func normalizeRollbackDataMode(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	if normalized == "" {
		return appruntime.RollbackDataModeArchiveData
	}
	return normalized
}

func parsePositiveOptionalInt(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return 0, errors.New("invalid positive int")
	}
	return v, nil
}

func readMigrationJournalTail(path string, step, limit int) ([]string, bool, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	lines := make([]string, 0, limit)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if step > 0 && !migrationLogMatchesStep(line, step) {
			continue
		}
		lines = append(lines, line)
		if len(lines) > limit {
			lines = lines[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, true, err
	}
	return lines, true, nil
}

func migrationLogMatchesStep(line string, step int) bool {
	var payload map[string]any
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		return false
	}
	raw, ok := payload["step"]
	if !ok {
		return false
	}
	switch v := raw.(type) {
	case float64:
		return v == float64(step)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		return err == nil && parsed == step
	default:
		return false
	}
}
