package replsession

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/terminal"
)

func TestServiceLifecycleAndIO(t *testing.T) {
	svc := NewService(terminal.NewManager())
	ctx := context.Background()

	created, err := svc.CreateSession(ctx, CreateSessionRequest{
		DeviceID:          "dev-1",
		OwnerActivationID: "terminal",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if created.Session.ID == "" {
		t.Fatalf("expected non-empty session id")
	}

	sessionID := created.Session.ID
	if _, err := svc.SendInput(ctx, SendInputRequest{
		SessionID: sessionID,
		DeviceID:  "dev-1",
		Input:     "echo replsession-test\n",
	}); err != nil {
		t.Fatalf("SendInput() error = %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, _ = svc.ReadAvailable(sessionID, 4096)
		output, _ := svc.Output(sessionID)
		if strings.Contains(output, "replsession-test") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	output, err := svc.Output(sessionID)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if !strings.Contains(output, "replsession-test") {
		t.Fatalf("output missing expected marker, got: %q", output)
	}

	if _, err := svc.AttachSession(ctx, AttachSessionRequest{
		SessionID: sessionID,
		DeviceID:  "dev-2",
	}); err != nil {
		t.Fatalf("AttachSession() error = %v", err)
	}
	if _, ok := svc.SessionIDForDevice("dev-2"); !ok {
		t.Fatalf("expected attached device mapping")
	}

	if _, err := svc.DetachSession(ctx, DetachSessionRequest{
		SessionID: sessionID,
		DeviceID:  "dev-2",
	}); err != nil {
		t.Fatalf("DetachSession() error = %v", err)
	}
	if _, ok := svc.SessionIDForDevice("dev-2"); ok {
		t.Fatalf("expected detached device mapping removed")
	}

	if _, err := svc.ResizeSession(ctx, ResizeSessionRequest{
		SessionID: sessionID,
		TTYWidth:  120,
		TTYHeight: 40,
	}); err != nil {
		t.Fatalf("ResizeSession() error = %v", err)
	}
	got, err := svc.GetSession(ctx, GetSessionRequest{SessionID: sessionID})
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if got.Session.TTYWidth != 120 || got.Session.TTYHeight != 40 {
		t.Fatalf("unexpected tty size = %dx%d", got.Session.TTYWidth, got.Session.TTYHeight)
	}

	if _, err := svc.TerminateSession(ctx, TerminateSessionRequest{SessionID: sessionID}); err != nil {
		t.Fatalf("TerminateSession() error = %v", err)
	}
	if _, err := svc.GetSession(ctx, GetSessionRequest{SessionID: sessionID}); err == nil {
		t.Fatalf("expected session lookup failure after terminate")
	}
}
