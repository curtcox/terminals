package transport

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"

	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
	"github.com/curtcox/terminals/terminal_server/internal/recording"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"google.golang.org/grpc"
)

// ErrControlNotConfigured indicates Connect was called before control wiring.
var ErrControlNotConfigured = errors.New("control service not configured")

// Server is a lifecycle wrapper around the gRPC control server.
type Server struct {
	addr        string
	boundAddr   string
	running     atomic.Bool
	control     *ControlService
	adapter     ProtoAdapter
	runtime     *scenario.Runtime
	deviceAudio DeviceAudioPublisher
	recording   recording.Manager
	webrtc      WebRTCSignalEngine
	bugReports  BugReportIntake

	mu         sync.Mutex
	listener   net.Listener
	grpcServer *grpc.Server
}

// NewServer returns a lifecycle-managed gRPC transport server.
func NewServer(addr string) *Server {
	return &Server{addr: addr}
}

// Address returns the bound address once started, otherwise the configured bind address.
func (s *Server) Address() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.boundAddr != "" {
		return s.boundAddr
	}
	return s.addr
}

// Start binds the gRPC listener, registers services, and starts serving.
func (s *Server) Start(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running.Load() {
		return nil
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	controlv1.RegisterTerminalControlServiceServer(grpcServer, newGeneratedControlService(s))

	s.listener = listener
	s.grpcServer = grpcServer
	s.boundAddr = listener.Addr().String()
	s.running.Store(true)

	go func(listener net.Listener, server *grpc.Server) {
		if err := server.Serve(listener); err != nil {
			s.mu.Lock()
			running := s.running.Load()
			address := s.boundAddr
			s.mu.Unlock()
			if running {
				eventlog.Emit(context.Background(), "transport.grpc.serve_failed", slog.LevelError, "grpc server serve failed",
					slog.String("component", "transport.grpc"),
					slog.String("address", address),
					slog.Any("error", err),
				)
			}
		}
	}(listener, grpcServer)

	eventlog.Emit(context.Background(), "transport.grpc.listener_ready", slog.LevelInfo, "grpc listener ready",
		slog.String("component", "transport.grpc"),
		slog.String("address", s.boundAddr),
	)
	return nil
}

// Stop drains active RPCs and stops the server.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running.Load() {
		s.mu.Unlock()
		return nil
	}
	grpcServer := s.grpcServer
	address := s.boundAddr
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		grpcServer.Stop()
		<-done
	}

	s.mu.Lock()
	s.running.Store(false)
	s.listener = nil
	s.grpcServer = nil
	s.boundAddr = ""
	s.mu.Unlock()

	eventlog.Emit(context.Background(), "transport.grpc.stopped", slog.LevelInfo, "grpc server stopped",
		slog.String("component", "transport.grpc"),
		slog.String("address", address),
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

// ConfigureBugReportIntake wires persisted diagnostics intake for Connect streams.
func (s *Server) ConfigureBugReportIntake(intake BugReportIntake) {
	s.bugReports = intake
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
	if s.bugReports != nil {
		handler.SetBugReportIntake(s.bugReports)
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
