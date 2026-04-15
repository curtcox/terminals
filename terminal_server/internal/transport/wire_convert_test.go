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
	if len(msg.Command.Arguments) != 0 {
		t.Fatalf("expected empty command arguments by default, got %+v", msg.Command.Arguments)
	}

	msg, err = InternalFromWireClient(WireClientMessage{
		Command: &WireCommandRequest{
			RequestID: "r2",
			DeviceID:  "d1",
			Action:    WireCommandActionStart,
			Kind:      WireCommandKindManual,
			Intent:    "photo frame",
			Arguments: []DataEntry{
				{Key: "device_ids", Value: "d1,d2"},
			},
		},
	})
	if err != nil {
		t.Fatalf("InternalFromWireClient(command arguments) error = %v", err)
	}
	if got := msg.Command.Arguments["device_ids"]; got != "d1,d2" {
		t.Fatalf("device_ids argument = %q, want d1,d2", got)
	}

	msg, err = InternalFromWireClient(WireClientMessage{
		WebRTCSignal: &WireWebRTCSignal{
			StreamID:   "stream-1",
			SignalType: "candidate",
			Payload:    "{\"candidate\":\"abc\"}",
		},
	})
	if err != nil {
		t.Fatalf("InternalFromWireClient(webrtc_signal) error = %v", err)
	}
	if msg.WebRTCSignal == nil {
		t.Fatalf("expected webrtc signal mapping")
	}
	if msg.WebRTCSignal.StreamID != "stream-1" || msg.WebRTCSignal.SignalType != "candidate" {
		t.Fatalf("unexpected webrtc signal mapping: %+v", msg.WebRTCSignal)
	}
	if msg.WebRTCSignal.Payload != "{\"candidate\":\"abc\"}" {
		t.Fatalf("unexpected webrtc payload: %q", msg.WebRTCSignal.Payload)
	}
}

func TestWireFromInternalServer(t *testing.T) {
	wire := WireFromInternalServer(ServerMessage{
		RegisterAck: &RegisterResponse{
			ServerID: "srv-1",
			Message:  "ok",
			Metadata: map[string]string{"photo_frame_asset_base_url": "http://home.local:50052/photo-frame"},
		},
		CommandAck: "cmd-1",
		UpdateUI: &UIUpdate{
			ComponentID: "terminal_output",
			Node: ui.Descriptor{
				Type:  "text",
				Props: map[string]string{"value": "patched"},
			},
		},
		StartStream: &StartStreamResponse{
			StreamID:       "stream-1",
			Kind:           "audio",
			SourceDeviceID: "d1",
			TargetDeviceID: "d2",
			Metadata: map[string]string{
				"codec": "opus",
			},
		},
		StopStream: &StopStreamResponse{
			StreamID: "stream-2",
		},
		RouteStream: &RouteStreamResponse{
			StreamID:       "route:d1|d2|audio",
			SourceDeviceID: "d1",
			TargetDeviceID: "d2",
			Kind:           "audio",
		},
		WebRTCSignal: &WebRTCSignalResponse{
			StreamID:   "stream-1",
			SignalType: "offer",
			Payload:    "{\"sdp\":\"v=0...\"}",
		},
		TransitionUI: &UITransition{
			Transition: "fade",
			DurationMS: 150,
		},
		ErrorCode: ErrorCodeInvalidCommandAction,
		Error:     "invalid command action",
		Data: map[string]string{
			"b": "2",
			"a": "1",
		},
	})
	if wire.RegisterAck == nil || wire.RegisterAck.ServerID != "srv-1" {
		t.Fatalf("unexpected register ack mapping: %+v", wire.RegisterAck)
	}
	if got := DecodeDataEntries(wire.RegisterAck.Metadata)["photo_frame_asset_base_url"]; got != "http://home.local:50052/photo-frame" {
		t.Fatalf("unexpected register ack metadata mapping: %+v", wire.RegisterAck.Metadata)
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
	if wire.StartStream == nil || wire.StartStream.StreamID != "stream-1" || wire.StartStream.Kind != "audio" {
		t.Fatalf("unexpected start_stream mapping: %+v", wire.StartStream)
	}
	if DecodeDataEntries(wire.StartStream.Metadata)["codec"] != "opus" {
		t.Fatalf("unexpected start_stream metadata: %+v", wire.StartStream.Metadata)
	}
	if wire.StopStream == nil || wire.StopStream.StreamID != "stream-2" {
		t.Fatalf("unexpected stop_stream mapping: %+v", wire.StopStream)
	}
	if wire.RouteStream == nil || wire.RouteStream.StreamID != "route:d1|d2|audio" {
		t.Fatalf("unexpected route_stream mapping: %+v", wire.RouteStream)
	}
	if wire.RouteStream.SourceDeviceID != "d1" || wire.RouteStream.TargetDeviceID != "d2" || wire.RouteStream.Kind != "audio" {
		t.Fatalf("unexpected route_stream fields: %+v", wire.RouteStream)
	}
	if wire.WebRTCSignal == nil || wire.WebRTCSignal.StreamID != "stream-1" || wire.WebRTCSignal.SignalType != "offer" {
		t.Fatalf("unexpected webrtc signal mapping: %+v", wire.WebRTCSignal)
	}
	if wire.WebRTCSignal.Payload != "{\"sdp\":\"v=0...\"}" {
		t.Fatalf("unexpected webrtc payload mapping: %q", wire.WebRTCSignal.Payload)
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
