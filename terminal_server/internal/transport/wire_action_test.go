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
