package transport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
	"golang.org/x/net/websocket"
)

// WebSocketServer hosts the browser control transport over websocket.
type WebSocketServer struct {
	addr           string
	path           string
	allowedOrigins []string
	transport      *Server

	running   atomic.Bool
	boundAddr string
	mu        sync.Mutex
	listener  net.Listener
	http      *http.Server
}

// NewWebSocketServer returns a websocket server bound to /control by default.
func NewWebSocketServer(addr string, transport *Server, allowedOrigins []string) *WebSocketServer {
	copied := append([]string(nil), allowedOrigins...)
	return &WebSocketServer{
		addr:           addr,
		path:           "/control",
		transport:      transport,
		allowedOrigins: copied,
	}
}

// Address returns the bound address once started, otherwise the configured bind address.
func (s *WebSocketServer) Address() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.boundAddr != "" {
		return s.boundAddr
	}
	return s.addr
}

// Path returns the websocket endpoint path.
func (s *WebSocketServer) Path() string {
	return s.path
}

// Running reports websocket server lifecycle state.
func (s *WebSocketServer) Running() bool {
	return s.running.Load()
}

// Start binds and serves websocket control traffic.
func (s *WebSocketServer) Start(context.Context) error {
	if s.transport == nil {
		return errors.New("websocket transport server not configured")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running.Load() {
		return nil
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle(s.path, websocket.Server{
		Handshake: s.handshake,
		Handler: websocket.Handler(func(conn *websocket.Conn) {
			defer func() { _ = conn.Close() }()
			req := conn.Request()
			ctx := context.Background()
			if req != nil {
				ctx = req.Context()
			}
			stream := NewWebSocketProtoStream(ctx, conn)
			if err := s.transport.Connect(stream); err != nil {
				eventlog.Emit(context.Background(), "transport.websocket.session.error", slog.LevelError, "websocket control session ended with error",
					slog.String("component", "transport.websocket"),
					slog.Any("error", err),
				)
			}
		}),
	})

	httpServer := &http.Server{Handler: mux}
	s.listener = listener
	s.http = httpServer
	s.boundAddr = listener.Addr().String()
	s.running.Store(true)

	go func(listener net.Listener, server *http.Server) {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.mu.Lock()
			running := s.running.Load()
			address := s.boundAddr
			s.mu.Unlock()
			if running {
				eventlog.Emit(context.Background(), "transport.websocket.serve_failed", slog.LevelError, "websocket server serve failed",
					slog.String("component", "transport.websocket"),
					slog.String("address", address),
					slog.Any("error", err),
				)
			}
		}
	}(listener, httpServer)

	eventlog.Emit(context.Background(), "transport.websocket.listener_ready", slog.LevelInfo, "websocket listener ready",
		slog.String("component", "transport.websocket"),
		slog.String("address", s.boundAddr),
		slog.String("path", s.path),
	)
	return nil
}

// Stop gracefully shuts down the websocket listener.
func (s *WebSocketServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running.Load() {
		s.mu.Unlock()
		return nil
	}
	httpServer := s.http
	address := s.boundAddr
	s.mu.Unlock()

	err := httpServer.Shutdown(ctx)

	s.mu.Lock()
	s.running.Store(false)
	s.listener = nil
	s.http = nil
	s.boundAddr = ""
	s.mu.Unlock()

	eventlog.Emit(context.Background(), "transport.websocket.stopped", slog.LevelInfo, "websocket server stopped",
		slog.String("component", "transport.websocket"),
		slog.String("address", address),
	)
	return err
}

func (s *WebSocketServer) handshake(cfg *websocket.Config, req *http.Request) error {
	_ = cfg
	if req == nil {
		return nil
	}
	origin := strings.TrimSpace(req.Header.Get("Origin"))
	if origin == "" {
		return nil
	}
	if sameOrigin(origin, req.Host) {
		return nil
	}
	for _, allowed := range s.allowedOrigins {
		trimmed := strings.TrimSpace(allowed)
		if trimmed == "" {
			continue
		}
		if trimmed == "*" || strings.EqualFold(trimmed, origin) {
			return nil
		}
	}
	return fmt.Errorf("origin not allowed: %s", origin)
}

func sameOrigin(origin, host string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if parsed.Host == "" {
		return false
	}
	return strings.EqualFold(parsed.Host, host)
}
