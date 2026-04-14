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

// MediaPeer describes the server-side WebRTC peer that the bridge has
// allocated for a SIP call session. It is the handle scenarios use to
// wire the client mic/speaker to the SIP audio path.
type MediaPeer struct {
	SessionID string
	StreamID  string
	Codec     string
}

// MediaTransport allocates and releases server-side WebRTC peers that
// bridge between a SIP session and a client audio stream. Implementations
// may stand up a real Pion peer connection or a test double.
type MediaTransport interface {
	Allocate(ctx context.Context, session Session) (MediaPeer, error)
	Release(ctx context.Context, peer MediaPeer) error
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
	media     MediaTransport

	mu         sync.Mutex
	started    bool
	registered bool
	counter    uint64
	sessions   map[string]Session
	peers      map[string]MediaPeer
}

// Option configures optional SIPBridge behavior.
type Option func(*SIPBridge)

// WithMediaTransport wires a MediaTransport into the bridge so that each
// outbound call also allocates a server-side WebRTC peer for the audio
// path between the originating client and the SIP party.
func WithMediaTransport(media MediaTransport) Option {
	return func(b *SIPBridge) {
		b.media = media
	}
}

// NewSIPBridge constructs a SIPBridge. If transport is nil a LogTransport
// with no sink is used so the bridge remains safe to call without a SIP
// stack attached.
func NewSIPBridge(reg Registration, transport Transport, opts ...Option) *SIPBridge {
	if transport == nil {
		transport = LogTransport{}
	}
	b := &SIPBridge{
		reg:       reg,
		transport: transport,
		sessions:  map[string]Session{},
		peers:     map[string]MediaPeer{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}
	return b
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
// be invoked again. Allocated WebRTC peers are released alongside their
// SIP sessions.
func (b *SIPBridge) Stop(ctx context.Context) error {
	b.mu.Lock()
	active := make([]Session, 0, len(b.sessions))
	for _, session := range b.sessions {
		active = append(active, session)
	}
	activePeers := make([]MediaPeer, 0, len(b.peers))
	for _, peer := range b.peers {
		activePeers = append(activePeers, peer)
	}
	b.sessions = map[string]Session{}
	b.peers = map[string]MediaPeer{}
	b.started = false
	b.registered = false
	media := b.media
	b.mu.Unlock()

	var firstErr error
	if media != nil {
		for _, peer := range activePeers {
			if err := media.Release(ctx, peer); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
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
// bridge has not been registered yet. When a MediaTransport is configured
// the bridge also allocates a server-side WebRTC peer for the call so the
// audio path between the client mic/speaker and the SIP party is in place
// before Call returns.
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
	media := b.media
	b.mu.Unlock()

	if err := b.transport.Invite(ctx, session); err != nil {
		b.mu.Lock()
		delete(b.sessions, session.ID)
		b.mu.Unlock()
		return fmt.Errorf("sip invite: %w", err)
	}

	if media != nil {
		peer, err := media.Allocate(ctx, session)
		if err != nil {
			// Roll back both the SIP session and the bridge bookkeeping so the
			// caller does not see a half-formed call.
			_ = b.transport.Bye(ctx, session)
			b.mu.Lock()
			delete(b.sessions, session.ID)
			b.mu.Unlock()
			return fmt.Errorf("media allocate: %w", err)
		}
		if peer.SessionID == "" {
			peer.SessionID = session.ID
		}
		b.mu.Lock()
		b.peers[session.ID] = peer
		b.mu.Unlock()
	}
	return nil
}

// Hangup terminates an active call. If sessionID is empty and exactly one
// call is in flight that call is terminated, preserving Telephony bridge
// ergonomics for scenarios that track only one outbound call at a time.
// Any server-side WebRTC peer that was allocated for the session is
// released alongside the SIP BYE.
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
	peer, hasPeer := b.peers[session.ID]
	delete(b.peers, session.ID)
	media := b.media
	b.mu.Unlock()

	var firstErr error
	if hasPeer && media != nil {
		if err := media.Release(ctx, peer); err != nil {
			firstErr = fmt.Errorf("media release: %w", err)
		}
	}
	if err := b.transport.Bye(ctx, session); err != nil {
		if firstErr == nil {
			firstErr = fmt.Errorf("sip bye: %w", err)
		}
	}
	return firstErr
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

// ActivePeers returns a snapshot of the WebRTC media peers currently
// allocated by the bridge.
func (b *SIPBridge) ActivePeers() []MediaPeer {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]MediaPeer, 0, len(b.peers))
	for _, peer := range b.peers {
		out = append(out, peer)
	}
	return out
}

// PeerForSession returns the WebRTC media peer associated with the given
// SIP session, if one has been allocated.
func (b *SIPBridge) PeerForSession(sessionID string) (MediaPeer, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	peer, ok := b.peers[sessionID]
	return peer, ok
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

// LogMediaTransport is a MediaTransport that fabricates a deterministic
// MediaPeer per session and records activity via an optional logging
// function. It exists so the bridge can stand up a WebRTC peer placeholder
// without a real WebRTC stack — useful for development, configs without an
// SFU, and tests that only need to verify allocation and release happen.
type LogMediaTransport struct {
	Logf  func(format string, args ...any)
	Codec string
}

// Allocate returns a deterministic MediaPeer keyed off the session ID.
func (t LogMediaTransport) Allocate(_ context.Context, session Session) (MediaPeer, error) {
	codec := strings.TrimSpace(t.Codec)
	if codec == "" {
		codec = "opus"
	}
	peer := MediaPeer{
		SessionID: session.ID,
		StreamID:  "sip:" + session.ID + ":audio",
		Codec:     codec,
	}
	t.logf(
		"sip-media: allocate session=%s stream=%s codec=%s",
		peer.SessionID,
		peer.StreamID,
		peer.Codec,
	)
	return peer, nil
}

// Release records the WebRTC peer teardown.
func (t LogMediaTransport) Release(_ context.Context, peer MediaPeer) error {
	t.logf("sip-media: release session=%s stream=%s", peer.SessionID, peer.StreamID)
	return nil
}

func (t LogMediaTransport) logf(format string, args ...any) {
	if t.Logf == nil {
		return
	}
	t.Logf(format, args...)
}
