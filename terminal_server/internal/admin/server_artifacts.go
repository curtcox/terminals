package admin

import (
	"net/http"
	"strings"
)

func (h *Handler) handleArtifacts(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"artifacts": h.capability.ListArtifacts()})
}

func (h *Handler) handleArtifactGet(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	artifactID := strings.TrimSpace(req.URL.Query().Get("artifact_id"))
	if artifactID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "artifact_id is required")
		return
	}
	artifact, ok := h.capability.GetArtifact(artifactID)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "artifact not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"artifact": artifact})
}

func (h *Handler) handleArtifactHistory(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	artifactID := strings.TrimSpace(req.URL.Query().Get("artifact_id"))
	if artifactID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "artifact_id is required")
		return
	}
	history, ok := h.capability.ArtifactHistory(artifactID)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "artifact not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"artifact_id": artifactID, "versions": history})
}

func (h *Handler) handleArtifactCreate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	artifact := h.capability.CreateArtifact(req.Form.Get("kind"), req.Form.Get("title"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "artifact": artifact})
}

func (h *Handler) handleArtifactPatch(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	artifact, ok := h.capability.PatchArtifact(req.Form.Get("artifact_id"), req.Form.Get("title"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "artifact not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "artifact": artifact})
}

func (h *Handler) handleArtifactReplace(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	artifact, ok := h.capability.ReplaceArtifact(req.Form.Get("artifact_id"), req.Form.Get("title"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "artifact not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "artifact": artifact})
}

func (h *Handler) handleArtifactTemplateSave(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	tmpl, ok := h.capability.SaveArtifactTemplate(req.Form.Get("name"), req.Form.Get("source_artifact_id"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "artifact template source not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "template": tmpl})
}

func (h *Handler) handleArtifactTemplateApply(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	artifact, ok := h.capability.ApplyArtifactTemplate(req.Form.Get("name"), req.Form.Get("target_artifact_id"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "artifact template or target not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "artifact": artifact})
}

func (h *Handler) handleCanvas(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"annotations": h.capability.ListCanvas(req.URL.Query().Get("canvas"))})
}

func (h *Handler) handleCanvasAnnotate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	annotation := h.capability.AnnotateCanvas(req.Form.Get("canvas"), req.Form.Get("text"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "annotation": annotation})
}

func (h *Handler) handleSearch(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"results": h.capability.Search(req.URL.Query().Get("q"))})
}

func (h *Handler) handleSearchTimeline(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": h.capability.SearchTimeline(req.URL.Query().Get("scope"))})
}

func (h *Handler) handleSearchRelated(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"results": h.capability.SearchRelated(req.URL.Query().Get("subject"))})
}

func (h *Handler) handleSearchRecent(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": h.capability.SearchRecent(req.URL.Query().Get("scope"), 20)})
}

func (h *Handler) handleMemoryRecall(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"memories": h.capability.Recall(req.URL.Query().Get("q"))})
}

func (h *Handler) handleMemoryStream(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"memories": h.capability.MemoryStream(req.URL.Query().Get("scope"))})
}

func (h *Handler) handleMemoryRemember(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	memory := h.capability.Remember(req.Form.Get("scope"), req.Form.Get("text"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "memory": memory})
}
