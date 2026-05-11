package admin

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/curtcox/terminals/terminal_server/internal/capability"
)

func sortPlacements(placements []map[string]any) {
	sort.Slice(placements, func(i, j int) bool {
		return fmt.Sprintf("%v", placements[i]["device_id"]) < fmt.Sprintf("%v", placements[j]["device_id"])
	})
}

func (h *Handler) handleUIViews(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	viewID := strings.TrimSpace(req.URL.Query().Get("view_id"))
	if viewID == "" {
		h.writeJSON(w, http.StatusOK, map[string]any{"views": h.capability.UIViewList()})
		return
	}
	view, ok := h.capability.UIViewGet(viewID)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "ui view not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"view": view})
}

func (h *Handler) handleUIViewUpsert(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	viewID := strings.TrimSpace(req.Form.Get("view_id"))
	if viewID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "view_id is required")
		return
	}
	view := h.capability.UIViewUpsert(
		viewID,
		strings.TrimSpace(req.Form.Get("root_id")),
		strings.TrimSpace(req.Form.Get("descriptor")),
	)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "view": view})
}

func (h *Handler) handleUIViewDelete(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	viewID := strings.TrimSpace(req.Form.Get("view_id"))
	if viewID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "view_id is required")
		return
	}
	deleted := h.capability.UIViewDelete(viewID)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted": deleted})
}

func (h *Handler) handleUIPush(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deviceID := strings.TrimSpace(req.Form.Get("device_id"))
	descriptor := strings.TrimSpace(req.Form.Get("descriptor"))
	if deviceID == "" || descriptor == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id and descriptor are required")
		return
	}
	snapshot := h.capability.UIPush(deviceID, descriptor, strings.TrimSpace(req.Form.Get("root_id")))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "snapshot": snapshot})
}

func (h *Handler) handleUIPatch(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deviceID := strings.TrimSpace(req.Form.Get("device_id"))
	componentID := strings.TrimSpace(req.Form.Get("component_id"))
	descriptor := strings.TrimSpace(req.Form.Get("descriptor"))
	if deviceID == "" || componentID == "" || descriptor == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id, component_id, and descriptor are required")
		return
	}
	snapshot := h.capability.UIPatch(deviceID, componentID, descriptor)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "snapshot": snapshot})
}

func (h *Handler) handleUITransition(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deviceID := strings.TrimSpace(req.Form.Get("device_id"))
	componentID := strings.TrimSpace(req.Form.Get("component_id"))
	transition := strings.TrimSpace(req.Form.Get("transition"))
	if deviceID == "" || componentID == "" || transition == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id, component_id, and transition are required")
		return
	}
	durationMS := 0
	if raw := strings.TrimSpace(req.Form.Get("duration_ms")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			h.writeJSONError(w, http.StatusBadRequest, "duration_ms must be an integer")
			return
		}
		durationMS = parsed
	}
	snapshot := h.capability.UITransition(deviceID, componentID, transition, durationMS)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "snapshot": snapshot})
}

func (h *Handler) handleUIBroadcast(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	cohortName := strings.TrimSpace(req.Form.Get("cohort"))
	descriptor := strings.TrimSpace(req.Form.Get("descriptor"))
	if cohortName == "" || descriptor == "" {
		h.writeJSONError(w, http.StatusBadRequest, "cohort and descriptor are required")
		return
	}
	cohort, ok := h.capability.CohortGet(cohortName)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "cohort not found")
		return
	}
	members := h.resolveCohortMembers(cohort.Selectors)
	broadcast := h.capability.UIBroadcast(cohort.Name, descriptor, strings.TrimSpace(req.Form.Get("patch_id")), members)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "broadcast": broadcast, "members": members})
}

func (h *Handler) handleUISubscribe(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deviceID := strings.TrimSpace(req.Form.Get("device_id"))
	target := strings.TrimSpace(req.Form.Get("to"))
	if deviceID == "" || target == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id and to are required")
		return
	}
	snapshot := h.capability.UISubscribe(deviceID, target)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "snapshot": snapshot})
}

func (h *Handler) handleUISnapshot(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	deviceID := strings.TrimSpace(req.URL.Query().Get("device_id"))
	if deviceID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	snapshot, ok := h.capability.UISnapshot(deviceID)
	if !ok {
		h.writeJSON(w, http.StatusOK, map[string]any{"snapshot": nil})
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"snapshot": snapshot})
}

func (h *Handler) handlePlacement(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	placements := make([]map[string]any, 0)
	for _, d := range h.devices.List() {
		placements = append(placements, map[string]any{
			"device_id": d.DeviceID,
			"zone":      d.Placement.Zone,
			"roles":     append([]string(nil), d.Placement.Roles...),
			"mobility":  d.Placement.Mobility,
			"affinity":  d.Placement.Affinity,
		})
	}
	sortPlacements(placements)
	h.writeJSON(w, http.StatusOK, map[string]any{"placements": placements})
}

func (h *Handler) handleInlineScenarios(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	name := strings.TrimSpace(req.URL.Query().Get("name"))
	if name == "" {
		h.writeJSON(w, http.StatusOK, map[string]any{"scenarios": h.capability.ScenarioList()})
		return
	}
	def, ok := h.capability.ScenarioGet(name)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "inline scenario not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"scenario": def})
}

func (h *Handler) handleInlineScenarioDefine(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	name := strings.TrimSpace(req.Form.Get("name"))
	if name == "" {
		h.writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}
	onEventKinds := req.Form["on_event_kind"]
	onEventCommands := req.Form["on_event_command"]
	if len(onEventKinds) != len(onEventCommands) {
		h.writeJSONError(w, http.StatusBadRequest, "on_event_kind and on_event_command counts must match")
		return
	}
	onEvents := make([]capability.InlineScenarioEventHook, 0, len(onEventKinds))
	for i := range onEventKinds {
		kind := strings.TrimSpace(onEventKinds[i])
		command := strings.TrimSpace(onEventCommands[i])
		if kind == "" || command == "" {
			h.writeJSONError(w, http.StatusBadRequest, "on_event_kind and on_event_command values must be non-empty")
			return
		}
		onEvents = append(onEvents, capability.InlineScenarioEventHook{Kind: kind, Command: command})
	}
	def := h.capability.ScenarioDefine(capability.InlineScenarioDefinition{
		Name:         name,
		MatchIntents: parseCSVValues(req.Form["match_intent"]),
		MatchEvents:  parseCSVValues(req.Form["match_event"]),
		Priority:     strings.TrimSpace(req.Form.Get("priority")),
		OnStart:      strings.TrimSpace(req.Form.Get("on_start")),
		OnInput:      strings.TrimSpace(req.Form.Get("on_input")),
		OnEvents:     onEvents,
		OnSuspend:    strings.TrimSpace(req.Form.Get("on_suspend")),
		OnResume:     strings.TrimSpace(req.Form.Get("on_resume")),
		OnStop:       strings.TrimSpace(req.Form.Get("on_stop")),
	})
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "scenario": def})
}

func (h *Handler) handleInlineScenarioUndefine(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	name := strings.TrimSpace(req.Form.Get("name"))
	if name == "" {
		h.writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}
	deleted := h.capability.ScenarioUndefine(name)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted": deleted})
}
