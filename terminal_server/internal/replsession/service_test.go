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
	if created.Session.Origin != SessionOriginHuman {
		t.Fatalf("session origin = %q, want %q", created.Session.Origin, SessionOriginHuman)
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

func TestCreateSessionMCPOriginMetadata(t *testing.T) {
	svc := NewService(terminal.NewManager())
	ctx := context.Background()
	created, err := svc.CreateSession(ctx, CreateSessionRequest{
		DeviceID:        "dev-mcp-1",
		Origin:          SessionOriginMCP,
		AgentIdentity:   "codex-desktop",
		AgentCapability: "mutating_via_elicitation",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if created.Session.Origin != SessionOriginMCP {
		t.Fatalf("origin = %q, want %q", created.Session.Origin, SessionOriginMCP)
	}
	if created.Session.AgentIdentity != "codex-desktop" {
		t.Fatalf("agent identity = %q", created.Session.AgentIdentity)
	}
	if created.Session.AgentCapability != "mutating_via_elicitation" {
		t.Fatalf("agent capability = %q", created.Session.AgentCapability)
	}
	_, _ = svc.TerminateSession(ctx, TerminateSessionRequest{SessionID: created.Session.ID})
}

func TestSelectionPersistence(t *testing.T) {
	svc := NewService(terminal.NewManager())
	ctx := context.Background()

	created, err := svc.CreateSession(ctx, CreateSessionRequest{DeviceID: "dev-1"})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	sessionID := created.Session.ID
	defer func() {
		_, _ = svc.TerminateSession(ctx, TerminateSessionRequest{SessionID: sessionID})
	}()

	if err := svc.SetSelection(sessionID, "ollama", "llama3.1"); err != nil {
		t.Fatalf("SetSelection() error = %v", err)
	}
	provider, model, err := svc.GetSelection(sessionID)
	if err != nil {
		t.Fatalf("GetSelection() error = %v", err)
	}
	if provider != "ollama" || model != "llama3.1" {
		t.Fatalf("selection = %s/%s, want ollama/llama3.1", provider, model)
	}
}

func TestUseCaseP2SessionMobilityAndCoexistence(t *testing.T) {
	svc := NewService(terminal.NewManager())
	ctx := context.Background()

	primary, err := svc.CreateSession(ctx, CreateSessionRequest{DeviceID: "device-a"})
	if err != nil {
		t.Fatalf("CreateSession(primary) error = %v", err)
	}
	defer func() {
		_, _ = svc.TerminateSession(ctx, TerminateSessionRequest{SessionID: primary.Session.ID})
	}()

	secondary, err := svc.CreateSession(ctx, CreateSessionRequest{DeviceID: "device-b"})
	if err != nil {
		t.Fatalf("CreateSession(secondary) error = %v", err)
	}
	defer func() {
		_, _ = svc.TerminateSession(ctx, TerminateSessionRequest{SessionID: secondary.Session.ID})
	}()

	if primary.Session.ID == secondary.Session.ID {
		t.Fatalf("expected distinct sessions for separate devices")
	}

	listed, err := svc.ListSessions(ctx, ListSessionsRequest{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(listed.Sessions) != 2 {
		t.Fatalf("ListSessions() count = %d, want 2", len(listed.Sessions))
	}

	attached, err := svc.AttachSession(ctx, AttachSessionRequest{
		SessionID: primary.Session.ID,
		DeviceID:  "device-c",
	})
	if err != nil {
		t.Fatalf("AttachSession() error = %v", err)
	}
	if len(attached.Session.AttachedDevices) != 2 {
		t.Fatalf("attached device count = %d, want 2", len(attached.Session.AttachedDevices))
	}

	if mapped, ok := svc.SessionIDForDevice("device-c"); !ok || mapped != primary.Session.ID {
		t.Fatalf("SessionIDForDevice(device-c) = %q,%v want %q,true", mapped, ok, primary.Session.ID)
	}

	detached, err := svc.DetachSession(ctx, DetachSessionRequest{
		SessionID: primary.Session.ID,
		DeviceID:  "device-c",
	})
	if err != nil {
		t.Fatalf("DetachSession() error = %v", err)
	}
	if len(detached.Session.AttachedDevices) != 1 {
		t.Fatalf("attached device count after detach = %d, want 1", len(detached.Session.AttachedDevices))
	}
	if _, ok := svc.SessionIDForDevice("device-c"); ok {
		t.Fatalf("expected detached device mapping removed")
	}
}
