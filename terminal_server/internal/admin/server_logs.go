package admin

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/curtcox/terminals/terminal_server/internal/eventlog/query"
)

func (h *Handler) handleLogs(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	args := logFilterArgs(req)
	records, err := query.Search(h.cfg.LogDir, args, h.now().UTC())
	if err != nil {
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(records) > 200 {
		records = records[len(records)-200:]
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := logsTemplate.Execute(w, map[string]any{
		"Count":   len(records),
		"Filters": strings.Join(args, " "),
		"Rows":    records,
	}); err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("render logs: %v", err))
	}
}

func (h *Handler) handleLogsJSONL(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	records, err := query.Search(h.cfg.LogDir, logFilterArgs(req), h.now().UTC())
	if err != nil {
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	enc := json.NewEncoder(w)
	for _, record := range records {
		if err := enc.Encode(record); err != nil {
			return
		}
	}
}

func (h *Handler) handleLogsTrace(w http.ResponseWriter, req *http.Request) {
	traceID := strings.TrimSpace(strings.TrimPrefix(req.URL.Path, "/admin/logs/trace/"))
	if traceID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "trace id is required")
		return
	}
	records, err := query.ReadAll(h.cfg.LogDir)
	if err != nil {
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	events := query.Trace(records, traceID)
	if req.URL.Query().Get("format") == "json" {
		h.writeJSON(w, http.StatusOK, map[string]any{"trace_id": traceID, "events": events})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := traceTemplate.Execute(w, map[string]any{
		"Title":  "Trace Timeline",
		"ID":     traceID,
		"Events": events,
		"Kind":   "trace",
	}); err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("render trace logs: %v", err))
	}
}

func (h *Handler) handleLogsActivation(w http.ResponseWriter, req *http.Request) {
	activationID := strings.TrimSpace(strings.TrimPrefix(req.URL.Path, "/admin/logs/activation/"))
	if activationID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "activation id is required")
		return
	}
	records, err := query.ReadAll(h.cfg.LogDir)
	if err != nil {
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	events := query.Activation(records, activationID)
	if req.URL.Query().Get("format") == "json" {
		h.writeJSON(w, http.StatusOK, map[string]any{"activation_id": activationID, "events": events})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := traceTemplate.Execute(w, map[string]any{
		"Title":  "Activation Timeline",
		"ID":     activationID,
		"Events": events,
		"Kind":   "activation",
	}); err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("render activation logs: %v", err))
	}
}

func logFilterArgs(req *http.Request) []string {
	out := make([]string, 0)
	values := req.URL.Query()
	for key, items := range values {
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if key == "q" {
				out = append(out, item)
				continue
			}
			out = append(out, key+"="+item)
		}
	}
	return out
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
    <h2>World Model</h2>
    <div class="row">
      <label>Device ID<input id="placement_device_id" placeholder="kitchen-1" /></label>
      <label>Zone<input id="placement_zone" placeholder="kitchen" /></label>
      <label>Roles (comma-separated)<input id="placement_roles" placeholder="kitchen_display,screen" /></label>
      <label>Mobility<input id="placement_mobility" placeholder="fixed" /></label>
      <label>Affinity<input id="placement_affinity" placeholder="home" /></label>
      <button id="placement_save_btn">Save placement</button>
    </div>
    <pre id="placement_result">{}</pre>
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

  <section>
    <h2>Activations</h2>
    <pre id="activations">[]</pre>
  </section>

  <section>
    <h2>Apps</h2>
    <div class="row">
      <label>App name<input id="app_name" placeholder="sound_watch" /></label>
      <button id="app_reload_btn">Reload</button>
      <button class="secondary" id="app_rollback_btn">Rollback</button>
    </div>
    <pre id="app_result">{}</pre>
    <pre id="apps">[]</pre>
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
  document.getElementById('activations').textContent = format(await loadJSON('/admin/api/activations'));
  document.getElementById('apps').textContent = format(await loadJSON('/admin/api/apps'));
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
async function savePlacement() {
  const body = new URLSearchParams();
  body.set('device_id', document.getElementById('placement_device_id').value.trim());
  body.set('zone', document.getElementById('placement_zone').value.trim());
  body.set('roles', document.getElementById('placement_roles').value.trim());
  body.set('mobility', document.getElementById('placement_mobility').value.trim());
  body.set('affinity', document.getElementById('placement_affinity').value.trim());
  const response = await fetch('/admin/api/devices/placement', { method: 'POST', body });
  const json = await response.json();
  document.getElementById('placement_result').textContent = format(json);
  await refresh();
}
async function appCommand(path) {
  const body = new URLSearchParams();
  body.set('app', document.getElementById('app_name').value.trim());
  const response = await fetch(path, { method: 'POST', body });
  const json = await response.json();
  document.getElementById('app_result').textContent = format(json);
  await refresh();
}
document.getElementById('start_btn').addEventListener('click', () => scenarioCommand('/admin/api/scenarios/start'));
document.getElementById('stop_btn').addEventListener('click', () => scenarioCommand('/admin/api/scenarios/stop'));
document.getElementById('placement_save_btn').addEventListener('click', () => savePlacement());
document.getElementById('app_reload_btn').addEventListener('click', () => appCommand('/admin/api/apps/reload'));
document.getElementById('app_rollback_btn').addEventListener('click', () => appCommand('/admin/api/apps/rollback'));
refresh();
setInterval(refresh, 3000);
</script>
</body>
</html>`))

var logsTemplate = template.Must(template.New("logs").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Terminals Event Logs</title>
  <style>
    body { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; background: #0b1220; color: #e2e8f0; margin: 0; }
    main { padding: 16px; }
    a { color: #7dd3fc; }
    table { width: 100%; border-collapse: collapse; margin-top: 12px; }
    th, td { border-bottom: 1px solid #1e293b; text-align: left; padding: 6px; font-size: 13px; vertical-align: top; }
    th { color: #93c5fd; }
    .err { color: #fda4af; }
  </style>
</head>
<body>
<main>
  <h1>Event Logs</h1>
  <p>matching events: {{.Count}} | filters: {{.Filters}}</p>
  <p><a href="/admin">Back to dashboard</a></p>
  <table>
    <thead><tr><th>ts</th><th>level</th><th>event</th><th>component</th><th>msg</th><th>trace</th><th>activation</th></tr></thead>
    <tbody>
      {{range .Rows}}
      <tr>
        <td>{{index . "ts"}}</td>
        <td>{{index . "level"}}</td>
        <td>{{index . "event"}}</td>
        <td>{{index . "component"}}</td>
        <td>{{index . "msg"}}</td>
        <td><a href="/admin/logs/trace/{{index . "trace_id"}}">{{index . "trace_id"}}</a></td>
        <td><a href="/admin/logs/activation/{{index . "activation_id"}}">{{index . "activation_id"}}</a></td>
      </tr>
      {{end}}
    </tbody>
  </table>
</main>
</body>
</html>`))

var traceTemplate = template.Must(template.New("trace").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{{.Title}}</title>
  <style>
    body { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; background: #0b1220; color: #e2e8f0; margin: 0; }
    main { padding: 16px; }
    a { color: #7dd3fc; }
    table { width: 100%; border-collapse: collapse; margin-top: 12px; }
    th, td { border-bottom: 1px solid #1e293b; text-align: left; padding: 6px; font-size: 13px; vertical-align: top; }
    th { color: #93c5fd; }
    .indent-1 { padding-left: 18px; }
    .indent-2 { padding-left: 36px; }
    .indent-3 { padding-left: 54px; }
  </style>
</head>
<body>
<main>
  <h1>{{.Title}}</h1>
  <p>{{.Kind}}: <strong>{{.ID}}</strong></p>
  <p><a href="/admin/logs">Back to logs</a></p>
  <table>
    <thead><tr><th>seq</th><th>ts</th><th>level</th><th>event</th><th>component</th><th>msg</th><th>span</th><th>parent</th></tr></thead>
    <tbody>
      {{range .Events}}
      <tr>
        <td>{{index . "seq"}}</td>
        <td>{{index . "ts"}}</td>
        <td>{{index . "level"}}</td>
        <td>{{index . "event"}}</td>
        <td>{{index . "component"}}</td>
        <td>{{index . "msg"}}</td>
        <td>{{index . "span_id"}}</td>
        <td>{{index . "parent_span_id"}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>
</main>
</body>
</html>`))
