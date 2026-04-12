package transport

import "testing"

func TestInternalFromWireClient(t *testing.T) {
	msg, err := InternalFromWireClient(WireClientMessage{
		Command: &WireCommandRequest{
			RequestID: "r1",
			DeviceID:  "d1",
			Action:    CommandActionStart,
			Kind:      CommandKindManual,
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
		Data: map[string]string{
			"b": "2",
			"a": "1",
		},
	})
	if wire.RegisterAck == nil || wire.RegisterAck.ServerID != "srv-1" {
		t.Fatalf("unexpected register ack mapping: %+v", wire.RegisterAck)
	}
	if wire.CommandAck != "cmd-1" {
		t.Fatalf("CommandAck = %q, want cmd-1", wire.CommandAck)
	}
	if len(wire.Data) != 2 || wire.Data[0].Key != "a" || wire.Data[1].Key != "b" {
		t.Fatalf("unexpected data order: %+v", wire.Data)
	}
}

func TestInternalFromWireClientInvalid(t *testing.T) {
	_, err := InternalFromWireClient(WireClientMessage{})
	if err != ErrInvalidWireMessage {
		t.Fatalf("err = %v, want %v", err, ErrInvalidWireMessage)
	}
}
