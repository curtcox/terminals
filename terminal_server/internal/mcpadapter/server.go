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
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)

	var connectionSessionID string
	for {
		select {
		case <-ctx.Done():
			if connectionSessionID != "" {
				s.closeSession(context.Background(), connectionSessionID)
			}
			return ctx.Err()
		default:
		}

		var req rpcRequest
		if err := dec.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				if connectionSessionID != "" {
					s.closeSession(context.Background(), connectionSessionID)
				}
				return nil
			}
			return err
		}
		resp, sessionFromResponse, ok := s.handleRPCRequest(
			ctx,
			req,
			rpcTransportStdio,
			requestContext{SessionID: connectionSessionID},
		)
		if sessionFromResponse != "" {
			connectionSessionID = sessionFromResponse
		}
		if !ok {
			continue
		}
		if err := enc.Encode(resp); err != nil {
			return err
		}
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
		toolResp, err := s.adapter.CallTool(callCtx, CallToolRequest{
			SessionID:          sessionID,
			ToolName:           toolName,
			Arguments:          args,
			MetaConfirmationID: confirmationID,
		})
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
	s.mu.Unlock()
	if ok {
		_, _ = s.sessions.DetachSession(ctx, replsession.DetachSessionRequest{
			SessionID: binding.ReplSessionID,
			DeviceID:  binding.DeviceID,
		})
	}
	s.adapter.CloseSession(sessionID)
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
	supportsElicitation := anyBool(caps["elicitation"])
	// Fail closed by default: fallback confirmation must be explicitly declared
	// by the client or mutating commands remain unavailable for the session.
	supportsFallback := anyBool(caps["terminals_fallback_confirmation"])
	if strings.EqualFold(string(transport), string(rpcTransportHTTP)) {
		return ClientCapabilities{SupportsElicitation: supportsElicitation, SupportsFallbackID: supportsFallback}
	}
	return ClientCapabilities{SupportsElicitation: supportsElicitation, SupportsFallbackID: supportsFallback}
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
