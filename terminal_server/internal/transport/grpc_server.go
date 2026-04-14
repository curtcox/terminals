package transport

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/curtcox/terminals/terminal_server/internal/scenario"
)

// ErrControlNotConfigured indicates Connect was called before control wiring.
var ErrControlNotConfigured = errors.New("control service not configured")

// Server is a small lifecycle wrapper for the future gRPC control server.
type Server struct {
	addr        string
	running     atomic.Bool
	control     *ControlService
	adapter     ProtoAdapter
	runtime     *scenario.Runtime
	deviceAudio DeviceAudioPublisher
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
	s.adapter = adapter
}

// ConfigureRuntime wires scenario runtime support for command handling.
func (s *Server) ConfigureRuntime(runtime *scenario.Runtime) {
	s.runtime = runtime
}

// ConfigureDeviceAudio wires a live device-audio publisher so every control
// stream created by Connect forwards inbound VoiceAudio chunks to scenarios
// subscribed via the scenario.Environment's DeviceAudio hub.
func (s *Server) ConfigureDeviceAudio(pub DeviceAudioPublisher) {
	s.deviceAudio = pub
}

// Connect handles a single bidirectional control stream session.
func (s *Server) Connect(stream ProtoStream) error {
	if s.control == nil || s.adapter == nil {
		return ErrControlNotConfigured
	}
	handler := NewStreamHandlerWithRuntime(s.control, s.runtime)
	if s.deviceAudio != nil {
		handler.SetDeviceAudioPublisher(s.deviceAudio)
	}
	return RunProtoSession(handler, s.control, stream, s.adapter)
}
