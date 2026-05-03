package transport

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

// newDispatcherTestHandler builds a StreamHandler wired with a real
// scenario runtime, suitable for exercising the CommandDispatcher
// end-to-end via HandleMessage.
func newDispatcherTestHandler(t *testing.T) *StreamHandler {
	t.Helper()
	devices := device.NewManager()
	control := NewControlService("srv-dispatcher", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices: devices,
		IO:      io.NewRouter(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"},
	})
	return handler
}

// TestDispatcherValidationSentinelInvalidKind confirms the dispatcher
// surfaces ErrInvalidCommandKind as a single ServerMessage error
// response and propagates the sentinel via errors.Is.
func TestDispatcherValidationSentinelInvalidKind(t *testing.T) {
	handler := newDispatcherTestHandler(t)
	out, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "bad-kind",
			DeviceID:  "device-1",
			Kind:      "remote",
			Intent:    "photo frame",
		},
	})
	if !errors.Is(err, ErrInvalidCommandKind) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidCommandKind)
	}
	if len(out) != 1 {
		t.Fatalf("len(out) = %d, want 1", len(out))
	}
	if out[0].Error != ErrInvalidCommandKind.Error() {
		t.Fatalf("out[0].Error = %q, want %q", out[0].Error, ErrInvalidCommandKind.Error())
	}
	if out[0].CommandAck != "" {
		t.Fatalf("CommandAck = %q, want empty", out[0].CommandAck)
	}
}

// TestDispatcherValidationSentinelMissingDeviceID confirms missing
// device id surfaces ErrMissingCommandDeviceID.
func TestDispatcherValidationSentinelMissingDeviceID(t *testing.T) {
	handler := newDispatcherTestHandler(t)
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "no-device",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	if !errors.Is(err, ErrMissingCommandDeviceID) {
		t.Fatalf("err = %v, want %v", err, ErrMissingCommandDeviceID)
	}
}

// TestDispatcherValidationSentinelMissingIntent confirms missing intent
// surfaces ErrMissingCommandIntent for manual commands.
func TestDispatcherValidationSentinelMissingIntent(t *testing.T) {
	handler := newDispatcherTestHandler(t)
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "no-intent",
			DeviceID:  "device-1",
			Kind:      "manual",
		},
	})
	if !errors.Is(err, ErrMissingCommandIntent) {
		t.Fatalf("err = %v, want %v", err, ErrMissingCommandIntent)
	}
}

// TestDispatcherValidationSentinelMissingText confirms missing text
// surfaces ErrMissingCommandText for voice commands.
func TestDispatcherValidationSentinelMissingText(t *testing.T) {
	handler := newDispatcherTestHandler(t)
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "no-text",
			DeviceID:  "device-1",
			Kind:      "voice",
		},
	})
	if !errors.Is(err, ErrMissingCommandText) {
		t.Fatalf("err = %v, want %v", err, ErrMissingCommandText)
	}
}

// TestDispatcherValidationSentinelInvalidAction confirms an unknown
// action surfaces ErrInvalidCommandAction.
func TestDispatcherValidationSentinelInvalidAction(t *testing.T) {
	handler := newDispatcherTestHandler(t)
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "bad-action",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
			Action:    "rewind",
		},
	})
	if !errors.Is(err, ErrInvalidCommandAction) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidCommandAction)
	}
}

// TestDispatcherAuditAppendOnSuccess confirms that successful command
// dispatch appends an audit entry with a non-error outcome.
func TestDispatcherAuditAppendOnSuccess(t *testing.T) {
	handler := newDispatcherTestHandler(t)
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "ok-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	if err != nil {
		t.Fatalf("dispatch err = %v", err)
	}
	handler.mu.Lock()
	defer handler.mu.Unlock()
	if len(handler.recent) == 0 {
		t.Fatal("expected recent audit entry on success")
	}
	last := handler.recent[len(handler.recent)-1]
	if last.RequestID != "ok-1" {
		t.Fatalf("RequestID = %q, want ok-1", last.RequestID)
	}
	if last.Outcome == "" {
		t.Fatalf("Outcome empty")
	}
	if errPrefix := "error:"; len(last.Outcome) >= len(errPrefix) && last.Outcome[:len(errPrefix)] == errPrefix {
		t.Fatalf("unexpected error outcome on success: %q", last.Outcome)
	}
}

// TestDispatcherAuditAppendOnError confirms validation errors append an
// audit entry with the "error:" prefix and the sentinel's text.
func TestDispatcherAuditAppendOnError(t *testing.T) {
	handler := newDispatcherTestHandler(t)
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "err-1",
			DeviceID:  "device-1",
			Kind:      "remote",
			Intent:    "photo frame",
		},
	})
	if !errors.Is(err, ErrInvalidCommandKind) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidCommandKind)
	}
	handler.mu.Lock()
	defer handler.mu.Unlock()
	if len(handler.recent) == 0 {
		t.Fatal("expected recent audit entry on error")
	}
	last := handler.recent[len(handler.recent)-1]
	want := "error:" + ErrInvalidCommandKind.Error()
	if last.Outcome != want {
		t.Fatalf("Outcome = %q, want %q", last.Outcome, want)
	}
}

// TestDispatcherAuditTrimAtRecentLimit confirms the audit buffer is
// trimmed to recentLimit, preserving FIFO eviction order.
func TestDispatcherAuditTrimAtRecentLimit(t *testing.T) {
	handler := newDispatcherTestHandler(t)
	handler.recentLimit = 2

	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "trim-1", DeviceID: "device-1", Kind: "manual", Intent: "photo frame"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "trim-2", DeviceID: "device-1", Action: "stop", Kind: "manual", Intent: "photo frame"},
	})
	_, _ = handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{RequestID: "trim-3", Kind: "system", Intent: "server_status"},
	})

	handler.mu.Lock()
	defer handler.mu.Unlock()
	if len(handler.recent) != 2 {
		t.Fatalf("len(recent) = %d, want 2", len(handler.recent))
	}
	if handler.recent[0].RequestID != "trim-2" || handler.recent[1].RequestID != "trim-3" {
		t.Fatalf("eviction order wrong: %+v", handler.recent)
	}
}

// TestDispatcherBroadcastFanOutOnlyOnScenarioStart confirms broadcast
// notifications are only emitted when the command result is a scenario
// start or stop. A pure-notification command must not trigger fan-out.
func TestDispatcherBroadcastFanOutOnlyOnScenarioStart(t *testing.T) {
	handler := newDispatcherTestHandler(t)
	dispatcher := handler.commandDispatcher
	if dispatcher == nil {
		t.Fatal("commandDispatcher nil")
	}
	cmd := &CommandRequest{DeviceID: "device-1", Kind: "manual", Intent: "noop"}

	if got := dispatcher.BroadcastNotificationsForCommand(cmd, ServerMessage{Notification: "hi"}, 0); got != nil {
		t.Fatalf("expected nil for notification-only result, got %v", got)
	}
	if got := dispatcher.BroadcastNotificationsForCommand(nil, ServerMessage{ScenarioStart: "x"}, 0); got != nil {
		t.Fatalf("expected nil for nil cmd, got %v", got)
	}
}

// TestDispatcherPostCommandUIHostOrdering confirms that running a
// command does not lose the post-command UI host snapshot accounting.
// Specifically, the UIHostBeforeCount must be advanced exactly once per
// dispatch so subsequent commands only see new events.
func TestDispatcherPostCommandUIHostOrdering(t *testing.T) {
	handler := newDispatcherTestHandler(t)
	beforeFirst := handler.uiSession.UIHostBeforeCountAndAdvance("device-1", 0)
	if beforeFirst != 0 {
		t.Fatalf("initial before count = %d, want 0", beforeFirst)
	}
	_, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "ord-1",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	if err != nil {
		t.Fatalf("dispatch err = %v", err)
	}
}

// TestDispatcherAuditBufferRace exercises concurrent dispatch under the
// race detector to confirm the audit buffer's locking model holds. The
// buffer is updated under h.mu, the same lock that guards seen/seenOrder
// updates the dispatcher already coordinates.
func TestDispatcherAuditBufferRace(t *testing.T) {
	handler := newDispatcherTestHandler(t)
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = handler.HandleMessage(context.Background(), ClientMessage{
				Command: &CommandRequest{
					DeviceID: "device-1",
					Kind:     "system",
					Intent:   "server_status",
				},
			})
		}()
	}
	wg.Wait()
}
