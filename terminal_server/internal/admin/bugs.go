package admin

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
	"github.com/curtcox/terminals/terminal_server/internal/diagnostics/bugreport"
	"google.golang.org/protobuf/encoding/protojson"
)

func (h *Handler) handleBugsListAPI(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.bugReports == nil {
		h.writeJSON(w, http.StatusOK, map[string]any{"bugs": []any{}})
		return
	}
	items, err := h.bugReports.ListFiltered(parseBugListFilter(req))
	if err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("list bug reports: %v", err))
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"bugs": items})
}

func (h *Handler) handleBugsListPage(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	items := []any{}
	filter := parseBugListFilter(req)
	if h.bugReports != nil {
		list, err := h.bugReports.ListFiltered(filter)
		if err != nil {
			h.writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("list bug reports: %v", err))
			return
		}
		for _, item := range list {
			items = append(items, item)
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := bugsListTemplate.Execute(w, map[string]any{"Reports": items, "Filter": filter}); err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("render bugs list: %v", err))
	}
}

func (h *Handler) handleBugDetailPage(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	reportID := strings.TrimSpace(strings.TrimPrefix(req.URL.Path, "/admin/bugs/"))
	if reportID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "report id is required")
		return
	}
	if h.bugReports == nil {
		h.writeJSONError(w, http.StatusNotFound, "bug reports unavailable")
		return
	}
	rec, ok, err := h.bugReports.Get(reportID)
	if err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("read bug report: %v", err))
		return
	}
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "bug report not found")
		return
	}
	pretty, _ := json.MarshalIndent(rec, "", "  ")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := bugDetailTemplate.Execute(w, map[string]any{
		"ReportID":  reportID,
		"JSON":      string(pretty),
		"Confirmed": rec.Confirmed,
	}); err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("render bug detail: %v", err))
	}
}

func (h *Handler) handleBugReportAPI(w http.ResponseWriter, req *http.Request) {
	if h.bugReports == nil {
		h.writeJSONError(w, http.StatusNotFound, "bug reports unavailable")
		return
	}
	pathValue := strings.TrimSpace(strings.TrimPrefix(req.URL.Path, "/admin/api/bugs/"))
	if pathValue == "" {
		h.writeJSONError(w, http.StatusBadRequest, "report id is required")
		return
	}
	if strings.HasSuffix(pathValue, "/confirm") {
		if req.Method != http.MethodPost {
			h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		reportID := strings.TrimSpace(strings.TrimSuffix(pathValue, "/confirm"))
		reportID = strings.TrimSuffix(reportID, "/")
		if reportID == "" {
			h.writeJSONError(w, http.StatusBadRequest, "report id is required")
			return
		}
		rec, ok, err := h.bugReports.Confirm(req.Context(), reportID, "admin")
		if err != nil {
			h.writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("confirm bug report: %v", err))
			return
		}
		if !ok {
			h.writeJSONError(w, http.StatusNotFound, "bug report not found")
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{"report": rec})
		return
	}

	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	reportID := strings.TrimSuffix(pathValue, "/")
	reportID = strings.TrimSpace(reportID)
	if reportID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "report id is required")
		return
	}
	rec, ok, err := h.bugReports.Get(reportID)
	if err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("read bug report: %v", err))
		return
	}
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "bug report not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"report": rec})
}

func (h *Handler) handleBugNewPage(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	subjectID := strings.TrimSpace(req.URL.Query().Get("device"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := bugNewTemplate.Execute(w, map[string]any{"SubjectDeviceID": subjectID}); err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("render bug intake page: %v", err))
	}
}

func (h *Handler) handleBugIntake(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.bugReports == nil {
		h.writeJSONError(w, http.StatusNotFound, "bug reports unavailable")
		return
	}

	report := &diagnosticsv1.BugReport{}
	contentType := strings.ToLower(strings.TrimSpace(req.Header.Get("Content-Type")))
	isJSON := strings.Contains(contentType, "application/json")
	if isJSON {
		payload, err := io.ReadAll(req.Body)
		if err != nil {
			h.writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("read json payload: %v", err))
			return
		}
		if err := protojson.Unmarshal(payload, report); err != nil {
			h.writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("decode bug report json: %v", err))
			return
		}
	} else {
		if err := req.ParseForm(); err != nil {
			h.writeJSONError(w, http.StatusBadRequest, "invalid form")
			return
		}
		report.ReporterDeviceId = strings.TrimSpace(req.FormValue("reporter_device_id"))
		report.SubjectDeviceId = strings.TrimSpace(req.FormValue("subject_device_id"))
		report.Description = strings.TrimSpace(req.FormValue("description"))
		report.Tags = splitCSV(req.FormValue("tags"))
		report.Source = parseBugSource(req.FormValue("source"))
		report.TimestampUnixMs = time.Now().UTC().UnixMilli()
	}

	ack, err := h.bugReports.File(req.Context(), report)
	if err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("file bug report: %v", err))
		return
	}

	if isJSON {
		h.writeJSON(w, http.StatusOK, map[string]any{"ack": ack})
		return
	}
	http.Redirect(w, req, path.Join("/admin/bugs", ack.GetReportId()), http.StatusSeeOther)
}

func splitCSV(raw string) []string {
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

func parseBugSource(raw string) diagnosticsv1.BugReportSource {
	key := strings.TrimSpace(strings.ToUpper(raw))
	if key == "" {
		return diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_ADMIN
	}
	if !strings.HasPrefix(key, "BUG_REPORT_SOURCE_") {
		key = "BUG_REPORT_SOURCE_" + key
	}
	if value, ok := diagnosticsv1.BugReportSource_value[key]; ok {
		return diagnosticsv1.BugReportSource(value)
	}
	return diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_OTHER
}

func parseBugListFilter(req *http.Request) bugreport.ListFilter {
	query := req.URL.Query()
	return bugreport.ListFilter{
		SubjectDeviceID:  strings.TrimSpace(query.Get("subject_device_id")),
		ReporterDeviceID: strings.TrimSpace(query.Get("reporter_device_id")),
		Source:           strings.TrimSpace(query.Get("source")),
		Tag:              strings.TrimSpace(query.Get("tag")),
		FromUnixMS:       parseInt64Query(query.Get("from_unix_ms")),
		ToUnixMS:         parseInt64Query(query.Get("to_unix_ms")),
		ConfirmedOnly:    parseBoolQuery(query.Get("confirmed")),
		PendingOnly:      parseBoolQuery(query.Get("pending")),
	}
}

func parseInt64Query(raw string) int64 {
	text := strings.TrimSpace(raw)
	if text == "" {
		return 0
	}
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func parseBoolQuery(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

var bugsListTemplate = template.Must(template.New("bugs_list").Parse(`<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Bug Reports</title>
</head>
<body>
  <h1>Bug Reports</h1>
  <p><a href="/admin">Back to dashboard</a> | <a href="/admin/bugs/new">File a report</a></p>
  <form method="get" action="/admin/bugs">
    <label>Subject <input name="subject_device_id" value="{{.Filter.SubjectDeviceID}}" /></label>
    <label>Reporter <input name="reporter_device_id" value="{{.Filter.ReporterDeviceID}}" /></label>
    <label>Source <input name="source" value="{{.Filter.Source}}" /></label>
    <label>Tag <input name="tag" value="{{.Filter.Tag}}" /></label>
    <label>Confirmed <input type="checkbox" name="confirmed" value="1" {{if .Filter.ConfirmedOnly}}checked{{end}} /></label>
    <label>Pending <input type="checkbox" name="pending" value="1" {{if .Filter.PendingOnly}}checked{{end}} /></label>
    <button type="submit">Apply</button>
  </form>
  <table border="1" cellspacing="0" cellpadding="6">
    <thead>
      <tr>
        <th>Report</th>
        <th>When</th>
        <th>Source</th>
        <th>Subject</th>
        <th>Reporter</th>
        <th>Offline</th>
        <th>Confirmed</th>
      </tr>
    </thead>
    <tbody>
      {{range .Reports}}
      <tr>
        <td><a href="/admin/bugs/{{.ReportID}}">{{.ReportID}}</a></td>
        <td>{{.TimestampUnixMS}}</td>
        <td>{{.Source}}</td>
        <td>{{.SubjectDeviceID}}</td>
        <td>{{.ReporterDeviceID}}</td>
        <td>{{.SubjectOffline}}</td>
        <td>{{.Confirmed}}</td>
      </tr>
      {{else}}
      <tr><td colspan="7">No reports yet.</td></tr>
      {{end}}
    </tbody>
  </table>
</body>
</html>`))

var bugDetailTemplate = template.Must(template.New("bug_detail").Parse(`<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Bug Report</title>
</head>
<body>
  <h1>Bug Report {{.ReportID}}</h1>
  <p><a href="/admin/bugs">Back to report list</a></p>
  {{if not .Confirmed}}
  <form method="post" action="/admin/api/bugs/{{.ReportID}}/confirm">
    <button type="submit">Confirm</button>
  </form>
  {{end}}
  <pre>{{.JSON}}</pre>
</body>
</html>`))

var bugNewTemplate = template.Must(template.New("bug_new").Parse(`<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>File Bug Report</title>
</head>
<body>
  <h1>File a Bug Report</h1>
  <p><a href="/admin/bugs">Back to report list</a></p>
  <form method="post" action="/bug/intake">
    <p><label>Reporter Device ID <input name="reporter_device_id" /></label></p>
    <p><label>Subject Device ID <input name="subject_device_id" value="{{.SubjectDeviceID}}" /></label></p>
    <p><label>Source <input name="source" value="admin" /></label></p>
    <p><label>Tags (comma-separated) <input name="tags" /></label></p>
    <p><label>Description<br><textarea name="description" rows="6" cols="70"></textarea></label></p>
    <p><button type="submit">Submit</button></p>
  </form>
</body>
</html>`))
