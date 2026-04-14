// Package telephony contains scenario-facing telephony bridge abstractions.
package telephony

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// ErrNotRegistered is returned when a call is attempted before a bridge
// has successfully registered with its SIP provider.
var ErrNotRegistered = errors.New("telephony: bridge not registered")

// ErrSessionNotFound is returned when Hangup is asked to terminate a
// session that the bridge does not know about.
var ErrSessionNotFound = errors.New("telephony: session not found")

// ErrMissingTarget is returned when Call is invoked without a dial target.
var ErrMissingTarget = errors.New("telephony: call target required")

// Registration captures the account details used when registering with a
// SIP provider.
type Registration struct {
	ServerURI   string
	Username    string
	DisplayName string
	Password    string
}

// Sanitized returns a copy of the registration with sensitive fields
// removed. Intended for logging or diagnostics.
func (r Registration) Sanitized() Registration {
	out := r
	out.Password = ""
	return out
}

// Session represents an active SIP call session managed by the bridge.
type Session struct {
	ID     string
	Target string
}

// Transport performs the lower-level SIP exchanges on behalf of a bridge.
// Implementations may talk to a real SIP stack or a test double.
type Transport interface {
	// Register sends the REGISTER transaction. It is called at most once per
	// successful Start.
	Register(ctx context.Context, reg Registration) error
	// Invite opens a new call session with the provider.
	Invite(ctx context.Context, session Session) error
	// Bye ends an active call session.
	Bye(ctx context.Context, session Session) error
	// Close releases any transport-owned resources.
	Close(ctx context.Context) error
}

// NoopBridge is a placeholder telephony bridge that accepts every operation
// without touching a SIP provider. It is useful for environments where the
// telephony bridge has not been configured yet.
type NoopBridge struct{}

// Call accepts any target and returns nil.
func (NoopBridge) Call(context.Context, string) error {
	return nil
}

// Hangup accepts any session and returns nil.
func (NoopBridge) Hangup(context.Context, string) error {
	return nil
}

// SIPBridge is a configurable scenario.TelephonyBridge implementation backed
// by a pluggable Transport. A single bridge registers once and tracks the
// lifecycle of any outbound calls placed through it.
type SIPBridge struct {
	reg       Registration
	transport Transport

	mu         sync.Mutex
	started    bool
	registered bool
	counter    uint64
	sessions   map[string]Session
}

// NewSIPBridge constructs a SIPBridge. If transport is nil a LogTransport
// with no sink is used so the bridge remains safe to call without a SIP
// stack attached.
func NewSIPBridge(reg Registration, transport Transport) *SIPBridge {
	if transport == nil {
		transport = LogTransport{}
	}
	return &SIPBridge{
		reg:       reg,
		transport: transport,
		sessions:  map[string]Session{},
	}
}

// Registration returns a sanitized copy of the registration used by the
// bridge. Password is omitted.
func (b *SIPBridge) Registration() Registration {
	return b.reg.Sanitized()
}

// Registered reports whether Start has completed successfully and the
// bridge is ready to place calls.
func (b *SIPBridge) Registered() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.registered
}

// Start performs the one-time SIP REGISTER transaction. Subsequent calls
// are a no-op once the bridge is registered.
func (b *SIPBridge) Start(ctx context.Context) error {
	b.mu.Lock()
	if b.started {
		b.mu.Unlock()
		return nil
	}
	b.started = true
	b.mu.Unlock()

	if err := b.transport.Register(ctx, b.reg); err != nil {
		b.mu.Lock()
		b.started = false
		b.mu.Unlock()
		return fmt.Errorf("sip register: %w", err)
	}

	b.mu.Lock()
	b.registered = true
	b.mu.Unlock()
	return nil
}

// Stop terminates any outstanding sessions and releases transport
// resources. The bridge returns to an unregistered state so that Start can
// be invoked again.
func (b *SIPBridge) Stop(ctx context.Context) error {
	b.mu.Lock()
	active := make([]Session, 0, len(b.sessions))
	for _, session := range b.sessions {
		active = append(active, session)
	}
	b.sessions = map[string]Session{}
	b.started = false
	b.registered = false
	b.mu.Unlock()

	var firstErr error
	for _, session := range active {
		if err := b.transport.Bye(ctx, session); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if err := b.transport.Close(ctx); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// Call places an outbound call and records the session. It fails if the
// bridge has not been registered yet.
func (b *SIPBridge) Call(ctx context.Context, target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return ErrMissingTarget
	}

	b.mu.Lock()
	if !b.registered {
		b.mu.Unlock()
		return ErrNotRegistered
	}
	session := Session{ID: b.nextSessionIDLocked(), Target: target}
	b.sessions[session.ID] = session
	b.mu.Unlock()

	if err := b.transport.Invite(ctx, session); err != nil {
		b.mu.Lock()
		delete(b.sessions, session.ID)
		b.mu.Unlock()
		return fmt.Errorf("sip invite: %w", err)
	}
	return nil
}

// Hangup terminates an active call. If sessionID is empty and exactly one
// call is in flight that call is terminated, preserving Telephony bridge
// ergonomics for scenarios that track only one outbound call at a time.
func (b *SIPBridge) Hangup(ctx context.Context, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)

	b.mu.Lock()
	var session Session
	switch {
	case sessionID == "" && len(b.sessions) == 1:
		for _, s := range b.sessions {
			session = s
		}
	case sessionID == "":
		b.mu.Unlock()
		return ErrSessionNotFound
	default:
		existing, ok := b.sessions[sessionID]
		if !ok {
			b.mu.Unlock()
			return ErrSessionNotFound
		}
		session = existing
	}
	delete(b.sessions, session.ID)
	b.mu.Unlock()

	if err := b.transport.Bye(ctx, session); err != nil {
		return fmt.Errorf("sip bye: %w", err)
	}
	return nil
}

// ActiveSessions returns a snapshot of the in-flight call sessions owned
// by the bridge.
func (b *SIPBridge) ActiveSessions() []Session {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Session, 0, len(b.sessions))
	for _, session := range b.sessions {
		out = append(out, session)
	}
	return out
}

func (b *SIPBridge) nextSessionIDLocked() string {
	b.counter++
	prefix := strings.TrimSpace(b.reg.Username)
	if prefix == "" {
		prefix = "sip"
	}
	return prefix + "-" + strconv.FormatUint(b.counter, 10)
}

// LogTransport is a Transport that records SIP activity via an optional
// logging function. A nil Logf turns every operation into a silent no-op,
// which keeps the bridge safe when no real SIP stack has been wired up.
type LogTransport struct {
	Logf func(format string, args ...any)
}

// Register records the REGISTER transaction.
func (t LogTransport) Register(_ context.Context, reg Registration) error {
	t.logf(
		"sip: register server=%s user=%s display=%s",
		reg.ServerURI,
		reg.Username,
		reg.DisplayName,
	)
	return nil
}

// Invite records the outbound INVITE transaction.
func (t LogTransport) Invite(_ context.Context, session Session) error {
	t.logf("sip: invite session=%s target=%s", session.ID, session.Target)
	return nil
}

// Bye records the terminating BYE transaction.
func (t LogTransport) Bye(_ context.Context, session Session) error {
	t.logf("sip: bye session=%s target=%s", session.ID, session.Target)
	return nil
}

// Close records transport shutdown.
func (t LogTransport) Close(_ context.Context) error {
	t.logf("sip: close")
	return nil
}

func (t LogTransport) logf(format string, args ...any) {
	if t.Logf == nil {
		return
	}
	t.Logf(format, args...)
}
