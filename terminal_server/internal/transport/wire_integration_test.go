package transport

import (
	"context"
	"strings"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestWireSessionProtocolViolationRecoverable(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			WireClientMessage{Heartbeat: &WireHeartbeatRequest{DeviceID: "device-1"}},
			WireClientMessage{Register: &WireRegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"}},
			WireClientMessage{Heartbeat: &WireHeartbeatRequest{DeviceID: "device-1"}},
		},
	}

	if err := RunProtoSession(handler, control, stream, WireProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}
	if len(stream.sent) < 3 {
		t.Fatalf("len(sent) = %d, want at least 3", len(stream.sent))
	}

	first, ok := stream.sent[0].(WireServerMessage)
	if !ok {
		t.Fatalf("first response type = %T, want WireServerMessage", stream.sent[0])
	}
	if first.ErrorCode != ErrorCodeProtocolViolation {
		t.Fatalf("ErrorCode = %q, want %q", first.ErrorCode, ErrorCodeProtocolViolation)
	}
	if !strings.Contains(first.Error, "register required") {
		t.Fatalf("Error = %q, expected register-required text", first.Error)
	}
}

func TestWireSessionCommandValidationErrorCode(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	handler := NewStreamHandlerWithRuntime(control, runtime)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			WireClientMessage{Register: &WireRegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"}},
			WireClientMessage{Command: &WireCommandRequest{
				RequestID: "bad-manual",
				DeviceID:  "device-1",
				Kind:      CommandKindManual,
				Intent:    "   ",
			}},
		},
	}

	if err := RunProtoSession(handler, control, stream, WireProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}
	if len(stream.sent) < 3 {
		t.Fatalf("len(sent) = %d, want at least 3", len(stream.sent))
	}
	last, ok := stream.sent[len(stream.sent)-1].(WireServerMessage)
	if !ok {
		t.Fatalf("last response type = %T, want WireServerMessage", stream.sent[len(stream.sent)-1])
	}
	if last.ErrorCode != ErrorCodeMissingIntent {
		t.Fatalf("ErrorCode = %q, want %q", last.ErrorCode, ErrorCodeMissingIntent)
	}
}

func TestWireSessionSystemDataDeterministicOrder(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			WireClientMessage{Register: &WireRegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen"}},
			WireClientMessage{Command: &WireCommandRequest{
				RequestID: "sys-help",
				Kind:      CommandKindSystem,
				Intent:    SystemIntentHelp,
			}},
		},
	}

	if err := RunProtoSession(handler, control, stream, WireProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}
	helpMsg, ok := stream.sent[len(stream.sent)-1].(WireServerMessage)
	if !ok {
		t.Fatalf("help response type = %T, want WireServerMessage", stream.sent[len(stream.sent)-1])
	}
	if len(helpMsg.Data) < 2 {
		t.Fatalf("expected help data entries, got %+v", helpMsg.Data)
	}
	for i := 1; i < len(helpMsg.Data); i++ {
		if helpMsg.Data[i-1].Key > helpMsg.Data[i].Key {
			t.Fatalf("data entries not sorted: %+v", helpMsg.Data)
		}
	}
}
