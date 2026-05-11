package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/curtcox/terminals/terminal_server/internal/replai"
	"github.com/curtcox/terminals/terminal_server/internal/replsession"
)

func (h *Handler) handleReplAIProviders(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	resp, err := h.ai.ListProviders(req.Context(), replai.ListProvidersRequest{})
	if err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"providers": resp.Providers})
}

func (h *Handler) handleReplAIModels(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	provider := strings.TrimSpace(req.URL.Query().Get("provider"))
	resp, err := h.ai.ListModels(req.Context(), replai.ListModelsRequest{Provider: provider})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, replai.ErrMissingProvider) || errors.Is(err, replai.ErrProviderNotFound) {
			status = http.StatusBadRequest
		}
		h.writeJSONError(w, status, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"provider": resp.Provider,
		"models":   resp.Models,
	})
}

func (h *Handler) handleReplAISelection(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	switch req.Method {
	case http.MethodGet:
		sessionID := strings.TrimSpace(req.URL.Query().Get("session_id"))
		resp, err := h.ai.GetSelection(req.Context(), replai.GetSelectionRequest{SessionID: sessionID})
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, replai.ErrMissingSessionID) ||
				errors.Is(err, replai.ErrMissingProvider) ||
				errors.Is(err, replai.ErrMissingModel) ||
				errors.Is(err, replai.ErrProviderNotFound) {
				status = http.StatusBadRequest
			}
			if errors.Is(err, replsession.ErrSessionNotFound) {
				status = http.StatusNotFound
			}
			h.writeJSONError(w, status, err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, resp)
	case http.MethodPost:
		if err := req.ParseForm(); err != nil {
			h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
			return
		}
		resp, err := h.ai.SetSelection(req.Context(), replai.SetSelectionRequest{
			SessionID: strings.TrimSpace(req.Form.Get("session_id")),
			Provider:  strings.TrimSpace(req.Form.Get("provider")),
			Model:     strings.TrimSpace(req.Form.Get("model")),
		})
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, replai.ErrMissingSessionID) ||
				errors.Is(err, replai.ErrMissingProvider) ||
				errors.Is(err, replai.ErrMissingModel) ||
				errors.Is(err, replai.ErrProviderNotFound) {
				status = http.StatusBadRequest
			}
			if errors.Is(err, replsession.ErrSessionNotFound) {
				status = http.StatusNotFound
			}
			h.writeJSONError(w, status, err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, resp)
	default:
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleReplAIAsk(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	resp, err := h.ai.Ask(req.Context(), replai.AskRequest{
		SessionID: strings.TrimSpace(req.Form.Get("session_id")),
		Prompt:    strings.TrimSpace(req.Form.Get("prompt")),
	})
	if err != nil {
		h.writeReplAIError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleReplAIGenerate(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	resp, err := h.ai.Generate(req.Context(), replai.GenerateRequest{
		SessionID:   strings.TrimSpace(req.Form.Get("session_id")),
		Description: strings.TrimSpace(req.Form.Get("description")),
	})
	if err != nil {
		h.writeReplAIError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleReplAIContext(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	switch req.Method {
	case http.MethodGet:
		sessionID := strings.TrimSpace(req.URL.Query().Get("session_id"))
		resp, err := h.ai.GetContext(req.Context(), replai.GetContextRequest{SessionID: sessionID})
		if err != nil {
			h.writeReplAIError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, resp)
	case http.MethodPost:
		if err := req.ParseForm(); err != nil {
			h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
			return
		}
		resp, err := h.ai.AddContext(req.Context(), replai.AddContextRequest{
			SessionID: strings.TrimSpace(req.Form.Get("session_id")),
			Ref:       strings.TrimSpace(req.Form.Get("ref")),
		})
		if err != nil {
			h.writeReplAIError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, resp)
	default:
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleReplAIPinContext(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	resp, err := h.ai.PinContext(req.Context(), replai.PinContextRequest{
		SessionID: strings.TrimSpace(req.Form.Get("session_id")),
		Ref:       strings.TrimSpace(req.Form.Get("ref")),
	})
	if err != nil {
		h.writeReplAIError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleReplAIUnpinContext(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	resp, err := h.ai.UnpinContext(req.Context(), replai.UnpinContextRequest{
		SessionID: strings.TrimSpace(req.Form.Get("session_id")),
		Ref:       strings.TrimSpace(req.Form.Get("ref")),
	})
	if err != nil {
		h.writeReplAIError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleReplAIClearContext(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	resp, err := h.ai.ClearContext(req.Context(), replai.ClearContextRequest{
		SessionID: strings.TrimSpace(req.Form.Get("session_id")),
	})
	if err != nil {
		h.writeReplAIError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleReplAIPolicy(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	switch req.Method {
	case http.MethodGet:
		sessionID := strings.TrimSpace(req.URL.Query().Get("session_id"))
		resp, err := h.ai.GetPolicy(req.Context(), replai.GetPolicyRequest{SessionID: sessionID})
		if err != nil {
			h.writeReplAIError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, resp)
	case http.MethodPost:
		if err := req.ParseForm(); err != nil {
			h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
			return
		}
		resp, err := h.ai.SetPolicy(req.Context(), replai.SetPolicyRequest{
			SessionID: strings.TrimSpace(req.Form.Get("session_id")),
			Policy:    strings.TrimSpace(req.Form.Get("policy")),
		})
		if err != nil {
			h.writeReplAIError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, resp)
	default:
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleReplAIHistory(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	sessionID := strings.TrimSpace(req.URL.Query().Get("session_id"))
	resp, err := h.ai.GetThread(req.Context(), replai.GetThreadRequest{SessionID: sessionID})
	if err != nil {
		h.writeReplAIError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleReplAIReset(w http.ResponseWriter, req *http.Request) {
	if h.ai == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl ai service not configured")
		return
	}
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	resp, err := h.ai.ResetThread(req.Context(), replai.ResetThreadRequest{
		SessionID: strings.TrimSpace(req.Form.Get("session_id")),
	})
	if err != nil {
		h.writeReplAIError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) writeReplAIError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, replai.ErrMissingSessionID) ||
		errors.Is(err, replai.ErrMissingProvider) ||
		errors.Is(err, replai.ErrMissingModel) ||
		errors.Is(err, replai.ErrProviderNotFound) ||
		errors.Is(err, replai.ErrMissingContextRef) ||
		errors.Is(err, replai.ErrUnsupportedApprovalPolicy) ||
		errors.Is(err, replai.ErrMissingPrompt) {
		status = http.StatusBadRequest
	}
	if errors.Is(err, replsession.ErrSessionNotFound) {
		status = http.StatusNotFound
	}
	h.writeJSONError(w, status, err.Error())
}

func (h *Handler) handleReplSessions(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.repl == nil {
		h.writeJSON(w, http.StatusOK, map[string]any{"sessions": []replsession.ReplSession{}})
		return
	}
	list, err := h.repl.ListSessions(req.Context(), replsession.ListSessionsRequest{})
	if err != nil {
		h.writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"sessions": list.Sessions})
}

func (h *Handler) handleReplSession(w http.ResponseWriter, req *http.Request) {
	if h.repl == nil {
		h.writeJSONError(w, http.StatusNotFound, "repl session service not configured")
		return
	}
	sessionID := strings.TrimSpace(strings.TrimPrefix(req.URL.Path, "/admin/api/repl/sessions/"))
	if sessionID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "session id is required")
		return
	}
	switch req.Method {
	case http.MethodGet:
		session, err := h.repl.GetSession(req.Context(), replsession.GetSessionRequest{SessionID: sessionID})
		if err != nil {
			h.writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{"session": session.Session})
	case http.MethodDelete:
		if _, err := h.repl.TerminateSession(req.Context(), replsession.TerminateSessionRequest{SessionID: sessionID}); err != nil {
			h.writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{
			"status":     "ok",
			"session_id": sessionID,
		})
	default:
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
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
			"log_dir":                    h.cfg.LogDir,
			"log_level":                  h.cfg.LogLevel,
			"log_max_bytes":              h.cfg.LogMaxBytes,
			"log_max_archives":           h.cfg.LogMaxArchives,
			"log_stderr":                 h.cfg.LogStderr,
			"photo_frame_dir":            h.cfg.PhotoFrameDir,
			"admin_http_host":            h.cfg.AdminHTTPHost,
			"admin_http_port":            h.cfg.AdminHTTPPort,
		},
		"timestamp_unix_ms": h.now().UTC().UnixMilli(),
	}
	h.writeJSON(w, http.StatusOK, status)
}
