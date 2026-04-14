package transport

import (
	"context"
	"sync"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

type transportTestTelephonyTransport struct {
	registers []telephony.Registration
	invites   []telephony.Session
	byes      []telephony.Session
}

type transportTestMediaTransport struct {
	mu          sync.Mutex
	allocations []telephony.Session
	releases    []telephony.MediaPeer
}

func (m *transportTestMediaTransport) Allocate(_ context.Context, s telephony.Session) (telephony.MediaPeer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allocations = append(m.allocations, s)
	return telephony.MediaPeer{
		SessionID: s.ID,
		StreamID:  "sip:" + s.ID + ":audio",
		Codec:     "opus",
	}, nil
}

func (m *transportTestMediaTransport) Release(_ context.Context, peer telephony.MediaPeer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.releases = append(m.releases, peer)
	return nil
}

func (m *transportTestMediaTransport) Allocations() []telephony.Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]telephony.Session, len(m.allocations))
	copy(out, m.allocations)
	return out
}

func (m *transportTestMediaTransport) Releases() []telephony.MediaPeer {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]telephony.MediaPeer, len(m.releases))
	copy(out, m.releases)
	return out
}

func (t *transportTestTelephonyTransport) Register(_ context.Context, reg telephony.Registration) error {
	t.registers = append(t.registers, reg)
	return nil
}

func (t *transportTestTelephonyTransport) Invite(_ context.Context, s telephony.Session) error {
	t.invites = append(t.invites, s)
	return nil
}

func (t *transportTestTelephonyTransport) Bye(_ context.Context, s telephony.Session) error {
	t.byes = append(t.byes, s)
	return nil
}

func (t *transportTestTelephonyTransport) Close(context.Context) error { return nil }

func TestControlStreamVoiceCallDrivesSIPBridge(t *testing.T) {
	sipTransport := &transportTestTelephonyTransport{}
	bridge := telephony.NewSIPBridge(telephony.Registration{
		ServerURI:   "sip:home.example",
		Username:    "alice",
		DisplayName: "Alice",
	}, sipTransport)
	if err := bridge.Start(context.Background()); err != nil {
		t.Fatalf("bridge.Start() error = %v", err)
	}
	if len(sipTransport.registers) != 1 {
		t.Fatalf("register count = %d, want 1", len(sipTransport.registers))
	}

	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	broadcaster := ui.NewMemoryBroadcaster()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Telephony: bridge,
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: broadcaster,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	})

	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-call-1",
			DeviceID:  "device-1",
			Kind:      "voice",
			Text:      "call 5551212",
		},
	})
	if err != nil {
		t.Fatalf("HandleMessage(command voice call) error = %v", err)
	}
	sawStart := false
	for _, msg := range out {
		if msg.ScenarioStart == "phone_call" {
			sawStart = true
			break
		}
	}
	if !sawStart {
		t.Fatalf("expected phone_call scenario start in output: %+v", out)
	}

	if len(sipTransport.invites) != 1 {
		t.Fatalf("invite count = %d, want 1", len(sipTransport.invites))
	}
	if got := sipTransport.invites[0].Target; got != "5551212" {
		t.Fatalf("invite target = %q, want 5551212", got)
	}

	active := bridge.ActiveSessions()
	if len(active) != 1 {
		t.Fatalf("active sessions = %d, want 1", len(active))
	}

	if err := bridge.Hangup(context.Background(), active[0].ID); err != nil {
		t.Fatalf("Hangup() error = %v", err)
	}
	if len(sipTransport.byes) != 1 {
		t.Fatalf("bye count = %d, want 1", len(sipTransport.byes))
	}

	events := broadcaster.Events()
	if len(events) == 0 {
		t.Fatalf("expected broadcast event announcing the call")
	}
	sawCalling := false
	for _, ev := range events {
		if ev.Message == "Calling 5551212" {
			sawCalling = true
			break
		}
	}
	if !sawCalling {
		t.Fatalf("expected 'Calling 5551212' broadcast; got %+v", events)
	}
}

func TestControlStreamVoiceCallUnregisteredBridgeReturnsError(t *testing.T) {
	sipTransport := &transportTestTelephonyTransport{}
	bridge := telephony.NewSIPBridge(telephony.Registration{
		ServerURI: "sip:home.example",
		Username:  "alice",
	}, sipTransport)
	// Intentionally do NOT call Start; bridge is unregistered.

	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	broadcaster := ui.NewMemoryBroadcaster()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Telephony: bridge,
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: broadcaster,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	})

	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-call-unreg",
			DeviceID:  "device-1",
			Kind:      "voice",
			Text:      "call 5551212",
		},
	})
	if err == nil {
		t.Fatalf("expected error surfacing unregistered bridge")
	}
	if len(sipTransport.invites) != 0 {
		t.Fatalf("invite count = %d, want 0", len(sipTransport.invites))
	}
}

// TestControlStreamVoiceCallAllocatesWebRTCMediaPeer drives the full voice
// call path through the control stream and confirms that the SIPBridge's
// configured MediaTransport allocates a server-side WebRTC peer for the
// outbound call, and releases it on Hangup.
func TestControlStreamVoiceCallAllocatesWebRTCMediaPeer(t *testing.T) {
	sipTransport := &transportTestTelephonyTransport{}
	mediaTransport := &transportTestMediaTransport{}
	bridge := telephony.NewSIPBridge(telephony.Registration{
		ServerURI:   "sip:home.example",
		Username:    "alice",
		DisplayName: "Alice",
	}, sipTransport, telephony.WithMediaTransport(mediaTransport))
	if err := bridge.Start(context.Background()); err != nil {
		t.Fatalf("bridge.Start() error = %v", err)
	}

	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	broadcaster := ui.NewMemoryBroadcaster()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Telephony: bridge,
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: broadcaster,
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	})

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-call-1",
			DeviceID:  "device-1",
			Kind:      "voice",
			Text:      "call 5551212",
		},
	}); err != nil {
		t.Fatalf("HandleMessage(command voice call) error = %v", err)
	}

	if got := len(mediaTransport.Allocations()); got != 1 {
		t.Fatalf("media allocate count = %d, want 1", got)
	}
	if got := mediaTransport.Allocations()[0].Target; got != "5551212" {
		t.Fatalf("allocate target = %q, want 5551212", got)
	}

	peers := bridge.ActivePeers()
	if len(peers) != 1 {
		t.Fatalf("active peers = %d, want 1", len(peers))
	}
	if peers[0].StreamID == "" {
		t.Fatalf("media peer stream id should not be empty: %+v", peers[0])
	}

	sessionID := bridge.ActiveSessions()[0].ID
	if peer, ok := bridge.PeerForSession(sessionID); !ok || peer.SessionID != sessionID {
		t.Fatalf("PeerForSession(%q) = (%+v, %t), want a peer keyed by session", sessionID, peer, ok)
	}

	if err := bridge.Hangup(context.Background(), sessionID); err != nil {
		t.Fatalf("Hangup() error = %v", err)
	}
	if got := len(mediaTransport.Releases()); got != 1 {
		t.Fatalf("media release count = %d, want 1", got)
	}
	if got := mediaTransport.Releases()[0].SessionID; got != sessionID {
		t.Fatalf("release session = %q, want %q", got, sessionID)
	}
	if got := len(bridge.ActivePeers()); got != 0 {
		t.Fatalf("active peers after hangup = %d, want 0", got)
	}
}
