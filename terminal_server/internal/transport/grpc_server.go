package transport

import (
	"context"
	"sync/atomic"
)

// Server is a small lifecycle wrapper for the future gRPC control server.
type Server struct {
	addr    string
	running atomic.Bool
}

// NewServer returns a lifecycle-managed transport server placeholder.
func NewServer(addr string) *Server {
	return &Server{addr: addr}
}

// Address returns the configured bind address.
func (s *Server) Address() string {
	return s.addr
}

// Start marks the server as running.
func (s *Server) Start(context.Context) error {
	s.running.Store(true)
	return nil
}

// Stop marks the server as stopped.
func (s *Server) Stop(context.Context) error {
	s.running.Store(false)
	return nil
}

// Running reports server lifecycle state.
func (s *Server) Running() bool {
	return s.running.Load()
}
