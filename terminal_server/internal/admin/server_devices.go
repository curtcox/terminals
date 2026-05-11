package admin

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

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
		Zone           string            `json:"zone,omitempty"`
		Roles          []string          `json:"roles,omitempty"`
		Mobility       string            `json:"mobility,omitempty"`
		Affinity       string            `json:"affinity,omitempty"`
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
			Zone:           d.Placement.Zone,
			Roles:          d.Placement.Roles,
			Mobility:       d.Placement.Mobility,
			Affinity:       d.Placement.Affinity,
			State:          string(d.State),
			LastHeartbeat:  d.LastHeartbeat.UTC().UnixMilli(),
			RegisteredAt:   d.RegisteredAt.UTC().UnixMilli(),
			ActiveScenario: activeByDevice[d.DeviceID],
			Capabilities:   d.Capabilities,
		})
	}

	h.writeJSON(w, http.StatusOK, map[string]any{"devices": views})
}

func (h *Handler) handleDevicePlacementUpdate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form")
		return
	}
	deviceID := strings.TrimSpace(req.FormValue("device_id"))
	if deviceID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	placement := device.PlacementMetadata{
		Zone:     strings.TrimSpace(req.FormValue("zone")),
		Roles:    parseDeviceIDs(req.FormValue("roles")),
		Mobility: strings.TrimSpace(req.FormValue("mobility")),
		Affinity: strings.TrimSpace(req.FormValue("affinity")),
	}
	if err := h.devices.UpdatePlacement(deviceID, placement); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	eventlog.Emit(req.Context(), "admin.action.applied", slog.LevelInfo, "admin placement update applied",
		slog.String("component", "admin.http"),
		slog.String("action", "device_placement.update"),
		slog.String("device_id", deviceID),
	)
	h.writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"device_id": deviceID,
		"placement": placement,
	})
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
	eventlog.Emit(req.Context(), "admin.action.applied", slog.LevelInfo, "admin scenario command applied",
		slog.String("component", "admin.http"),
		slog.String("action", "scenario."+action),
		slog.String("scenario", matched),
		slog.Int("target_device_count", len(deviceIDs)),
	)

	h.writeJSON(w, http.StatusOK, map[string]any{
		"status":            "ok",
		"action":            action,
		"scenario":          matched,
		"requested":         scenarioName,
		"target_device_ids": deviceIDs,
	})
}

func (h *Handler) handleActivations(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	activeByDevice := h.runtime.Engine.ActiveSnapshot()
	suspendedByDevice := h.runtime.Engine.SuspendedSnapshot()

	claimsByDevice := map[string][]iorouter.Claim{}
	suspendedClaimsByDevice := map[string][]iorouter.Claim{}
	if routeIO, ok := h.runtime.Env.IO.(interface{ Claims() *iorouter.ClaimManager }); ok {
		claims := routeIO.Claims()
		if claims != nil {
			for _, d := range h.devices.List() {
				deviceID := d.DeviceID
				claimsByDevice[deviceID] = claims.Snapshot(deviceID)
				suspendedClaimsByDevice[deviceID] = claims.SuspendedSnapshot(deviceID)
			}
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"active_by_device":           activeByDevice,
		"suspended_by_device":        suspendedByDevice,
		"claims_by_device":           claimsByDevice,
		"suspended_claims_by_device": suspendedClaimsByDevice,
		"event_tail":                 h.runtime.EventTail(50),
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

func parseSelectors(raw string) []string {
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

func parseCSVValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			out = append(out, trimmed)
		}
	}
	return out
}

func (h *Handler) resolveCohortMembers(selectors []string) []string {
	devices := h.devices.List()
	members := make([]string, 0, len(devices))
	for _, d := range devices {
		if deviceMatchesSelectors(d, selectors) {
			members = append(members, d.DeviceID)
		}
	}
	sort.Strings(members)
	return members
}

func deviceMatchesSelectors(d device.Device, selectors []string) bool {
	for _, selector := range selectors {
		selector = strings.ToLower(strings.TrimSpace(selector))
		if selector == "" {
			continue
		}
		key, value, ok := strings.Cut(selector, ":")
		if !ok || strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return false
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "id", "device":
			if !strings.EqualFold(d.DeviceID, value) {
				return false
			}
		case "zone":
			if !strings.EqualFold(d.Placement.Zone, value) {
				return false
			}
		case "role":
			matched := false
			for _, role := range d.Placement.Roles {
				if strings.EqualFold(role, value) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		case "platform":
			if !strings.EqualFold(d.Platform, value) {
				return false
			}
		case "type":
			if !strings.EqualFold(d.DeviceType, value) {
				return false
			}
		case "state":
			if !strings.EqualFold(string(d.State), value) {
				return false
			}
		case "mobility":
			if !strings.EqualFold(d.Placement.Mobility, value) {
				return false
			}
		case "affinity":
			if !strings.EqualFold(d.Placement.Affinity, value) {
				return false
			}
		default:
			return false
		}
	}
	return true
}
