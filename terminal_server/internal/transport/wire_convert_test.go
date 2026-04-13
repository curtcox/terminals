package transport

import (
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestInternalFromWireClient(t *testing.T) {
	msg, err := InternalFromWireClient(WireClientMessage{
		Command: &WireCommandRequest{
			RequestID: "r1",
			DeviceID:  "d1",
			Action:    WireCommandActionStart,
			Kind:      WireCommandKindManual,
			Intent:    "photo frame",
		},
	})
	if err != nil {
		t.Fatalf("InternalFromWireClient() error = %v", err)
	}
	if msg.Command == nil || msg.Command.RequestID != "r1" {
		t.Fatalf("unexpected command mapping: %+v", msg.Command)
	}
}

func TestWireFromInternalServer(t *testing.T) {
	wire := WireFromInternalServer(ServerMessage{
		RegisterAck: &RegisterResponse{ServerID: "srv-1", Message: "ok"},
		CommandAck:  "cmd-1",
		UpdateUI: &UIUpdate{
			ComponentID: "terminal_output",
			Node: ui.Descriptor{
				Type: "text",
				Props: map[string]string{"value": "patched"},
			},
		},
		RouteStream: &RouteStreamResponse{
			StreamID:       "route:d1|d2|audio",
			SourceDeviceID: "d1",
			TargetDeviceID: "d2",
			Kind:           "audio",
		},
		TransitionUI: &UITransition{
			Transition: "fade",
			DurationMS: 150,
		},
		ErrorCode:   ErrorCodeInvalidCommandAction,
		Error:       "invalid command action",
		Data: map[string]string{
			"b": "2",
			"a": "1",
		},
	})
	if wire.RegisterAck == nil || wire.RegisterAck.ServerID != "srv-1" {
		t.Fatalf("unexpected register ack mapping: %+v", wire.RegisterAck)
	}
	if wire.CommandResult == nil || wire.CommandResult.RequestID != "cmd-1" {
		t.Fatalf("unexpected command result mapping: %+v", wire.CommandResult)
	}
	if wire.Error == nil || wire.Error.Code != WireControlErrorCodeInvalidCommandAction || wire.Error.Message == "" {
		t.Fatalf("unexpected error fields: %+v", wire.Error)
	}
	if wire.UpdateUI == nil || wire.UpdateUI.ComponentID != "terminal_output" {
		t.Fatalf("unexpected update_ui mapping: %+v", wire.UpdateUI)
	}
	if wire.UpdateUI.Node.Type != "text" || DecodeDataEntries(wire.UpdateUI.Node.Props)["value"] != "patched" {
		t.Fatalf("unexpected update_ui node mapping: %+v", wire.UpdateUI.Node)
	}
	if wire.RouteStream == nil || wire.RouteStream.StreamID != "route:d1|d2|audio" {
		t.Fatalf("unexpected route_stream mapping: %+v", wire.RouteStream)
	}
	if wire.RouteStream.SourceDeviceID != "d1" || wire.RouteStream.TargetDeviceID != "d2" || wire.RouteStream.Kind != "audio" {
		t.Fatalf("unexpected route_stream fields: %+v", wire.RouteStream)
	}
	if wire.TransitionUI == nil || wire.TransitionUI.Transition != "fade" || wire.TransitionUI.DurationMS != 150 {
		t.Fatalf("unexpected transition_ui mapping: %+v", wire.TransitionUI)
	}
	if len(wire.CommandResult.Data) != 2 || wire.CommandResult.Data[0].Key != "a" || wire.CommandResult.Data[1].Key != "b" {
		t.Fatalf("unexpected data order: %+v", wire.CommandResult.Data)
	}
}

func TestInternalFromWireClientInvalid(t *testing.T) {
	_, err := InternalFromWireClient(WireClientMessage{})
	if err != ErrInvalidWireMessage {
		t.Fatalf("err = %v, want %v", err, ErrInvalidWireMessage)
	}
}
