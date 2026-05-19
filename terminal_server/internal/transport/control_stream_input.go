package transport

import (
	"context"
	"strings"
	"time"

	diagnosticsv1 "github.com/curtcox/terminals/terminal_server/gen/go/diagnostics/v1"
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
	logicalComponentID := logicalInputComponentID(componentID)
	if h.rejectUnknownScopedInput(deviceID, componentID, action) {
		return nil, nil
	}
	return h.dispatchInput(ctx, deviceID, componentID, logicalComponentID, action, in)
}

func logicalInputComponentID(componentID string) string {
	_, _, logicalComponentID, _ := parseScopedComponentID(componentID)
	if logicalComponentID == "" {
		return componentID
	}
	return logicalComponentID
}

func (h *StreamHandler) rejectUnknownScopedInput(deviceID, componentID, action string) bool {
	if !requiresScopedUIActionComponent(action) || !h.uiOwners.HasKnownActivation(deviceID) {
		return false
	}
	_, reason, ok := h.uiOwners.Resolve(deviceID, componentID)
	if ok {
		return false
	}
	if h.metrics != nil {
		h.metrics.IncUnknownUIActionComponent(reason)
	}
	return true
}

func (h *StreamHandler) dispatchInput(
	ctx context.Context,
	deviceID, componentID, logicalComponentID, action string,
	in *InputRequest,
) ([]ServerMessage, error) {
	if handled, out, err := h.handleInputEarly(ctx, deviceID, componentID, action, in.Value); handled {
		return out, err
	}
	if h.shouldDropMainInputWhileOverlayOpen(deviceID, in) {
		return nil, nil
	}
	if handled, out, err := h.handleInputUIChange(deviceID, componentID, logicalComponentID, action, in.Value); handled {
		return out, err
	}
	if action == SystemIntentTerminalRefresh {
		handled, out, err := h.handleInputSystemRefresh(ctx, deviceID)
		if handled {
			return out, err
		}
	}
	if action != "" && (logicalComponentID != "terminal_input" || action != "submit") {
		if out, routed, err := h.routeScenarioUIAction(ctx, deviceID, action); routed {
			return out, err
		}
	}
	if handled, out, err := h.handleInputTerminalSubmit(ctx, deviceID, componentID, logicalComponentID, in); handled {
		return out, err
	}
	return nil, nil
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
