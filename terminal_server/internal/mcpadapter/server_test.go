package mcpadapter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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
		t.Fatalf("supports fallback = true, want false when no fallback carrier is declared")
	}
	withFallback := parseClientCapabilities(map[string]any{
		"capabilities": map[string]any{
			"terminals_fallback_confirmation": true,
		},
	}, rpcTransportHTTP)
	if !withFallback.SupportsFallbackID {
		t.Fatalf("supports fallback = false, want true when explicitly declared")
	}
	withElicitation := parseClientCapabilities(map[string]any{
		"capabilities": map[string]any{
			"elicitation": true,
		},
	}, rpcTransportHTTP)
	if withElicitation.SupportsElicitation {
		t.Fatalf("supports elicitation = true, want false for http request/response transport")
	}
}

func TestHTTPToolCallCanStreamChunksOverSSE(t *testing.T) {
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
		Params:  mustRawJSON(t, map[string]any{"clientInfo": map[string]any{"name": "codex"}}),
	})
	sessionID := strings.TrimSpace(anyString(parseAnyMap(initResp.Result)["session_id"]))
	if sessionID == "" {
		t.Fatalf("initialize response missing session_id")
	}

	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("2"),
		Method:  "tools/call",
		Params: mustRawJSON(t, map[string]any{
			"name":      "echo",
			"arguments": map[string]any{"text": "http-stream"},
		}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(request) error = %v", err)
	}
	httpReq, err := http.NewRequest(http.MethodPost, httpServer.URL, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set(HeaderSessionID, sessionID)
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		t.Fatalf("http.Do() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("content-type = %q, want text/event-stream", ct)
	}

	scanner := bufio.NewScanner(resp.Body)
	type sseEvent struct {
		Event string
		Data  string
	}
	events := make([]sseEvent, 0, 4)
	current := sseEvent{}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if strings.TrimSpace(current.Event) != "" {
				events = append(events, current)
			}
			current = sseEvent{}
			continue
		}
		if strings.HasPrefix(line, "event:") {
			current.Event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			current.Data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan event stream: %v", err)
	}

	sawChunk := false
	sawFinal := false
	for _, event := range events {
		switch event.Event {
		case "notifications/tools/call_output":
			var payload map[string]any
			if err := json.Unmarshal([]byte(event.Data), &payload); err != nil {
				t.Fatalf("decode stream chunk event: %v", err)
			}
			if strings.Contains(anyString(payload["chunk"]), "http-stream") {
				sawChunk = true
			}
		case "jsonrpc/response":
			var rpcResp rpcResponse
			if err := json.Unmarshal([]byte(event.Data), &rpcResp); err != nil {
				t.Fatalf("decode final rpc response event: %v", err)
			}
			result := parseAnyMap(rpcResp.Result)
			content, _ := result["content"].([]any)
			if len(content) == 0 {
				t.Fatalf("missing content in final rpc response: %+v", result)
			}
			text := anyString(parseAnyMap(content[0])["text"])
			if !strings.Contains(text, "http-stream") {
				t.Fatalf("final rpc response missing stream text: %q", text)
			}
			sawFinal = true
		}
	}
	if !sawChunk {
		t.Fatalf("missing stream chunk notification event: %+v", events)
	}
	if !sawFinal {
		t.Fatalf("missing final jsonrpc/response event: %+v", events)
	}
}

func TestReplDiscoveryResultIncludesMetadataInVisiblePayload(t *testing.T) {
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
		Params:  mustRawJSON(t, map[string]any{"clientInfo": map[string]any{"name": "codex"}}),
	})
	sessionID := strings.TrimSpace(anyString(parseAnyMap(initResp.Result)["session_id"]))
	if sessionID == "" {
		t.Fatalf("initialize response missing session_id")
	}

	callResp := postRPC(t, httpServer.URL, sessionID, rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("2"),
		Method:  "tools/call",
		Params: mustRawJSON(t, map[string]any{
			"name":      ToolReplComplete,
			"arguments": map[string]any{"prefix": "sessions ", "limit": 5},
		}),
	})
	callResult := parseAnyMap(callResp.Result)
	content, _ := callResult["content"].([]any)
	if len(content) == 0 {
		t.Fatalf("tools/call content missing")
	}
	text := anyString(parseAnyMap(content[0])["text"])
	var payload map[string]any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("discovery payload is not json: %v text=%q", err, text)
	}
	metadata := parseAnyMap(payload["metadata"])
	if len(metadata) == 0 {
		t.Fatalf("visible payload missing metadata: %v", payload)
	}
	matches, _ := metadata["matches"].([]any)
	if len(matches) == 0 {
		if asStrings, ok := metadata["matches"].([]string); ok {
			if len(asStrings) == 0 {
				t.Fatalf("metadata.matches missing completion values: %v", metadata)
			}
			return
		}
		t.Fatalf("metadata.matches missing completion values: %v", metadata)
	}
}

func TestMutatingFallbackIsAvailableWithoutProbeHandshake(t *testing.T) {
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
	if got := strings.TrimSpace(anyString(initResult["mutating_capability"])); got != string(MutatingViaFallback) {
		t.Fatalf("mutating_capability = %q, want %q", got, MutatingViaFallback)
	}

	mutatingCall := postRPC(t, httpServer.URL, sessionID, rpcRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage("2"),
		Method:  "tools/call",
		Params: mustRawJSON(t, map[string]any{
			"name":      "app_reload",
			"arguments": map[string]any{"app": "demo"},
		}),
	})
	result := parseAnyMap(mutatingCall.Result)
	meta := parseAnyMap(result["_meta"])
	if status := strings.TrimSpace(anyString(meta["status"])); status != "confirmation_required" {
		t.Fatalf("status = %q, want confirmation_required", status)
	}
	if confirmationID := strings.TrimSpace(anyString(meta["confirmation_id"])); confirmationID == "" {
		t.Fatalf("missing confirmation_id in fallback response")
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

func TestStdioElicitationRoundTrip(t *testing.T) {
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

	serverInR, serverInW := io.Pipe()
	serverOutR, serverOutW := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var serveErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		serveErr = server.ServeStdio(ctx, serverInR, serverOutW)
	}()

	enc := json.NewEncoder(serverInW)
	dec := json.NewDecoder(serverOutR)
	msgs := make(chan map[string]any, 16)
	readErrs := make(chan error, 1)
	go func() {
		for {
			var msg map[string]any
			if err := dec.Decode(&msg); err != nil {
				readErrs <- err
				close(msgs)
				return
			}
			msgs <- msg
		}
	}()

	writeClientRPC := func(req map[string]any) {
		t.Helper()
		if err := enc.Encode(req); err != nil {
			t.Fatalf("encode client rpc: %v", err)
		}
	}
	readServerRPC := func() map[string]any {
		t.Helper()
		select {
		case msg, ok := <-msgs:
			if !ok {
				select {
				case err := <-readErrs:
					t.Fatalf("decode server rpc: %v", err)
				default:
					t.Fatalf("decode server rpc: closed")
				}
			}
			return msg
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout waiting for server rpc")
		}
		return nil
	}

	writeClientRPC(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"clientInfo": map[string]any{"name": "codex", "version": "1"},
			"capabilities": map[string]any{
				"elicitation": true,
			},
		},
	})
	initResp := readServerRPC()
	initResult := parseAnyMap(initResp["result"])
	sessionID := strings.TrimSpace(anyString(initResult["session_id"]))
	if sessionID == "" {
		t.Fatalf("missing session id in initialize response: %+v", initResp)
	}
	if capValue := strings.TrimSpace(anyString(initResult["mutating_capability"])); capValue != string(MutatingViaElicitation) {
		t.Fatalf("mutating_capability=%q, want %q", capValue, MutatingViaElicitation)
	}

	writeClientRPC(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "app_reload",
			"arguments": map[string]any{"app": "demo"},
		},
	})

	elicitReq := readServerRPC()
	if method := strings.TrimSpace(anyString(elicitReq["method"])); method != "elicitation/create" {
		t.Fatalf("method=%q, want elicitation/create", method)
	}
	elicitID := elicitReq["id"]
	if elicitID == nil {
		t.Fatalf("elicitation request missing id: %+v", elicitReq)
	}
	writeClientRPC(map[string]any{
		"jsonrpc": "2.0",
		"id":      elicitID,
		"result": map[string]any{
			"approved": true,
		},
	})

	callResp := readServerRPC()
	callResult := parseAnyMap(callResp["result"])
	meta := parseAnyMap(callResult["_meta"])
	if code := strings.TrimSpace(anyString(meta["error_code"])); code == "approval_rejected" || code == "elicit_unavailable" {
		t.Fatalf("unexpected approval failure meta=%+v", meta)
	}

	_ = serverInW.Close()
	_ = serverOutR.Close()
	wg.Wait()
	if serveErr != nil && serveErr != context.Canceled {
		t.Fatalf("ServeStdio() error = %v", serveErr)
	}
}

func TestStdioOperationalStreamNotification(t *testing.T) {
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

	serverInR, serverInW := io.Pipe()
	serverOutR, serverOutW := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- server.ServeStdio(ctx, serverInR, serverOutW)
	}()

	enc := json.NewEncoder(serverInW)
	dec := json.NewDecoder(serverOutR)
	msgs := make(chan map[string]any, 16)
	readErrs := make(chan error, 1)
	go func() {
		for {
			var msg map[string]any
			if err := dec.Decode(&msg); err != nil {
				readErrs <- err
				close(msgs)
				return
			}
			msgs <- msg
		}
	}()
	send := func(msg map[string]any) {
		t.Helper()
		if err := enc.Encode(msg); err != nil {
			t.Fatalf("encode client rpc: %v", err)
		}
	}
	recv := func() map[string]any {
		t.Helper()
		select {
		case msg, ok := <-msgs:
			if !ok {
				select {
				case err := <-readErrs:
					t.Fatalf("decode server rpc: %v", err)
				default:
					t.Fatalf("decode server rpc: closed")
				}
			}
			return msg
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout waiting for server rpc")
		}
		return nil
	}

	send(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{"clientInfo": map[string]any{"name": "codex"}},
	})
	_ = recv() // initialize response

	send(map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "echo",
			"arguments": map[string]any{"text": "stream-notify"},
		},
	})

	gotFinal := false
	sawChunk := false
	for i := 0; i < 5; i++ {
		msg := recv()
		if method := strings.TrimSpace(anyString(msg["method"])); method == "notifications/tools/call_output" {
			params := parseAnyMap(msg["params"])
			if strings.TrimSpace(anyString(params["request_id"])) != "3" {
				t.Fatalf("notification request_id=%q, want 3", anyString(params["request_id"]))
			}
			if !strings.Contains(anyString(params["chunk"]), "stream-notify") {
				t.Fatalf("notification chunk=%q missing stream output", anyString(params["chunk"]))
			}
			sawChunk = true
			continue
		}
		if strings.TrimSpace(anyString(msg["id"])) == "3" {
			gotFinal = true
			break
		}
		if _, ok := msg["result"]; ok {
			gotFinal = true
			break
		}
	}
	if !gotFinal {
		t.Fatalf("did not receive final tools/call response")
	}
	if !sawChunk {
		t.Fatalf("did not receive streamed output notification")
	}

	_ = serverInW.Close()
	_ = serverOutR.Close()
	if err := <-done; err != nil && err != context.Canceled {
		t.Fatalf("ServeStdio() error = %v", err)
	}
}
