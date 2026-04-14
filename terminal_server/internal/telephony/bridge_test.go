package telephony

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

type recordingTransport struct {
	mu         sync.Mutex
	regs       []Registration
	invites    []Session
	byes       []Session
	closes     int
	failOnBye  bool
	failOnInvt bool
	failOnReg  bool
}

func (r *recordingTransport) Register(_ context.Context, reg Registration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failOnReg {
		return errors.New("register failed")
	}
	r.regs = append(r.regs, reg)
	return nil
}

func (r *recordingTransport) Invite(_ context.Context, s Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failOnInvt {
		return errors.New("invite failed")
	}
	r.invites = append(r.invites, s)
	return nil
}

func (r *recordingTransport) Bye(_ context.Context, s Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failOnBye {
		return errors.New("bye failed")
	}
	r.byes = append(r.byes, s)
	return nil
}

func (r *recordingTransport) Close(_ context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closes++
	return nil
}

func newTestBridge(t *testing.T, tr Transport) *SIPBridge {
	t.Helper()
	bridge := NewSIPBridge(Registration{
		ServerURI:   "sip:home.example",
		Username:    "alice",
		DisplayName: "Alice",
		Password:    "secret",
	}, tr)
	if err := bridge.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	return bridge
}

func TestSIPBridgeRegisterOnStart(t *testing.T) {
	tr := &recordingTransport{}
	bridge := newTestBridge(t, tr)

	if !bridge.Registered() {
		t.Fatalf("bridge should be registered after Start")
	}
	if got := len(tr.regs); got != 1 {
		t.Fatalf("register count = %d, want 1", got)
	}
	if got := tr.regs[0].Username; got != "alice" {
		t.Fatalf("register username = %q, want alice", got)
	}

	if err := bridge.Start(context.Background()); err != nil {
		t.Fatalf("second Start() error = %v", err)
	}
	if got := len(tr.regs); got != 1 {
		t.Fatalf("register count after second Start = %d, want 1", got)
	}
}

func TestSIPBridgeCallTracksSession(t *testing.T) {
	tr := &recordingTransport{}
	bridge := newTestBridge(t, tr)

	if err := bridge.Call(context.Background(), "5551212"); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if got := len(tr.invites); got != 1 {
		t.Fatalf("invite count = %d, want 1", got)
	}
	if got := tr.invites[0].Target; got != "5551212" {
		t.Fatalf("invite target = %q, want 5551212", got)
	}
	if got := tr.invites[0].ID; !strings.HasPrefix(got, "alice-") {
		t.Fatalf("session id = %q, want prefix alice-", got)
	}

	sessions := bridge.ActiveSessions()
	if len(sessions) != 1 {
		t.Fatalf("active sessions = %d, want 1", len(sessions))
	}
}

func TestSIPBridgeCallRequiresRegistration(t *testing.T) {
	tr := &recordingTransport{}
	bridge := NewSIPBridge(Registration{ServerURI: "sip:h", Username: "alice"}, tr)

	err := bridge.Call(context.Background(), "5551212")
	if !errors.Is(err, ErrNotRegistered) {
		t.Fatalf("Call() error = %v, want ErrNotRegistered", err)
	}
	if len(tr.invites) != 0 {
		t.Fatalf("invite count = %d, want 0", len(tr.invites))
	}
}

func TestSIPBridgeCallRejectsEmptyTarget(t *testing.T) {
	bridge := newTestBridge(t, &recordingTransport{})
	if err := bridge.Call(context.Background(), "   "); !errors.Is(err, ErrMissingTarget) {
		t.Fatalf("Call() error = %v, want ErrMissingTarget", err)
	}
}

func TestSIPBridgeCallInviteFailureReleasesSession(t *testing.T) {
	tr := &recordingTransport{failOnInvt: true}
	bridge := newTestBridge(t, tr)

	if err := bridge.Call(context.Background(), "5551212"); err == nil {
		t.Fatalf("Call() expected error")
	}
	if got := len(bridge.ActiveSessions()); got != 0 {
		t.Fatalf("active sessions = %d, want 0", got)
	}
}

func TestSIPBridgeHangupByID(t *testing.T) {
	tr := &recordingTransport{}
	bridge := newTestBridge(t, tr)

	if err := bridge.Call(context.Background(), "5551212"); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if err := bridge.Call(context.Background(), "5553434"); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if got := len(bridge.ActiveSessions()); got != 2 {
		t.Fatalf("active sessions = %d, want 2", got)
	}

	target := tr.invites[0].ID
	if err := bridge.Hangup(context.Background(), target); err != nil {
		t.Fatalf("Hangup() error = %v", err)
	}
	if got := len(tr.byes); got != 1 {
		t.Fatalf("bye count = %d, want 1", got)
	}
	if got := tr.byes[0].ID; got != target {
		t.Fatalf("bye session id = %q, want %q", got, target)
	}
	remaining := bridge.ActiveSessions()
	if len(remaining) != 1 || remaining[0].Target != "5553434" {
		t.Fatalf("remaining = %+v, want single call to 5553434", remaining)
	}
}

func TestSIPBridgeHangupImplicitSingleSession(t *testing.T) {
	tr := &recordingTransport{}
	bridge := newTestBridge(t, tr)

	if err := bridge.Call(context.Background(), "5551212"); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if err := bridge.Hangup(context.Background(), ""); err != nil {
		t.Fatalf("Hangup() error = %v", err)
	}
	if got := len(tr.byes); got != 1 {
		t.Fatalf("bye count = %d, want 1", got)
	}
	if got := len(bridge.ActiveSessions()); got != 0 {
		t.Fatalf("active sessions = %d, want 0", got)
	}
}

func TestSIPBridgeHangupUnknownSession(t *testing.T) {
	tr := &recordingTransport{}
	bridge := newTestBridge(t, tr)

	if err := bridge.Hangup(context.Background(), "missing"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Hangup() error = %v, want ErrSessionNotFound", err)
	}
	if err := bridge.Hangup(context.Background(), ""); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Hangup() with empty id error = %v, want ErrSessionNotFound", err)
	}
}

func TestSIPBridgeStopClearsSessions(t *testing.T) {
	tr := &recordingTransport{}
	bridge := newTestBridge(t, tr)

	if err := bridge.Call(context.Background(), "5551212"); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if err := bridge.Call(context.Background(), "5553434"); err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	if err := bridge.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if bridge.Registered() {
		t.Fatalf("bridge should not be registered after Stop")
	}
	if got := len(bridge.ActiveSessions()); got != 0 {
		t.Fatalf("active sessions = %d, want 0", got)
	}
	if got := len(tr.byes); got != 2 {
		t.Fatalf("bye count = %d, want 2", got)
	}
	if tr.closes != 1 {
		t.Fatalf("close count = %d, want 1", tr.closes)
	}

	// After Stop the bridge must reject new calls until Started again.
	if err := bridge.Call(context.Background(), "5559999"); !errors.Is(err, ErrNotRegistered) {
		t.Fatalf("Call after Stop error = %v, want ErrNotRegistered", err)
	}
}

func TestSIPBridgeStartPropagatesRegisterError(t *testing.T) {
	tr := &recordingTransport{failOnReg: true}
	bridge := NewSIPBridge(Registration{ServerURI: "sip:h", Username: "alice"}, tr)

	err := bridge.Start(context.Background())
	if err == nil {
		t.Fatalf("Start() expected error")
	}
	if !strings.Contains(err.Error(), "register") {
		t.Fatalf("Start() error = %v, want contains register", err)
	}
	if bridge.Registered() {
		t.Fatalf("bridge should not be registered after failed Start")
	}
}

func TestSIPBridgeSanitizedRegistration(t *testing.T) {
	bridge := NewSIPBridge(Registration{
		ServerURI:   "sip:home.example",
		Username:    "alice",
		DisplayName: "Alice",
		Password:    "secret",
	}, LogTransport{})

	reg := bridge.Registration()
	if reg.Password != "" {
		t.Fatalf("sanitized password = %q, want empty", reg.Password)
	}
	if reg.Username != "alice" {
		t.Fatalf("sanitized username = %q, want alice", reg.Username)
	}
}

func TestLogTransportSafeWithoutSink(t *testing.T) {
	tr := LogTransport{}
	ctx := context.Background()
	if err := tr.Register(ctx, Registration{}); err != nil {
		t.Fatalf("Register error = %v", err)
	}
	if err := tr.Invite(ctx, Session{ID: "s1", Target: "t"}); err != nil {
		t.Fatalf("Invite error = %v", err)
	}
	if err := tr.Bye(ctx, Session{ID: "s1"}); err != nil {
		t.Fatalf("Bye error = %v", err)
	}
	if err := tr.Close(ctx); err != nil {
		t.Fatalf("Close error = %v", err)
	}
}

func TestLogTransportPassesThroughSink(t *testing.T) {
	var captured []string
	tr := LogTransport{Logf: func(format string, _ ...any) {
		captured = append(captured, format)
	}}
	ctx := context.Background()
	_ = tr.Register(ctx, Registration{})
	_ = tr.Invite(ctx, Session{})
	_ = tr.Bye(ctx, Session{})
	_ = tr.Close(ctx)
	if len(captured) != 4 {
		t.Fatalf("log calls = %d, want 4", len(captured))
	}
}

func TestNoopBridgeReturnsNil(t *testing.T) {
	b := NoopBridge{}
	if err := b.Call(context.Background(), "5551212"); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if err := b.Hangup(context.Background(), "sid"); err != nil {
		t.Fatalf("Hangup() error = %v", err)
	}
}
