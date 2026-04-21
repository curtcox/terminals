package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/mcpadapter"
)

func TestProxyMCPStdioBridgesFallbackConfirmationThroughElicitation(t *testing.T) {
	var (
		mu                    sync.Mutex
		httpCallCount         int
		sawFallbackCapability bool
		sawConfirmationID     bool
	)
	mcpHTTP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		method := strings.TrimSpace(anyString(req["method"]))

		mu.Lock()
		httpCallCount++
		currentCall := httpCallCount
		mu.Unlock()

		if sid := strings.TrimSpace(r.Header.Get(mcpadapter.HeaderSessionID)); sid != "" {
			w.Header().Set(mcpadapter.HeaderSessionID, sid)
		}

		switch {
		case method == "initialize":
			caps := parseAnyMap(parseAnyMap(req["params"])["capabilities"])
			sawFallbackCapability = anyBool(caps["terminals_fallback_confirmation"])
			w.Header().Set(mcpadapter.HeaderSessionID, "mcp-proxy-1")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]any{
					"session_id":          "mcp-proxy-1",
					"mutating_capability": "mutating_via_fallback",
				},
			})
			return
		case method == "tools/call" && currentCall == 2:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]any{
					"_meta": map[string]any{
						"status":          "confirmation_required",
						"confirmation_id": "cnf-1",
					},
				},
			})
			return
		case method == "tools/call" && currentCall == 3:
			sawConfirmationID = strings.TrimSpace(r.Header.Get(mcpadapter.HeaderConfirmationID)) == "cnf-1"
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]any{
					"content": []map[string]any{{"type": "text", "text": "ok"}},
					"_meta":   map[string]any{"status": "ok"},
				},
			})
			return
		default:
			t.Fatalf("unexpected request: method=%q call=%d", method, currentCall)
		}
	}))
	defer mcpHTTP.Close()

	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- proxyMCPStdio(ctx, inR, outW, mcpHTTP.URL)
	}()

	enc := json.NewEncoder(inW)
	dec := json.NewDecoder(outR)
	readMsg := func() map[string]any {
		t.Helper()
		msgCh := make(chan map[string]any, 1)
		errCh := make(chan error, 1)
		go func() {
			var msg map[string]any
			if err := dec.Decode(&msg); err != nil {
				errCh <- err
				return
			}
			msgCh <- msg
		}()
		select {
		case msg := <-msgCh:
			return msg
		case err := <-errCh:
			t.Fatalf("decode proxy output: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout waiting for proxy output")
		}
		return nil
	}

	if err := enc.Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"capabilities": map[string]any{"elicitation": true},
		},
	}); err != nil {
		t.Fatalf("encode initialize: %v", err)
	}
	initResp := readMsg()
	initResult := parseAnyMap(initResp["result"])
	if got := strings.TrimSpace(anyString(initResult["mutating_capability"])); got != "mutating_via_elicitation" {
		t.Fatalf("mutating_capability=%q, want mutating_via_elicitation", got)
	}

	if err := enc.Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "app_reload",
			"arguments": map[string]any{"app": "demo"},
		},
	}); err != nil {
		t.Fatalf("encode tools/call: %v", err)
	}

	elicitReq := readMsg()
	if method := strings.TrimSpace(anyString(elicitReq["method"])); method != "elicitation/create" {
		t.Fatalf("method=%q, want elicitation/create", method)
	}
	if err := enc.Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      elicitReq["id"],
		"result":  map[string]any{"approved": true},
	}); err != nil {
		t.Fatalf("encode elicitation response: %v", err)
	}

	callResp := readMsg()
	result := parseAnyMap(callResp["result"])
	meta := parseAnyMap(result["_meta"])
	if got := strings.TrimSpace(anyString(meta["status"])); got != "ok" {
		t.Fatalf("status=%q, want ok", got)
	}

	if err := inW.Close(); err != nil {
		t.Fatalf("close input: %v", err)
	}
	cancel()

	select {
	case err := <-done:
		if err != nil && err != context.Canceled {
			t.Fatalf("proxyMCPStdio() error=%v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for proxy to finish")
	}

	mu.Lock()
	defer mu.Unlock()
	if !sawFallbackCapability {
		t.Fatalf("initialize did not inject terminals_fallback_confirmation capability")
	}
	if !sawConfirmationID {
		t.Fatalf("approved replay missing %s header", mcpadapter.HeaderConfirmationID)
	}
}

func parseAnyMap(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	out, ok := v.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return out
}

func anyString(v any) string {
	s, _ := v.(string)
	return s
}

func anyBool(v any) bool {
	b, _ := v.(bool)
	return b
}

func TestRegisterAckMetadataIncludesServerBuildInfo(t *testing.T) {
	t.Setenv("TERMINALS_BUILD_SHA", "ea99b3f38658")
	t.Setenv("TERMINALS_BUILD_DATE", "2026-04-21T14:55:56Z")

	metadata := registerAckMetadata("http://home.local:50052/photo-frame")
	if got := metadata[registerMetadataPhotoFrameAssetBaseURLKey]; got != "http://home.local:50052/photo-frame" {
		t.Fatalf("photo frame metadata = %q, want configured value", got)
	}
	if got := metadata[registerMetadataServerBuildSHAKey]; got != "ea99b3f38658" {
		t.Fatalf("server build sha metadata = %q, want ea99b3f38658", got)
	}
	if got := metadata[registerMetadataServerBuildDateKey]; got != "2026-04-21T14:55:56Z" {
		t.Fatalf("server build date metadata = %q, want 2026-04-21T14:55:56Z", got)
	}
}

func TestRegisterAckMetadataDefaultsUnknownBuildInfo(t *testing.T) {
	t.Setenv("TERMINALS_BUILD_SHA", "")
	t.Setenv("TERMINALS_BUILD_DATE", "")

	metadata := registerAckMetadata("http://home.local:50052/photo-frame")
	if got := metadata[registerMetadataServerBuildSHAKey]; got != "unknown" {
		t.Fatalf("server build sha metadata = %q, want unknown", got)
	}
	if got := metadata[registerMetadataServerBuildDateKey]; got != "unknown" {
		t.Fatalf("server build date metadata = %q, want unknown", got)
	}
}
