package repl

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestDescribeAndComplete(t *testing.T) {
	in := strings.NewReader("describe app reload\ncomplete app r\nexit\n")
	var out bytes.Buffer

	err := Run(context.Background(), in, &out, Options{Prompt: "repl>"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "app reload <app> [--json]") {
		t.Fatalf("missing describe usage output: %q", text)
	}
	if !strings.Contains(text, "classification: mutating") {
		t.Fatalf("missing mutating classification in describe output: %q", text)
	}
	if !strings.Contains(text, "app reload") {
		t.Fatalf("missing completion match: %q", text)
	}
}

func TestCommandSpecsExposeOperationalAndDiscouragedFlags(t *testing.T) {
	specs := CommandSpecs()
	if len(specs) == 0 {
		t.Fatalf("CommandSpecs() returned no commands")
	}
	sleep, ok := DescribeCommand("sleep")
	if !ok {
		t.Fatalf("DescribeCommand(sleep) not found")
	}
	if sleep.Classification != CommandClassificationOperational {
		t.Fatalf("sleep classification = %q, want %q", sleep.Classification, CommandClassificationOperational)
	}
	aiUse, ok := DescribeCommand("ai use")
	if !ok {
		t.Fatalf("DescribeCommand(ai use) not found")
	}
	if !aiUse.DiscouragedForAgents {
		t.Fatalf("ai use should be discouraged_for_agents")
	}
}

func TestExecuteCommandDocsMarkdownMode(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	tempWD := t.TempDir()
	if chdirErr := os.Chdir(tempWD); chdirErr != nil {
		t.Fatalf("Chdir(%q) error = %v", tempWD, chdirErr)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	result, err := ExecuteCommand(context.Background(), "docs search app", ExecuteOptions{
		DocsMode: DocsRenderModeMarkdown,
	})
	if err != nil {
		t.Fatalf("ExecuteCommand(docs search app) error = %v", err)
	}
	if strings.Contains(result.Output, "search results for") {
		t.Fatalf("markdown docs search should omit terminal preamble, got %q", result.Output)
	}
	if !strings.Contains(result.Output, "- `repl/commands/app`") {
		t.Fatalf("markdown docs search missing topic bullet, got %q", result.Output)
	}
}

func TestCompletePrefixFiltering(t *testing.T) {
	logMatches := Complete("l", 20)
	if len(logMatches) == 0 || logMatches[0] != "logs tail" {
		t.Fatalf("l completions = %#v, want logs-prefixed command(s)", logMatches)
	}

	matches := Complete("sessions s", 20)
	if len(matches) == 0 {
		t.Fatalf("expected completions for sessions s")
	}
	for _, match := range matches {
		if !strings.HasPrefix(match, "sessions s") {
			t.Fatalf("completion %q does not match sessions s prefix", match)
		}
	}

	matches = Complete("docs o", 20)
	if len(matches) != 1 || matches[0] != "docs open" {
		t.Fatalf("docs o completions = %#v, want [docs open]", matches)
	}

	matches = Complete("sessions ", 20)
	if len(matches) == 0 {
		t.Fatalf("expected subcommands for sessions prefix with trailing space")
	}
	for _, match := range matches {
		if !strings.HasPrefix(match, "sessions ") {
			t.Fatalf("completion %q does not remain under sessions namespace", match)
		}
	}
}

func TestDescribeOperationalLogCommands(t *testing.T) {
	logsTail, ok := DescribeCommand("logs tail")
	if !ok {
		t.Fatalf("DescribeCommand(logs tail) not found")
	}
	if logsTail.Classification != CommandClassificationOperational {
		t.Fatalf("logs tail classification = %q, want %q", logsTail.Classification, CommandClassificationOperational)
	}

	observeTail, ok := DescribeCommand("observe tail")
	if !ok {
		t.Fatalf("DescribeCommand(observe tail) not found")
	}
	if observeTail.Classification != CommandClassificationOperational {
		t.Fatalf("observe tail classification = %q, want %q", observeTail.Classification, CommandClassificationOperational)
	}
}

func TestMutatingCommandsUseAdminAPIs(t *testing.T) {
	admin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodDelete && req.URL.Path == "/admin/api/repl/sessions/repl-9":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","session_id":"repl-9"}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/apps/reload":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","version":"1.2.3"}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/apps/rollback":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","version":"1.2.2"}`))
		default:
			http.NotFound(w, req)
		}
	}))
	defer admin.Close()

	in := strings.NewReader("sessions terminate repl-9\napp reload sound_watch\napp rollback sound_watch\nexit\n")
	var out bytes.Buffer

	err := Run(context.Background(), in, &out, Options{
		Prompt:       "repl>",
		AdminBaseURL: admin.URL,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "OK  terminated session repl-9") {
		t.Fatalf("missing terminate success output: %q", text)
	}
	if !strings.Contains(text, "OK  app=sound_watch action=reload version=1.2.3") {
		t.Fatalf("missing reload success output: %q", text)
	}
	if !strings.Contains(text, "OK  app=sound_watch action=rollback version=1.2.2") {
		t.Fatalf("missing rollback success output: %q", text)
	}
}

func TestAICommandsUseAdminAPIs(t *testing.T) {
	admin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/repl/ai/providers":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"providers":[{"name":"ollama","default_model":"llama3.1","models":["llama3.1"]},{"name":"openrouter","default_model":"anthropic/claude-sonnet-4-6","models":["anthropic/claude-sonnet-4-6"]}]}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/repl/ai/models":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"provider":"ollama","models":["llama3.1","qwen3"]}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/repl/ai/selection":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"session_id":"repl-9","provider":"openrouter","model":"anthropic/claude-sonnet-4-6"}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/repl/ai/selection":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"session_id":"repl-9","provider":"openrouter","model":"anthropic/claude-sonnet-4-6"}`))
		default:
			http.NotFound(w, req)
		}
	}))
	defer admin.Close()

	in := strings.NewReader("ai providers\nai models ollama\nai use openrouter anthropic/claude-sonnet-4-6\nai status\nexit\n")
	var out bytes.Buffer

	err := Run(context.Background(), in, &out, Options{
		Prompt:       "repl>",
		AdminBaseURL: admin.URL,
		SessionID:    "repl-9",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "ollama") || !strings.Contains(text, "openrouter") {
		t.Fatalf("missing provider output: %q", text)
	}
	if !strings.Contains(text, "llama3.1") {
		t.Fatalf("missing model output: %q", text)
	}
	if !strings.Contains(text, "sticky for repl-9") {
		t.Fatalf("missing ai use confirmation: %q", text)
	}
	if !strings.Contains(text, "session: repl-9") {
		t.Fatalf("missing ai status output: %q", text)
	}
}

func TestDescribeIncludesCapabilityClosureCommands(t *testing.T) {
	commands := []string{
		"identity ls",
		"identity resolve",
		"session create",
		"message post",
		"board pin",
		"artifact create",
		"canvas annotate",
		"search query",
		"memory remember",
		"placement ls",
		"recent ls",
		"store put",
		"bus emit",
	}
	for _, command := range commands {
		if _, ok := DescribeCommand(command); !ok {
			t.Fatalf("DescribeCommand(%q) not found", command)
		}
	}
}

func TestDocsExamplesIncludeCapabilityClosureTopics(t *testing.T) {
	result, err := ExecuteCommand(context.Background(), "docs examples", ExecuteOptions{})
	if err != nil {
		t.Fatalf("ExecuteCommand(docs examples) error = %v", err)
	}
	required := []string{
		"start-room-chat",
		"send-direct-message",
		"pin-family-bulletin",
		"remote-help-session",
		"shared-lesson-session",
		"annotate-shared-canvas",
		"search-household-memory",
		"review-learner-progress",
		"resume-multiplayer-session",
	}
	for _, topic := range required {
		if !strings.Contains(result.Output, topic) {
			t.Fatalf("docs examples missing %q in output: %q", topic, result.Output)
		}
	}
}

func TestCapabilityClosureGroupsUseAdminAPIs(t *testing.T) {
	admin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/identity":
			_, _ = w.Write([]byte(`{"identities":[{"id":"alice"}]}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/identity/resolve":
			_, _ = w.Write([]byte(`{"audience":"group:family","identities":[{"id":"alice"}]}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/session/create":
			_, _ = w.Write([]byte(`{"status":"ok","session":{"id":"sess-1"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/message/post":
			_, _ = w.Write([]byte(`{"status":"ok","message":{"id":"msg-1"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/board/pin":
			_, _ = w.Write([]byte(`{"status":"ok","item":{"id":"pin-1"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/artifact/create":
			_, _ = w.Write([]byte(`{"status":"ok","artifact":{"id":"art-1"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/canvas/annotate":
			_, _ = w.Write([]byte(`{"status":"ok","annotation":{"id":"ann-1"}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/search":
			_, _ = w.Write([]byte(`{"results":[{"id":"msg-1"}]}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/memory/remember":
			_, _ = w.Write([]byte(`{"status":"ok","memory":{"id":"mem-1"}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/placement":
			_, _ = w.Write([]byte(`{"placements":[{"device_id":"d1","zone":"kitchen"}]}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/recent":
			_, _ = w.Write([]byte(`{"items":[{"id":"evt-1","kind":"message"}]}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/store/put":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/bus/emit":
			_, _ = w.Write([]byte(`{"status":"ok","event":{"id":"bus-1"}}`))
		default:
			http.NotFound(w, req)
		}
	}))
	defer admin.Close()

	in := strings.NewReader(strings.Join([]string{
		"identity ls",
		"identity resolve group:family",
		"session create help room",
		"message post room-1 hello",
		"board pin family reminder",
		"artifact create lesson-1 math",
		"canvas annotate canvas-1 note",
		"search query hello",
		"memory remember kitchen milk",
		"placement ls",
		"recent ls",
		"store put notes key1 value1",
		"bus emit event alarm",
		"exit",
	}, "\n") + "\n")
	var out bytes.Buffer
	err := Run(context.Background(), in, &out, Options{Prompt: "repl>", AdminBaseURL: admin.URL})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "alice") {
		t.Fatalf("identity output missing: %q", text)
	}
	if !strings.Contains(text, "group:family") {
		t.Fatalf("identity resolve output missing audience label: %q", text)
	}
	if !strings.Contains(text, "sess-1") {
		t.Fatalf("session create output missing: %q", text)
	}
	if !strings.Contains(text, "msg-1") {
		t.Fatalf("message output missing: %q", text)
	}
	if !strings.Contains(text, "pin-1") {
		t.Fatalf("board output missing: %q", text)
	}
	if !strings.Contains(text, "art-1") {
		t.Fatalf("artifact output missing: %q", text)
	}
	if !strings.Contains(text, "ann-1") {
		t.Fatalf("canvas output missing: %q", text)
	}
	if !strings.Contains(text, "evt-1") {
		t.Fatalf("recent output missing: %q", text)
	}
	if !strings.Contains(text, "bus-1") {
		t.Fatalf("bus output missing: %q", text)
	}
}
