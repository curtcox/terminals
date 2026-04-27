package admin

import (
	"encoding/json"
	"net/http"

	"github.com/curtcox/terminals/terminal_server/internal/trust"
)

// trustService is the subset of trust.Service used by the admin handler.
type trustService interface {
	InstallerKeyID() string
	AddKey(keyID string, roles []string, ceiling *trust.VoucherCeiling, note string) error
	ConfirmKey(keyID string) error
	RevokeKey(keyID, reason string) ([]string, error)
	ArchiveKey(keyID string) error
	ListKeys() []*trust.KeyRecord
	GetKey(keyID string) (*trust.KeyRecord, error)
	VerifyChain() error
	LogEntries() []*trust.LogEntry
}

func (h *Handler) handleTrustKeys(w http.ResponseWriter, r *http.Request) {
	if h.trust == nil {
		http.Error(w, "trust service not configured", http.StatusNotImplemented)
		return
	}
	switch r.Method {
	case http.MethodGet:
		recs := h.trust.ListKeys()
		out := make([]map[string]any, 0, len(recs))
		for _, rec := range recs {
			out = append(out, keyRecordToJSON(rec))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"installer_key_id": h.trust.InstallerKeyID(),
			"keys":             out,
		})
	case http.MethodPost:
		var req struct {
			KeyID string   `json:"key_id"`
			Roles []string `json:"roles"`
			Note  string   `json:"note"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := h.trust.AddKey(req.KeyID, req.Roles, nil, req.Note); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"key_id": req.KeyID, "state": "candidate"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleTrustKeyConfirm(w http.ResponseWriter, r *http.Request) {
	if h.trust == nil {
		http.Error(w, "trust service not configured", http.StatusNotImplemented)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		KeyID string `json:"key_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.trust.ConfirmKey(req.KeyID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"key_id": req.KeyID, "state": "active"})
}

func (h *Handler) handleTrustKeyRevoke(w http.ResponseWriter, r *http.Request) {
	if h.trust == nil {
		http.Error(w, "trust service not configured", http.StatusNotImplemented)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		KeyID  string `json:"key_id"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	affected, err := h.trust.RevokeKey(req.KeyID, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"key_id":        req.KeyID,
		"state":         "revoked",
		"affected_apps": affected,
	})
}

func (h *Handler) handleTrustKeyArchive(w http.ResponseWriter, r *http.Request) {
	if h.trust == nil {
		http.Error(w, "trust service not configured", http.StatusNotImplemented)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		KeyID string `json:"key_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.trust.ArchiveKey(req.KeyID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"key_id": req.KeyID, "state": "archived"})
}

func (h *Handler) handleTrustVerify(w http.ResponseWriter, r *http.Request) {
	if h.trust == nil {
		http.Error(w, "trust service not configured", http.StatusNotImplemented)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	entries := h.trust.LogEntries()
	chainErr := h.trust.VerifyChain()
	status := "ok"
	errMsg := ""
	if chainErr != nil {
		status = "broken"
		errMsg = chainErr.Error()
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"chain_status":  status,
		"chain_error":   errMsg,
		"entry_count":   len(entries),
		"installer_key": h.trust.InstallerKeyID(),
	})
}

func (h *Handler) handleTrustLog(w http.ResponseWriter, r *http.Request) {
	if h.trust == nil {
		http.Error(w, "trust service not configured", http.StatusNotImplemented)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	entries := h.trust.LogEntries()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"entries": entries})
}

func keyRecordToJSON(rec *trust.KeyRecord) map[string]any {
	m := map[string]any{
		"key_id":            rec.KeyID,
		"roles":             rec.Roles,
		"state":             rec.State,
		"first_observed_at": rec.FirstObservedAt,
		"note":              rec.Note,
	}
	if rec.Ceiling != nil {
		m["voucher_ceiling"] = map[string]any{
			"max_tier":        rec.Ceiling.MaxTier,
			"allowed_testing": rec.Ceiling.AllowedTesting,
			"max_expiry_days": rec.Ceiling.MaxExpiryDays,
		}
	}
	return m
}
