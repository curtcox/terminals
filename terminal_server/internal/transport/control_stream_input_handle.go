package transport

import (
	"context"
	"log/slog"
	"strings"

	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
	"github.com/curtcox/terminals/terminal_server/internal/replsession"
)

func (h *StreamHandler) handleInputEarly(
	ctx context.Context,
	deviceID, componentID, action, value string,
) (bool, []ServerMessage, error) {
	if strings.HasPrefix(action, bugReportActionPrefix) {
		out, err := h.handleBugReportUIAction(ctx, deviceID, action, strings.TrimSpace(value))
		return true, out, err
	}
	if responses, handled := h.handleChatInput(deviceID, componentID, action, value); handled {
		return true, responses, nil
	}
	if responses, handled, err := h.handleMenuOverlayInput(ctx, deviceID, componentID, action); handled {
		return true, responses, err
	}
	return false, nil, nil
}

func (h *StreamHandler) handleInputUIChange(
	deviceID, componentID, logicalComponentID, action, value string,
) (bool, []ServerMessage, error) {
	switch action {
	case "change":
		if logicalComponentID == "terminal_input" {
			if sessionID, ok := h.replSessionIDForDevice(deviceID); ok {
				_ = h.replSessions.SetDraft(sessionID, deviceID, value)
			}
			return true, nil, nil
		}
		if update, ok := h.renderTerminalUIAction(deviceID, componentID, action, value); ok {
			return true, []ServerMessage{{UpdateUI: update}}, nil
		}
		return true, nil, nil
	case "toggle", "select":
		if update, ok := h.renderTerminalUIAction(deviceID, componentID, action, value); ok {
			return true, []ServerMessage{{UpdateUI: update}}, nil
		}
		return true, nil, nil
	default:
		return false, nil, nil
	}
}

func (h *StreamHandler) handleInputSystemRefresh(ctx context.Context, deviceID string) (bool, []ServerMessage, error) {
	cmd := &CommandRequest{
		DeviceID: deviceID,
		Kind:     CommandKindManual,
		Intent:   SystemIntentTerminalRefresh,
	}
	commandResult, err := h.handleCommand(ctx, cmd)
	if err != nil {
		return true, nil, err
	}
	return true, h.commandResponses(ctx, cmd, commandResult), nil
}

func (h *StreamHandler) handleInputTerminalSubmit(
	ctx context.Context,
	deviceID, componentID, logicalComponentID string,
	in *InputRequest,
) (bool, []ServerMessage, error) {
	sessionID, ok := h.replSessionIDForDevice(deviceID)
	if !ok {
		return false, nil, nil
	}
	text, fromKey := inputTerminalText(h, sessionID, deviceID, logicalComponentID, in)
	if text == "" || (!fromKey && strings.TrimSpace(text) == "") {
		return true, nil, nil
	}
	if fromKey {
		emitTerminalKeyInput(ctx, deviceID, componentID, text)
	}
	if !fromKey && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if _, err := h.replSessions.SendInput(ctx, replsession.SendInputRequest{
		SessionID: sessionID,
		DeviceID:  deviceID,
		Input:     text,
	}); err != nil {
		return true, nil, err
	}
	if logicalComponentID == "terminal_input" {
		_ = h.replSessions.ClearDraft(sessionID, deviceID)
	}
	h.readTerminalOutput(deviceID, sessionID)
	return true, []ServerMessage{{UpdateUI: h.terminalOutputUpdate(sessionID)}}, nil
}

func inputTerminalText(
	h *StreamHandler,
	sessionID, deviceID, logicalComponentID string,
	in *InputRequest,
) (string, bool) {
	text := in.Value
	fromKey := false
	if text == "" && logicalComponentID == "terminal_input" {
		draft, err := h.replSessions.Draft(sessionID, deviceID)
		if err == nil {
			text = draft
		}
	}
	if text == "" {
		text = in.KeyText
		fromKey = text != ""
	}
	if fromKey {
		text = normalizeTerminalKeyText(text)
	}
	return text, fromKey
}

func emitTerminalKeyInput(ctx context.Context, deviceID, componentID, text string) {
	eventlog.Emit(ctx, "terminal.input.received", slog.LevelDebug, "terminal key input received",
		slog.String("component", "transport.input"),
		slog.String("device_id", deviceID),
		slog.String("component_id", componentID),
		slog.Int("text_len", len(text)),
		slog.String("text", strings.NewReplacer("\n", "\\n", "\r", "\\r", "\b", "\\b", "\x7f", "<DEL>").Replace(text)),
	)
}
