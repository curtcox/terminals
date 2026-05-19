package mcpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
)

func (s *Server) processStdioMessage(ctx context.Context, conn *stdioConnection, raw map[string]any, wg *sync.WaitGroup) {
	if method := strings.TrimSpace(anyString(raw["method"])); method == "" {
		conn.routeResponse(raw)
		return
	}
	reqRaw, err := json.Marshal(raw)
	if err != nil {
		return
	}
	var req rpcRequest
	if err := json.Unmarshal(reqRaw, &req); err != nil {
		return
	}
	if strings.EqualFold(strings.TrimSpace(req.Method), "initialize") {
		s.dispatchStdioRPC(ctx, conn, req, false, wg)
		return
	}
	wg.Add(1)
	go func(req rpcRequest) {
		defer wg.Done()
		s.dispatchStdioRPC(ctx, conn, req, true, wg)
	}(req)
}

func (s *Server) dispatchStdioRPC(ctx context.Context, conn *stdioConnection, req rpcRequest, async bool, wg *sync.WaitGroup) {
	_ = async
	_ = wg
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
}

func (s *Server) closeStdioSession(conn *stdioConnection) {
	if sessionID := conn.getSessionID(); sessionID != "" {
		s.closeSession(context.Background(), sessionID)
	}
}

func readStdioMessage(dec *json.Decoder) (map[string]any, error) {
	var raw map[string]any
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func isStdioEOF(err error) bool {
	return errors.Is(err, io.EOF)
}
