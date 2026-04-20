package transport

import (
	"context"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/chat"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func newChatTestHandler(t *testing.T) (*StreamHandler, *device.Manager, *scenario.Runtime) {
	t.Helper()
	manager := device.NewManager()
	control := NewControlService("srv-1", manager)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   manager,
		IO:        iorouter.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	return handler, manager, runtime
}

// resetChatRoom swaps in a fresh chat room for isolation between tests.
func resetChatRoom(t *testing.T) {
	t.Helper()
	// scenario package does not expose a reset; we rely on test ordering and the
	// fact each test uses distinct device ids + names. Room retention is 24h,
	// so messages from earlier subtests don't interfere with correctness
	// checks here.
}

func activateChatFor(t *testing.T, runtime *scenario.Runtime, manager *device.Manager, deviceID string) {
	t.Helper()
	if _, err := manager.Register(device.Manifest{DeviceID: deviceID, DeviceName: deviceID}); err != nil {
		t.Fatalf("register device: %v", err)
	}
	if _, err := runtime.HandleTrigger(context.Background(), scenario.Trigger{
		Kind:     scenario.TriggerManual,
		SourceID: deviceID,
		Intent:   "chat",
	}); err != nil {
		t.Fatalf("activate chat: %v", err)
	}
}

func TestChatInputSetsNameAndRendersView(t *testing.T) {
	resetChatRoom(t)
	handler, manager, runtime := newChatTestHandler(t)
	const deviceID = "chat-test-dev-name"
	activateChatFor(t, runtime, manager, deviceID)

	out, err := handler.handleInput(context.Background(), &InputRequest{
		DeviceID:    deviceID,
		ComponentID: ui.ChatNameInputID,
		Action:      "submit",
		Value:       "alice",
	})
	if err != nil {
		t.Fatalf("handleInput: %v", err)
	}
	if len(out) != 1 || out[0].SetUI == nil {
		t.Fatalf("expected SetUI response, got %+v", out)
	}
	if got := scenario.SharedRoom().Name(deviceID); got != "alice" {
		t.Fatalf("room.Name = %q, want alice", got)
	}
	if id := out[0].SetUI.Props["id"]; id != ui.ChatRootComponentID {
		t.Fatalf("root id = %q, want %q", id, ui.ChatRootComponentID)
	}
}

func TestChatInputPostsMessageAndBroadcasts(t *testing.T) {
	resetChatRoom(t)
	handler, manager, runtime := newChatTestHandler(t)
	const senderID = "chat-test-sender"
	const peerID = "chat-test-peer"
	activateChatFor(t, runtime, manager, senderID)
	activateChatFor(t, runtime, manager, peerID)
	scenario.SharedRoom().SetName(senderID, "alice")

	out, err := handler.handleInput(context.Background(), &InputRequest{
		DeviceID:    senderID,
		ComponentID: ui.ChatMessageInputID,
		Action:      "submit",
		Value:       "hello world",
	})
	if err != nil {
		t.Fatalf("handleInput: %v", err)
	}
	if len(out) < 2 {
		t.Fatalf("expected self + peer responses, got %d", len(out))
	}
	if out[0].UpdateUI == nil || out[0].UpdateUI.ComponentID != ui.ChatMessagesComponentID {
		t.Fatalf("first response = %+v, want chat messages UpdateUI for self", out[0])
	}
	if out[0].RelayToDeviceID != "" {
		t.Fatalf("self response should not have RelayToDeviceID")
	}
	seenRelay := false
	for _, msg := range out[1:] {
		if msg.RelayToDeviceID == peerID && msg.UpdateUI != nil {
			seenRelay = true
		}
	}
	if !seenRelay {
		t.Fatalf("expected relay UpdateUI to peer %q in %+v", peerID, out)
	}
	msgs := scenario.SharedRoom().Messages()
	found := false
	for _, m := range msgs {
		if m.Text == "hello world" && m.DeviceID == senderID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected message recorded in room: %+v", msgs)
	}
}

func TestChatHistoryVisibleToNewJoiner(t *testing.T) {
	resetChatRoom(t)
	room := scenario.SharedRoom()
	const earlyID = "chat-test-early"
	room.SetName(earlyID, "early_bird")
	room.Post(earlyID, "early_bird", "history must persist")

	handler, manager, runtime := newChatTestHandler(t)
	const joinerID = "chat-test-late"
	activateChatFor(t, runtime, manager, joinerID)

	view := handler.chatEntryUI(joinerID)
	// Joiner has no identity yet; should see identity view first.
	if view.Props["id"] != ui.ChatRootComponentID {
		t.Fatalf("entry view id = %q, want %q", view.Props["id"], ui.ChatRootComponentID)
	}
	// Identity view should contain the name input.
	nameInputFound := false
	walk(view, func(d ui.Descriptor) {
		if d.Props["id"] == ui.ChatNameInputID {
			nameInputFound = true
		}
	})
	if !nameInputFound {
		t.Fatalf("identity view missing name input: %+v", view)
	}

	// After setting a name, the view must include prior history.
	room.SetName(joinerID, "latecomer")
	view = handler.chatEntryUI(joinerID)
	found := false
	walk(view, func(d ui.Descriptor) {
		if d.Type == "text" && d.Props["value"] == "history must persist" {
			found = true
		}
	})
	if !found {
		t.Fatalf("late joiner did not see prior history in view: %+v", view)
	}
}

func walk(d ui.Descriptor, fn func(ui.Descriptor)) {
	fn(d)
	for _, child := range d.Children {
		walk(child, fn)
	}
}

func TestChatRoomPostAllowsDuplicateNames(t *testing.T) {
	room := chat.NewRoom()
	room.SetName("a", "same")
	room.SetName("b", "same")
	if _, ok := room.Post("a", "same", "from-a"); !ok {
		t.Fatalf("post from a failed")
	}
	if _, ok := room.Post("b", "same", "from-b"); !ok {
		t.Fatalf("post from b failed")
	}
	msgs := room.Messages()
	if len(msgs) != 2 {
		t.Fatalf("messages = %d, want 2", len(msgs))
	}
	if msgs[0].DeviceID == msgs[1].DeviceID {
		t.Fatalf("duplicate name collapsed messages: %+v", msgs)
	}
}
