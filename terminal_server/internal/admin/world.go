package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/curtcox/terminals/terminal_server/internal/world"
)

type worldCalibrationDevice struct {
	Geometry world.DeviceGeometry     `json:"geometry"`
	History  []world.CalibrationEvent `json:"history"`
}

func (h *Handler) handleWorldCalibration(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.world == nil {
		h.writeJSONError(w, http.StatusServiceUnavailable, "world model unavailable")
		return
	}

	limit := parseHistoryLimit(req.URL.Query().Get("history_limit"))
	deviceID := strings.TrimSpace(req.URL.Query().Get("device_id"))
	geometries := h.world.ListGeometries(req.Context())
	out := make([]worldCalibrationDevice, 0, len(geometries))
	for _, geometry := range geometries {
		if deviceID != "" && geometry.DeviceID != deviceID {
			continue
		}
		history, err := h.world.CalibrationHistory(req.Context(), geometry.DeviceID, limit)
		if err != nil && !errors.Is(err, world.ErrNotFound) {
			h.writeJSONError(w, http.StatusInternalServerError, "read calibration history")
			return
		}
		if errors.Is(err, world.ErrNotFound) {
			history = nil
		}
		out = append(out, worldCalibrationDevice{Geometry: geometry, History: history})
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"devices":       out,
		"history_limit": limit,
	})
}

func (h *Handler) handleWorldVerify(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.world == nil {
		h.writeJSONError(w, http.StatusServiceUnavailable, "world model unavailable")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "parse form")
		return
	}

	deviceID := strings.TrimSpace(req.FormValue("device_id"))
	method := strings.TrimSpace(req.FormValue("method"))
	if deviceID == "" || method == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id and method are required")
		return
	}
	if err := h.world.VerifyDevice(req.Context(), deviceID, method); err != nil {
		if errors.Is(err, world.ErrNotFound) {
			h.writeJSONError(w, http.StatusNotFound, "device not found")
			return
		}
		h.writeJSONError(w, http.StatusInternalServerError, "verify device")
		return
	}

	limit := parseHistoryLimit(req.FormValue("history_limit"))
	geometries := h.world.ListGeometries(req.Context())
	for _, geometry := range geometries {
		if geometry.DeviceID != deviceID {
			continue
		}
		history, err := h.world.CalibrationHistory(req.Context(), deviceID, limit)
		if err != nil {
			h.writeJSONError(w, http.StatusInternalServerError, "read calibration history")
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{
			"device": worldCalibrationDevice{
				Geometry: geometry,
				History:  history,
			},
		})
		return
	}

	h.writeJSONError(w, http.StatusNotFound, "device not found")
}

func parseHistoryLimit(raw string) int {
	limit, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || limit <= 0 {
		return 20
	}
	if limit > 200 {
		return 200
	}
	return limit
}
