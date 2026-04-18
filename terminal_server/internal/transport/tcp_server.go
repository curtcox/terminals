package transport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"

	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
)

// TCPServer hosts the control transport over framed TCP sockets.
type TCPServer struct {
	addr      string
	transport *Server

	running   atomic.Bool
	boundAddr string
	mu        sync.Mutex
	listener  net.Listener
}

// NewTCPServer returns a TCP control server.
func NewTCPServer(addr string, transport *Server) *TCPServer {
	return &TCPServer{
		addr:      addr,
		transport: transport,
	}
}

// Address returns the bound address once started, otherwise configured address.
func (s *TCPServer) Address() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.boundAddr != "" {
		return s.boundAddr
	}
	return s.addr
}

// Running reports lifecycle state.
func (s *TCPServer) Running() bool {
	return s.running.Load()
}

// Start binds and accepts TCP control sessions.
func (s *TCPServer) Start(context.Context) error {
	if s.transport == nil {
		return errors.New("tcp transport server not configured")
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
	s.listener = listener
	s.boundAddr = listener.Addr().String()
	s.running.Store(true)

	go s.serve(listener)
	eventlog.Emit(context.Background(), "transport.tcp.listener_ready", slog.LevelInfo, "tcp listener ready",
		slog.String("component", "transport.tcp"),
		slog.String("address", s.boundAddr),
	)
	return nil
}

func (s *TCPServer) serve(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			s.mu.Lock()
			running := s.running.Load()
			address := s.boundAddr
			s.mu.Unlock()
			if running {
				eventlog.Emit(context.Background(), "transport.tcp.accept_failed", slog.LevelError, "tcp accept failed",
					slog.String("component", "transport.tcp"),
					slog.String("address", address),
					slog.Any("error", err),
				)
			}
			return
		}
		go func(conn net.Conn) {
			defer func() { _ = conn.Close() }()
			stream := NewTCPEnvelopeStream(context.Background(), conn)
			if err := s.transport.ConnectEnvelope(stream); err != nil {
				eventlog.Emit(context.Background(), "transport.tcp.session.error", slog.LevelError, "tcp control session ended with error",
					slog.String("component", "transport.tcp"),
					slog.Any("error", err),
				)
			}
		}(conn)
	}
}

// Stop shuts down the TCP listener.
func (s *TCPServer) Stop(context.Context) error {
	s.mu.Lock()
	if !s.running.Load() {
		s.mu.Unlock()
		return nil
	}
	listener := s.listener
	address := s.boundAddr
	s.mu.Unlock()

	err := listener.Close()

	s.mu.Lock()
	s.running.Store(false)
	s.listener = nil
	s.boundAddr = ""
	s.mu.Unlock()

	eventlog.Emit(context.Background(), "transport.tcp.stopped", slog.LevelInfo, "tcp server stopped",
		slog.String("component", "transport.tcp"),
		slog.String("address", address),
	)
	if err != nil && !errors.Is(err, net.ErrClosed) {
		return fmt.Errorf("close tcp listener: %w", err)
	}
	return nil
}
