package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/appruntime"
	"github.com/curtcox/terminals/terminal_server/internal/config"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestAppsEndpointsListReloadAndRollback(t *testing.T) {
	appRoot := createTestAppPackage(t, "sound_watch", "1.0.0")
	if err := os.MkdirAll(filepath.Join(appRoot, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appRoot, "migrate", "0001_v1_to_v2.tal"), []byte("def migrate():\n    return\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migrate step) error = %v", err)
	}
	appRuntime := appruntime.NewRuntime()
	if _, err := appRuntime.LoadPackage(context.Background(), appRoot); err != nil {
		t.Fatalf("LoadPackage(v1) error = %v", err)
	}

	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "kitchen-1", DeviceName: "Kitchen"})
	control := transport.NewControlService("HomeServer", devices)
	engine := scenario.NewEngine()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	h := NewHandler(control, runtime, nil, nil, appRuntime, func() {}, devices, config.Config{MDNSName: "HomeServer"}, nil, nil)

	listReq := httptest.NewRequest(http.MethodGet, "/admin/api/apps", nil)
	listW := httptest.NewRecorder()
	h.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("apps list status = %d, want 200 body=%s", listW.Code, listW.Body.String())
	}
	var listed map[string][]map[string]any
	if err := json.Unmarshal(listW.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode apps list: %v", err)
	}
	if len(listed["apps"]) != 1 || listed["apps"][0]["name"] != "sound_watch" {
		t.Fatalf("apps list = %+v, want one sound_watch app", listed["apps"])
	}

	time.Sleep(5 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(appRoot, "manifest.toml"), []byte(
		"name = \"sound_watch\"\nversion = \"1.1.0\"\nlanguage = \"tal/1\"\nexports = [\"watch\"]\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v2) error = %v", err)
	}
	reloadReq := httptest.NewRequest(http.MethodPost, "/admin/api/apps/reload", strings.NewReader(url.Values{
		"app": {"sound_watch"},
	}.Encode()))
	reloadReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reloadW := httptest.NewRecorder()
	h.ServeHTTP(reloadW, reloadReq)
	if reloadW.Code != http.StatusOK {
		t.Fatalf("reload status = %d, want 200 body=%s", reloadW.Code, reloadW.Body.String())
	}
	var reloaded map[string]any
	if err := json.Unmarshal(reloadW.Body.Bytes(), &reloaded); err != nil {
		t.Fatalf("decode reload: %v", err)
	}
	if reloaded["version"] != "1.1.0" {
		t.Fatalf("reloaded version = %v, want 1.1.0", reloaded["version"])
	}

	rollbackReq := httptest.NewRequest(http.MethodPost, "/admin/api/apps/rollback", strings.NewReader(url.Values{
		"app": {"sound_watch"},
	}.Encode()))
	rollbackReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rollbackW := httptest.NewRecorder()
	h.ServeHTTP(rollbackW, rollbackReq)
	if rollbackW.Code != http.StatusOK {
		t.Fatalf("rollback status = %d, want 200 body=%s", rollbackW.Code, rollbackW.Body.String())
	}
	var rolledBack map[string]any
	if err := json.Unmarshal(rollbackW.Body.Bytes(), &rolledBack); err != nil {
		t.Fatalf("decode rollback: %v", err)
	}
	if rolledBack["version"] != "1.0.0" {
		t.Fatalf("rolled back version = %v, want 1.0.0", rolledBack["version"])
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/admin/api/apps/migrate/status?app=sound_watch", nil)
	statusW := httptest.NewRecorder()
	h.ServeHTTP(statusW, statusReq)
	if statusW.Code != http.StatusOK {
		t.Fatalf("migrate status code = %d, want 200 body=%s", statusW.Code, statusW.Body.String())
	}
	var statusBody map[string]any
	if err := json.Unmarshal(statusW.Body.Bytes(), &statusBody); err != nil {
		t.Fatalf("decode migrate status: %v", err)
	}
	migration, _ := statusBody["migration"].(map[string]any)
	if fmt.Sprint(migration["app"]) != "sound_watch" {
		t.Fatalf("migration app = %v, want sound_watch", migration["app"])
	}
	if fmt.Sprint(migration["verdict"]) != "idle" {
		t.Fatalf("migration verdict = %v, want idle", migration["verdict"])
	}
	if fmt.Sprint(migration["last_step"]) != "0" {
		t.Fatalf("migration last_step = %v, want 0", migration["last_step"])
	}
	if fmt.Sprint(migration["requires_drain"]) != "false" {
		t.Fatalf("migration requires_drain = %v, want false", migration["requires_drain"])
	}
	if fmt.Sprint(migration["drain_ready"]) != "true" {
		t.Fatalf("migration drain_ready = %v, want true", migration["drain_ready"])
	}
	if fmt.Sprint(migration["drain_timeout_seconds"]) != "90" {
		t.Fatalf("migration drain_timeout_seconds = %v, want 90", migration["drain_timeout_seconds"])
	}
	if fmt.Sprint(migration["drain_blocked_since"]) != "" {
		t.Fatalf("migration drain_blocked_since = %v, want empty", migration["drain_blocked_since"])
	}
	pendingRecords, _ := migration["pending_records"].([]any)
	if len(pendingRecords) != 0 {
		t.Fatalf("migration pending_records len = %d, want 0", len(pendingRecords))
	}
	journalPath := fmt.Sprint(migration["journal_path"])
	if journalPath == "" {
		t.Fatalf("migration journal_path empty")
	}
	journalFile := filepath.Join(appRoot, filepath.FromSlash(journalPath))
	if err := os.MkdirAll(filepath.Dir(journalFile), 0o755); err != nil {
		t.Fatalf("MkdirAll(journal dir) error = %v", err)
	}
	if err := os.WriteFile(journalFile, []byte("{\"step\":1,\"event\":\"start\"}\n{\"step\":2,\"event\":\"commit\"}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(journal) error = %v", err)
	}

	logsReq := httptest.NewRequest(http.MethodGet, "/admin/api/apps/migrate/logs?app=sound_watch&step=2", nil)
	logsW := httptest.NewRecorder()
	h.ServeHTTP(logsW, logsReq)
	if logsW.Code != http.StatusOK {
		t.Fatalf("migrate logs code = %d, want 200 body=%s", logsW.Code, logsW.Body.String())
	}
	var logsBody map[string]any
	if err := json.Unmarshal(logsW.Body.Bytes(), &logsBody); err != nil {
		t.Fatalf("decode migrate logs: %v", err)
	}
	if fmt.Sprint(logsBody["journal_exists"]) != "true" {
		t.Fatalf("migrate logs journal_exists = %v, want true", logsBody["journal_exists"])
	}
	lines, _ := logsBody["lines"].([]any)
	if len(lines) != 1 || !strings.Contains(fmt.Sprint(lines[0]), `"step":2`) {
		t.Fatalf("migrate logs lines = %v, want only step 2 entry", lines)
	}

	invalidLogsReq := httptest.NewRequest(http.MethodGet, "/admin/api/apps/migrate/logs?app=sound_watch&step=0", nil)
	invalidLogsW := httptest.NewRecorder()
	h.ServeHTTP(invalidLogsW, invalidLogsReq)
	if invalidLogsW.Code != http.StatusBadRequest {
		t.Fatalf("invalid migrate logs code = %d, want 400 body=%s", invalidLogsW.Code, invalidLogsW.Body.String())
	}

	retryReq := httptest.NewRequest(http.MethodPost, "/admin/api/apps/migrate/retry", strings.NewReader(url.Values{"app": {"sound_watch"}}.Encode()))
	retryReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	retryW := httptest.NewRecorder()
	h.ServeHTTP(retryW, retryReq)
	if retryW.Code != http.StatusOK {
		t.Fatalf("migrate retry code = %d, want 200 body=%s", retryW.Code, retryW.Body.String())
	}
	var retryBody map[string]any
	if err := json.Unmarshal(retryW.Body.Bytes(), &retryBody); err != nil {
		t.Fatalf("decode migrate retry: %v", err)
	}
	if retryBody["status"] != "ok" {
		t.Fatalf("migrate retry status = %v, want ok", retryBody["status"])
	}
	retryMigration, _ := retryBody["migration"].(map[string]any)
	if fmt.Sprint(retryMigration["verdict"]) != "ok" {
		t.Fatalf("migrate retry verdict = %v, want ok", retryMigration["verdict"])
	}
	if fmt.Sprint(retryMigration["last_error"]) != "" {
		t.Fatalf("migrate retry last_error = %v, want empty", retryMigration["last_error"])
	}

	drainReadyReq := httptest.NewRequest(http.MethodPost, "/admin/api/apps/migrate/drain-ready", strings.NewReader(url.Values{"app": {"sound_watch"}, "ready": {"true"}}.Encode()))
	drainReadyReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	drainReadyW := httptest.NewRecorder()
	h.ServeHTTP(drainReadyW, drainReadyReq)
	if drainReadyW.Code != http.StatusOK {
		t.Fatalf("migrate drain-ready code = %d, want 200 body=%s", drainReadyW.Code, drainReadyW.Body.String())
	}
	var drainReadyBody map[string]any
	if err := json.Unmarshal(drainReadyW.Body.Bytes(), &drainReadyBody); err != nil {
		t.Fatalf("decode migrate drain-ready: %v", err)
	}
	if drainReadyBody["status"] != "ok" {
		t.Fatalf("migrate drain-ready status = %v, want ok", drainReadyBody["status"])
	}
	if fmt.Sprint(drainReadyBody["action"]) != "drain-ready" {
		t.Fatalf("migrate drain-ready action = %v, want drain-ready", drainReadyBody["action"])
	}
	if fmt.Sprint(drainReadyBody["ready"]) != "true" {
		t.Fatalf("migrate drain-ready ready = %v, want true", drainReadyBody["ready"])
	}

	abortReq := httptest.NewRequest(http.MethodPost, "/admin/api/apps/migrate/abort", strings.NewReader(url.Values{"app": {"sound_watch"}, "to": {"baseline"}}.Encode()))
	abortReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	abortW := httptest.NewRecorder()
	h.ServeHTTP(abortW, abortReq)
	if abortW.Code != http.StatusOK {
		t.Fatalf("migrate abort code = %d, want 200 body=%s", abortW.Code, abortW.Body.String())
	}
	var abortBody map[string]any
	if err := json.Unmarshal(abortW.Body.Bytes(), &abortBody); err != nil {
		t.Fatalf("decode migrate abort: %v", err)
	}
	if abortBody["status"] != "ok" {
		t.Fatalf("migrate abort status = %v, want ok", abortBody["status"])
	}
	if fmt.Sprint(abortBody["to"]) != "baseline" {
		t.Fatalf("migrate abort to = %v, want baseline", abortBody["to"])
	}

	invalidAbortReq := httptest.NewRequest(http.MethodPost, "/admin/api/apps/migrate/abort", strings.NewReader(url.Values{"app": {"sound_watch"}, "to": {"bad"}}.Encode()))
	invalidAbortReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	invalidAbortW := httptest.NewRecorder()
	h.ServeHTTP(invalidAbortW, invalidAbortReq)
	if invalidAbortW.Code != http.StatusBadRequest {
		t.Fatalf("invalid migrate abort code = %d, want 400 body=%s", invalidAbortW.Code, invalidAbortW.Body.String())
	}
}

func TestAppsRollbackRejectsKeepDataWithoutDowngradeSteps(t *testing.T) {
	appRoot := createTestAppPackage(t, "sound_watch_keep", "1.0.0")
	appRuntime := appruntime.NewRuntime()
	if _, err := appRuntime.LoadPackage(context.Background(), appRoot); err != nil {
		t.Fatalf("LoadPackage(v1) error = %v", err)
	}

	time.Sleep(5 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(appRoot, "manifest.toml"), []byte(
		"name = \"sound_watch_keep\"\nversion = \"1.1.0\"\nlanguage = \"tal/1\"\nexports = [\"watch\"]\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v2) error = %v", err)
	}
	if _, changed, err := appRuntime.ReloadPackage(context.Background(), "sound_watch_keep"); err != nil || !changed {
		t.Fatalf("ReloadPackage(v2) = (%v, %v), want changed true no error", err, changed)
	}

	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "kitchen-1", DeviceName: "Kitchen"})
	control := transport.NewControlService("HomeServer", devices)
	engine := scenario.NewEngine()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	h := NewHandler(control, runtime, nil, nil, appRuntime, func() {}, devices, config.Config{MDNSName: "HomeServer"}, nil, nil)

	rollbackReq := httptest.NewRequest(http.MethodPost, "/admin/api/apps/rollback", strings.NewReader(url.Values{
		"app":       {"sound_watch_keep"},
		"keep_data": {"1"},
	}.Encode()))
	rollbackReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rollbackW := httptest.NewRecorder()
	h.ServeHTTP(rollbackW, rollbackReq)
	if rollbackW.Code != http.StatusBadRequest {
		t.Fatalf("rollback status = %d, want 400 body=%s", rollbackW.Code, rollbackW.Body.String())
	}
	if !strings.Contains(rollbackW.Body.String(), appruntime.ErrRollbackKeepDataRequiresDowngrade.Error()) {
		t.Fatalf("rollback body missing keep-data downgrade error: %s", rollbackW.Body.String())
	}
}

func TestAppsRollbackBlockedByReconcilePendingReturnsMigrationStatus(t *testing.T) {
	appRoot := createTestAppPackage(t, "sound_watch_pending", "1.0.0")
	appRuntime := appruntime.NewRuntime()
	if _, err := appRuntime.LoadPackage(context.Background(), appRoot); err != nil {
		t.Fatalf("LoadPackage(v1) error = %v", err)
	}

	time.Sleep(5 * time.Millisecond)
	if err := os.MkdirAll(filepath.Join(appRoot, "migrate"), 0o755); err != nil {
		t.Fatalf("MkdirAll(migrate) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appRoot, "migrate", "0001_1.0.0_to_1.1.0.tal"), []byte("def migrate(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(migration step) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(appRoot, "manifest.toml"), []byte(
		"name = \"sound_watch_pending\"\nversion = \"1.1.0\"\nlanguage = \"tal/1\"\nexports = [\"watch\"]\n\n[migrate]\ndeclared_steps = 1\n\n[[migrate.step]]\nfrom = \"1.0.0\"\nto = \"1.1.0\"\ncompatibility = \"compatible\"\ndrain_policy = \"none\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest v2) error = %v", err)
	}
	if _, changed, err := appRuntime.ReloadPackage(context.Background(), "sound_watch_pending"); err != nil || !changed {
		t.Fatalf("ReloadPackage(v2) = (%v, %v), want changed true no error", err, changed)
	}
	status, err := appRuntime.RetryMigration("sound_watch_pending")
	if err != nil {
		t.Fatalf("RetryMigration() error = %v", err)
	}
	if status.JournalPath == "" {
		t.Fatalf("RetryMigration() journal_path empty")
	}
	journalFile := filepath.Join(appRoot, filepath.FromSlash(status.JournalPath))
	if err := os.MkdirAll(filepath.Dir(journalFile), 0o755); err != nil {
		t.Fatalf("MkdirAll(journal dir) error = %v", err)
	}
	journalEntry := `{"event":"artifact_inverse_failed","record_id":"artifact:photo-frame","recommended_resolution":"manual"}` + "\n"
	file, err := os.OpenFile(filepath.Clean(journalFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("OpenFile(journal) error = %v", err)
	}
	if _, err := file.WriteString(journalEntry); err != nil {
		_ = file.Close()
		t.Fatalf("WriteString(journal) error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close(journal) error = %v", err)
	}
	if status, err := appRuntime.AbortMigration("sound_watch_pending", appruntime.MigrationAbortToBaseline); !errors.Is(err, appruntime.ErrMigrationReconcilePending) {
		journal, _ := os.ReadFile(journalFile)
		t.Fatalf("AbortMigration(to baseline) = (%+v, %v), want ErrMigrationReconcilePending; journal=%s", status, err, string(journal))
	}

	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "kitchen-1", DeviceName: "Kitchen"})
	control := transport.NewControlService("HomeServer", devices)
	engine := scenario.NewEngine()
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	h := NewHandler(control, runtime, nil, nil, appRuntime, func() {}, devices, config.Config{MDNSName: "HomeServer"}, nil, nil)

	rollbackReq := httptest.NewRequest(http.MethodPost, "/admin/api/apps/rollback", strings.NewReader(url.Values{
		"app": {"sound_watch_pending"},
	}.Encode()))
	rollbackReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rollbackW := httptest.NewRecorder()
	h.ServeHTTP(rollbackW, rollbackReq)
	if rollbackW.Code != http.StatusConflict {
		t.Fatalf("rollback status = %d, want 409 body=%s", rollbackW.Code, rollbackW.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rollbackW.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode rollback body: %v", err)
	}
	if body["status"] != "blocked" || body["error"] != appruntime.ErrMigrationReconcilePending.Error() {
		t.Fatalf("rollback body status/error = %v/%v, want blocked/%q", body["status"], body["error"], appruntime.ErrMigrationReconcilePending.Error())
	}
	migration, _ := body["migration"].(map[string]any)
	if fmt.Sprint(migration["verdict"]) != "reconcile_pending" {
		t.Fatalf("rollback migration verdict = %v, want reconcile_pending", migration["verdict"])
	}
	pendingRecords, _ := migration["pending_records"].([]any)
	if len(pendingRecords) != 1 {
		t.Fatalf("pending_records len = %d, want 1", len(pendingRecords))
	}
	pending, _ := pendingRecords[0].(map[string]any)
	if fmt.Sprint(pending["record_id"]) != "artifact:photo-frame" || fmt.Sprint(pending["recommended_resolution"]) != "manual" {
		t.Fatalf("pending record = %+v, want artifact:photo-frame/manual", pending)
	}
}
