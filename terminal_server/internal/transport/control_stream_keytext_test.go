package transport

import "testing"

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
