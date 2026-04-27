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

func TestStorePutWithTTLPassesFormValues(t *testing.T) {
	var capturedTTL string
	var capturedValue string
	admin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/store/put":
			if err := req.ParseForm(); err != nil {
				t.Fatalf("ParseForm() error = %v", err)
			}
			capturedTTL = req.Form.Get("ttl")
			capturedValue = req.Form.Get("value")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			http.NotFound(w, req)
		}
	}))
	defer admin.Close()

	in := strings.NewReader("store put notes key1 hello world --ttl 5s\nexit\n")
	var out bytes.Buffer
	err := Run(context.Background(), in, &out, Options{Prompt: "repl>", AdminBaseURL: admin.URL})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if capturedTTL != "5s" {
		t.Fatalf("store put ttl form value = %q, want 5s", capturedTTL)
	}
	if capturedValue != "hello world" {
		t.Fatalf("store put value form value = %q, want hello world", capturedValue)
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
		"identity show",
		"identity groups",
		"identity resolve",
		"identity prefs",
		"identity ack ls",
		"identity ack show",
		"identity ack record",
		"session create",
		"session show",
		"session members",
		"session join",
		"session leave",
		"session attach",
		"session detach",
		"session control request",
		"session control grant",
		"session control revoke",
		"message rooms",
		"message room new",
		"message room show",
		"message post",
		"message get",
		"message dm",
		"message thread",
		"message unread",
		"message ack",
		"board post",
		"board pin",
		"artifact create",
		"artifact patch",
		"artifact replace",
		"artifact template save",
		"artifact template apply",
		"canvas annotate",
		"search query",
		"search timeline",
		"search related",
		"search recent",
		"memory remember",
		"memory stream",
		"placement ls",
		"cohort ls",
		"cohort show",
		"cohort put",
		"cohort del",
		"ui push",
		"ui patch",
		"ui transition",
		"ui broadcast",
		"ui subscribe",
		"ui snapshot",
		"ui views ls",
		"ui views show",
		"ui views rm",
		"recent ls",
		"store ns ls",
		"store put",
		"store del",
		"store watch",
		"store bind",
		"bus emit",
		"bus replay",
		"handlers ls",
		"handlers on",
		"handlers off",
		"scenarios ls",
		"scenarios show",
		"scenarios define",
		"scenarios undefine",
		"sim device new",
		"sim device rm",
		"sim input",
		"sim ui",
		"sim expect",
		"sim record",
		"scripts dry-run",
		"scripts run",
	}
	for _, command := range commands {
		if _, ok := DescribeCommand(command); !ok {
			t.Fatalf("DescribeCommand(%q) not found", command)
		}
	}
}

func TestSessionShowMembersCommandsUseAdminAPIs(t *testing.T) {
	admin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/session/show":
			_, _ = w.Write([]byte(`{"session":{"id":"sess-1","kind":"help","target":"room-1","participants":[{"identity_id":"alice"},{"identity_id":"bob"}]}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/session/members":
			_, _ = w.Write([]byte(`{"session_id":"sess-1","participants":[{"identity_id":"alice"},{"identity_id":"bob"}]}`))
		default:
			http.NotFound(w, req)
		}
	}))
	defer admin.Close()

	in := strings.NewReader("session show sess-1\nsession members sess-1\nexit\n")
	var out bytes.Buffer
	err := Run(context.Background(), in, &out, Options{Prompt: "repl>", AdminBaseURL: admin.URL})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "sess-1") {
		t.Fatalf("missing session id output: %q", text)
	}
	if !strings.Contains(text, "help") || !strings.Contains(text, "room-1") {
		t.Fatalf("missing session metadata output: %q", text)
	}
	if !strings.Contains(text, "alice") || !strings.Contains(text, "bob") {
		t.Fatalf("missing session members output: %q", text)
	}
}

func TestSessionJoinLeaveCommandsUseAdminAPIs(t *testing.T) {
	admin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/session/join":
			_, _ = w.Write([]byte(`{"status":"ok","session":{"id":"sess-1","participants":[{"identity_id":"alice"}]}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/session/leave":
			_, _ = w.Write([]byte(`{"status":"ok","session":{"id":"sess-1","participants":[]}}`))
		default:
			http.NotFound(w, req)
		}
	}))
	defer admin.Close()

	in := strings.NewReader("session join sess-1 alice\nsession leave sess-1 alice\nexit\n")
	var out bytes.Buffer
	err := Run(context.Background(), in, &out, Options{Prompt: "repl>", AdminBaseURL: admin.URL})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "OK  session=sess-1 participant=alice action=join") {
		t.Fatalf("missing session join output: %q", text)
	}
	if !strings.Contains(text, "OK  session=sess-1 participant=alice action=leave") {
		t.Fatalf("missing session leave output: %q", text)
	}
}

func TestSessionAttachDetachAndControlCommandsUseAdminAPIs(t *testing.T) {
	admin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/session/attach":
			_, _ = w.Write([]byte(`{"status":"ok","session":{"id":"sess-1","attached_devices":["device:kitchen-display"]}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/session/detach":
			_, _ = w.Write([]byte(`{"status":"ok","session":{"id":"sess-1","attached_devices":[]}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/session/control/request":
			_, _ = w.Write([]byte(`{"status":"ok","session":{"id":"sess-1","control_requests":[{"participant_id":"alice","control_type":"keyboard"}]}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/session/control/grant":
			_, _ = w.Write([]byte(`{"status":"ok","session":{"id":"sess-1","control_grants":[{"participant_id":"alice","granted_by":"moderator","control_type":"keyboard"}]}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/session/control/revoke":
			_, _ = w.Write([]byte(`{"status":"ok","session":{"id":"sess-1","control_grants":[]}}`))
		default:
			http.NotFound(w, req)
		}
	}))
	defer admin.Close()

	in := strings.NewReader(strings.Join([]string{
		"session attach sess-1 device:kitchen-display",
		"session detach sess-1 device:kitchen-display",
		"session control request sess-1 alice keyboard",
		"session control grant sess-1 alice moderator keyboard",
		"session control revoke sess-1 alice moderator",
		"exit",
	}, "\n") + "\n")
	var out bytes.Buffer
	err := Run(context.Background(), in, &out, Options{Prompt: "repl>", AdminBaseURL: admin.URL})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "OK  session=sess-1 device=device:kitchen-display action=attach") {
		t.Fatalf("missing session attach output: %q", text)
	}
	if !strings.Contains(text, "OK  session=sess-1 device=device:kitchen-display action=detach") {
		t.Fatalf("missing session detach output: %q", text)
	}
	if !strings.Contains(text, "OK  session=sess-1 participant=alice action=control.request type=keyboard") {
		t.Fatalf("missing session control request output: %q", text)
	}
	if !strings.Contains(text, "OK  session=sess-1 participant=alice action=control.grant by=moderator type=keyboard") {
		t.Fatalf("missing session control grant output: %q", text)
	}
	if !strings.Contains(text, "OK  session=sess-1 participant=alice action=control.revoke by=moderator") {
		t.Fatalf("missing session control revoke output: %q", text)
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
		"red-alert-broadcast",
		"timer-and-reminder",
		"presence-query",
		"sim-only-assertion",
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
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/identity/show":
			_, _ = w.Write([]byte(`{"identity":{"id":"alice","display_name":"Alice"}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/identity/groups":
			_, _ = w.Write([]byte(`{"groups":["family","operators"]}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/identity/resolve":
			_, _ = w.Write([]byte(`{"audience":"group:family","identities":[{"id":"alice"}]}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/identity/prefs":
			_, _ = w.Write([]byte(`{"identity":"alice","preferences":{"notifications":"normal"}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/identity/ack":
			_, _ = w.Write([]byte(`{"subject_ref":"message:msg-1","acknowledgements":[{"actor_ref":"device:kitchen-screen","mode":"dismissed"}]}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/identity/ack":
			_, _ = w.Write([]byte(`{"status":"ok","ack":{"actor_ref":"device:kitchen-screen","mode":"dismissed"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/session/create":
			_, _ = w.Write([]byte(`{"status":"ok","session":{"id":"sess-1"}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/message/rooms":
			_, _ = w.Write([]byte(`{"rooms":[{"id":"room-1","name":"kitchen"}]}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/message/room":
			_, _ = w.Write([]byte(`{"status":"ok","room":{"id":"room-2","name":"family"}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/message/room":
			_, _ = w.Write([]byte(`{"room":{"id":"room-1","name":"kitchen"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/message/post":
			_, _ = w.Write([]byte(`{"status":"ok","message":{"id":"msg-1"}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/message/get":
			_, _ = w.Write([]byte(`{"message":{"id":"msg-1","room":"room-1","text":"hello"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/message/dm":
			_, _ = w.Write([]byte(`{"status":"ok","message":{"id":"msg-dm-1","target_ref":"person:mom"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/message/thread":
			_, _ = w.Write([]byte(`{"status":"ok","message":{"id":"msg-2","thread_root_ref":"msg-1"}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/message/unread":
			_, _ = w.Write([]byte(`{"identity_id":"alice","messages":[]}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/message/ack":
			_, _ = w.Write([]byte(`{"status":"ok","ack":{"message_id":"msg-1"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/board/post":
			_, _ = w.Write([]byte(`{"status":"ok","item":{"id":"post-1"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/board/pin":
			_, _ = w.Write([]byte(`{"status":"ok","item":{"id":"pin-1"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/artifact/create":
			_, _ = w.Write([]byte(`{"status":"ok","artifact":{"id":"art-1"}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/artifact/get":
			_, _ = w.Write([]byte(`{"artifact":{"id":"art-1","version":2,"title":"math advanced"}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/artifact/history":
			_, _ = w.Write([]byte(`{"artifact_id":"art-1","versions":[{"version":1,"action":"create"},{"version":2,"action":"patch"}]}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/artifact/patch":
			_, _ = w.Write([]byte(`{"status":"ok","artifact":{"id":"art-1","title":"math advanced"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/artifact/replace":
			_, _ = w.Write([]byte(`{"status":"ok","artifact":{"id":"art-1","title":"math replacement"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/artifact/template/save":
			_, _ = w.Write([]byte(`{"status":"ok","template":{"name":"lesson-base","source_artifact_id":"art-1"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/artifact/template/apply":
			_, _ = w.Write([]byte(`{"status":"ok","artifact":{"id":"art-1","title":"fractions basics"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/canvas/annotate":
			_, _ = w.Write([]byte(`{"status":"ok","annotation":{"id":"ann-1"}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/search":
			_, _ = w.Write([]byte(`{"results":[{"id":"msg-1"}]}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/search/timeline":
			_, _ = w.Write([]byte(`{"items":[{"id":"timeline-1","kind":"message"}]}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/search/related":
			_, _ = w.Write([]byte(`{"results":[{"id":"related-1"}]}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/search/recent":
			_, _ = w.Write([]byte(`{"items":[{"id":"recent-1","kind":"memory"}]}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/memory/remember":
			_, _ = w.Write([]byte(`{"status":"ok","memory":{"id":"mem-1"}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/memory/stream":
			_, _ = w.Write([]byte(`{"memories":[{"id":"mem-1","scope":"kitchen"}]}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/placement":
			_, _ = w.Write([]byte(`{"placements":[{"device_id":"d1","zone":"kitchen"}]}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/cohort" && req.URL.Query().Get("name") == "":
			_, _ = w.Write([]byte(`{"cohorts":[{"name":"family-screens","selectors":["role:screen","zone:kitchen"]}]}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/cohort" && req.URL.Query().Get("name") == "family-screens":
			_, _ = w.Write([]byte(`{"cohort":{"name":"family-screens","selectors":["role:screen","zone:kitchen"]},"members":["d1"]}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/cohort/upsert":
			_, _ = w.Write([]byte(`{"status":"ok","cohort":{"name":"family-screens","selectors":["role:screen","zone:kitchen"]},"members":["d1"]}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/cohort/del":
			_, _ = w.Write([]byte(`{"status":"ok","deleted":true}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/ui/push":
			_, _ = w.Write([]byte(`{"status":"ok","snapshot":{"device_id":"d1","root_id":"root-main"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/ui/patch":
			_, _ = w.Write([]byte(`{"status":"ok","snapshot":{"device_id":"d1","last_patch_component_id":"banner"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/ui/transition":
			_, _ = w.Write([]byte(`{"status":"ok","snapshot":{"device_id":"d1","last_transition":"fade"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/ui/broadcast":
			_, _ = w.Write([]byte(`{"status":"ok","broadcast":{"cohort":"family-screens"},"members":["d1"]}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/ui/subscribe":
			_, _ = w.Write([]byte(`{"status":"ok","snapshot":{"device_id":"d1","subscriptions":["cohort:family-screens"]}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/ui/snapshot":
			_, _ = w.Write([]byte(`{"snapshot":{"device_id":"d1","root_id":"root-main"}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/ui/views" && req.URL.Query().Get("view_id") == "":
			_, _ = w.Write([]byte(`{"views":[{"view_id":"kitchen-home","root_id":"root-main"}]}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/ui/views" && req.URL.Query().Get("view_id") == "kitchen-home":
			_, _ = w.Write([]byte(`{"view":{"view_id":"kitchen-home","root_id":"root-main","descriptor":"{\"type\":\"stack\"}"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/ui/views/del":
			_, _ = w.Write([]byte(`{"status":"ok","deleted":true}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/recent":
			_, _ = w.Write([]byte(`{"items":[{"id":"evt-1","kind":"message"}]}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/store/ns":
			_, _ = w.Write([]byte(`{"namespaces":[{"name":"notes","record_count":2}]}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/store/put":
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/store/watch":
			_, _ = w.Write([]byte(`{"namespace":"notes","prefix":"key","records":[{"namespace":"notes","key":"key1","value":"value1"}]}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/store/bind":
			_, _ = w.Write([]byte(`{"status":"ok","record":{"namespace":"notes","key":"key1","binding":"device-1:chat"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/store/del":
			_, _ = w.Write([]byte(`{"status":"ok","deleted":true}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/bus/emit":
			_, _ = w.Write([]byte(`{"status":"ok","event":{"id":"bus-1"}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/bus/replay":
			_, _ = w.Write([]byte(`{"events":[{"id":"bus-1","kind":"event","name":"alarm"}]}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/handlers":
			_, _ = w.Write([]byte(`{"handlers":[{"id":"handler-1","selector":"scenario=chat","action":"submit","run_command":"store put notes key value"}]}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/handlers/on":
			_, _ = w.Write([]byte(`{"status":"ok","handler":{"id":"handler-2","selector":"scenario=chat","action":"submit","emit_kind":"intent","emit_name":"alert_ack"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/handlers/off":
			_, _ = w.Write([]byte(`{"status":"ok","deleted":true}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/scenarios/inline" && req.URL.Query().Get("name") == "":
			_, _ = w.Write([]byte(`{"scenarios":[{"name":"red_alert","priority":"high","match_intents":["red alert"],"match_events":["alarm.triggered"]}]}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/scenarios/inline" && req.URL.Query().Get("name") == "red_alert":
			_, _ = w.Write([]byte(`{"scenario":{"name":"red_alert","priority":"high","on_start":"ui broadcast all_screens banner"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/scenarios/inline/define":
			_, _ = w.Write([]byte(`{"status":"ok","scenario":{"name":"red_alert"}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/scenarios/inline/undefine":
			_, _ = w.Write([]byte(`{"status":"ok","deleted":true}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/sim/devices/new":
			_, _ = w.Write([]byte(`{"status":"ok","device":{"device_id":"sim-kitchen","caps":["display","keyboard"]}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/sim/input":
			_, _ = w.Write([]byte(`{"status":"ok","input":{"id":"simin-1","device_id":"sim-kitchen","component_id":"chat_input","action":"submit"}}`))
		case req.Method == http.MethodGet && req.URL.Path == "/admin/api/sim/ui":
			_, _ = w.Write([]byte(`{"device":{"device_id":"sim-kitchen","caps":["display","keyboard"]},"snapshot":{"device_id":"sim-kitchen","root_id":"sim-root"},"inputs":[{"id":"simin-1","action":"submit"}]}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/sim/expect":
			_, _ = w.Write([]byte(`{"status":"ok","result":{"device_id":"sim-kitchen","kind":"ui","selector":"hello","matched":true}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/sim/record":
			_, _ = w.Write([]byte(`{"status":"ok","result":{"device_id":"sim-kitchen","duration":"5s","inputs":[{"id":"simin-1"}],"messages":[{"id":"bus-1"}]}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/sim/devices/rm":
			_, _ = w.Write([]byte(`{"status":"ok","deleted":true}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/scripts/dry-run":
			_, _ = w.Write([]byte(`{"status":"ok","result":{"path":"/tmp/smoke.term","command_count":2,"skipped_count":1}}`))
		case req.Method == http.MethodPost && req.URL.Path == "/admin/api/scripts/run":
			_, _ = w.Write([]byte(`{"status":"ok","result":{"path":"/tmp/smoke.term","command_count":2,"executed_count":2,"failed_count":0}}`))
		default:
			http.NotFound(w, req)
		}
	}))
	defer admin.Close()

	in := strings.NewReader(strings.Join([]string{
		"identity ls",
		"identity show alice",
		"identity groups",
		"identity resolve group:family",
		"identity prefs alice",
		"identity ack record message:msg-1 --actor device:kitchen-screen --mode dismissed",
		"identity ack show message:msg-1",
		"session create help room",
		"message rooms",
		"message room new family",
		"message room show kitchen",
		"message post room-1 hello",
		"message get msg-1",
		"message dm mom come downstairs",
		"message thread msg-1 follow up",
		"message unread alice room-1",
		"message ack alice msg-1",
		"board post family grocery update",
		"board pin family reminder",
		"artifact create lesson-1 math",
		"artifact show art-1",
		"artifact history art-1",
		"artifact patch art-1 math advanced",
		"artifact replace art-1 math replacement",
		"artifact template save lesson-base art-1",
		"artifact template apply lesson-base art-1",
		"canvas annotate canvas-1 note",
		"search query hello",
		"search timeline message",
		"search related board_post_42",
		"search recent memory",
		"memory remember kitchen milk",
		"memory stream kitchen",
		"placement ls",
		"cohort ls",
		"cohort show family-screens",
		"cohort put family-screens --selectors zone:kitchen,role:screen",
		"cohort del family-screens",
		"ui push d1 '{\"type\":\"stack\"}' --root root-main",
		"ui patch d1 banner '{\"type\":\"text\"}'",
		"ui transition d1 banner fade --duration-ms 150",
		"ui broadcast family-screens '{\"type\":\"banner\"}' --patch alert-banner",
		"ui subscribe d1 --to cohort:family-screens",
		"ui snapshot d1",
		"ui views ls",
		"ui views show kitchen-home",
		"ui views rm kitchen-home",
		"recent ls",
		"store ns ls",
		"store put notes key1 value1",
		"store watch notes --prefix key",
		"store bind notes key1 --to device-1:chat",
		"store del notes key1",
		"bus emit event alarm",
		"bus replay bus-1 bus-1 --kind event",
		"handlers ls",
		"handlers on scenario=chat submit --emit intent alert_ack",
		"handlers off handler-2",
		"scenarios ls",
		"scenarios show red_alert",
		"scenarios define red_alert --match intent=red_alert --match event=alarm.triggered --priority high --on-start 'ui broadcast all_screens banner' --on-event alarm.triggered 'bus emit event alarm_ack'",
		"scenarios undefine red_alert",
		"sim device new sim-kitchen --caps display,keyboard",
		"sim input sim-kitchen chat_input submit hello",
		"sim ui sim-kitchen",
		"sim expect sim-kitchen ui hello --within 5s",
		"sim record sim-kitchen --duration 5s",
		"sim device rm sim-kitchen",
		"scripts dry-run /tmp/smoke.term",
		"scripts run /tmp/smoke.term",
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
	if !strings.Contains(text, "operators") {
		t.Fatalf("identity groups output missing operators group: %q", text)
	}
	if !strings.Contains(text, "notifications") {
		t.Fatalf("identity prefs output missing preferences payload: %q", text)
	}
	if !strings.Contains(text, "action=ack.record") {
		t.Fatalf("identity ack record output missing: %q", text)
	}
	if !strings.Contains(text, "sess-1") {
		t.Fatalf("session create output missing: %q", text)
	}
	if !strings.Contains(text, "msg-1") {
		t.Fatalf("message output missing: %q", text)
	}
	if !strings.Contains(text, "msg-dm-1") {
		t.Fatalf("direct message output missing: %q", text)
	}
	if !strings.Contains(text, "room=room-2") || !strings.Contains(text, "action=create") {
		t.Fatalf("message room create output missing: %q", text)
	}
	if !strings.Contains(text, "\"name\": \"kitchen\"") {
		t.Fatalf("message room show output missing kitchen payload: %q", text)
	}
	if !strings.Contains(text, "action=thread") {
		t.Fatalf("message thread output missing: %q", text)
	}
	if !strings.Contains(text, "post-1") {
		t.Fatalf("board post output missing: %q", text)
	}
	if !strings.Contains(text, "pin-1") {
		t.Fatalf("board output missing: %q", text)
	}
	if !strings.Contains(text, "art-1") {
		t.Fatalf("artifact output missing: %q", text)
	}
	if !strings.Contains(text, `"version": 2`) {
		t.Fatalf("artifact show output missing version payload: %q", text)
	}
	if !strings.Contains(text, `"action": "patch"`) {
		t.Fatalf("artifact history output missing patch entry: %q", text)
	}
	if !strings.Contains(text, "action=replace") {
		t.Fatalf("artifact replace output missing: %q", text)
	}
	if !strings.Contains(text, "template=lesson-base source=art-1 action=save") {
		t.Fatalf("artifact template save output missing: %q", text)
	}
	if !strings.Contains(text, "template=lesson-base target=art-1 action=apply") {
		t.Fatalf("artifact template apply output missing: %q", text)
	}
	if !strings.Contains(text, "ann-1") {
		t.Fatalf("canvas output missing: %q", text)
	}
	if !strings.Contains(text, "timeline-1") {
		t.Fatalf("search timeline output missing: %q", text)
	}
	if !strings.Contains(text, "related-1") {
		t.Fatalf("search related output missing: %q", text)
	}
	if !strings.Contains(text, "recent-1") {
		t.Fatalf("search recent output missing: %q", text)
	}
	if !strings.Contains(text, "evt-1") {
		t.Fatalf("recent output missing: %q", text)
	}
	if !strings.Contains(text, "family-screens") {
		t.Fatalf("cohort output missing cohort name: %q", text)
	}
	if !strings.Contains(text, "\"members\": [\n    \"d1\"\n  ]") && !strings.Contains(text, `"members":["d1"]`) {
		t.Fatalf("cohort show output missing members payload: %q", text)
	}
	if !strings.Contains(text, "selectors=zone:kitchen,role:screen") {
		t.Fatalf("cohort put output missing selectors summary: %q", text)
	}
	if !strings.Contains(text, "cohort=family-screens") {
		t.Fatalf("cohort delete output missing cohort summary: %q", text)
	}
	if !strings.Contains(text, "action=push device=d1") {
		t.Fatalf("ui push output missing summary: %q", text)
	}
	if !strings.Contains(text, "action=patch device=d1 component=banner") {
		t.Fatalf("ui patch output missing summary: %q", text)
	}
	if !strings.Contains(text, "action=transition device=d1 transition=fade") {
		t.Fatalf("ui transition output missing summary: %q", text)
	}
	if !strings.Contains(text, "action=broadcast cohort=family-screens") {
		t.Fatalf("ui broadcast output missing summary: %q", text)
	}
	if !strings.Contains(text, "action=subscribe device=d1 to=cohort:family-screens") {
		t.Fatalf("ui subscribe output missing summary: %q", text)
	}
	if !strings.Contains(text, `"snapshot": {`) {
		t.Fatalf("ui snapshot output missing JSON payload: %q", text)
	}
	if !strings.Contains(text, "kitchen-home") {
		t.Fatalf("ui views output missing view id: %q", text)
	}
	if !strings.Contains(text, "view=kitchen-home") {
		t.Fatalf("ui views rm output missing view summary: %q", text)
	}
	if !strings.Contains(text, "notes") {
		t.Fatalf("store namespace output missing: %q", text)
	}
	if !strings.Contains(text, "bound_to=device-1:chat") {
		t.Fatalf("store bind output missing: %q", text)
	}
	if !strings.Contains(text, "deleted=true") {
		t.Fatalf("store delete output missing: %q", text)
	}
	if !strings.Contains(text, "bus-1") {
		t.Fatalf("bus output missing: %q", text)
	}
	if !strings.Contains(text, "handler-1") {
		t.Fatalf("handlers ls output missing handler id: %q", text)
	}
	if !strings.Contains(text, "red_alert") {
		t.Fatalf("scenarios output missing: %q", text)
	}
	if !strings.Contains(text, "action=define scenario=red_alert") {
		t.Fatalf("scenarios define output missing: %q", text)
	}
	if !strings.Contains(text, "deleted=true scenario=red_alert") {
		t.Fatalf("scenarios undefine output missing: %q", text)
	}
	if !strings.Contains(text, "action=sim.device.new device=sim-kitchen") {
		t.Fatalf("sim device new output missing: %q", text)
	}
	if !strings.Contains(text, "action=sim.input device=sim-kitchen component=chat_input event=submit") {
		t.Fatalf("sim input output missing: %q", text)
	}
	if !strings.Contains(text, `"device_id": "sim-kitchen"`) && !strings.Contains(text, `"device_id":"sim-kitchen"`) {
		t.Fatalf("sim ui output missing device payload: %q", text)
	}
	if !strings.Contains(text, "action=sim.device.rm device=sim-kitchen deleted=true") {
		t.Fatalf("sim device rm output missing: %q", text)
	}
	if !strings.Contains(text, "action=sim.expect device=sim-kitchen kind=ui matched=true") {
		t.Fatalf("sim expect output missing: %q", text)
	}
	if !strings.Contains(text, "action=sim.record device=sim-kitchen inputs=1 messages=1") {
		t.Fatalf("sim record output missing: %q", text)
	}
	if !strings.Contains(text, "action=scripts.dry-run path=/tmp/smoke.term commands=2 skipped=1") {
		t.Fatalf("scripts dry-run output missing: %q", text)
	}
	if !strings.Contains(text, "action=scripts.run path=/tmp/smoke.term commands=2 executed=2 failed=0") {
		t.Fatalf("scripts run output missing: %q", text)
	}
	if !strings.Contains(text, "handler=handler-2 selector=scenario=chat action=submit") {
		t.Fatalf("handlers on output missing summary: %q", text)
	}
	if !strings.Contains(text, "handler=handler-2") {
		t.Fatalf("handlers off output missing handler id: %q", text)
	}
}
