package admin

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/capability"
)

func (h *Handler) handleRecent(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": h.capability.ListRecent()})
}

func (h *Handler) handleSimDevices(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"devices": h.capability.SimDeviceList()})
}

func (h *Handler) handleSimDeviceNew(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deviceID := strings.TrimSpace(req.Form.Get("device_id"))
	if deviceID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	device := h.capability.SimDeviceUpsert(deviceID, parseCSVValues(req.Form["caps"]))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "device": device})
}

func (h *Handler) handleSimDeviceRemove(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deviceID := strings.TrimSpace(req.Form.Get("device_id"))
	if deviceID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	deleted := h.capability.SimDeviceDelete(deviceID)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted": deleted})
}

func (h *Handler) handleSimInput(w http.ResponseWriter, req *http.Request) {
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
	action := strings.TrimSpace(req.Form.Get("action"))
	if deviceID == "" || componentID == "" || action == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id, component_id, and action are required")
		return
	}
	input, ok := h.capability.SimRecordInput(deviceID, componentID, action, req.Form.Get("value"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "sim device not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "input": input})
}

func (h *Handler) handleSimUI(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	deviceID := strings.TrimSpace(req.URL.Query().Get("device_id"))
	if deviceID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	device, ok := h.capability.SimDeviceGet(deviceID)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "sim device not found")
		return
	}
	snapshot, hasSnapshot := h.capability.UISnapshot(device.DeviceID)
	if !hasSnapshot {
		snapshot = capability.UISnapshot{DeviceID: device.DeviceID}
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"device":   device,
		"snapshot": snapshot,
		"inputs":   h.capability.SimInputs(device.DeviceID),
	})
}

func (h *Handler) handleSimExpect(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deviceID := strings.TrimSpace(req.Form.Get("device_id"))
	kind := strings.TrimSpace(req.Form.Get("kind"))
	selector := strings.TrimSpace(req.Form.Get("selector"))
	if deviceID == "" || kind == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id and kind are required")
		return
	}
	within := time.Duration(0)
	if rawWithin := strings.TrimSpace(req.Form.Get("within")); rawWithin != "" {
		parsedWithin, err := time.ParseDuration(rawWithin)
		if err != nil || parsedWithin <= 0 {
			h.writeJSONError(w, http.StatusBadRequest, "within must be a positive duration")
			return
		}
		within = parsedWithin
	}
	result, ok := h.capability.SimExpect(deviceID, kind, selector, within)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "sim device not found")
		return
	}
	if !result.Matched {
		h.writeJSON(w, http.StatusConflict, map[string]any{"status": "failed", "result": result})
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "result": result})
}

func (h *Handler) handleSimRecord(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deviceID := strings.TrimSpace(req.Form.Get("device_id"))
	if deviceID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	duration := time.Duration(0)
	if rawDuration := strings.TrimSpace(req.Form.Get("duration")); rawDuration != "" {
		parsedDuration, err := time.ParseDuration(rawDuration)
		if err != nil || parsedDuration <= 0 {
			h.writeJSONError(w, http.StatusBadRequest, "duration must be a positive duration")
			return
		}
		duration = parsedDuration
	}
	record, ok := h.capability.SimRecord(deviceID, duration)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "sim device not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "result": record})
}

func (h *Handler) handleScriptsDryRun(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	path := strings.TrimSpace(req.Form.Get("path"))
	if path == "" {
		h.writeJSONError(w, http.StatusBadRequest, "path is required")
		return
	}
	content, err := os.ReadFile(path)
	if err != nil {
		h.writeJSONError(w, http.StatusNotFound, "script not found")
		return
	}
	result := h.capability.ScriptDryRun(path, string(content))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "result": result})
}

func (h *Handler) handleScriptsRun(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	path := strings.TrimSpace(req.Form.Get("path"))
	if path == "" {
		h.writeJSONError(w, http.StatusBadRequest, "path is required")
		return
	}
	content, err := os.ReadFile(path)
	if err != nil {
		h.writeJSONError(w, http.StatusNotFound, "script not found")
		return
	}
	result := h.capability.ScriptRun(path, string(content))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "result": result})
}

func (h *Handler) handleStoreGet(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	record, ok := h.capability.StoreGet(req.URL.Query().Get("namespace"), req.URL.Query().Get("key"))
	if !ok {
		h.writeJSON(w, http.StatusOK, map[string]any{"record": nil})
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"record": record})
}

func (h *Handler) handleStoreNamespaces(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"namespaces": h.capability.StoreNamespaces()})
}

func (h *Handler) handleStoreList(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"records": h.capability.StoreList(req.URL.Query().Get("namespace"))})
}

func (h *Handler) handleStorePut(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	ttl := time.Duration(0)
	ttlRaw := strings.TrimSpace(req.Form.Get("ttl"))
	if ttlRaw != "" {
		parsedTTL, err := time.ParseDuration(ttlRaw)
		if err != nil || parsedTTL <= 0 {
			h.writeJSONError(w, http.StatusBadRequest, "ttl must be a positive duration")
			return
		}
		ttl = parsedTTL
	}
	record := h.capability.StorePut(req.Form.Get("namespace"), req.Form.Get("key"), req.Form.Get("value"), ttl)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "record": record})
}

func (h *Handler) handleStoreDelete(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	deleted := h.capability.StoreDelete(req.Form.Get("namespace"), req.Form.Get("key"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted": deleted})
}

func (h *Handler) handleStoreWatch(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	namespace := req.URL.Query().Get("namespace")
	prefix := req.URL.Query().Get("prefix")
	h.writeJSON(w, http.StatusOK, map[string]any{
		"namespace": namespace,
		"prefix":    prefix,
		"records":   h.capability.StoreWatch(namespace, prefix),
	})
}

func (h *Handler) handleStoreBind(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	binding := strings.TrimSpace(req.Form.Get("to"))
	parts := strings.SplitN(binding, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		h.writeJSONError(w, http.StatusBadRequest, "to must be formatted as <device>:<scenario>")
		return
	}
	record, ok := h.capability.StoreBind(req.Form.Get("namespace"), req.Form.Get("key"), binding)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "store record not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "record": record})
}

func (h *Handler) handleBusTail(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit := 0
	if rawLimit := strings.TrimSpace(req.URL.Query().Get("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			h.writeJSONError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = parsedLimit
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"events": h.capability.BusTail(req.URL.Query().Get("kind"), req.URL.Query().Get("name"), limit)})
}

func (h *Handler) handleBusEmit(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	event := h.capability.BusEmit(req.Form.Get("kind"), req.Form.Get("name"), req.Form.Get("payload"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "event": event})
}

func (h *Handler) handleBusReplay(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit := 0
	if rawLimit := strings.TrimSpace(req.URL.Query().Get("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			h.writeJSONError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = parsedLimit
	}
	events := h.capability.BusReplay(
		req.URL.Query().Get("from"),
		req.URL.Query().Get("to"),
		req.URL.Query().Get("kind"),
		req.URL.Query().Get("name"),
		limit,
	)
	h.writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (h *Handler) handleHandlers(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"handlers": h.capability.HandlerList()})
}

func (h *Handler) handleHandlersOn(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	selector := strings.TrimSpace(req.Form.Get("selector"))
	action := strings.TrimSpace(req.Form.Get("action"))
	if selector == "" || action == "" {
		h.writeJSONError(w, http.StatusBadRequest, "selector and action are required")
		return
	}
	runCommand := strings.TrimSpace(req.Form.Get("run"))
	emitKind := strings.TrimSpace(req.Form.Get("emit_kind"))
	emitName := strings.TrimSpace(req.Form.Get("emit_name"))
	emitPayload := strings.TrimSpace(req.Form.Get("emit_payload"))

	hasRun := runCommand != ""
	hasEmit := emitKind != "" || emitName != "" || emitPayload != ""
	if hasRun == hasEmit {
		h.writeJSONError(w, http.StatusBadRequest, "provide exactly one target: run or emit_kind/emit_name")
		return
	}

	var handler capability.HandlerRegistration
	if hasRun {
		handler = h.capability.HandlerOnRun(selector, action, runCommand)
	} else {
		if emitName == "" {
			h.writeJSONError(w, http.StatusBadRequest, "emit_name is required when using emit")
			return
		}
		handler = h.capability.HandlerOnEmit(selector, action, emitKind, emitName, emitPayload)
	}

	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "handler": handler})
}

func (h *Handler) handleHandlersOff(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	handlerID := strings.TrimSpace(req.Form.Get("handler_id"))
	if handlerID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "handler_id is required")
		return
	}
	deleted := h.capability.HandlerOff(handlerID)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted": deleted})
}
