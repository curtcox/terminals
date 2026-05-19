// Package main is the entry point for the terminal server binary.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/admin"
	"github.com/curtcox/terminals/terminal_server/internal/ai"
	"github.com/curtcox/terminals/terminal_server/internal/appruntime"
	"github.com/curtcox/terminals/terminal_server/internal/audio"
	"github.com/curtcox/terminals/terminal_server/internal/config"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/diagnostics/bugreport"
	"github.com/curtcox/terminals/terminal_server/internal/discovery"
	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/mcpadapter"
	"github.com/curtcox/terminals/terminal_server/internal/observation"
	"github.com/curtcox/terminals/terminal_server/internal/placement"
	"github.com/curtcox/terminals/terminal_server/internal/recording"
	"github.com/curtcox/terminals/terminal_server/internal/repl"
	"github.com/curtcox/terminals/terminal_server/internal/replai"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/trust"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
	"github.com/curtcox/terminals/terminal_server/internal/world"
)

// CONTENTS:
//   line  51  const ( // registerMetadata keys
//   line  57  func registerAckMetadata(photoBaseURL string) map[string]string
//   line  65  func normalizeBuildMetadataValue(raw string) string
//   line  73  func main()
//   line 402  func runREPL(stdin io.Reader, stdout, stderr io.Writer) int
//   line 417  func runMCPStdio(stdin io.Reader, stdout, stderr io.Writer) int
//   line 453  func proxyMCPStdio(ctx context.Context, in io.Reader, out io.Writer, mcpURL string) error
//   line 559  func postMCPRPC(ctx context.Context, client *http.Client, mcpURL string, payload map[string]any, sessionID string, confirmationID string) (map[string]any, string, error)
//   line 601  func elicitViaProxy(ctx context.Context, dec *json.Decoder, enc *json.Encoder, proxyRequestID int, originalRequest map[string]any, confirmationMeta map[string]any) (bool, error)
//   line 692  func approvalRejectedResponse(id any) map[string]any
//   line 712  func cloneMapAny(src map[string]any) map[string]any
//   line 723  func mcpAnyMap(v any) map[string]any
//   line 733  func mcpAnyBool(v any) bool
//   line 745  func mcpAnyString(v any) string
//   line 750  func mcpCapabilityEnabled(v any) bool
//   line 766  func rpcIDMatches(id any, want int) bool
//   line 789  type scenarioAnalyzerRunner struct
//   line 794  func (r scenarioAnalyzerRunner) StartAnalyzer(ctx context.Context, sourceDeviceID string, analyzer string, emit func(iorouter.AnalyzerEvent)) (func(), error)
//   line 848  func copyStringMap(in map[string]string) map[string]string
//   line 859  func loadAppPackages(ctx context.Context, runtime *appruntime.Runtime)
//   line 882  func newServerAppRuntime() *appruntime.Runtime
//   line 888  type worldModelAdapter struct
//   line 892  func (w worldModelAdapter) LocateEntity(ctx context.Context, query scenario.EntityQuery) (*iorouter.LocationEstimate, error)
//   line 905  func (w worldModelAdapter) WhoIsHome(ctx context.Context) ([]scenario.EntityRecord, error)
//   line 927  func (w worldModelAdapter) VerifyDevice(ctx context.Context, deviceID string, method string) error
//   line 934  func (w worldModelAdapter) RecentObservations(ctx context.Context, zone string, kind string, since time.Time) ([]iorouter.Observation, error)
//   line 944  func buildTelephonyBridge(ctx context.Context, cfg config.SIPConfig) (scenario.TelephonyBridge, error)
//   line 973  func runDueTimerLoop(ctx context.Context, runtime *scenario.Runtime, interval time.Duration)
//   line 1000 func runLivenessLoop(ctx context.Context, control *transport.ControlService, timeout, interval time.Duration)
//   line 1027 func configurePhotoFrame(handler *transport.StreamHandler, cfg config.Config, baseURL string)
//   line 1048 func loadPhotoFrameSlides(dir, baseURL string) ([]string, error)
//   line 1091 func startPhotoFrameAssetServer(cfg config.Config) (*http.Server, string, error)
//   line 1121 func newPhotoFrameAssetHandler(dir string) http.Handler
//   line 1137 func photoFrameAssetBaseURL(cfg config.Config) string
//   line 1151 func mcpEndpointURL(cfg config.Config) string
//   line 1162 func firstModel(models []string) string

const (
	registerMetadataPhotoFrameAssetBaseURLKey = "photo_frame_asset_base_url"
	registerMetadataServerBuildSHAKey         = "server_build_sha"
	registerMetadataServerBuildDateKey        = "server_build_date"
)

func registerAckMetadata(photoBaseURL string) map[string]string {
	return map[string]string{
		registerMetadataPhotoFrameAssetBaseURLKey: photoBaseURL,
		registerMetadataServerBuildSHAKey:         normalizeBuildMetadataValue(os.Getenv("TERMINALS_BUILD_SHA")),
		registerMetadataServerBuildDateKey:        normalizeBuildMetadataValue(os.Getenv("TERMINALS_BUILD_DATE")),
	}
}

func normalizeBuildMetadataValue(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "unknown"
	}
	return trimmed
}

func main() {
	if len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) == "repl" {
		os.Exit(runREPL(os.Stdin, os.Stdout, os.Stderr))
	}
	if len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) == "mcp-stdio" {
		os.Exit(runMCPStdio(os.Stdin, os.Stdout, os.Stderr))
	}

	if err := runServer(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

type serverRuntime struct {
	cfg               config.Config
	logger            *slog.Logger
	deviceManager     *device.Manager
	ioRouter          *iorouter.Router
	scenarioEngine    *scenario.Engine
	controlService    *transport.ControlService
	scheduler         *storage.MemoryScheduler
	scenarioRuntime   *scenario.Runtime
	controlStream     *transport.StreamHandler
	appRuntime        *appruntime.Runtime
	telephonyBridge   scenario.TelephonyBridge
	photoServer       *http.Server
	adminServer       *http.Server
	grpcServer        *transport.Server
	websocketServer   *transport.WebSocketServer
	tcpServer         *transport.TCPServer
	httpControlServer *transport.HTTPControlServer
	mdns              *discovery.MDNSAdvertiser
}

func runServer() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	evtLogger, err := configureEventLogger(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = evtLogger.Flush() }()
	logger := eventlog.Component("main")
	logger.Info("config loaded", "event", "config.loaded")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx, endSpan := eventlog.WithSpan(ctx, "server:start")
	defer endSpan()

	runtime, err := newServerRuntime(ctx, cfg, logger)
	if err != nil {
		logger.Error("configure server runtime", "event", "server.configure.failed", "error", err)
		return nil
	}
	if err := runtime.start(ctx); err != nil {
		return nil
	}
	runtime.logReady()
	runtime.startHousekeeping(ctx)

	<-ctx.Done()
	logger.Info("terminal server shutting down", "event", "server.stopping")
	runtime.shutdown()
	logger.Info("terminal server stopped", "event", "server.stopped")
	return nil
}

func configureEventLogger(cfg config.Config) (*eventlog.Logger, error) {
	evtLogger, err := eventlog.New(eventlog.Config{
		Dir:           cfg.LogDir,
		Level:         cfg.LogLevel,
		MaxBytes:      cfg.LogMaxBytes,
		MaxArchives:   cfg.LogMaxArchives,
		MirrorStderr:  cfg.LogStderr,
		ServerID:      cfg.MDNSName,
		ServerVersion: cfg.Version,
	})
	if err != nil {
		return nil, fmt.Errorf("init event logger: %w", err)
	}
	eventlog.SetDefault(evtLogger)
	log.SetFlags(0)
	log.SetOutput(evtLogger.StdLogAdapter("legacy"))
	return evtLogger, nil
}

func newServerRuntime(ctx context.Context, cfg config.Config, logger *slog.Logger) (*serverRuntime, error) {
	deviceManager := device.NewManager()
	ioRouter := iorouter.NewRouter()
	audioHub := audio.NewHub()
	scenarioEngine := scenario.NewEngine()
	controlService := transport.NewControlService(cfg.MDNSName, deviceManager)
	store := storage.NewMemoryStore()
	scheduler := storage.NewMemoryScheduler()
	broadcaster := ui.NewMemoryBroadcaster()
	uiHost := ui.NewMemoryHost()
	observationStore := observation.NewStore(4096)
	worldModel := world.NewModel()
	appRuntime := newServerAppRuntime()
	loadAppPackages(ctx, appRuntime)
	registerAppScenarioDefinitions(scenarioEngine, appRuntime)
	telephonyBridge, err := buildTelephonyBridge(ctx, cfg.SIP)
	if err != nil {
		return nil, fmt.Errorf("configure telephony bridge: %w", err)
	}
	aiBackends := ai.NewNoopBackends()
	// Swap the noop sound classifier for a real RMS-based silence detector
	// so Phase-6 "tell me when X stops" scenarios work without an external
	// model. Other noop backends remain until real providers are selected.
	aiBackends.Sound = ai.NewSilenceClassifier(ai.SilenceClassifierConfig{})
	environment := &scenario.Environment{
		Devices:     deviceManager,
		IO:          ioRouter,
		AI:          ai.LLMQueryAdapter{LLM: aiBackends.LLM},
		LLM:         scenarioLLM{backend: aiBackends.LLM},
		Vision:      scenarioVisionAnalyzer{backend: aiBackends.Vision},
		Sound:       scenarioSoundClassifier{backend: aiBackends.Sound},
		STT:         scenarioSTT{backend: aiBackends.STT},
		WakeWord:    scenario.PrefixWakeWordDetector{Prefixes: cfg.WakeWordPrefixes},
		TTS:         scenarioTTS{backend: aiBackends.TTS},
		Telephony:   telephonyBridge,
		Storage:     store,
		Scheduler:   scheduler,
		Broadcast:   broadcaster,
		UI:          uiHost,
		DeviceAudio: scenarioDeviceAudio{hub: audioHub},
		Placement:   placement.NewManagerBackedEngine(deviceManager, ioRouter.Claims()),
		Observe:     observationStore,
		World:       worldModelAdapter{model: worldModel},
	}
	ioRouter.MediaPlanner().SetAnalyzerRunner(scenarioAnalyzerRunner{
		Sound:       environment.Sound,
		DeviceAudio: environment.DeviceAudio,
	})
	scenario.RegisterBuiltins(scenarioEngine)
	scenarioRuntime := scenario.NewRuntime(scenarioEngine, environment)
	ioRouter.MediaPlanner().SetAnalyzerSink(func(event iorouter.AnalyzerEvent) {
		_, _ = scenarioRuntime.HandleEvent(context.Background(), strings.TrimSpace(event.Subject), scenario.EventRecord{
			Kind:       strings.TrimSpace(event.Kind),
			Subject:    strings.TrimSpace(event.Subject),
			Attributes: copyStringMap(event.Attributes),
			Source:     scenario.SourceEvent,
			OccurredAt: event.OccurredAt,
		})
	})
	ioRouter.MediaPlanner().SetObservationSink(func(observation iorouter.Observation) {
		observationStore.AddObservation(context.Background(), observation)
		worldModel.AddObservation(context.Background(), observation)
	})
	if err := scenarioRuntime.RecoverActivations(ctx); err != nil {
		logger.Error("recover scenario activations", "event", "scenario.recovery.failed", "error", err)
	}

	controlStream, mcpServer, err := newControlStreamAndMCP(cfg, controlService)
	if err != nil {
		return nil, fmt.Errorf("configure mcp adapter: %w", err)
	}
	replAIService := replai.NewService(controlStream.ReplSessions(), replai.Config{
		DefaultProvider: cfg.AI.DefaultProvider,
		DefaultModel:    cfg.AI.DefaultModel,
		LLM:             aiBackends.LLM,
		Providers: []replai.ProviderConfig{
			{
				Name:         "openrouter",
				DefaultModel: firstModel(cfg.AI.OpenRouter.Models),
				Models:       cfg.AI.OpenRouter.Models,
			},
			{
				Name:         "ollama",
				DefaultModel: firstModel(cfg.AI.Ollama.Models),
				Models:       cfg.AI.Ollama.Models,
			},
		},
	})
	bugReports := bugreport.NewService(cfg.LogDir, deviceManager, scenarioRuntime)
	webrtcEngine, err := transport.NewPionWebRTCSignalEngine()
	if err != nil {
		return nil, fmt.Errorf("configure webrtc signal engine: %w", err)
	}
	controlStream.SetWebRTCSignalEngine(webrtcEngine)
	photoServer, photoBaseURL, err := startPhotoFrameAssetServer(cfg)
	if err != nil {
		return nil, fmt.Errorf("start photo frame asset server: %w", err)
	}
	adminHandler := admin.NewHandler(
		controlService,
		scenarioRuntime,
		controlStream.ReplSessions(),
		replAIService,
		appRuntime,
		func() { registerAppScenarioDefinitions(scenarioEngine, appRuntime) },
		deviceManager,
		cfg,
		worldModel,
		trust.NewService(),
	)
	adminMux := http.NewServeMux()
	adminMux.Handle("/mcp", mcpServer)
	adminMux.Handle("/mcp/", mcpServer)
	adminMux.Handle("/", adminHandler)
	adminServer, err := startAdminServer(cfg, adminMux)
	if err != nil {
		return nil, fmt.Errorf("start admin dashboard: %w", err)
	}
	controlService.SetRegisterMetadata(registerAckMetadata(photoBaseURL))
	configurePhotoFrame(controlStream, cfg, photoBaseURL)
	recordingManager, err := recording.NewDiskManager(cfg.RecordingDir)
	if err != nil {
		return nil, fmt.Errorf("configure recording manager: %w", err)
	}
	controlStream.SetRecordingManager(recordingManager)
	grpcServer := transport.NewServer(cfg.GRPCAddress())
	grpcServer.ConfigureControl(controlService, transport.GeneratedProtoAdapter{})
	grpcServer.ConfigureRuntime(scenarioRuntime)
	grpcServer.ConfigureDeviceAudio(audioHub)
	grpcServer.ConfigureRecording(recordingManager)
	grpcServer.ConfigureWebRTCSignalEngine(webrtcEngine)
	grpcServer.ConfigureBugReportIntake(bugReports)
	websocketServer := transport.NewWebSocketServer(cfg.ControlWSAddress(), grpcServer, cfg.ControlWSAllowedOrigins)
	tcpServer := transport.NewTCPServer(cfg.ControlTCPAddress(), grpcServer)
	httpControlServer := transport.NewHTTPControlServer(cfg.ControlHTTPAddress(), grpcServer)
	mdns := discovery.NewMDNSAdvertiser()

	return &serverRuntime{
		cfg:               cfg,
		logger:            logger,
		deviceManager:     deviceManager,
		ioRouter:          ioRouter,
		scenarioEngine:    scenarioEngine,
		controlService:    controlService,
		scheduler:         scheduler,
		scenarioRuntime:   scenarioRuntime,
		controlStream:     controlStream,
		appRuntime:        appRuntime,
		telephonyBridge:   telephonyBridge,
		photoServer:       photoServer,
		adminServer:       adminServer,
		grpcServer:        grpcServer,
		websocketServer:   websocketServer,
		tcpServer:         tcpServer,
		httpControlServer: httpControlServer,
		mdns:              mdns,
	}, nil
}

func newControlStreamAndMCP(cfg config.Config, controlService *transport.ControlService) (*transport.StreamHandler, *mcpadapter.Server, error) {
	controlStream := transport.NewStreamHandler(controlService)
	adminBaseURL := fmt.Sprintf("http://127.0.0.1:%d", cfg.AdminHTTPPort)
	controlStream.SetTerminalREPLAdminURL(adminBaseURL)
	mcpAdapter := mcpadapter.New(mcpadapter.Config{
		AdminBaseURL:    adminBaseURL,
		ConfirmationTTL: time.Duration(cfg.Agent.Approval.ConfirmationTTLSeconds) * time.Second,
		MinHumanLatency: time.Duration(cfg.Agent.Approval.MinHumanLatencyMS) * time.Millisecond,
		OperationalMax:  cfg.Agent.Operational.MaxStreams,
		OperationalTTL:  time.Duration(cfg.Agent.Operational.StreamTTLSeconds) * time.Second,
		UnsafeConfirmation: func(ctx context.Context, event mcpadapter.UnsafeConfirmationEvent) {
			eventlog.Emit(ctx, "unsafe_confirmation_protocol", slog.LevelWarn, "unsafe confirmation protocol observed",
				slog.String("session_origin", "mcp"),
				slog.String("session_id", event.SessionID),
				slog.String("client_id", event.ClientID),
				slog.String("tool", event.ToolName),
				slog.String("command_hash", event.CommandHash),
				slog.Int64("latency_ms", event.Latency.Milliseconds()),
				slog.String("path", event.Path),
			)
		},
	})
	mcpServer, err := mcpadapter.NewServer(mcpadapter.ServerConfig{
		Adapter:      mcpAdapter,
		Sessions:     controlStream.ReplSessions(),
		AdminBaseURL: adminBaseURL,
	})
	if err != nil {
		return nil, nil, err
	}
	return controlStream, mcpServer, nil
}

func (r *serverRuntime) start(ctx context.Context) error {
	r.logger.Info("terminal server starting", "event", "server.starting", "grpc_address", r.grpcServer.Address())
	if err := r.mdns.Start(ctx, discovery.ServiceInfo{
		ServiceType: r.cfg.MDNSService,
		Name:        r.cfg.MDNSName,
		Port:        r.cfg.GRPCPort,
		Version:     r.cfg.Version,
		GRPC:        r.cfg.GRPCAddress(),
		WebSocket:   fmt.Sprintf("ws://%s%s", r.cfg.ControlWSAddress(), r.websocketServer.Path()),
		TCP:         r.cfg.ControlTCPAddress(),
		HTTP:        fmt.Sprintf("http://%s", r.cfg.ControlHTTPAddress()),
		MCP:         mcpEndpointURL(r.cfg),
		Priority:    []string{"grpc", "websocket", "tcp", "http", "mcp"},
	}); err != nil {
		r.logger.Error("start mDNS", "event", "discovery.mdns.failed", "error", err)
		return err
	}
	if err := r.startTransports(ctx); err != nil {
		return err
	}
	return r.validateBootstrap()
}

func (r *serverRuntime) startTransports(ctx context.Context) error {
	if err := r.grpcServer.Start(ctx); err != nil {
		r.logger.Error("start transport", "event", "transport.grpc.start_failed", "error", err)
		return err
	}
	if err := r.websocketServer.Start(ctx); err != nil {
		r.logger.Error("start websocket transport", "event", "transport.websocket.start_failed", "error", err)
		return err
	}
	if err := r.tcpServer.Start(ctx); err != nil {
		r.logger.Error("start tcp transport", "event", "transport.tcp.start_failed", "error", err)
		return err
	}
	if err := r.httpControlServer.Start(ctx); err != nil {
		r.logger.Error("start http fallback transport", "event", "transport.http.start_failed", "error", err)
		return err
	}
	return nil
}

func (r *serverRuntime) validateBootstrap() error {
	if len(r.deviceManager.List()) != 0 {
		r.logger.Error("unexpected initial device registry state", "event", "server.bootstrap.invalid_state")
		return errors.New("unexpected initial device registry state")
	}
	if len(r.ioRouter.Routes()) != 0 {
		r.logger.Error("unexpected initial route registry state", "event", "server.bootstrap.invalid_state")
		return errors.New("unexpected initial route registry state")
	}
	if _, ok := r.scenarioEngine.Active("bootstrap"); ok {
		r.logger.Error("unexpected initial scenario state", "event", "server.bootstrap.invalid_state")
		return errors.New("unexpected initial scenario state")
	}
	if len(r.scheduler.List()) != 0 {
		r.logger.Error("unexpected initial scheduler state", "event", "server.bootstrap.invalid_state")
		return errors.New("unexpected initial scheduler state")
	}
	return nil
}

func (r *serverRuntime) logReady() {
	r.logger.Info("control service ready", "event", "server.started", "server_id", r.cfg.MDNSName)
	r.logger.Info("websocket control ready", "event", "transport.websocket.ready", "websocket_address", r.websocketServer.Address(), "path", r.websocketServer.Path())
	r.logger.Info("tcp control ready", "event", "transport.tcp.ready", "tcp_address", r.tcpServer.Address())
	r.logger.Info("http fallback control ready", "event", "transport.http.ready", "http_address", r.httpControlServer.Address())
	if r.adminServer != nil {
		r.logger.Info("admin dashboard available", "event", "admin.http.ready", "addr", r.adminServer.Addr)
	}
	r.logger.Info("control stream handler initialized", "event", "transport.stream.ready")
	r.logger.Info("recording manager initialized", "event", "recording.started", "dir", r.cfg.RecordingDir)
	r.logger.Info("scenario runtime initialized", "event", "scenario.definition.registered", "builtin_scenarios", 3)
	r.logger.Info(
		"app runtime initialized",
		"event", "appruntime.package.loaded",
		"packages", len(r.appRuntime.ListPackages()),
		"definitions", len(r.appRuntime.Definitions()),
	)
	r.logger.Info(
		"housekeeping configured",
		"event", "housekeeping.configured",
		"heartbeat_timeout_seconds", r.cfg.HeartbeatTimeoutSeconds,
		"liveness_interval_seconds", r.cfg.LivenessReconcileIntervalSecs,
		"due_timer_interval_seconds", r.cfg.DueTimerProcessIntervalSecs,
	)
	_ = r.controlStream
}

func (r *serverRuntime) startHousekeeping(ctx context.Context) {
	go runDueTimerLoop(ctx, r.scenarioRuntime, time.Duration(r.cfg.DueTimerProcessIntervalSecs)*time.Second)
	go runLivenessLoop(
		ctx,
		r.controlService,
		time.Duration(r.cfg.HeartbeatTimeoutSeconds)*time.Second,
		time.Duration(r.cfg.LivenessReconcileIntervalSecs)*time.Second,
	)
}

func (r *serverRuntime) shutdown() {
	shutdownCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := r.httpControlServer.Stop(shutdownCtx); err != nil {
		r.logger.Error("stop http fallback transport", "event", "transport.http.stop_failed", "error", err)
	}
	if err := r.tcpServer.Stop(shutdownCtx); err != nil {
		r.logger.Error("stop tcp transport", "event", "transport.tcp.stop_failed", "error", err)
	}
	if err := r.websocketServer.Stop(shutdownCtx); err != nil {
		r.logger.Error("stop websocket transport", "event", "transport.websocket.stop_failed", "error", err)
	}
	if err := r.grpcServer.Stop(shutdownCtx); err != nil {
		r.logger.Error("stop transport", "event", "transport.grpc.stop_failed", "error", err)
	}
	if r.photoServer != nil {
		if err := r.photoServer.Shutdown(shutdownCtx); err != nil {
			r.logger.Error("stop photo frame asset server", "event", "transport.http.photo_frame.stop_failed", "error", err)
		}
	}
	if r.adminServer != nil {
		if err := r.adminServer.Shutdown(shutdownCtx); err != nil {
			r.logger.Error("stop admin dashboard", "event", "admin.http.stop_failed", "error", err)
		}
	}
	if err := r.mdns.Stop(shutdownCtx); err != nil {
		r.logger.Error("stop mDNS", "event", "discovery.mdns.stop_failed", "error", err)
	}
	if bridge, ok := r.telephonyBridge.(*telephony.SIPBridge); ok {
		if err := bridge.Stop(shutdownCtx); err != nil {
			r.logger.Error("stop telephony bridge", "event", "telephony.bridge.stop_failed", "error", err)
		}
	}
}

func runREPL(stdin io.Reader, stdout, stderr io.Writer) int {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := repl.Run(ctx, stdin, stdout, repl.Options{
		Prompt:       "repl>",
		AdminBaseURL: strings.TrimSpace(os.Getenv("TERMINALS_REPL_ADMIN_URL")),
		SessionID:    strings.TrimSpace(os.Getenv("TERMINALS_REPL_SESSION_ID")),
	}); err != nil {
		_, _ = fmt.Fprintf(stderr, "repl: %v\n", err)
		return 1
	}
	return 0
}

func runMCPStdio(stdin io.Reader, stdout, stderr io.Writer) int {
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "load config: %v\n", err)
		return 1
	}
	adminBaseURL := fmt.Sprintf("http://127.0.0.1:%d", cfg.AdminHTTPPort)
	statusURL := strings.TrimSuffix(adminBaseURL, "/") + "/admin/api/status"
	req, err := http.NewRequest(http.MethodGet, statusURL, nil)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "mcp-stdio: build status probe: %v\n", err)
		return 1
	}
	probeCtx, probeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer probeCancel()
	req = req.WithContext(probeCtx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp == nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		_, _ = fmt.Fprintf(stderr, "mcp-stdio: no running server detected at %s\n", statusURL)
		return 1
	}
	_ = resp.Body.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	mcpURL := strings.TrimSuffix(adminBaseURL, "/") + "/mcp"
	if err := proxyMCPStdio(ctx, stdin, stdout, mcpURL); err != nil && !errors.Is(err, context.Canceled) {
		_, _ = fmt.Fprintf(stderr, "mcp-stdio: %v\n", err)
		return 1
	}
	return 0
}

func proxyMCPStdio(ctx context.Context, in io.Reader, out io.Writer, mcpURL string) error {
	dec := json.NewDecoder(bufio.NewReader(in))
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	proxy := &mcpStdioProxy{
		decoder:        dec,
		encoder:        enc,
		client:         &http.Client{Timeout: 30 * time.Second},
		mcpURL:         mcpURL,
		proxyRequestID: 1000000,
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var req map[string]any
		if err := dec.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		method := strings.TrimSpace(mcpAnyString(req["method"]))
		if method == "" {
			// Client responses are consumed only while awaiting an in-flight elicitation.
			continue
		}
		if err := proxy.handleRequest(ctx, method, req); err != nil {
			return err
		}
	}
}

type mcpStdioProxy struct {
	decoder                   *json.Decoder
	encoder                   *json.Encoder
	client                    *http.Client
	mcpURL                    string
	sessionID                 string
	clientSupportsElicitation bool
	proxyRequestID            int
}

func (p *mcpStdioProxy) handleRequest(ctx context.Context, method string, req map[string]any) error {
	payload := cloneMapAny(req)
	if strings.EqualFold(method, "initialize") {
		p.clientSupportsElicitation = prepareMCPInitializeForProxy(payload)
	}
	resp, err := p.forward(ctx, payload, "")
	if err != nil {
		return err
	}
	resp, err = p.rewriteResponse(ctx, method, req, payload, resp)
	if err != nil {
		return err
	}
	if req["id"] == nil {
		return nil
	}
	return p.encoder.Encode(resp)
}

func (p *mcpStdioProxy) forward(ctx context.Context, payload map[string]any, confirmationID string) (map[string]any, error) {
	resp, nextSessionID, err := postMCPRPC(ctx, p.client, p.mcpURL, payload, p.sessionID, confirmationID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(nextSessionID) != "" {
		p.sessionID = strings.TrimSpace(nextSessionID)
	}
	return resp, nil
}

func (p *mcpStdioProxy) rewriteResponse(
	ctx context.Context,
	method string,
	original map[string]any,
	payload map[string]any,
	resp map[string]any,
) (map[string]any, error) {
	if strings.EqualFold(method, "initialize") && p.clientSupportsElicitation {
		return rewriteMCPInitializeResponseForElicitation(resp), nil
	}
	if !strings.EqualFold(method, "tools/call") || !p.clientSupportsElicitation {
		return resp, nil
	}
	p.proxyRequestID++
	nextResp, updatedSessionID, handled, err := handleMCPToolConfirmation(
		ctx,
		mcpToolConfirmationRequest{
			decoder:        p.decoder,
			encoder:        p.encoder,
			client:         p.client,
			mcpURL:         p.mcpURL,
			payload:        payload,
			sessionID:      p.sessionID,
			proxyRequestID: p.proxyRequestID,
			original:       original,
			response:       resp,
		},
	)
	if err != nil {
		return nil, err
	}
	if handled {
		p.sessionID = updatedSessionID
		return nextResp, nil
	}
	return resp, nil
}

func prepareMCPInitializeForProxy(payload map[string]any) bool {
	params := mcpAnyMap(payload["params"])
	caps := mcpAnyMap(params["capabilities"])
	clientSupportsElicitation := mcpCapabilityEnabled(caps["elicitation"]) || mcpCapabilityEnabled(caps["mcp_elicitation"])
	if !clientSupportsElicitation {
		return false
	}
	// Drive elicitation from the proxy against the client, using the
	// server's fallback confirmation_id protocol as the wire format.
	// The server HTTP path has no elicitation hook; if it classified
	// the session as mutating_via_elicitation it would reject mutating
	// tool calls with elicit_unavailable.
	delete(caps, "elicitation")
	delete(caps, "mcp_elicitation")
	caps["terminals_fallback_confirmation"] = true
	params["capabilities"] = caps
	payload["params"] = params
	return true
}

func rewriteMCPInitializeResponseForElicitation(resp map[string]any) map[string]any {
	result := mcpAnyMap(mcpAnyMap(resp)["result"])
	if !strings.EqualFold(strings.TrimSpace(mcpAnyString(result["mutating_capability"])), "mutating_via_fallback") {
		return resp
	}
	result["mutating_capability"] = "mutating_via_elicitation"
	respMap := mcpAnyMap(resp)
	respMap["result"] = result
	return respMap
}

type mcpToolConfirmationRequest struct {
	decoder        *json.Decoder
	encoder        *json.Encoder
	client         *http.Client
	mcpURL         string
	payload        map[string]any
	sessionID      string
	proxyRequestID int
	original       map[string]any
	response       map[string]any
}

func handleMCPToolConfirmation(ctx context.Context, req mcpToolConfirmationRequest) (map[string]any, string, bool, error) {
	meta := mcpConfirmationMeta(req.response)
	confirmationID := strings.TrimSpace(mcpAnyString(meta["confirmation_id"]))
	if confirmationID == "" {
		return req.response, req.sessionID, false, nil
	}
	approved, err := elicitViaProxy(ctx, req.decoder, req.encoder, req.proxyRequestID, req.original, meta)
	if err != nil {
		return nil, req.sessionID, false, err
	}
	if !approved {
		return approvalRejectedResponse(req.original["id"]), req.sessionID, true, nil
	}
	resp, nextSessionID, err := postMCPRPC(ctx, req.client, req.mcpURL, req.payload, req.sessionID, confirmationID)
	if err != nil {
		return nil, req.sessionID, false, err
	}
	if strings.TrimSpace(nextSessionID) == "" {
		nextSessionID = req.sessionID
	}
	return resp, strings.TrimSpace(nextSessionID), true, nil
}

func mcpConfirmationMeta(resp map[string]any) map[string]any {
	result := mcpAnyMap(mcpAnyMap(resp)["result"])
	meta := mcpAnyMap(result["_meta"])
	if !strings.EqualFold(strings.TrimSpace(mcpAnyString(meta["status"])), "confirmation_required") {
		return nil
	}
	return meta
}

func postMCPRPC(
	ctx context.Context,
	client *http.Client,
	mcpURL string,
	payload map[string]any,
	sessionID string,
	confirmationID string,
) (map[string]any, string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, mcpURL, strings.NewReader(string(raw)))
	if err != nil {
		return nil, "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(sessionID) != "" {
		httpReq.Header.Set(mcpadapter.HeaderSessionID, strings.TrimSpace(sessionID))
	}
	if strings.TrimSpace(confirmationID) != "" {
		httpReq.Header.Set(mcpadapter.HeaderConfirmationID, strings.TrimSpace(confirmationID))
	}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = httpResp.Body.Close() }()
	nextSessionID := strings.TrimSpace(httpResp.Header.Get(mcpadapter.HeaderSessionID))
	if httpResp.StatusCode == http.StatusNoContent {
		return map[string]any{}, nextSessionID, nil
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, nextSessionID, fmt.Errorf("mcp http status %d", httpResp.StatusCode)
	}
	var rpcResp map[string]any
	if err := json.NewDecoder(httpResp.Body).Decode(&rpcResp); err != nil {
		return nil, nextSessionID, err
	}
	return rpcResp, nextSessionID, nil
}

func elicitViaProxy(
	ctx context.Context,
	dec *json.Decoder,
	enc *json.Encoder,
	proxyRequestID int,
	originalRequest map[string]any,
	confirmationMeta map[string]any,
) (bool, error) {
	if err := enc.Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      proxyRequestID,
		"method":  "elicitation/create",
		"params":  mcpElicitationParams(originalRequest, confirmationMeta),
	}); err != nil {
		return false, err
	}
	return awaitMCPApproval(ctx, dec, proxyRequestID)
}

func mcpElicitationParams(originalRequest map[string]any, confirmationMeta map[string]any) map[string]any {
	toolName := strings.TrimSpace(mcpAnyString(mcpAnyMap(originalRequest["params"])["name"]))
	rendered := strings.TrimSpace(mcpAnyString(confirmationMeta["rendered_command"]))
	classification := mcpConfirmationClassification(confirmationMeta)
	promptLabel := strings.ReplaceAll(classification, "_", " ")
	return map[string]any{
		"message": fmt.Sprintf("Approve %s command?\n\nTool: %s\nCommand: %s", promptLabel, toolName, rendered),
		"requestedSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"approved": map[string]any{
					"type":        "boolean",
					"title":       "Approve",
					"description": "Approve this command",
				},
			},
			"required": []string{"approved"},
		},
		// Preserved for older/custom clients that key off these fields.
		"title":            fmt.Sprintf("Approve %s command", promptLabel),
		"tool_name":        toolName,
		"rendered_command": rendered,
		"classification":   classification,
	}
}

func mcpConfirmationClassification(meta map[string]any) string {
	classification := strings.TrimSpace(mcpAnyString(meta["classification"]))
	if classification == "" {
		return string(repl.CommandClassificationMutating)
	}
	return classification
}

func awaitMCPApproval(ctx context.Context, dec *json.Decoder, proxyRequestID int) (bool, error) {
	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}
		var msg map[string]any
		if err := dec.Decode(&msg); err != nil {
			return false, err
		}
		if strings.TrimSpace(mcpAnyString(msg["method"])) != "" {
			continue
		}
		if !rpcIDMatches(msg["id"], proxyRequestID) {
			continue
		}
		return mcpApprovalResult(mcpAnyMap(msg["result"])), nil
	}
}

func mcpApprovalResult(result map[string]any) bool {
	// Per MCP spec, an accept response has action=="accept" and the
	// schema-shaped answer under `content`. Anything else (decline,
	// cancel, error, or ambiguous shape) counts as not-approved.
	action := strings.ToLower(strings.TrimSpace(mcpAnyString(result["action"])))
	content := mcpAnyMap(result["content"])
	switch action {
	case "accept", "accepted", "approve", "approved", "yes":
		if _, hasApproved := content["approved"]; hasApproved {
			return mcpAnyBool(content["approved"])
		}
		return true
	case "decline", "declined", "reject", "rejected", "no", "cancel", "cancelled", "canceled":
		return false
	}
	return mcpAnyBool(result["approved"])
}

func approvalRejectedResponse(id any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": "operation rejected by user",
				},
			},
			"_meta": map[string]any{
				"status":        "error",
				"error_code":    "approval_rejected",
				"error_message": "operation rejected by user",
			},
		},
	}
}

func cloneMapAny(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func mcpAnyMap(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	if typed, ok := v.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func mcpAnyBool(v any) bool {
	switch typed := v.(type) {
	case bool:
		return typed
	case string:
		raw := strings.ToLower(strings.TrimSpace(typed))
		return raw == "true" || raw == "1" || raw == "yes"
	default:
		return false
	}
}

func mcpAnyString(v any) string {
	s, _ := v.(string)
	return s
}

func mcpCapabilityEnabled(v any) bool {
	switch typed := v.(type) {
	case bool:
		return typed
	case string:
		raw := strings.ToLower(strings.TrimSpace(typed))
		return raw == "true" || raw == "1" || raw == "yes"
	case map[string]any:
		return true
	case []any:
		return len(typed) > 0
	default:
		return false
	}
}

func rpcIDMatches(id any, want int) bool {
	switch typed := id.(type) {
	case float64:
		return int(typed) == want
	case float32:
		return int(typed) == want
	case int:
		return typed == want
	case int32:
		return int(typed) == want
	case int64:
		return int(typed) == want
	case json.Number:
		n, err := typed.Int64()
		return err == nil && int(n) == want
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(typed))
		return err == nil && n == want
	default:
		return false
	}
}

type scenarioAnalyzerRunner struct {
	Sound       scenario.SoundClassifier
	DeviceAudio scenario.DeviceAudioSubscriber
}

func (r scenarioAnalyzerRunner) StartAnalyzer(
	ctx context.Context,
	sourceDeviceID string,
	analyzer string,
	emit func(iorouter.AnalyzerEvent),
) (func(), error) {
	if r.Sound == nil || r.DeviceAudio == nil || strings.TrimSpace(sourceDeviceID) == "" {
		return func() {}, nil
	}
	if analyzer != "" && analyzer != "sound" {
		return func() {}, nil
	}

	audioSub, err := r.DeviceAudio.SubscribeAudio(ctx, sourceDeviceID)
	if err != nil {
		return nil, err
	}
	stream, err := r.Sound.Classify(ctx, audioSub)
	if err != nil {
		_ = audioSub.Close()
		return nil, err
	}

	childCtx, cancel := context.WithCancel(ctx)
	go func() {
		defer cancel()
		defer func() { _ = audioSub.Close() }()
		for {
			select {
			case <-childCtx.Done():
				return
			case event, ok := <-stream:
				if !ok {
					return
				}
				emit(iorouter.AnalyzerEvent{
					Kind:    "sound.detected",
					Subject: strings.TrimSpace(sourceDeviceID),
					Attributes: map[string]string{
						"label":      strings.TrimSpace(event.Label),
						"confidence": fmt.Sprintf("%.4f", event.Confidence),
					},
					OccurredAt: time.Now().UTC(),
				})
			}
		}
	}()

	return func() {
		cancel()
		_ = audioSub.Close()
	}, nil
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func loadAppPackages(ctx context.Context, runtime *appruntime.Runtime) {
	logger := eventlog.Component("appruntime")
	if runtime == nil {
		return
	}
	root := "apps"
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		appRoot := filepath.Join(root, entry.Name())
		if _, err := runtime.LoadPackage(ctx, appRoot); err != nil {
			logger.Warn("skip app package", "event", "appruntime.package.skipped", "package", entry.Name(), "error", err)
			continue
		}
		logger.Info("loaded app package", "event", "appruntime.package.loaded", "package", entry.Name())
	}
}

func newServerAppRuntime() *appruntime.Runtime {
	runtime := appruntime.NewRuntime()
	runtime.SetMigrationDryRunGateEnabled(true)
	return runtime
}

type worldModelAdapter struct {
	model *world.Model
}

func (w worldModelAdapter) LocateEntity(ctx context.Context, query scenario.EntityQuery) (*iorouter.LocationEstimate, error) {
	if w.model == nil {
		return nil, world.ErrNotFound
	}
	return w.model.LocateEntity(ctx, world.EntityQuery{
		Person:        query.Person,
		Object:        query.Object,
		BluetoothMAC:  query.BluetoothMAC,
		LastKnownOnly: query.LastKnownOnly,
		MinConfidence: query.MinConfidence,
	})
}

func (w worldModelAdapter) WhoIsHome(ctx context.Context) ([]scenario.EntityRecord, error) {
	if w.model == nil {
		return nil, nil
	}
	records, err := w.model.WhoIsHome(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]scenario.EntityRecord, 0, len(records))
	for _, record := range records {
		out = append(out, scenario.EntityRecord{
			EntityID:    record.EntityID,
			Kind:        string(record.Kind),
			DisplayName: record.DisplayName,
			LastKnown:   record.LastKnown,
			LastSeenAt:  record.LastSeenAt,
			Confidence:  record.Confidence,
		})
	}
	return out, nil
}

func (w worldModelAdapter) VerifyDevice(ctx context.Context, deviceID string, method string) error {
	if w.model == nil {
		return world.ErrNotFound
	}
	return w.model.VerifyDevice(ctx, deviceID, method)
}

func (w worldModelAdapter) RecentObservations(ctx context.Context, zone string, kind string, since time.Time) ([]iorouter.Observation, error) {
	if w.model == nil {
		return nil, nil
	}
	return w.model.RecentObservations(ctx, zone, kind, since)
}

// buildTelephonyBridge returns the configured telephony bridge for the
// server runtime. When SIP is disabled a NoopBridge is returned so
// scenarios continue to function without a SIP provider.
func buildTelephonyBridge(ctx context.Context, cfg config.SIPConfig) (scenario.TelephonyBridge, error) {
	logger := eventlog.Component("telephony.sip")
	if !cfg.Enabled {
		logger.Info("telephony bridge disabled", "event", "telephony.bridge.disabled")
		return telephony.NoopBridge{}, nil
	}
	bridge := telephony.NewSIPBridge(
		telephony.Registration{
			ServerURI:   cfg.ServerURI,
			Username:    cfg.Username,
			DisplayName: cfg.DisplayName,
			Password:    cfg.Password,
		},
		telephony.LogTransport{Logger: logger},
		telephony.WithMediaTransport(telephony.LogMediaTransport{Logger: logger}),
	)
	if err := bridge.Start(ctx); err != nil {
		return nil, err
	}
	logger.Info(
		"telephony bridge registered",
		"event", "telephony.bridge.registered",
		"server", cfg.ServerURI,
		"user", cfg.Username,
		"display", cfg.DisplayName,
	)
	return bridge, nil
}

func runDueTimerLoop(ctx context.Context, runtime *scenario.Runtime, interval time.Duration) {
	logger := eventlog.Component("housekeeping")
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			processed, err := runtime.ProcessDueTimers(ctx, now.UTC())
			if err != nil {
				logger.Error("due timer loop error", "event", "housekeeping.due_timers.failed", "error", err)
				continue
			}
			if processed > 0 {
				logger.Info("due timer loop processed", "event", "housekeeping.due_timers.processed", "processed", processed)
			} else {
				logger.Debug("due timer loop idle", "event", "housekeeping.due_timers.processed", "processed", 0)
			}
		}
	}
}

func runLivenessLoop(
	ctx context.Context,
	control *transport.ControlService,
	timeout, interval time.Duration,
) {
	logger := eventlog.Component("housekeeping")
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			updatedIDs := control.ReconcileLivenessDeviceIDs(timeout)
			if len(updatedIDs) > 0 {
				logger.Info("liveness reconcile updated", "event", "housekeeping.liveness.reconciled", "updated", len(updatedIDs))
			} else {
				logger.Debug("liveness reconcile idle", "event", "housekeeping.liveness.reconciled", "updated", 0)
			}
		}
	}
}

func configurePhotoFrame(handler *transport.StreamHandler, cfg config.Config, baseURL string) {
	logger := eventlog.Component("transport.photo_frame")
	if handler == nil {
		return
	}
	interval := time.Duration(cfg.PhotoFrameIntervalSeconds) * time.Second
	slides, err := loadPhotoFrameSlides(cfg.PhotoFrameDir, baseURL)
	if err != nil {
		logger.Error("photo frame slide discovery failed", "event", "photo_frame.slide_discovery.failed", "dir", cfg.PhotoFrameDir, "error", err)
	}
	handler.SetPhotoFrameSettings(slides, interval)
	logger.Info(
		"photo frame configured",
		"event", "photo_frame.configured",
		"slides", len(slides),
		"dir", cfg.PhotoFrameDir,
		"interval_seconds", cfg.PhotoFrameIntervalSeconds,
		"base_url", baseURL,
	)
}

func loadPhotoFrameSlides(dir, baseURL string) ([]string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, nil
	}
	baseURL = strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	slides := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		default:
			continue
		}
		absPath := filepath.Join(dir, name)
		absPath, err = filepath.Abs(absPath)
		if err != nil {
			return nil, fmt.Errorf("resolve photo frame slide %q: %w", absPath, err)
		}
		slideURL, joinErr := url.JoinPath(baseURL, path.Base(absPath))
		if joinErr != nil {
			return nil, fmt.Errorf("build photo frame url for %q: %w", absPath, joinErr)
		}
		slides = append(slides, slideURL)
	}
	sort.Strings(slides)
	return slides, nil
}

func startPhotoFrameAssetServer(cfg config.Config) (*http.Server, string, error) {
	logger := eventlog.Component("transport.photo_frame")
	baseURL := photoFrameAssetBaseURL(cfg)
	if strings.TrimSpace(cfg.PhotoFrameDir) == "" {
		return nil, baseURL, nil
	}
	if strings.TrimSpace(cfg.PhotoFramePublicBaseURL) != "" {
		return nil, baseURL, nil
	}

	mux := http.NewServeMux()
	mux.Handle("/photo-frame/", newPhotoFrameAssetHandler(cfg.PhotoFrameDir))

	address := net.JoinHostPort(strings.TrimSpace(cfg.PhotoFrameHTTPHost), strconv.Itoa(cfg.PhotoFrameHTTPPort))
	server := &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("photo frame asset server listening", "event", "transport.http.photo_frame.ready", "address", address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("photo frame asset server error", "event", "transport.http.photo_frame.error", "error", err)
		}
	}()

	return server, baseURL, nil
}

func newPhotoFrameAssetHandler(dir string) http.Handler {
	fileServer := http.StripPrefix("/photo-frame/", http.FileServer(http.Dir(dir)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=60")
		fileServer.ServeHTTP(w, r)
	})
}

func photoFrameAssetBaseURL(cfg config.Config) string {
	if configured := strings.TrimSpace(cfg.PhotoFramePublicBaseURL); configured != "" {
		return strings.TrimSuffix(configured, "/")
	}
	publicHost := strings.TrimSpace(cfg.MDNSName)
	if publicHost == "" {
		publicHost = "localhost"
	}
	if !strings.Contains(publicHost, ".") {
		publicHost += ".local"
	}
	return fmt.Sprintf("http://%s:%d/photo-frame", publicHost, cfg.PhotoFrameHTTPPort)
}

func mcpEndpointURL(cfg config.Config) string {
	publicHost := strings.TrimSpace(cfg.MDNSName)
	if publicHost == "" {
		publicHost = "localhost"
	}
	if !strings.Contains(publicHost, ".") {
		publicHost += ".local"
	}
	return fmt.Sprintf("http://%s:%d/mcp", publicHost, cfg.AdminHTTPPort)
}

func firstModel(models []string) string {
	if len(models) == 0 {
		return ""
	}
	return strings.TrimSpace(models[0])
}
