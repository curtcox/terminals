package transport

import (
	"context"
	"strings"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/replsession"
)

func (h *StreamHandler) commandTerminalRefresh(_ context.Context, cmd *CommandRequest) (ServerMessage, bool) {
	if cmd == nil {
		return ServerMessage{}, false
	}
	targetDeviceID := ""
	switch cmd.Kind {
	case CommandKindManual:
		if strings.TrimSpace(cmd.Intent) != SystemIntentTerminalRefresh {
			return ServerMessage{}, false
		}
		targetDeviceID = strings.TrimSpace(cmd.DeviceID)
	case CommandKindSystem:
		parsed, err := ParseSystemIntent(cmd.Intent)
		if err != nil || parsed.Name != SystemIntentTerminalRefresh {
			return ServerMessage{}, false
		}
		targetDeviceID = strings.TrimSpace(parsed.Arg)
		if targetDeviceID == "" {
			targetDeviceID = strings.TrimSpace(cmd.DeviceID)
		}
	default:
		return ServerMessage{}, false
	}
	if targetDeviceID == "" {
		return ServerMessage{}, false
	}
	update, err := h.pollTerminalOutput(targetDeviceID, true)
	if err != nil || update == nil {
		return ServerMessage{}, false
	}
	return *update, true
}

func (h *StreamHandler) ensureTerminalSession(ctx context.Context, deviceID string) (string, error) {
	if strings.TrimSpace(deviceID) == "" {
		return "", ErrMissingCommandDeviceID
	}
	if h.replSessions == nil {
		h.replSessions = replsession.NewService(h.terminals)
	}
	if existingID, ok := h.replSessions.SessionIDForDevice(deviceID); ok {
		return h.replSessions.Output(existingID)
	}

	h.mu.Lock()
	replAdminURL := h.terminalReplAdminURL
	h.mu.Unlock()
	session, err := h.replSessions.CreateSession(ctx, replsession.CreateSessionRequest{
		DeviceID:          deviceID,
		OwnerActivationID: "terminal",
		ReplAdminURL:      replAdminURL,
	})
	if err != nil {
		return "", err
	}
	return h.replSessions.Output(session.Session.ID)
}

func (h *StreamHandler) terminateTerminalForDevice(deviceID string) {
	if strings.TrimSpace(deviceID) == "" {
		return
	}
	if h.replSessions == nil {
		return
	}
	sessionID, ok := h.replSessions.SessionIDForDevice(deviceID)
	if !ok {
		return
	}
	_, _ = h.replSessions.TerminateSession(context.Background(), replsession.TerminateSessionRequest{
		SessionID: sessionID,
	})
}

func (h *StreamHandler) readTerminalOutput(_ string, sessionID string) string {
	readDeadline, readInterval := h.terminalReadSettings()
	deadline := time.Now().Add(readDeadline)
	for time.Now().Before(deadline) {
		_, err := h.replSessions.ReadAvailable(sessionID, 4096)
		if err != nil {
			break
		}
		time.Sleep(readInterval)
	}
	output, _ := h.replSessions.Output(sessionID)
	return output
}

func (h *StreamHandler) replSessionIDForDevice(deviceID string) (string, bool) {
	if h.replSessions == nil {
		return "", false
	}
	return h.replSessions.SessionIDForDevice(deviceID)
}

func (h *StreamHandler) terminalReadSettings() (time.Duration, time.Duration) {
	readDeadline := h.terminalReadDeadline
	readInterval := h.terminalReadInterval

	if readDeadline <= 0 {
		readDeadline = defaultTerminalReadDeadline
	}
	if readInterval <= 0 {
		readInterval = defaultTerminalReadInterval
	}
	if readInterval > readDeadline {
		readInterval = readDeadline
	}
	return readDeadline, readInterval
}
