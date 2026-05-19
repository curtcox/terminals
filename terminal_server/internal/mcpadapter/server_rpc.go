package mcpadapter

import (
	"context"
	"strings"
)

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
		return s.handleRPCCancelled(req, rc)
	case "notifications/initialized":
		return rpcResponse{}, "", false
	case "initialize":
		return s.handleRPCInitialize(ctx, req, transport, rc)
	case "tools/list":
		return s.handleRPCToolsList(req, rc)
	case "tools/call":
		return s.handleRPCToolsCall(ctx, req, transport, rc)
	case "shutdown":
		return s.handleRPCShutdown(ctx, req, rc)
	default:
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &rpcError{Code: -32601, Message: "method not found"},
		}, rc.SessionID, hasRPCID(id)
	}
}

func (s *Server) handleRPCCancelled(req rpcRequest, rc requestContext) (rpcResponse, string, bool) {
	params := parseAnyMap(req.Params)
	requestID := requestIDFromParams(params)
	if requestID != "" && strings.TrimSpace(rc.SessionID) != "" {
		s.cancelInflight(rc.SessionID, requestID)
	}
	return rpcResponse{}, rc.SessionID, false
}

func (s *Server) handleRPCInitialize(
	ctx context.Context,
	req rpcRequest,
	transport rpcTransport,
	rc requestContext,
) (rpcResponse, string, bool) {
	id := req.ID
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
	if transport == rpcTransportStdio && rc.Stdio != nil {
		s.mu.Lock()
		s.stdio[info.SessionID] = rc.Stdio
		s.mu.Unlock()
	}
	return rpcResponse{JSONRPC: "2.0", ID: id, Result: result}, info.SessionID, hasRPCID(id)
}

func (s *Server) handleRPCToolsList(req rpcRequest, rc requestContext) (rpcResponse, string, bool) {
	id := req.ID
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
}

func (s *Server) handleRPCToolsCall(
	ctx context.Context,
	req rpcRequest,
	transport rpcTransport,
	rc requestContext,
) (rpcResponse, string, bool) {
	id := req.ID
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
	stream := toolCallStream(transport, rc, sessionID, requestID)
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
	return rpcResponse{JSONRPC: "2.0", ID: id, Result: toolResult(toolResp)}, sessionID, hasRPCID(id)
}

func toolCallStream(transport rpcTransport, rc requestContext, sessionID, requestID string) func(string) error {
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
	if transport == rpcTransportHTTP && rc.HTTPStream != nil && requestID != "" {
		stream = func(chunk string) error {
			if strings.TrimSpace(chunk) == "" {
				return nil
			}
			return rc.HTTPStream("notifications/tools/call_output", map[string]any{
				"session_id": sessionID,
				"request_id": requestID,
				"chunk":      chunk,
			})
		}
	}
	return stream
}

func (s *Server) handleRPCShutdown(ctx context.Context, req rpcRequest, rc requestContext) (rpcResponse, string, bool) {
	id := req.ID
	sessionID := strings.TrimSpace(rc.SessionID)
	if sessionID != "" {
		s.closeSession(ctx, sessionID)
	}
	return rpcResponse{JSONRPC: "2.0", ID: id, Result: map[string]any{"status": "ok"}}, "", hasRPCID(id)
}
