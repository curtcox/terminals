package transport

import "testing"

func TestInternalActionFromWire(t *testing.T) {
	if got := internalActionFromWire(WireCommandActionStart); got != CommandActionStart {
		t.Fatalf("start mapping = %q, want %q", got, CommandActionStart)
	}
	if got := internalActionFromWire(WireCommandActionStop); got != CommandActionStop {
		t.Fatalf("stop mapping = %q, want %q", got, CommandActionStop)
	}
	if got := internalActionFromWire(WireCommandActionUnspecified); got != "" {
		t.Fatalf("unspecified mapping = %q, want empty", got)
	}
}

func TestInternalKindFromWire(t *testing.T) {
	if got := internalKindFromWire(WireCommandKindVoice); got != CommandKindVoice {
		t.Fatalf("voice mapping = %q, want %q", got, CommandKindVoice)
	}
	if got := internalKindFromWire(WireCommandKindManual); got != CommandKindManual {
		t.Fatalf("manual mapping = %q, want %q", got, CommandKindManual)
	}
	if got := internalKindFromWire(WireCommandKindSystem); got != CommandKindSystem {
		t.Fatalf("system mapping = %q, want %q", got, CommandKindSystem)
	}
	if got := internalKindFromWire(WireCommandKindUnspecified); got != "" {
		t.Fatalf("unspecified mapping = %q, want empty", got)
	}
}

func TestWireErrorCodeFromInternal(t *testing.T) {
	if got := wireErrorCodeFromInternal(ErrorCodeInvalidClientMessage); got != WireControlErrorCodeInvalidClientMessage {
		t.Fatalf("invalid client mapping = %d", got)
	}
	if got := wireErrorCodeFromInternal(ErrorCodeProtocolViolation); got != WireControlErrorCodeProtocolViolation {
		t.Fatalf("protocol violation mapping = %d", got)
	}
	if got := wireErrorCodeFromInternal("something_else"); got != WireControlErrorCodeUnknown {
		t.Fatalf("unknown mapping = %d, want unknown", got)
	}
}
