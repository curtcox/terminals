package transport

import (
	"context"
	"errors"
	"sync/atomic"
)

var (
	// ErrControlNotConfigured indicates Connect was called before control wiring.
	ErrControlNotConfigured = errors.New("control service not configured")
)

// Server is a small lifecycle wrapper for the future gRPC control server.
type Server struct {
	addr    string
	running atomic.Bool
	control *ControlService
	handler *StreamHandler
	adapter ProtoAdapter
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

// ConfigureControl wires the control dependencies needed by Connect.
func (s *Server) ConfigureControl(control *ControlService, adapter ProtoAdapter) {
	s.control = control
	s.handler = NewStreamHandler(control)
	s.adapter = adapter
}

// Connect handles a single bidirectional control stream session.
func (s *Server) Connect(stream ProtoStream) error {
	if s.control == nil || s.handler == nil || s.adapter == nil {
		return ErrControlNotConfigured
	}
	return RunProtoSession(s.handler, s.control, stream, s.adapter)
}
