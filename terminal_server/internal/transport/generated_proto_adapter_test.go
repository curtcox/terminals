package transport

import (
	"testing"

	capabilitiesv1 "github.com/curtcox/terminals/terminal_server/gen/go/capabilities/v1"
	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	iov1 "github.com/curtcox/terminals/terminal_server/gen/go/io/v1"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestGeneratedProtoAdapterToInternalRegister(t *testing.T) {
	adapter := GeneratedProtoAdapter{}
	msg, err := adapter.ToInternal(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Register{
			Register: &controlv1.RegisterDevice{
				Capabilities: &capabilitiesv1.DeviceCapabilities{
					DeviceId: "device-1",
					Identity: &capabilitiesv1.DeviceIdentity{
						DeviceName: "Kitchen Display",
						DeviceType: "tablet",
						Platform:   "android",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ToInternal() error = %v", err)
	}
	if msg.Register == nil {
		t.Fatalf("expected register message")
	}
	if msg.Register.DeviceID != "device-1" {
		t.Fatalf("device_id = %q, want %q", msg.Register.DeviceID, "device-1")
	}
	if msg.Register.DeviceName != "Kitchen Display" {
		t.Fatalf("device_name = %q, want %q", msg.Register.DeviceName, "Kitchen Display")
	}
	if msg.Register.Capabilities["platform"] != "android" {
		t.Fatalf("platform capability = %q, want %q", msg.Register.Capabilities["platform"], "android")
	}
}

func TestGeneratedProtoAdapterToInternalInput(t *testing.T) {
	adapter := GeneratedProtoAdapter{}
	msg, err := adapter.ToInternal(&controlv1.ConnectRequest{
		Payload: &controlv1.ConnectRequest_Input{
			Input: &iov1.InputEvent{
				DeviceId: "device-2",
				Payload: &iov1.InputEvent_UiAction{
					UiAction: &iov1.UIAction{
						ComponentId: "terminal_input",
						Action:      "submit",
						Value:       "echo hello",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ToInternal() error = %v", err)
	}
	if msg.Input == nil {
		t.Fatalf("expected input message")
	}
	if msg.Input.DeviceID != "device-2" {
		t.Fatalf("input device_id = %q, want device-2", msg.Input.DeviceID)
	}
	if msg.Input.ComponentID != "terminal_input" || msg.Input.Action != "submit" {
		t.Fatalf("unexpected input mapping: %+v", msg.Input)
	}
	if msg.Input.Value != "echo hello" {
		t.Fatalf("input value = %q, want echo hello", msg.Input.Value)
	}
}

func TestGeneratedProtoAdapterFromInternal(t *testing.T) {
	adapter := GeneratedProtoAdapter{}

	envelope, err := adapter.FromInternal(ServerMessage{
		CommandAck:    "req-1",
		ScenarioStart: "photo_frame",
		Data: map[string]string{
			"a": "1",
			"b": "2",
		},
	})
	if err != nil {
		t.Fatalf("FromInternal() error = %v", err)
	}

	resp, ok := envelope.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("response envelope type = %T, want *controlv1.ConnectResponse", envelope)
	}
	result := resp.GetCommandResult()
	if result == nil {
		t.Fatalf("expected command_result payload")
	}
	if result.GetRequestId() != "req-1" {
		t.Fatalf("request_id = %q, want %q", result.GetRequestId(), "req-1")
	}
	if result.GetData()["a"] != "1" || result.GetData()["b"] != "2" {
		t.Fatalf("unexpected data map: %+v", result.GetData())
	}

	envelope, err = adapter.FromInternal(ServerMessage{
		SetUI: &ui.Descriptor{
			Type: "stack",
			Children: []ui.Descriptor{
				{
					Type: "text",
					Props: map[string]string{
						"value": "hello",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("FromInternal() set_ui error = %v", err)
	}
	resp, ok = envelope.(*controlv1.ConnectResponse)
	if !ok {
		t.Fatalf("set_ui envelope type = %T, want *controlv1.ConnectResponse", envelope)
	}
	if resp.GetSetUi() == nil || resp.GetSetUi().GetRoot() == nil {
		t.Fatalf("expected set_ui root payload")
	}
	if resp.GetSetUi().GetRoot().GetText() != nil {
		t.Fatalf("stack root should not be text widget")
	}
	if len(resp.GetSetUi().GetRoot().GetChildren()) != 1 {
		t.Fatalf("children count = %d, want 1", len(resp.GetSetUi().GetRoot().GetChildren()))
	}
	if got := resp.GetSetUi().GetRoot().GetChildren()[0].GetText().GetValue(); got != "hello" {
		t.Fatalf("text value = %q, want %q", got, "hello")
	}
}
