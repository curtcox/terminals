package admin

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
)

func (h *Handler) handleChatMessages(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	msgs := scenario.SharedRoom().Messages()
	out := make([]map[string]any, 0, len(msgs))
	for _, msg := range msgs {
		out = append(out, map[string]any{
			"id":        msg.ID,
			"device_id": msg.DeviceID,
			"name":      msg.Name,
			"text":      msg.Text,
			"at":        msg.At.UTC().Format(time.RFC3339),
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"messages":     out,
		"participants": scenario.SharedRoom().Participants(),
	})
}

func (h *Handler) handleChatSend(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		DeviceID string `json:"device_id"`
		Name     string `json:"name"`
		Text     string `json:"text"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	body.DeviceID = strings.TrimSpace(body.DeviceID)
	body.Name = strings.TrimSpace(body.Name)
	body.Text = strings.TrimSpace(body.Text)
	if body.DeviceID == "" || body.Text == "" {
		h.writeJSONError(w, http.StatusBadRequest, "device_id and text are required")
		return
	}
	room := scenario.SharedRoom()
	if body.Name != "" {
		room.SetName(body.DeviceID, body.Name)
	}
	msg, ok := room.Post(body.DeviceID, body.Name, body.Text)
	if !ok {
		h.writeJSONError(w, http.StatusBadRequest, "message rejected")
		return
	}
	transport.BroadcastChatMessagesUpdate()
	h.writeJSON(w, http.StatusOK, map[string]any{
		"message": map[string]any{
			"id":        msg.ID,
			"device_id": msg.DeviceID,
			"name":      msg.Name,
			"text":      msg.Text,
			"at":        msg.At.UTC().Format(time.RFC3339),
		},
	})
}
