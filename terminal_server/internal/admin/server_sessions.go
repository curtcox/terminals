package admin

import (
	"net/http"
	"strings"
)

func (h *Handler) handleInteractiveSessions(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"sessions": h.capability.ListSessions()})
}

func (h *Handler) handleInteractiveSessionShow(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	sessionID := strings.TrimSpace(req.URL.Query().Get("session_id"))
	session, ok := h.capability.GetSession(sessionID)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"session": session})
}

func (h *Handler) handleInteractiveSessionMembers(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	sessionID := strings.TrimSpace(req.URL.Query().Get("session_id"))
	participants, ok := h.capability.ListSessionParticipants(sessionID)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"session_id":   sessionID,
		"participants": participants,
	})
}

func (h *Handler) handleInteractiveSessionCreate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	session := h.capability.CreateSession(req.Form.Get("kind"), req.Form.Get("target"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "session": session})
}

func (h *Handler) handleInteractiveSessionJoin(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	session, ok := h.capability.JoinSession(req.Form.Get("session_id"), req.Form.Get("participant"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "session": session})
}

func (h *Handler) handleInteractiveSessionLeave(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	session, ok := h.capability.LeaveSession(req.Form.Get("session_id"), req.Form.Get("participant"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "session": session})
}

func (h *Handler) handleInteractiveSessionAttachDevice(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	session, ok := h.capability.AttachDevice(req.Form.Get("session_id"), req.Form.Get("device_ref"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "session": session})
}

func (h *Handler) handleInteractiveSessionDetachDevice(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	session, ok := h.capability.DetachDevice(req.Form.Get("session_id"), req.Form.Get("device_ref"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "session": session})
}

func (h *Handler) handleInteractiveSessionRequestControl(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	session, ok := h.capability.RequestControl(req.Form.Get("session_id"), req.Form.Get("participant"), req.Form.Get("control_type"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "session": session})
}

func (h *Handler) handleInteractiveSessionGrantControl(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	session, ok := h.capability.GrantControl(req.Form.Get("session_id"), req.Form.Get("participant"), req.Form.Get("granted_by"), req.Form.Get("control_type"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "session": session})
}

func (h *Handler) handleInteractiveSessionRevokeControl(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	session, ok := h.capability.RevokeControl(req.Form.Get("session_id"), req.Form.Get("participant"), req.Form.Get("revoked_by"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "session": session})
}
