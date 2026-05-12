package transport

import (
	"context"
	"log/slog"
	"strings"
	"time"

	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
	"github.com/curtcox/terminals/terminal_server/internal/replsession"
)

func (h *StreamHandler) handleInput(ctx context.Context, in *InputRequest) ([]ServerMessage, error) {
	if in == nil {
		return nil, ErrInvalidClientMessage
	}
	deviceID := strings.TrimSpace(in.DeviceID)
	if deviceID == "" {
		return nil, ErrMissingCommandDeviceID
	}

	action := strings.ToLower(strings.TrimSpace(in.Action))
	componentID := strings.TrimSpace(in.ComponentID)
	_, _, logicalComponentID, _ := parseScopedComponentID(componentID)
	if logicalComponentID == "" {
		logicalComponentID = componentID
	}
	if requiresScopedUIActionComponent(action) && h.uiOwners.HasKnownActivation(deviceID) {
		if _, reason, ok := h.uiOwners.Resolve(deviceID, componentID); !ok {
			if h.metrics != nil {
				h.metrics.IncUnknownUIActionComponent(reason)
			}
			return nil, nil
		}
	}

	if strings.HasPrefix(action, bugReportActionPrefix) {
		return h.handleBugReportUIAction(ctx, deviceID, action, strings.TrimSpace(in.Value))
	}

	if responses, handled := h.handleChatInput(deviceID, componentID, action, in.Value); handled {
		return responses, nil
	}
	if responses, handled, err := h.handleMenuOverlayInput(ctx, deviceID, componentID, action); handled {
		return responses, err
	}
	if h.shouldDropMainInputWhileOverlayOpen(deviceID, in) {
		return nil, nil
	}

	switch action {
	case "change":
		if logicalComponentID == "terminal_input" {
			if sessionID, ok := h.replSessionIDForDevice(deviceID); ok {
				_ = h.replSessions.SetDraft(sessionID, deviceID, in.Value)
			}
			return nil, nil
		}
		if update, ok := h.renderTerminalUIAction(deviceID, componentID, action, in.Value); ok {
			return []ServerMessage{{UpdateUI: update}}, nil
		}
		return nil, nil
	case "toggle", "select":
		if update, ok := h.renderTerminalUIAction(deviceID, componentID, action, in.Value); ok {
			return []ServerMessage{{UpdateUI: update}}, nil
		}
		return nil, nil
	case SystemIntentTerminalRefresh:
		cmd := &CommandRequest{
			DeviceID: deviceID,
			Kind:     CommandKindManual,
			Intent:   SystemIntentTerminalRefresh,
		}
		commandResult, err := h.handleCommand(ctx, cmd)
		if err != nil {
			return nil, err
		}
		return h.commandResponses(ctx, cmd, commandResult), nil
	}

	if action != "" && (logicalComponentID != "terminal_input" || action != "submit") {
		if out, routed, err := h.routeScenarioUIAction(ctx, deviceID, action); routed {
			return out, err
		}
	}

	sessionID, ok := h.replSessionIDForDevice(deviceID)
	if !ok {
		return nil, nil
	}

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
	if text == "" || (!fromKey && strings.TrimSpace(text) == "") {
		return nil, nil
	}
	if fromKey {
		text = normalizeTerminalKeyText(text)
		eventlog.Emit(ctx, "terminal.input.received", slog.LevelDebug, "terminal key input received",
			slog.String("component", "transport.input"),
			slog.String("device_id", deviceID),
			slog.String("component_id", componentID),
			slog.Int("text_len", len(text)),
			slog.String("text", strings.NewReplacer("\n", "\\n", "\r", "\\r", "\b", "\\b", "\x7f", "<DEL>").Replace(text)),
		)
	}
	if !fromKey && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if _, err := h.replSessions.SendInput(ctx, replsession.SendInputRequest{
		SessionID: sessionID,
		DeviceID:  deviceID,
		Input:     text,
	}); err != nil {
		return nil, err
	}
	if logicalComponentID == "terminal_input" {
		_ = h.replSessions.ClearDraft(sessionID, deviceID)
	}

	h.readTerminalOutput(deviceID, sessionID)
	return []ServerMessage{{
		UpdateUI: h.terminalOutputUpdate(sessionID),
	}}, nil
}

func requiresScopedUIActionComponent(action string) bool {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "open", "close", "corner.open":
		return true
	default:
		return false
	}
}

func (h *StreamHandler) handleBugReportUIAction(ctx context.Context, reporterDeviceID, action, value string) ([]ServerMessage, error) {
	source, subjectDeviceID := parseBugReportUIAction(action, reporterDeviceID)
	if subjectDeviceID == "" {
		subjectDeviceID = strings.TrimSpace(value)
	}
	if subjectDeviceID == "" {
		subjectDeviceID = reporterDeviceID
	}
	ackMsg, err := h.diagnostics.HandleBugReport(ctx, &diagnosticsv1.BugReport{
		ReporterDeviceId: reporterDeviceID,
		SubjectDeviceId:  subjectDeviceID,
		Source:           source,
		Description:      "Filed from server-driven report button",
		Tags:             []string{"other"},
		TimestampUnixMs:  time.Now().UTC().UnixMilli(),
	})
	if err != nil {
		return nil, err
	}
	return []ServerMessage{
		ackMsg,
		{
			Notification: "Bug report filed: " + ackMsg.BugReportAck.GetReportId(),
		},
	}, nil
}

func parseBugReportUIAction(action, defaultSubjectDeviceID string) (diagnosticsv1.BugReportSource, string) {
	source := diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_SCREEN_BUTTON
	subjectDeviceID := strings.TrimSpace(defaultSubjectDeviceID)
	action = strings.TrimSpace(action)
	if action == "" {
		return source, subjectDeviceID
	}

	head := action
	if parts := strings.SplitN(action, ":", 2); len(parts) == 2 {
		head = strings.TrimSpace(parts[0])
		if parsedSubject := strings.TrimSpace(parts[1]); parsedSubject != "" {
			subjectDeviceID = parsedSubject
		}
	}

	modality := ""
	if parts := strings.SplitN(strings.ToLower(head), ".", 2); len(parts) == 2 {
		modality = strings.TrimSpace(parts[1])
	}

	switch modality {
	case "gesture":
		source = diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_GESTURE
	case "shake":
		source = diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_SHAKE
	case "keyboard":
		source = diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_KEYBOARD
	case "voice":
		source = diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_VOICE
	case "qr":
		source = diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_QR
	case "nfc":
		source = diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_NFC
	case "sip":
		source = diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_SIP
	case "admin":
		source = diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_ADMIN
	case "screen", "screen_button", "button", "":
		source = diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_SCREEN_BUTTON
	default:
		source = diagnosticsv1.BugReportSource_BUG_REPORT_SOURCE_OTHER
	}

	return source, subjectDeviceID
}
