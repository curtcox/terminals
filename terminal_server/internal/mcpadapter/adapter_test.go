package mcpadapter

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestToolsIncludeRegistryAndDiscoveryTools(t *testing.T) {
	adapter := New(Config{})
	tools := adapter.Tools()
	if len(tools) == 0 {
		t.Fatalf("expected generated tools")
	}
	foundComplete := false
	foundDescribe := false
	foundAppReload := false
	foundLogsTail := false
	for _, tool := range tools {
		switch tool.Name {
		case ToolReplComplete:
			foundComplete = true
		case ToolReplDescribe:
			foundDescribe = true
		case "app_reload":
			foundAppReload = true
			if tool.Classification != "mutating" {
				t.Fatalf("app_reload classification = %q", tool.Classification)
			}
		case "logs_tail":
			foundLogsTail = true
			if tool.Classification != "operational" {
				t.Fatalf("logs_tail classification = %q", tool.Classification)
			}
		}
		if strings.Contains(tool.Name, "confirm") || strings.Contains(tool.Name, "force") {
			t.Fatalf("tool catalog should not expose confirm/force controls: %s", tool.Name)
		}
	}
	if !foundComplete || !foundDescribe || !foundAppReload || !foundLogsTail {
		t.Fatalf("missing expected tools: complete=%v describe=%v app_reload=%v logs_tail=%v", foundComplete, foundDescribe, foundAppReload, foundLogsTail)
	}
}

func TestCapabilityNegotiationAndUnsupportedMutations(t *testing.T) {
	adapter := New(Config{})
	session := adapter.OpenSession("repl-mcp-1", "codex", ClientCapabilities{})
	if session.Capability != MutatingUnavailable {
		t.Fatalf("capability = %q, want %q", session.Capability, MutatingUnavailable)
	}
	resp, err := adapter.CallTool(context.Background(), CallToolRequest{
		SessionID: "repl-mcp-1",
		ToolName:  "app_reload",
		Arguments: map[string]any{"app": "sound_watch"},
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if resp.ErrorCode != "unsupported_client" {
		t.Fatalf("error code = %q, want unsupported_client", resp.ErrorCode)
	}
}

func TestFallbackConfirmationFlow(t *testing.T) {
	adapter := New(Config{
		Now: func() time.Time {
			return time.Unix(1710000000, 0)
		},
	})
	adapter.OpenSession("repl-mcp-2", "codex", ClientCapabilities{SupportsFallbackID: true})

	first, err := adapter.CallTool(context.Background(), CallToolRequest{
		SessionID: "repl-mcp-2",
		ToolName:  "app_reload",
		Arguments: map[string]any{"app": "sound_watch"},
	})
	if err != nil {
		t.Fatalf("first CallTool() error = %v", err)
	}
	if first.Status != "confirmation_required" || first.ConfirmationID == "" {
		t.Fatalf("expected confirmation_required, got %+v", first)
	}
	second, err := adapter.CallTool(context.Background(), CallToolRequest{
		SessionID:          "repl-mcp-2",
		ToolName:           "app_reload",
		Arguments:          map[string]any{"app": "sound_watch"},
		MetaConfirmationID: first.ConfirmationID,
	})
	if err != nil {
		t.Fatalf("second CallTool() error = %v", err)
	}
	if second.Status == "confirmation_required" {
		t.Fatalf("expected confirmation to be accepted")
	}
	if second.ErrorCode != "command_failed" && second.Status != "ok" {
		t.Fatalf("unexpected response after confirmation: %+v", second)
	}
}

func TestElicitationPath(t *testing.T) {
	called := false
	adapter := New(Config{
		Elicit: func(_ context.Context, req ElicitRequest) (ElicitResponse, error) {
			called = true
			if req.ToolName != "app_reload" {
				t.Fatalf("tool = %q", req.ToolName)
			}
			return ElicitResponse{Approved: false}, nil
		},
	})
	adapter.OpenSession("repl-mcp-3", "claude", ClientCapabilities{SupportsElicitation: true})
	resp, err := adapter.CallTool(context.Background(), CallToolRequest{
		SessionID: "repl-mcp-3",
		ToolName:  "app_reload",
		Arguments: map[string]any{"app": "sound_watch"},
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if !called {
		t.Fatalf("expected elicitation callback")
	}
	if resp.Status != "rejected" {
		t.Fatalf("status = %q, want rejected", resp.Status)
	}
}

func TestReplDiscoveryToolCalls(t *testing.T) {
	adapter := New(Config{})
	adapter.OpenSession("repl-mcp-4", "codex", ClientCapabilities{SupportsFallbackID: true})
	completeResp, err := adapter.CallTool(context.Background(), CallToolRequest{
		SessionID: "repl-mcp-4",
		ToolName:  ToolReplComplete,
		Arguments: map[string]any{"prefix": "app r", "limit": 5},
	})
	if err != nil {
		t.Fatalf("repl_complete error = %v", err)
	}
	if completeResp.Status != "ok" {
		t.Fatalf("repl_complete status = %q", completeResp.Status)
	}
	matches, _ := completeResp.Metadata["matches"].([]string)
	if len(matches) == 0 {
		t.Fatalf("expected completion matches")
	}
	describeResp, err := adapter.CallTool(context.Background(), CallToolRequest{
		SessionID: "repl-mcp-4",
		ToolName:  ToolReplDescribe,
		Arguments: map[string]any{"command": "app reload"},
	})
	if err != nil {
		t.Fatalf("repl_describe error = %v", err)
	}
	if describeResp.Status != "ok" {
		t.Fatalf("repl_describe status = %q", describeResp.Status)
	}
}

func TestReplDescribeWithoutCommandReturnsRegistrySummary(t *testing.T) {
	adapter := New(Config{})
	adapter.OpenSession("repl-mcp-describe-1", "codex", ClientCapabilities{})

	resp, err := adapter.CallTool(context.Background(), CallToolRequest{
		SessionID: "repl-mcp-describe-1",
		ToolName:  ToolReplDescribe,
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q, want ok", resp.Status)
	}
	commands, ok := resp.Metadata["commands"]
	if !ok {
		t.Fatalf("missing commands metadata in repl_describe summary response")
	}
	switch typed := commands.(type) {
	case []any:
		if len(typed) == 0 {
			t.Fatalf("expected non-empty command registry summary")
		}
	default:
		if fmt.Sprintf("%v", typed) == "[]" {
			t.Fatalf("expected non-empty command registry summary")
		}
	}
}

func TestOptionalPositionalArgsAreNotRequiredInSchemas(t *testing.T) {
	adapter := New(Config{})
	tools := adapter.Tools()

	var appLogs Tool
	var logsTail Tool
	for _, tool := range tools {
		switch tool.Name {
		case "app_logs":
			appLogs = tool
		case "logs_tail":
			logsTail = tool
		}
	}
	if appLogs.Name == "" || logsTail.Name == "" {
		t.Fatalf("missing expected tools app_logs or logs_tail")
	}

	appLogsRequired := requiredSchemaFields(appLogs.ArgumentsSchema)
	if !slices.Contains(appLogsRequired, "app") {
		t.Fatalf("app_logs required fields = %#v, want app", appLogsRequired)
	}
	if slices.Contains(appLogsRequired, "query") {
		t.Fatalf("app_logs should not require optional query; required=%#v", appLogsRequired)
	}

	logsTailRequired := requiredSchemaFields(logsTail.ArgumentsSchema)
	if slices.Contains(logsTailRequired, "query") {
		t.Fatalf("logs_tail should not require optional query; required=%#v", logsTailRequired)
	}
}

func TestOptionalPositionalArgsAreOptionalAtCallTime(t *testing.T) {
	adapter := New(Config{})
	adapter.OpenSession("repl-mcp-optional-1", "codex", ClientCapabilities{})

	logsResp, err := adapter.CallTool(context.Background(), CallToolRequest{
		SessionID: "repl-mcp-optional-1",
		ToolName:  "logs_tail",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("logs_tail CallTool() error = %v", err)
	}
	if logsResp.ErrorCode == "invalid_arguments" {
		t.Fatalf("logs_tail returned invalid_arguments for optional query: %+v", logsResp)
	}

	appLogsResp, err := adapter.CallTool(context.Background(), CallToolRequest{
		SessionID: "repl-mcp-optional-1",
		ToolName:  "app_logs",
		Arguments: map[string]any{"app": "sound_watch"},
	})
	if err != nil {
		t.Fatalf("app_logs CallTool() error = %v", err)
	}
	if appLogsResp.ErrorCode == "invalid_arguments" {
		t.Fatalf("app_logs returned invalid_arguments with required-only args: %+v", appLogsResp)
	}
}

func TestOperationalBudgetConcurrentLimit(t *testing.T) {
	adapter := New(Config{
		OperationalMax: 1,
		OperationalTTL: 2 * time.Second,
	})
	adapter.OpenSession("repl-mcp-op-1", "codex", ClientCapabilities{SupportsFallbackID: true})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = adapter.CallTool(context.Background(), CallToolRequest{
			SessionID: "repl-mcp-op-1",
			ToolName:  "sleep",
			Arguments: map[string]any{"seconds": "0.2"},
		})
	}()
	time.Sleep(25 * time.Millisecond)

	resp, err := adapter.CallTool(context.Background(), CallToolRequest{
		SessionID: "repl-mcp-op-1",
		ToolName:  "sleep",
		Arguments: map[string]any{"seconds": "0.2"},
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if resp.ErrorCode != "rate_limited" {
		t.Fatalf("error code = %q, want rate_limited", resp.ErrorCode)
	}
	wg.Wait()
}

func TestOperationalBudgetTTL(t *testing.T) {
	adapter := New(Config{
		OperationalMax: 2,
		OperationalTTL: 25 * time.Millisecond,
	})
	adapter.OpenSession("repl-mcp-op-2", "codex", ClientCapabilities{SupportsFallbackID: true})
	resp, err := adapter.CallTool(context.Background(), CallToolRequest{
		SessionID: "repl-mcp-op-2",
		ToolName:  "sleep",
		Arguments: map[string]any{"seconds": "0.2"},
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if resp.ErrorCode != "operational_ttl_exceeded" {
		t.Fatalf("error code = %q, want operational_ttl_exceeded", resp.ErrorCode)
	}
}

func TestCallToolStreamEmitsOutputChunks(t *testing.T) {
	adapter := New(Config{})
	adapter.OpenSession("repl-mcp-stream-1", "codex", ClientCapabilities{})
	chunks := make([]string, 0, 2)
	resp, err := adapter.CallToolStream(context.Background(), CallToolRequest{
		SessionID: "repl-mcp-stream-1",
		ToolName:  "echo",
		Arguments: map[string]any{"text": "hello-stream"},
	}, func(chunk string) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("CallToolStream() error = %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q, want ok", resp.Status)
	}
	if len(chunks) == 0 {
		t.Fatalf("expected at least one output chunk")
	}
	if !strings.Contains(strings.Join(chunks, ""), "hello-stream") {
		t.Fatalf("stream chunks missing echoed text: %q", strings.Join(chunks, ""))
	}
}

func requiredSchemaFields(schema map[string]any) []string {
	raw, ok := schema["required"]
	if !ok {
		return nil
	}
	typed, ok := raw.([]string)
	if ok {
		return typed
	}
	anySlice, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(anySlice))
	for _, item := range anySlice {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
