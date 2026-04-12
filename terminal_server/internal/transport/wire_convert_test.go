package transport

import "testing"

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
	if wire.Error == nil || wire.Error.Code != ErrorCodeInvalidCommandAction || wire.Error.Message == "" {
		t.Fatalf("unexpected error fields: %+v", wire.Error)
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
