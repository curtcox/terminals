package transport

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"

	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
	"github.com/curtcox/terminals/terminal_server/internal/recording"
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
	recording   recording.Manager
	webrtc      WebRTCSignalEngine
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
	eventlog.Emit(context.Background(), "transport.grpc.listener_ready", slog.LevelInfo, "grpc listener ready",
		slog.String("component", "transport.grpc"),
		slog.String("address", s.addr),
	)
	return nil
}

// Stop marks the server as stopped.
func (s *Server) Stop(context.Context) error {
	s.running.Store(false)
	eventlog.Emit(context.Background(), "transport.grpc.stopped", slog.LevelInfo, "grpc server stopped",
		slog.String("component", "transport.grpc"),
		slog.String("address", s.addr),
	)
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

// ConfigureRecording wires a recording manager so every control stream
// handler tracks StartStream/StopStream lifecycle for route recording.
func (s *Server) ConfigureRecording(mgr recording.Manager) {
	s.recording = mgr
}

// ConfigureWebRTCSignalEngine wires a server-side signaling engine for
// server-managed WebRTC routes.
func (s *Server) ConfigureWebRTCSignalEngine(engine WebRTCSignalEngine) {
	s.webrtc = engine
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
	if s.recording != nil {
		handler.SetRecordingManager(s.recording)
	}
	if s.webrtc != nil {
		handler.SetWebRTCSignalEngine(s.webrtc)
	}
	ctx, end := eventlog.WithSpan(context.Background(), "grpc:connect")
	defer end()
	eventlog.Emit(ctx, "transport.grpc.request.started", slog.LevelInfo, "control stream connect started",
		slog.String("component", "transport.grpc"),
		slog.Group("grpc", slog.String("method", "Control.Connect"), slog.String("status", "started")),
	)
	err := RunProtoSession(handler, s.control, stream, s.adapter)
	status := "ok"
	level := slog.LevelInfo
	if err != nil {
		status = "error"
		level = slog.LevelError
	}
	eventlog.Emit(ctx, "transport.grpc.request.finished", level, "control stream connect finished",
		slog.String("component", "transport.grpc"),
		slog.Group("grpc", slog.String("method", "Control.Connect"), slog.String("status", status)),
		slog.Any("error", err),
	)
	return err
}
