// Package admin serves a lightweight web dashboard and JSON admin APIs.
package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/config"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
)

// Handler serves a lightweight admin dashboard and JSON control APIs.
type Handler struct {
	control *transport.ControlService
	runtime *scenario.Runtime
	devices *device.Manager
	cfg     config.Config
	now     func() time.Time
}

// NewHandler builds an admin handler with dashboard and API routes.
func NewHandler(control *transport.ControlService, runtime *scenario.Runtime, devices *device.Manager, cfg config.Config) http.Handler {
	h := &Handler{
		control: control,
		runtime: runtime,
		devices: devices,
		cfg:     cfg,
		now:     time.Now,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/admin", h.handleDashboard)
	mux.HandleFunc("/admin/api/status", h.handleStatus)
	mux.HandleFunc("/admin/api/devices", h.handleDevices)
	mux.HandleFunc("/admin/api/scenarios", h.handleScenarios)
	mux.HandleFunc("/admin/api/scenarios/start", h.handleStartScenario)
	mux.HandleFunc("/admin/api/scenarios/stop", h.handleStopScenario)
	return mux
}

func (h *Handler) handleDashboard(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := dashboardTemplate.Execute(w, map[string]string{
		"ServerID": strings.TrimSpace(h.cfg.MDNSName),
	}); err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("render dashboard: %v", err))
	}
}

func (h *Handler) handleStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	status := map[string]any{
		"server":  h.control.StatusData(),
		"runtime": h.runtime.StatusData(),
		"config": map[string]any{
			"grpc_address":               h.cfg.GRPCAddress(),
			"mdns_service":               h.cfg.MDNSService,
			"mdns_name":                  h.cfg.MDNSName,
			"version":                    h.cfg.Version,
			"heartbeat_timeout_seconds":  h.cfg.HeartbeatTimeoutSeconds,
			"liveness_interval_seconds":  h.cfg.LivenessReconcileIntervalSecs,
			"due_timer_interval_seconds": h.cfg.DueTimerProcessIntervalSecs,
			"recording_dir":              h.cfg.RecordingDir,
			"photo_frame_dir":            h.cfg.PhotoFrameDir,
			"admin_http_host":            h.cfg.AdminHTTPHost,
			"admin_http_port":            h.cfg.AdminHTTPPort,
		},
		"timestamp_unix_ms": h.now().UTC().UnixMilli(),
	}
	h.writeJSON(w, http.StatusOK, status)
}

func (h *Handler) handleDevices(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	activeByDevice := h.runtime.Engine.ActiveSnapshot()
	type deviceView struct {
		DeviceID       string            `json:"device_id"`
		DeviceName     string            `json:"device_name"`
		DeviceType     string            `json:"device_type"`
		Platform       string            `json:"platform"`
		State          string            `json:"state"`
		LastHeartbeat  int64             `json:"last_heartbeat_unix_ms"`
		RegisteredAt   int64             `json:"registered_at_unix_ms"`
		ActiveScenario string            `json:"active_scenario,omitempty"`
		Capabilities   map[string]string `json:"capabilities"`
	}

	devices := h.devices.List()
	views := make([]deviceView, 0, len(devices))
	for _, d := range devices {
		views = append(views, deviceView{
			DeviceID:       d.DeviceID,
			DeviceName:     d.DeviceName,
			DeviceType:     d.DeviceType,
			Platform:       d.Platform,
			State:          string(d.State),
			LastHeartbeat:  d.LastHeartbeat.UTC().UnixMilli(),
			RegisteredAt:   d.RegisteredAt.UTC().UnixMilli(),
			ActiveScenario: activeByDevice[d.DeviceID],
			Capabilities:   d.Capabilities,
		})
	}

	h.writeJSON(w, http.StatusOK, map[string]any{"devices": views})
}

func (h *Handler) handleScenarios(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	activeByDevice := h.runtime.Engine.ActiveSnapshot()
	registry := h.runtime.Engine.RegistrySnapshot()
	activeDevicesByScenario := make(map[string][]string)
	for deviceID, scenarioName := range activeByDevice {
		activeDevicesByScenario[scenarioName] = append(activeDevicesByScenario[scenarioName], deviceID)
	}

	type scenarioView struct {
		Name          string   `json:"name"`
		Priority      int      `json:"priority"`
		ActiveDevices []string `json:"active_devices"`
	}
	views := make([]scenarioView, 0, len(registry))
	for _, reg := range registry {
		activeDevices := activeDevicesByScenario[reg.Name]
		sort.Strings(activeDevices)
		views = append(views, scenarioView{
			Name:          reg.Name,
			Priority:      int(reg.Priority),
			ActiveDevices: activeDevices,
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"scenarios": views})
}

func (h *Handler) handleStartScenario(w http.ResponseWriter, req *http.Request) {
	h.handleScenarioCommand(w, req, true)
}

func (h *Handler) handleStopScenario(w http.ResponseWriter, req *http.Request) {
	h.handleScenarioCommand(w, req, false)
}

func (h *Handler) handleScenarioCommand(w http.ResponseWriter, req *http.Request, start bool) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form")
		return
	}

	scenarioName := strings.TrimSpace(req.FormValue("scenario"))
	if scenarioName == "" {
		h.writeJSONError(w, http.StatusBadRequest, "scenario is required")
		return
	}

	deviceIDs := parseDeviceIDs(req.FormValue("device_ids"))
	if deviceID := strings.TrimSpace(req.FormValue("device_id")); deviceID != "" {
		deviceIDs = append(deviceIDs, deviceID)
	}
	deviceIDs = normalizeDeviceIDs(deviceIDs)

	ctx, cancel := context.WithTimeout(req.Context(), 3*time.Second)
	defer cancel()

	var (
		matched string
		err     error
		action  = "stop"
	)
	if start {
		action = "start"
		matched, err = h.runtime.StartScenario(ctx, scenarioName, deviceIDs)
	} else {
		matched, err = h.runtime.StopScenario(ctx, scenarioName, deviceIDs)
	}
	if err != nil {
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"status":            "ok",
		"action":            action,
		"scenario":          matched,
		"requested":         scenarioName,
		"target_device_ids": deviceIDs,
	})
}

func parseDeviceIDs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func normalizeDeviceIDs(deviceIDs []string) []string {
	out := make([]string, 0, len(deviceIDs))
	seen := make(map[string]struct{}, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		deviceID = strings.TrimSpace(deviceID)
		if deviceID == "" {
			continue
		}
		if _, exists := seen[deviceID]; exists {
			continue
		}
		seen[deviceID] = struct{}{}
		out = append(out, deviceID)
	}
	return out
}

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

var dashboardTemplate = template.Must(template.New("admin").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Terminals Admin Dashboard</title>
  <style>
    :root { color-scheme: light; }
    body { margin: 0; font-family: ui-sans-serif, system-ui, sans-serif; background: #f5f7fb; color: #0f172a; }
    main { max-width: 1080px; margin: 0 auto; padding: 20px; }
    h1 { margin: 0 0 16px; }
    section { background: #fff; border: 1px solid #dbe2ee; border-radius: 12px; padding: 14px; margin-bottom: 14px; }
    pre { margin: 0; overflow: auto; max-height: 360px; background: #0f172a; color: #e2e8f0; padding: 12px; border-radius: 8px; }
    label { display: block; margin-bottom: 8px; }
    input { width: 100%; max-width: 420px; padding: 8px; border: 1px solid #cbd5e1; border-radius: 8px; }
    .row { display: flex; gap: 10px; flex-wrap: wrap; align-items: flex-end; }
    button { padding: 8px 14px; border: 1px solid #334155; border-radius: 8px; background: #1e293b; color: #fff; cursor: pointer; }
    .secondary { background: #fff; color: #1e293b; }
  </style>
</head>
<body>
<main>
  <h1>Terminals Admin Dashboard</h1>
  <p>Server: <strong>{{.ServerID}}</strong></p>

  <section>
    <h2>Scenario Control</h2>
    <div class="row">
      <label>Scenario name<input id="scenario" placeholder="terminal" /></label>
      <label>Device IDs (comma-separated)<input id="device_ids" placeholder="kitchen-1,hall-2" /></label>
      <button id="start_btn">Start</button>
      <button class="secondary" id="stop_btn">Stop</button>
    </div>
    <pre id="scenario_result">{}</pre>
  </section>

  <section>
    <h2>Status</h2>
    <pre id="status">{}</pre>
  </section>

  <section>
    <h2>Devices</h2>
    <pre id="devices">[]</pre>
  </section>

  <section>
    <h2>Scenarios</h2>
    <pre id="scenarios">[]</pre>
  </section>
</main>
<script>
async function loadJSON(path) {
  const response = await fetch(path);
  return await response.json();
}
function format(json) {
  return JSON.stringify(json, null, 2);
}
async function refresh() {
  document.getElementById('status').textContent = format(await loadJSON('/admin/api/status'));
  document.getElementById('devices').textContent = format(await loadJSON('/admin/api/devices'));
  document.getElementById('scenarios').textContent = format(await loadJSON('/admin/api/scenarios'));
}
async function scenarioCommand(path) {
  const scenario = document.getElementById('scenario').value.trim();
  const deviceIDs = document.getElementById('device_ids').value.trim();
  const body = new URLSearchParams();
  body.set('scenario', scenario);
  body.set('device_ids', deviceIDs);
  const response = await fetch(path, { method: 'POST', body });
  const json = await response.json();
  document.getElementById('scenario_result').textContent = format(json);
  await refresh();
}
document.getElementById('start_btn').addEventListener('click', () => scenarioCommand('/admin/api/scenarios/start'));
document.getElementById('stop_btn').addEventListener('click', () => scenarioCommand('/admin/api/scenarios/stop'));
refresh();
setInterval(refresh, 3000);
</script>
</body>
</html>`))
