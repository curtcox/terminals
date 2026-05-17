package transport

import (
	"context"
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

// TestGeneratedSessionStandbyModeActivatedByVoiceCommand validates D3:
// a voice command activates standby/clock mode and the server notifies the
// requesting device.
func TestGeneratedSessionStandbyModeActivatedByVoiceCommand(t *testing.T) {
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
							DeviceId: "tablet-bedroom",
							Identity: &capabilitiesv1.DeviceIdentity{DeviceName: "Bedroom Tablet"},
						},
					},
				},
			},
			&controlv1.ConnectRequest{
				Payload: &controlv1.ConnectRequest_Command{
					Command: &controlv1.CommandRequest{
						RequestId: "standby-cmd",
						DeviceId:  "tablet-bedroom",
						Kind:      controlv1.CommandKind_COMMAND_KIND_MANUAL,
						Intent:    "standby mode",
					},
				},
			},
		},
	}

	if err := RunProtoSession(handler, control, stream, GeneratedProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var scenarioStarted bool
	for _, msg := range stream.sent {
		resp, ok := msg.(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if resp.GetCommandResult() != nil && resp.GetCommandResult().GetScenarioStart() == "standby" {
			scenarioStarted = true
		}
	}
	if !scenarioStarted {
		t.Fatalf("expected standby scenario start in responses; got %d messages: %+v", len(stream.sent), stream.sent)
	}
}

// TestWireSessionStandbyModeActivatedByVoiceCommand validates D3 over the wire
// adapter: a voice intent activates standby mode and the server confirms.
func TestWireSessionStandbyModeActivatedByVoiceCommand(t *testing.T) {
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
			WireClientMessage{Register: &WireRegisterRequest{
				DeviceID:   "tablet-hall",
				DeviceName: "Hall Tablet",
			}},
			WireClientMessage{Command: &WireCommandRequest{
				RequestID: "standby-wire",
				DeviceID:  "tablet-hall",
				Kind:      WireCommandKindManual,
				Intent:    "standby",
			}},
		},
	}

	if err := RunProtoSession(handler, control, stream, WireProtoAdapter{}); err != nil {
		t.Fatalf("RunProtoSession() error = %v", err)
	}

	var scenarioStarted bool
	for _, msg := range stream.sent {
		wire, ok := msg.(WireServerMessage)
		if !ok {
			continue
		}
		if wire.CommandResult != nil && wire.CommandResult.ScenarioStart == "standby" {
			scenarioStarted = true
		}
	}
	if !scenarioStarted {
		t.Fatalf("expected standby scenario start in wire responses; got %d messages: %+v", len(stream.sent), stream.sent)
	}
}
