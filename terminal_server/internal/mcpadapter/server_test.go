package mcpadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/replsession"
)

type fakeSessionService struct {
	created  []replsession.CreateSessionRequest
	detached []replsession.DetachSessionRequest
}

func (f *fakeSessionService) CreateSession(_ context.Context, req replsession.CreateSessionRequest) (*replsession.CreateSessionResponse, error) {
	f.created = append(f.created, req)
	return &replsession.CreateSessionResponse{
		Session: replsession.ReplSession{
			ID: "repl-created-1",
		},
	}, nil
}

func (f *fakeSessionService) DetachSession(_ context.Context, req replsession.DetachSessionRequest) (*replsession.DetachSessionResponse, error) {
	f.detached = append(f.detached, req)
	return &replsession.DetachSessionResponse{}, nil
}

func TestHTTPInitializeCallAndShutdown(t *testing.T) {
	adapter := New(Config{})
	sessions := &fakeSessionService{}
	server, err := NewServer(ServerConfig{
		Adapter:      adapter,
		Sessions:     sessions,
		AdminBaseURL: "http://127.0.0.1:50053",
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	httpServer := httptest.NewServer(http.HandlerFunc(server.ServeHTTP))
	defer httpServer.Close()

	initResp := postRPC(t, httpServer.URL, "", rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "initialize",
		Params: mustRawJSON(t, map[string]any{
			"clientInfo": map[string]any{"name": "codex", "version": "1"},
		}),
	})
	initResult := parseAnyMap(initResp.Result)
	sessionID := strings.TrimSpace(anyString(initResult["session_id"]))
	if sessionID == "" {
		t.Fatalf("initialize response missing session_id: %+v", initResp)
	}
	if len(sessions.created) != 1 {
		t.Fatalf("CreateSession called %d times, want 1", len(sessions.created))
	}

	callResp := postRPC(t, httpServer.URL, sessionID, rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("2"),
		Method:  "tools/call",
		Params: mustRawJSON(t, map[string]any{
			"name":      "echo",
			"arguments": map[string]any{"text": "hello-mcp"},
		}),
	})
	callResult := parseAnyMap(callResp.Result)
	content, _ := callResult["content"].([]any)
	if len(content) == 0 {
		t.Fatalf("tools/call content missing: %+v", callResult)
	}
	first, _ := content[0].(map[string]any)
	if !strings.Contains(anyString(first["text"]), "hello-mcp") {
		t.Fatalf("tools/call output = %q, want hello-mcp", anyString(first["text"]))
	}

	_ = postRPC(t, httpServer.URL, sessionID, rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("3"),
		Method:  "shutdown",
	})
	if len(sessions.detached) != 1 {
		t.Fatalf("DetachSession called %d times, want 1", len(sessions.detached))
	}
}

func TestStdioInitializeAndEOFDetaches(t *testing.T) {
	adapter := New(Config{})
	sessions := &fakeSessionService{}
	server, err := NewServer(ServerConfig{
		Adapter:      adapter,
		Sessions:     sessions,
		AdminBaseURL: "http://127.0.0.1:50053",
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	var in bytes.Buffer
	writeRPCLine(t, &in, rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "initialize",
		Params:  mustRawJSON(t, map[string]any{"clientInfo": map[string]any{"name": "codex"}}),
	})
	var out bytes.Buffer
	if err := server.ServeStdio(context.Background(), &in, &out); err != nil {
		t.Fatalf("ServeStdio() error = %v", err)
	}
	if len(sessions.created) != 1 {
		t.Fatalf("CreateSession called %d times, want 1", len(sessions.created))
	}
	if len(sessions.detached) != 1 {
		t.Fatalf("DetachSession called %d times, want 1", len(sessions.detached))
	}
}

func postRPC(t *testing.T, url, sessionID string, req rpcRequest) rpcResponse {
	t.Helper()
	return postRPCWithConfirmation(t, url, sessionID, "", req)
}

func postRPCWithConfirmation(t *testing.T, url, sessionID, confirmationID string, req rpcRequest) rpcResponse {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal(request) error = %v", err)
	}
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(sessionID) != "" {
		httpReq.Header.Set(HeaderSessionID, strings.TrimSpace(sessionID))
	}
	if strings.TrimSpace(confirmationID) != "" {
		httpReq.Header.Set(HeaderConfirmationID, strings.TrimSpace(confirmationID))
	}
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		t.Fatalf("http.Do() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var rpcResp rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		t.Fatalf("decode rpc response error = %v", err)
	}
	if rpcResp.Error != nil {
		t.Fatalf("rpc error: %+v", rpcResp.Error)
	}
	return rpcResp
}

func mustRawJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return json.RawMessage(b)
}

func writeRPCLine(t *testing.T, out *bytes.Buffer, req rpcRequest) {
	t.Helper()
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal(req) error = %v", err)
	}
	out.Write(b)
	out.WriteString("\n")
}

func TestParseClientCapabilitiesFailClosedFallback(t *testing.T) {
	caps := parseClientCapabilities(map[string]any{
		"capabilities": map[string]any{},
	}, rpcTransportHTTP)
	if caps.SupportsFallbackID {
		t.Fatalf("supports fallback = true, want false by default")
	}
	withFallback := parseClientCapabilities(map[string]any{
		"capabilities": map[string]any{
			"terminals_fallback_confirmation": true,
		},
	}, rpcTransportHTTP)
	if !withFallback.SupportsFallbackID {
		t.Fatalf("supports fallback = false, want true when explicitly declared")
	}
}

func TestFallbackProbeRequiredBeforeMutatingFallback(t *testing.T) {
	adapter := New(Config{})
	sessions := &fakeSessionService{}
	server, err := NewServer(ServerConfig{
		Adapter:      adapter,
		Sessions:     sessions,
		AdminBaseURL: "http://127.0.0.1:50053",
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	httpServer := httptest.NewServer(http.HandlerFunc(server.ServeHTTP))
	defer httpServer.Close()

	initResp := postRPC(t, httpServer.URL, "", rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "initialize",
		Params: mustRawJSON(t, map[string]any{
			"clientInfo": map[string]any{"name": "codex", "version": "1"},
			"capabilities": map[string]any{
				"terminals_fallback_confirmation": true,
			},
		}),
	})
	initResult := parseAnyMap(initResp.Result)
	sessionID := strings.TrimSpace(anyString(initResult["session_id"]))
	if sessionID == "" {
		t.Fatalf("initialize response missing session_id")
	}
	if got := strings.TrimSpace(anyString(initResult["mutating_capability"])); got != string(MutatingUnavailable) {
		t.Fatalf("mutating_capability = %q, want %q before probe", got, MutatingUnavailable)
	}
	probeToken := strings.TrimSpace(anyString(initResult["fallback_probe_token"]))
	if probeToken == "" {
		t.Fatalf("expected fallback_probe_token")
	}

	mutatingBeforeProbe := postRPC(t, httpServer.URL, sessionID, rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("2"),
		Method:  "tools/call",
		Params: mustRawJSON(t, map[string]any{
			"name":      "app_reload",
			"arguments": map[string]any{"app": "demo"},
		}),
	})
	beforeResult := parseAnyMap(mutatingBeforeProbe.Result)
	beforeMeta := parseAnyMap(beforeResult["_meta"])
	if code := strings.TrimSpace(anyString(beforeMeta["error_code"])); code != "unsupported_client" {
		t.Fatalf("error_code = %q, want unsupported_client before probe", code)
	}

	_ = postRPCWithConfirmation(t, httpServer.URL, sessionID, probeToken, rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("3"),
		Method:  "tools/call",
		Params: mustRawJSON(t, map[string]any{
			"name":      "echo",
			"arguments": map[string]any{"text": "probe"},
		}),
	})

	mutatingAfterProbe := postRPC(t, httpServer.URL, sessionID, rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("4"),
		Method:  "tools/call",
		Params: mustRawJSON(t, map[string]any{
			"name":      "app_reload",
			"arguments": map[string]any{"app": "demo"},
		}),
	})
	afterResult := parseAnyMap(mutatingAfterProbe.Result)
	afterMeta := parseAnyMap(afterResult["_meta"])
	if status := strings.TrimSpace(anyString(afterMeta["status"])); status != "confirmation_required" {
		t.Fatalf("status = %q, want confirmation_required after probe", status)
	}
}

func TestHTTPCancelNotificationCancelsInflightToolCall(t *testing.T) {
	adapter := New(Config{
		OperationalTTL: 5 * time.Second,
	})
	sessions := &fakeSessionService{}
	server, err := NewServer(ServerConfig{
		Adapter:      adapter,
		Sessions:     sessions,
		AdminBaseURL: "http://127.0.0.1:50053",
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	httpServer := httptest.NewServer(http.HandlerFunc(server.ServeHTTP))
	defer httpServer.Close()

	initResp := postRPC(t, httpServer.URL, "", rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "initialize",
		Params:  mustRawJSON(t, map[string]any{"clientInfo": map[string]any{"name": "codex", "version": "1"}}),
	})
	initResult := parseAnyMap(initResp.Result)
	sessionID := strings.TrimSpace(anyString(initResult["session_id"]))
	if sessionID == "" {
		t.Fatalf("initialize response missing session_id")
	}

	done := make(chan rpcResponse, 1)
	go func() {
		done <- postRPC(t, httpServer.URL, sessionID, rpcRequest{
			JSONRPC: "2.0",
			ID:      json.RawMessage("99"),
			Method:  "tools/call",
			Params: mustRawJSON(t, map[string]any{
				"name":      "sleep",
				"arguments": map[string]any{"seconds": "2"},
			}),
		})
	}()

	time.Sleep(100 * time.Millisecond)
	postRPCNotification(t, httpServer.URL, sessionID, rpcRequest{
		JSONRPC: "2.0",
		Method:  "notifications/cancelled",
		Params:  mustRawJSON(t, map[string]any{"requestId": 99}),
	})

	select {
	case resp := <-done:
		result := parseAnyMap(resp.Result)
		meta := parseAnyMap(result["_meta"])
		if code := strings.TrimSpace(anyString(meta["error_code"])); code != "command_failed" {
			t.Fatalf("error_code = %q, want command_failed (canceled)", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for canceled tools/call response")
	}
}

func postRPCNotification(t *testing.T, url, sessionID string, req rpcRequest) {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal(request) error = %v", err)
	}
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(sessionID) != "" {
		httpReq.Header.Set(HeaderSessionID, strings.TrimSpace(sessionID))
	}
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		t.Fatalf("http.Do() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("notification status=%d body=%s", resp.StatusCode, string(raw))
	}
}
