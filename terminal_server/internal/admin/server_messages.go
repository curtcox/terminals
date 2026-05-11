package admin

import (
	"net/http"
	"strings"
)

func (h *Handler) handleMessages(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"messages": h.capability.ListMessages(req.URL.Query().Get("room"))})
}

func (h *Handler) handleMessageRooms(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"rooms": h.capability.ListMessageRooms()})
}

func (h *Handler) handleMessageRoom(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		roomRef := strings.TrimSpace(req.URL.Query().Get("room"))
		if roomRef == "" {
			h.writeJSONError(w, http.StatusBadRequest, "room is required")
			return
		}
		room, ok := h.capability.GetMessageRoom(roomRef)
		if !ok {
			h.writeJSONError(w, http.StatusNotFound, "room not found")
			return
		}
		h.writeJSON(w, http.StatusOK, map[string]any{"room": room})
	case http.MethodPost:
		if err := req.ParseForm(); err != nil {
			h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
			return
		}
		name := strings.TrimSpace(req.Form.Get("name"))
		if name == "" {
			h.writeJSONError(w, http.StatusBadRequest, "name is required")
			return
		}
		room := h.capability.CreateMessageRoom(name)
		h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "room": room})
	default:
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleMessageGet(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	messageID := strings.TrimSpace(req.URL.Query().Get("message_id"))
	if messageID == "" {
		h.writeJSONError(w, http.StatusBadRequest, "message_id is required")
		return
	}
	message, ok := h.capability.GetMessage(messageID)
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "message not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"message": message})
}

func (h *Handler) handleMessageUnread(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	identityID := strings.TrimSpace(req.URL.Query().Get("identity_id"))
	h.writeJSON(w, http.StatusOK, map[string]any{
		"identity_id": identityID,
		"messages":    h.capability.ListUnreadMessages(identityID, req.URL.Query().Get("room")),
	})
}

func (h *Handler) handleMessagePost(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	message := h.capability.PostMessage(req.Form.Get("room"), req.Form.Get("text"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "message": message})
}

func (h *Handler) handleMessageDirect(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	message := h.capability.SendDirectMessage(req.Form.Get("target_ref"), req.Form.Get("text"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "message": message})
}

func (h *Handler) handleMessageThread(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	message, ok := h.capability.ReplyMessageThread(req.Form.Get("root_ref"), req.Form.Get("text"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "root message not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "message": message})
}

func (h *Handler) handleMessageAck(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	ack, ok := h.capability.AcknowledgeMessage(req.Form.Get("identity_id"), req.Form.Get("message_id"))
	if !ok {
		h.writeJSONError(w, http.StatusNotFound, "message not found")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "ack": ack})
}

func (h *Handler) handleBoard(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": h.capability.ListBoard(req.URL.Query().Get("board"))})
}

func (h *Handler) handleBoardPost(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	item := h.capability.PostBoard(req.Form.Get("board"), req.Form.Get("text"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "item": item})
}

func (h *Handler) handleBoardPin(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		h.writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := req.ParseForm(); err != nil {
		h.writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}
	item := h.capability.PinBoard(req.Form.Get("board"), req.Form.Get("text"))
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "item": item})
}
