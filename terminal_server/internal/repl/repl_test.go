package repl

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunEchoHelpAndExit(t *testing.T) {
	in := strings.NewReader("help\necho hi repl\nexit\n")
	var out bytes.Buffer

	err := Run(context.Background(), in, &out, Options{Prompt: "repl>"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "control-plane only") {
		t.Fatalf("missing banner: %q", text)
	}
	if !strings.Contains(text, "echo <text>") {
		t.Fatalf("missing help output: %q", text)
	}
	if !strings.Contains(text, "hi repl") {
		t.Fatalf("missing echo output: %q", text)
	}
	if !strings.Contains(text, "bye") {
		t.Fatalf("missing exit output: %q", text)
	}
}

func TestRunSemicolonSleepPrintf(t *testing.T) {
	in := strings.NewReader("sleep 0; printf '\\x6f\\x6b\\n'\nexit\n")
	var out bytes.Buffer

	err := Run(context.Background(), in, &out, Options{Prompt: "repl>"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "ok") {
		t.Fatalf("expected printf escape output, got %q", out.String())
	}
}
