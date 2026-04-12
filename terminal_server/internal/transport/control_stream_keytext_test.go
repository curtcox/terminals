package transport

import (
	"testing"
	"time"
)

func TestNormalizeTerminalKeyText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "no-control", in: "abc", want: "abc"},
		{name: "single-backspace", in: "a\b", want: "a\x7f"},
		{name: "multiple-backspace", in: "ab\b\bc", want: "ab\x7f\x7fc"},
		{name: "mixed-with-newline", in: "hello\b\n", want: "hello\x7f\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeTerminalKeyText(tc.in)
			if got != tc.want {
				t.Fatalf("normalizeTerminalKeyText(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestTerminalReadSettingsDefaultsAndBounds(t *testing.T) {
	handler := &StreamHandler{}

	deadline, interval := handler.terminalReadSettings()
	if deadline != defaultTerminalReadDeadline {
		t.Fatalf("default deadline = %v, want %v", deadline, defaultTerminalReadDeadline)
	}
	if interval != defaultTerminalReadInterval {
		t.Fatalf("default interval = %v, want %v", interval, defaultTerminalReadInterval)
	}

	handler.terminalReadDeadline = 40 * time.Millisecond
	handler.terminalReadInterval = 100 * time.Millisecond
	deadline, interval = handler.terminalReadSettings()
	if deadline != 40*time.Millisecond {
		t.Fatalf("custom deadline = %v, want 40ms", deadline)
	}
	if interval != 40*time.Millisecond {
		t.Fatalf("bounded interval = %v, want 40ms", interval)
	}
}
