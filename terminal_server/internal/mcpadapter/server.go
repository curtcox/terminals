package mcpadapter

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
	"github.com/curtcox/terminals/terminal_server/internal/repl"
	"github.com/curtcox/terminals/terminal_server/internal/replsession"
)

const (
	HeaderSessionID      = "Mcp-Session-Id"
	HeaderConfirmationID = "Mcp-Confirmation-Id"
)

var (
	// ErrNilAdapter indicates MCP server construction was attempted without an adapter.
	ErrNilAdapter = errors.New("nil mcp adapter")
	// ErrNilSessionService indicates MCP server construction was attempted without a repl session service.
	ErrNilSessionService = errors.New("nil repl session service")
	// ErrMissingSession indicates a request was made without a valid MCP session id.
	ErrMissingSession = errors.New("missing mcp session id")
)

type sessionService interface {
	CreateSession(context.Context, replsession.CreateSessionRequest) (*replsession.CreateSessionResponse, error)
	DetachSession(context.Context, replsession.DetachSessionRequest) (*replsession.DetachSessionResponse, error)
}

// ServerConfig configures JSON-RPC MCP serving over stdio and HTTP.
type ServerConfig struct {
	Adapter      *Adapter
	Sessions     sessionService
	AdminBaseURL string
	Now          func() time.Time
}

// Server hosts MCP JSON-RPC transport glue for the REPL-backed adapter.
type Server struct {
	adapter      *Adapter
	sessions     sessionService
	adminBaseURL string
	now          func() time.Time

	nextID atomic.Uint64

	mu       sync.Mutex
	bindings map[string]sessionBinding
	inflight map[string]context.CancelFunc
	probes   map[string]string
	stdio    map[string]*stdioConnection
}

type sessionBinding struct {
	DeviceID      string
	ReplSessionID string
}

// NewServer constructs an MCP server wrapper around an adapter.
func NewServer(cfg ServerConfig) (*Server, error) {
	if cfg.Adapter == nil {
		return nil, ErrNilAdapter
	}
	if cfg.Sessions == nil {
		return nil, ErrNilSessionService
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &Server{
		adapter:      cfg.Adapter,
		sessions:     cfg.Sessions,
		adminBaseURL: strings.TrimSpace(cfg.AdminBaseURL),
		now:          now,
		bindings:     map[string]sessionBinding{},
		inflight:     map[string]context.CancelFunc{},
		probes:       map[string]string{},
		stdio:        map[string]*stdioConnection{},
	}, nil
}

// ServeHTTP handles Streamable-HTTP style JSON-RPC MCP calls on one endpoint.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req rpcRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid json-rpc request", http.StatusBadRequest)
		return
	}
	sessionID := strings.TrimSpace(r.Header.Get(HeaderSessionID))
	confirmationID := strings.TrimSpace(r.Header.Get(HeaderConfirmationID))
	resp, sessionFromResponse, ok := s.handleRPCRequest(
		r.Context(),
		req,
		rpcTransportHTTP,
		requestContext{SessionID: sessionID, ConfirmationID: confirmationID},
	)
	if sessionFromResponse != "" {
		w.Header().Set(HeaderSessionID, sessionFromResponse)
	}
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// ServeStdio handles stdio JSON-RPC MCP traffic on the current process.
func (s *Server) ServeStdio(ctx context.Context, in io.Reader, out io.Writer) error {
	dec := json.NewDecoder(bufio.NewReader(in))
	conn := newStdioConnection(out, s.nextID.Add(1))
	s.adapter.SetElicitHook(s.elicitViaStdio)
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			if sessionID := conn.getSessionID(); sessionID != "" {
				s.closeSession(context.Background(), sessionID)
			}
			wg.Wait()
			return ctx.Err()
		default:
		}

		var raw map[string]any
		if err := dec.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				if sessionID := conn.getSessionID(); sessionID != "" {
					s.closeSession(context.Background(), sessionID)
				}
				wg.Wait()
				return nil
			}
			wg.Wait()
			return err
		}

		if method := strings.TrimSpace(anyString(raw["method"])); method == "" {
			if conn.routeResponse(raw) {
				continue
			}
			continue
		}
		reqRaw, err := json.Marshal(raw)
		if err != nil {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(reqRaw, &req); err != nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(req.Method), "initialize") {
			resp, sessionFromResponse, ok := s.handleRPCRequest(
				ctx,
				req,
				rpcTransportStdio,
				requestContext{SessionID: conn.getSessionID(), Stdio: conn},
			)
			if sessionFromResponse != "" {
				conn.setSessionID(sessionFromResponse)
			}
			if ok {
				_ = conn.writeRPCResponse(resp)
			}
			continue
		}
		wg.Add(1)
		go func(req rpcRequest) {
			defer wg.Done()
			resp, sessionFromResponse, ok := s.handleRPCRequest(
				ctx,
				req,
				rpcTransportStdio,
				requestContext{SessionID: conn.getSessionID(), Stdio: conn},
			)
			if sessionFromResponse != "" {
				conn.setSessionID(sessionFromResponse)
			}
			if !ok {
				return
			}
			_ = conn.writeRPCResponse(resp)
		}(req)
	}
}

type rpcTransport string

const (
	rpcTransportStdio rpcTransport = "stdio"
	rpcTransportHTTP  rpcTransport = "http"
)

type requestContext struct {
	SessionID      string
	ConfirmationID string
	Stdio          *stdioConnection
}

func (s *Server) handleRPCRequest(
	ctx context.Context,
	req rpcRequest,
	transport rpcTransport,
	rc requestContext,
) (rpcResponse, string, bool) {
	id := req.ID
	method := strings.TrimSpace(req.Method)
	if method == "" {
		return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: -32600, Message: "missing method"}}, "", hasRPCID(id)
	}
	switch method {
	case "notifications/cancelled":
		params := parseAnyMap(req.Params)
		requestID := requestIDFromParams(params)
		if requestID != "" && strings.TrimSpace(rc.SessionID) != "" {
			s.cancelInflight(rc.SessionID, requestID)
		}
		return rpcResponse{}, rc.SessionID, false
	case "notifications/initialized":
		return rpcResponse{}, "", false
	case "initialize":
		params := parseAnyMap(req.Params)
		clientIdentity := parseClientIdentity(params)
		caps := parseClientCapabilities(params, transport)
		requireFallbackProbe := !caps.SupportsElicitation && caps.SupportsFallbackID
		if requireFallbackProbe {
			caps.SupportsFallbackID = false
		}
		sessionID := strings.TrimSpace(rc.SessionID)
		if sessionID == "" {
			sessionID = strings.TrimSpace(anyString(params["session_id"]))
		}
		info, err := s.openSession(ctx, sessionID, clientIdentity, caps)
		if err != nil {
			return rpcResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &rpcError{Code: -32603, Message: err.Error()},
			}, "", hasRPCID(id)
		}
		result := map[string]any{
			"protocolVersion": "2025-03-26",
			"serverInfo": map[string]any{
				"name":    "terminals-mcp",
				"version": "1",
			},
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"session_id":          info.SessionID,
			"mutating_capability": string(info.Capability),
			"registry_version":    s.adapter.RegistryVersion(),
		}
		if requireFallbackProbe {
			token := s.issueFallbackProbe(info.SessionID)
			result["fallback_probe_required"] = true
			result["fallback_probe_token"] = token
		}
		if transport == rpcTransportStdio && rc.Stdio != nil {
			s.mu.Lock()
			s.stdio[info.SessionID] = rc.Stdio
			s.mu.Unlock()
		}
		return rpcResponse{JSONRPC: "2.0", ID: id, Result: result}, info.SessionID, hasRPCID(id)
	case "tools/list":
		tools := s.adapter.Tools()
		out := make([]map[string]any, 0, len(tools))
		for _, tool := range tools {
			out = append(out, map[string]any{
				"name":        tool.Name,
				"description": tool.Description,
				"inputSchema": tool.ArgumentsSchema,
			})
		}
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result:  map[string]any{"tools": out, "registry_version": s.adapter.RegistryVersion()},
		}, rc.SessionID, hasRPCID(id)
	case "tools/call":
		sessionID := strings.TrimSpace(rc.SessionID)
		if sessionID == "" {
			return rpcResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &rpcError{Code: -32001, Message: ErrMissingSession.Error()},
			}, "", hasRPCID(id)
		}
		params := parseAnyMap(req.Params)
		toolName := strings.TrimSpace(anyString(params["name"]))
		args := parseAnyMap(params["arguments"])
		confirmationID := strings.TrimSpace(rc.ConfirmationID)
		if confirmationID == "" {
			meta := parseAnyMap(params["_meta"])
			confirmationID = strings.TrimSpace(anyString(meta["terminals_confirmation_id"]))
		}
		if confirmed := s.confirmFallbackProbe(sessionID, confirmationID); confirmed {
			confirmationID = ""
		}
		callCtx := ctx
		requestID := rpcIDString(id)
		if requestID != "" {
			var cancel context.CancelFunc
			callCtx, cancel = context.WithCancel(ctx)
			s.addInflight(sessionID, requestID, cancel)
			defer func() {
				cancel()
				s.removeInflight(sessionID, requestID)
			}()
		}
		stream := func(string) error { return nil }
		if transport == rpcTransportStdio && rc.Stdio != nil && requestID != "" {
			stream = func(chunk string) error {
				if strings.TrimSpace(chunk) == "" {
					return nil
				}
				return rc.Stdio.sendNotification("notifications/tools/call_output", map[string]any{
					"session_id": sessionID,
					"request_id": requestID,
					"chunk":      chunk,
				})
			}
		}
		toolResp, err := s.adapter.CallToolStream(callCtx, CallToolRequest{
			SessionID:          sessionID,
			ToolName:           toolName,
			Arguments:          args,
			MetaConfirmationID: confirmationID,
		}, stream)
		if err != nil {
			return rpcResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &rpcError{Code: -32002, Message: err.Error()},
			}, sessionID, hasRPCID(id)
		}
		s.emitCallLog(ctx, sessionID, toolName, toolResp)
		result := toolResult(toolResp)
		return rpcResponse{JSONRPC: "2.0", ID: id, Result: result}, sessionID, hasRPCID(id)
	case "shutdown":
		sessionID := strings.TrimSpace(rc.SessionID)
		if sessionID != "" {
			s.closeSession(ctx, sessionID)
		}
		return rpcResponse{JSONRPC: "2.0", ID: id, Result: map[string]any{"status": "ok"}}, "", hasRPCID(id)
	default:
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &rpcError{Code: -32601, Message: "method not found"},
		}, rc.SessionID, hasRPCID(id)
	}
}

func (s *Server) openSession(ctx context.Context, requestedID, clientIdentity string, caps ClientCapabilities) (SessionInfo, error) {
	sessionID := strings.TrimSpace(requestedID)
	if sessionID == "" {
		sessionID = fmt.Sprintf("mcp-%d", s.nextID.Add(1))
	}
	info := s.adapter.OpenSession(sessionID, clientIdentity, caps)
	deviceID := "mcp-device-" + sanitizeID(sessionID)
	createReq := ReplSessionCreateRequest(deviceID, "mcp", info)
	createReq.ReplAdminURL = s.adminBaseURL
	created, err := s.sessions.CreateSession(ctx, createReq)
	if err != nil {
		s.adapter.CloseSession(sessionID)
		return SessionInfo{}, err
	}
	s.mu.Lock()
	s.bindings[sessionID] = sessionBinding{
		DeviceID:      deviceID,
		ReplSessionID: created.Session.ID,
	}
	s.mu.Unlock()
	return info, nil
}

func (s *Server) closeSession(ctx context.Context, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	s.cancelAllInflightForSession(sessionID)
	s.mu.Lock()
	binding, ok := s.bindings[sessionID]
	if ok {
		delete(s.bindings, sessionID)
	}
	delete(s.probes, sessionID)
	delete(s.stdio, sessionID)
	s.mu.Unlock()
	if ok {
		_, _ = s.sessions.DetachSession(ctx, replsession.DetachSessionRequest{
			SessionID: binding.ReplSessionID,
			DeviceID:  binding.DeviceID,
		})
	}
	s.adapter.CloseSession(sessionID)
}

func (s *Server) issueFallbackProbe(sessionID string) string {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ""
	}
	token := fmt.Sprintf("probe_%d", s.nextID.Add(1))
	s.mu.Lock()
	s.probes[sessionID] = token
	s.mu.Unlock()
	return token
}

func (s *Server) confirmFallbackProbe(sessionID, suppliedToken string) bool {
	sessionID = strings.TrimSpace(sessionID)
	suppliedToken = strings.TrimSpace(suppliedToken)
	if sessionID == "" || suppliedToken == "" {
		return false
	}
	s.mu.Lock()
	expected := strings.TrimSpace(s.probes[sessionID])
	if expected == "" || expected != suppliedToken {
		s.mu.Unlock()
		return false
	}
	delete(s.probes, sessionID)
	s.mu.Unlock()
	s.adapter.SetSessionCapability(sessionID, MutatingViaFallback)
	return true
}

func (s *Server) addInflight(sessionID, requestID string, cancel context.CancelFunc) {
	if cancel == nil {
		return
	}
	key := inflightKey(sessionID, requestID)
	if key == "" {
		return
	}
	s.mu.Lock()
	s.inflight[key] = cancel
	s.mu.Unlock()
}

func (s *Server) removeInflight(sessionID, requestID string) {
	key := inflightKey(sessionID, requestID)
	if key == "" {
		return
	}
	s.mu.Lock()
	delete(s.inflight, key)
	s.mu.Unlock()
}

func (s *Server) cancelInflight(sessionID, requestID string) {
	key := inflightKey(sessionID, requestID)
	if key == "" {
		return
	}
	s.mu.Lock()
	cancel := s.inflight[key]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Server) cancelAllInflightForSession(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	s.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(s.inflight))
	prefix := sessionID + "|"
	for key, cancel := range s.inflight {
		if strings.HasPrefix(key, prefix) {
			cancels = append(cancels, cancel)
			delete(s.inflight, key)
		}
	}
	s.mu.Unlock()
	for _, cancel := range cancels {
		if cancel != nil {
			cancel()
		}
	}
}

func inflightKey(sessionID, requestID string) string {
	sessionID = strings.TrimSpace(sessionID)
	requestID = strings.TrimSpace(requestID)
	if sessionID == "" || requestID == "" {
		return ""
	}
	return sessionID + "|" + requestID
}

func (s *Server) emitCallLog(ctx context.Context, sessionID, toolName string, resp CallToolResponse) {
	if strings.TrimSpace(toolName) == "" {
		return
	}
	status := strings.TrimSpace(resp.Status)
	if status == "" {
		status = "unknown"
	}
	level := slog.LevelInfo
	if status == "error" || status == "rejected" {
		level = slog.LevelWarn
	}
	attrs := []slog.Attr{
		slog.String("session_origin", "mcp"),
		slog.String("session_id", strings.TrimSpace(sessionID)),
		slog.String("tool", toolName),
		slog.String("status", status),
	}
	if resp.ErrorCode != "" {
		attrs = append(attrs, slog.String("error_code", resp.ErrorCode))
	}
	if resp.Classification != repl.CommandClassification("") {
		attrs = append(attrs, slog.String("classification", string(resp.Classification)))
	}
	eventlog.Emit(ctx, "mcp.tool.call", level, "mcp tool call", attrs...)
}

func parseClientCapabilities(params map[string]any, transport rpcTransport) ClientCapabilities {
	caps := parseAnyMap(params["capabilities"])
	supportsElicitation := capabilityEnabled(caps["elicitation"]) || capabilityEnabled(caps["mcp_elicitation"])
	supportsFallback := capabilityEnabled(caps["terminals_fallback_confirmation"]) ||
		capabilityEnabled(caps["fallback_confirmation"]) ||
		capabilityEnabled(caps["confirmation_id"])
	if !supportsFallback {
		// Probe by default on supported transports so clients that can carry
		// confirmation IDs do not need a custom initialize capability flag.
		supportsFallback = true
	}
	if strings.EqualFold(string(transport), string(rpcTransportHTTP)) {
		// HTTP transport in this server is request/response only, so server-originated
		// elicitation requests cannot be round-tripped. Keep mutating fail-closed unless
		// fallback confirmation is available.
		return ClientCapabilities{SupportsElicitation: false, SupportsFallbackID: supportsFallback}
	}
	return ClientCapabilities{SupportsElicitation: supportsElicitation, SupportsFallbackID: supportsFallback}
}

func capabilityEnabled(v any) bool {
	switch typed := v.(type) {
	case bool:
		return typed
	case string:
		raw := strings.ToLower(strings.TrimSpace(typed))
		return raw == "true" || raw == "1" || raw == "yes"
	case map[string]any:
		return true
	case []any:
		return len(typed) > 0
	case json.RawMessage:
		var decoded any
		if err := json.Unmarshal(typed, &decoded); err != nil {
			return false
		}
		return capabilityEnabled(decoded)
	default:
		return false
	}
}

func (s *Server) elicitViaStdio(ctx context.Context, req ElicitRequest) (ElicitResponse, error) {
	s.mu.Lock()
	conn := s.stdio[strings.TrimSpace(req.SessionID)]
	s.mu.Unlock()
	if conn == nil {
		return ElicitResponse{Approved: false}, nil
	}
	result, err := conn.sendRequestAndAwait(ctx, "elicitation/create", map[string]any{
		"title":             "Approve mutating command",
		"tool_name":         req.ToolName,
		"rendered_command":  req.RenderedCommand,
		"classification":    string(req.Classification),
		"session_id":        req.SessionID,
		"approval_required": true,
	})
	if err != nil {
		return ElicitResponse{Approved: false}, nil
	}
	if anyBool(result["approved"]) {
		return ElicitResponse{Approved: true}, nil
	}
	action := strings.ToLower(strings.TrimSpace(anyString(result["action"])))
	switch action {
	case "approve", "approved", "accept", "accepted", "yes":
		return ElicitResponse{Approved: true}, nil
	default:
		return ElicitResponse{Approved: false}, nil
	}
}

func parseClientIdentity(params map[string]any) string {
	info := parseAnyMap(params["clientInfo"])
	if len(info) == 0 {
		return ""
	}
	name := strings.TrimSpace(anyString(info["name"]))
	version := strings.TrimSpace(anyString(info["version"]))
	if name == "" {
		return ""
	}
	if version == "" {
		return name
	}
	return name + "/" + version
}

func parseAnyMap(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	switch typed := v.(type) {
	case map[string]any:
		return typed
	case json.RawMessage:
		var out map[string]any
		if err := json.Unmarshal(typed, &out); err == nil && out != nil {
			return out
		}
	}
	return map[string]any{}
}

func requestIDFromParams(params map[string]any) string {
	if len(params) == 0 {
		return ""
	}
	if reqID := rpcIDStringValue(params["requestId"]); reqID != "" {
		return reqID
	}
	if reqID := rpcIDStringValue(params["request_id"]); reqID != "" {
		return reqID
	}
	return rpcIDStringValue(params["id"])
}

func rpcIDString(id json.RawMessage) string {
	var decoded any
	if err := json.Unmarshal(id, &decoded); err != nil {
		return strings.TrimSpace(string(id))
	}
	return rpcIDStringValue(decoded)
}

func rpcIDStringValue(v any) string {
	switch typed := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%.0f", typed))
	case int:
		return strings.TrimSpace(fmt.Sprintf("%d", typed))
	case int64:
		return strings.TrimSpace(fmt.Sprintf("%d", typed))
	case json.RawMessage:
		return rpcIDString(typed)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func toolResult(resp CallToolResponse) map[string]any {
	meta := map[string]any{
		"status":               resp.Status,
		"error_code":           resp.ErrorCode,
		"error_message":        resp.ErrorMessage,
		"confirmation_id":      resp.ConfirmationID,
		"classification":       string(resp.Classification),
		"rendered_command":     resp.RenderedCommand,
		"expires_at_unix_ms":   resp.ExpiresAt.UnixMilli(),
		"raw_tool_metadata":    resp.Metadata,
		"confirmation_expired": (!resp.ExpiresAt.IsZero()) && time.Now().UTC().After(resp.ExpiresAt.UTC()),
	}
	text := strings.TrimSpace(resp.Output)
	if text == "" {
		payload := map[string]any{
			"status":           resp.Status,
			"error_code":       resp.ErrorCode,
			"error_message":    resp.ErrorMessage,
			"confirmation_id":  resp.ConfirmationID,
			"rendered_command": resp.RenderedCommand,
		}
		if len(resp.Metadata) > 0 {
			payload["metadata"] = resp.Metadata
		}
		if !resp.ExpiresAt.IsZero() {
			payload["expires_at"] = resp.ExpiresAt.UTC().Format(time.RFC3339)
		}
		body, _ := json.Marshal(payload)
		text = string(body)
	}
	result := map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": text,
			},
		},
		"_meta": meta,
	}
	if resp.Status == "error" || resp.Status == "rejected" || resp.ErrorCode != "" {
		result["isError"] = true
	}
	return result
}

func sanitizeID(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "0"
	}
	var b strings.Builder
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		b.WriteString("_")
	}
	out := b.String()
	if out == "" {
		return "0"
	}
	return out
}

type stdioConnection struct {
	enc       *json.Encoder
	writeMu   sync.Mutex
	pendingMu sync.Mutex
	pending   map[string]chan map[string]any
	nextID    atomic.Uint64
	sessionMu sync.Mutex
	sessionID string
}

func newStdioConnection(out io.Writer, seed uint64) *stdioConnection {
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	conn := &stdioConnection{
		enc:     enc,
		pending: map[string]chan map[string]any{},
	}
	conn.nextID.Store(seed)
	return conn
}

func (c *stdioConnection) setSessionID(sessionID string) {
	c.sessionMu.Lock()
	c.sessionID = strings.TrimSpace(sessionID)
	c.sessionMu.Unlock()
}

func (c *stdioConnection) getSessionID() string {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()
	return c.sessionID
}

func (c *stdioConnection) writeRPCResponse(resp rpcResponse) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.enc.Encode(resp)
}

func (c *stdioConnection) sendNotification(method string, params map[string]any) error {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.enc.Encode(payload)
}

func (c *stdioConnection) sendRequestAndAwait(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	id := strconv.FormatUint(c.nextID.Add(1), 10)
	ch := make(chan map[string]any, 1)
	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()
	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
	}()

	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	c.writeMu.Lock()
	err := c.enc.Encode(payload)
	c.writeMu.Unlock()
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-ch:
		if errPayload, ok := resp["error"].(map[string]any); ok && len(errPayload) > 0 {
			return nil, errors.New(strings.TrimSpace(anyString(errPayload["message"])))
		}
		return parseAnyMap(resp["result"]), nil
	}
}

func (c *stdioConnection) routeResponse(raw map[string]any) bool {
	id := strings.TrimSpace(anyString(raw["id"]))
	if id == "" {
		return false
	}
	c.pendingMu.Lock()
	ch := c.pending[id]
	c.pendingMu.Unlock()
	if ch == nil {
		return false
	}
	select {
	case ch <- raw:
	default:
	}
	return true
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func hasRPCID(id json.RawMessage) bool {
	return len(strings.TrimSpace(string(id))) > 0
}
