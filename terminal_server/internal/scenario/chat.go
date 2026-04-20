package scenario

import (
	"context"
	"strings"
	"sync"

	"github.com/curtcox/terminals/terminal_server/internal/chat"
)

// SharedChatRoom is the process-global chat room. It is created lazily so
// tests can independently construct their own rooms; the transport layer
// uses the same instance for all chat activations.
var (
	sharedChatRoomOnce sync.Once
	sharedChatRoom     *chat.Room
)

// SharedRoom returns the process-global in-memory chat room.
func SharedRoom() *chat.Room {
	sharedChatRoomOnce.Do(func() {
		sharedChatRoom = chat.NewRoom()
	})
	return sharedChatRoom
}

// ChatScenario joins the requesting device to the shared chat room. The
// scenario owns participant membership; all UI broadcasting is handled by
// the transport layer which calls into the room directly.
type ChatScenario struct {
	trigger Trigger
	room    *chat.Room
}

// Name returns the stable scenario identifier.
func (s *ChatScenario) Name() string { return "chat" }

// Match records trigger metadata when chat mode is requested.
func (s *ChatScenario) Match(trigger Trigger) bool {
	if !intentMatches(trigger.Intent, "chat", "open chat", "join chat") {
		return false
	}
	s.trigger = trigger
	return true
}

// Start registers the device as a chat participant.
func (s *ChatScenario) Start(ctx context.Context, env *Environment) error {
	room := s.resolvedRoom()
	deviceID := strings.TrimSpace(s.trigger.SourceID)
	if deviceID != "" {
		room.Join(deviceID)
	}
	if env != nil && env.Broadcast != nil && deviceID != "" {
		return env.Broadcast.Notify(ctx, []string{deviceID}, "Chat active")
	}
	return nil
}

// Stop removes the participant from the active set. Identity + message
// history are retained per the room's retention policy.
func (s *ChatScenario) Stop() error {
	room := s.resolvedRoom()
	deviceID := strings.TrimSpace(s.trigger.SourceID)
	if deviceID != "" {
		room.Leave(deviceID)
	}
	return nil
}

func (s *ChatScenario) resolvedRoom() *chat.Room {
	if s.room != nil {
		return s.room
	}
	return SharedRoom()
}
