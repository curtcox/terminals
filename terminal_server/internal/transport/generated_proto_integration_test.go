package transport

import (
	"context"
	"strings"
	"testing"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestGeneratedSessionProtocolViolationRecoverable(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Heartbeat{
					Heartbeat: &controlv1.Heartbeat{DeviceId: "device-1"},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Heartbeat{
					Heartbeat: &controlv1.Heartbeat{DeviceId: "device-1"},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}
	if len(stream.sent) < 3 {
		t.Fatalf("len(sent) = %d, want at least 3", len(stream.sent))
	}

	first, ok := stream.sent[0].(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("first response type = %T, want *controlv1.ConnectResponse", stream.sent[0])
	}
	if first.GetError() == nil || first.GetError().GetCode() != controlv1.ControlErrorCode_CONTROL_ERROR_CODE_PROTOCOL_VIOLATION {
		t.Fatalf("error code = %+v, want protocol violation", first.GetError())
	}
	if !strings.Contains(first.GetError().GetMessage(), "register required") {
		t.Fatalf("error message = %q, expected register-required text", first.GetError().GetMessage())
	}
}

func TestGeneratedSessionCommandValidationErrorCode(t *testing.T) {
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
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "bad-manual",
						DeviceId:  "device-1",
						Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
						Intent:    "   ",
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}
	if len(stream.sent) < 3 {
		t.Fatalf("len(sent) = %d, want at least 3", len(stream.sent))
	}
	last, ok := stream.sent[len(stream.sent)-1].(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("last response type = %T, want *controlv1.ConnectResponse", stream.sent[len(stream.sent)-1])
	}
	if last.GetError() == nil || last.GetError().GetCode() != controlv1.ControlErrorCode_CONTROL_ERROR_CODE_MISSING_COMMAND_INTENT {
		t.Fatalf("error code = %+v, want missing_command_intent", last.GetError())
	}
}

func TestGeneratedSessionSystemDataPayload(t *testing.T) {
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	handler := NewStreamHandler(control)
	stream := &fakeProtoStream{
		ctx: context.Background(),
		recvQueue: []ProtoClientEnvelope{
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Register{
					Register: &controlv1.RegisterDevice{
						Capabilities: &capabilitiesv1.DeviceCapabilities{
							DeviceId: "device-1",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Kitchen"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "sys-help",
						Kind:      controlv1.CommandKind_COMMAND_KIND_SYSTEM,
						Intent:    SystemIntentHelp,
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}
	last, ok := stream.sent[len(stream.sent)-1].(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("help response type = %T, want *controlv1.ConnectResponse", stream.sent[len(stream.sent)-1])
	}
	if last.GetCommandResult() == nil {
		t.Fatalf("expected command result payload")
	}
	data := last.GetCommandResult().GetData()
	if data["system_intents"] == "" {
		t.Fatalf("system_intents entry missing from system help payload: %+v", data)
	}
	if data["command_kinds"] == "" || data["command_actions"] == "" {
		t.Fatalf("command metadata missing from system help payload: %+v", data)
	}
}
