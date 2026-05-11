package admin

import (
	"net/http"
	"strings"
)

func (h *Handler) handleIdentity(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"identities": h.capability.ListIdentities()})
}

func (h *Handler) handleIdentityShow(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	identityRef := strings.TrimSpace(req.URL.Query().Get("identity"))
	if identityRef == "" {
		h.writeJSONError(w, http.StatusBadRequest, "identity is required")
		return
	}
	identity, ok := h.capability.GetIdentity(identityRef)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "identity not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"identity": identity})
}

func (h *Handler) handleIdentityGroups(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"groups": h.capability.ListGroups()})
}

func (h *Handler) handleIdentityPreferences(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	identityRef := strings.TrimSpace(req.URL.Query().Get("identity"))
	if identityRef == "" {
		h.writeJSONError(w, http.StatusBadRequest, "identity is required")
		return
	}
	prefs, ok := h.capability.GetPreferences(identityRef)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "identity not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"identity": identityRef, "preferences": prefs})
}

func (h *Handler) handleIdentityResolve(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	audience := strings.TrimSpace(req.URL.Query().Get("audience"))
	h.writeJSON(w, http.StatusOK, map[string]any{
		"audience":   audience,
		"identities": h.capability.ResolveAudience(audience),
	})
}

func (h *Handler) handleIdentityAcknowledgements(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		subjectRef := strings.TrimSpace(req.URL.Query().Get("subject_ref"))
		h.writeJSON(w, http.StatusOK, map[string]any{
			"subject_ref":      subjectRef,
			"acknowledgements": h.capability.GetAcknowledgements(subjectRef),
		})
	case http.MethodPost:
		if err := req.ParseForm(); err != nil {
			h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
			return
		}
		ack, ok := h.capability.RecordAcknowledgement(req.Form.Get("subject_ref"), req.Form.Get("actor"), req.Form.Get("mode"))
		if !ok {
			h.writeJSONError(w, http.StatusBadRequest, "invalid acknowledgement")
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "ack": ack})
	default:
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleCohorts(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	name := strings.TrimSpace(req.URL.Query().Get("name"))
	if name == "" {
		h.writeJSON(w, http.StatusOK, map[string]any{"cohorts": h.capability.CohortList()})
		return
	}
	cohort, ok := h.capability.CohortGet(name)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "cohort not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"cohort":  cohort,
		"members": h.resolveCohortMembers(cohort.Selectors),
	})
}

func (h *Handler) handleCohortUpsert(w http.ResponseWriter, req *http.Request) {
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
	selectors := parseSelectors(req.Form.Get("selectors"))
	cohort := h.capability.CohortUpsert(name, selectors)
	h.writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"cohort":  cohort,
		"members": h.resolveCohortMembers(cohort.Selectors),
	})
}

func (h *Handler) handleCohortDelete(w http.ResponseWriter, req *http.Request) {
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
	deleted := h.capability.CohortDelete(name)
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted": deleted})
}
