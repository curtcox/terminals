// Package main is the entry point for the terminal server binary.
package main

import (
	"context"
	"fmt"
	"log"
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
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/observation"
	"github.com/curtcox/terminals/terminal_server/internal/placement"
	"github.com/curtcox/terminals/terminal_server/internal/recording"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
	"github.com/curtcox/terminals/terminal_server/internal/world"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

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
		_, _ = fmt.Fprintf(os.Stderr, "init event logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = evtLogger.Flush() }()
	eventlog.SetDefault(evtLogger)
	log.SetFlags(0)
	log.SetOutput(evtLogger.StdLogAdapter("legacy"))
	logger := eventlog.Component("main")
	logger.Info("config loaded", "event", "config.loaded")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx, endSpan := eventlog.WithSpan(ctx, "server:start")
	defer endSpan()

	deviceManager := device.NewManager()
	ioRouter := io.NewRouter()
	audioHub := audio.NewHub()
	scenarioEngine := scenario.NewEngine()
	controlService := transport.NewControlService(cfg.MDNSName, deviceManager)
	store := storage.NewMemoryStore()
	scheduler := storage.NewMemoryScheduler()
	broadcaster := ui.NewMemoryBroadcaster()
	observationStore := observation.NewStore(4096)
	worldModel := world.NewModel()
	appRuntime := appruntime.NewRuntime()
	loadAppPackages(ctx, appRuntime)
	registerAppScenarioDefinitions(scenarioEngine, appRuntime)
	telephonyBridge, err := buildTelephonyBridge(ctx, cfg.SIP)
	if err != nil {
		logger.Error("configure telephony bridge", "event", "telephony.bridge.failed", "error", err)
		return
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
		DeviceAudio: scenarioDeviceAudio{hub: audioHub},
		Placement:   placement.NewManagerBackedEngine(deviceManager),
		Observe:     observationStore,
		World:       worldModelAdapter{model: worldModel},
	}
	ioRouter.MediaPlanner().SetAnalyzerRunner(scenarioAnalyzerRunner{
		Sound:       environment.Sound,
		DeviceAudio: environment.DeviceAudio,
	})
	scenario.RegisterBuiltins(scenarioEngine)
	scenarioRuntime := scenario.NewRuntime(scenarioEngine, environment)
	ioRouter.MediaPlanner().SetAnalyzerSink(func(event io.AnalyzerEvent) {
		_, _ = scenarioRuntime.HandleEvent(context.Background(), strings.TrimSpace(event.Subject), scenario.EventRecord{
			Kind:       strings.TrimSpace(event.Kind),
			Subject:    strings.TrimSpace(event.Subject),
			Attributes: copyStringMap(event.Attributes),
			Source:     scenario.SourceEvent,
			OccurredAt: event.OccurredAt,
		})
	})
	ioRouter.MediaPlanner().SetObservationSink(func(observation io.Observation) {
		observationStore.AddObservation(context.Background(), observation)
	})
	if err := scenarioRuntime.RecoverActivations(ctx); err != nil {
		logger.Error("recover scenario activations", "event", "scenario.recovery.failed", "error", err)
	}
	controlStream := transport.NewStreamHandler(controlService)
	bugReports := bugreport.NewService(cfg.LogDir, deviceManager, scenarioRuntime)
	webrtcEngine, err := transport.NewPionWebRTCSignalEngine()
	if err != nil {
		logger.Error("configure webrtc signal engine", "event", "transport.webrtc.configure_failed", "error", err)
		return
	}
	controlStream.SetWebRTCSignalEngine(webrtcEngine)
	photoServer, photoBaseURL, err := startPhotoFrameAssetServer(cfg)
	if err != nil {
		logger.Error("start photo frame asset server", "event", "transport.http.photo_frame.start_failed", "error", err)
		return
	}
	adminServer, err := startAdminServer(cfg, admin.NewHandler(
		controlService,
		scenarioRuntime,
		appRuntime,
		func() { registerAppScenarioDefinitions(scenarioEngine, appRuntime) },
		deviceManager,
		cfg,
	))
	if err != nil {
		logger.Error("start admin dashboard", "event", "admin.http.start_failed", "error", err)
		return
	}
	controlService.SetRegisterMetadata(map[string]string{
		"photo_frame_asset_base_url": photoBaseURL,
	})
	configurePhotoFrame(controlStream, cfg, photoBaseURL)
	recordingManager, err := recording.NewDiskManager(cfg.RecordingDir)
	if err != nil {
		logger.Error("configure recording manager", "event", "recording.configure_failed", "error", err)
		return
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

	logger.Info("terminal server starting", "event", "server.starting", "grpc_address", grpcServer.Address())

	if err := mdns.Start(ctx, discovery.ServiceInfo{
		ServiceType: cfg.MDNSService,
		Name:        cfg.MDNSName,
		Port:        cfg.GRPCPort,
		Version:     cfg.Version,
		GRPC:        cfg.GRPCAddress(),
		WebSocket:   fmt.Sprintf("ws://%s%s", cfg.ControlWSAddress(), websocketServer.Path()),
		TCP:         cfg.ControlTCPAddress(),
		HTTP:        fmt.Sprintf("http://%s", cfg.ControlHTTPAddress()),
		Priority:    []string{"grpc", "websocket", "tcp", "http"},
	}); err != nil {
		logger.Error("start mDNS", "event", "discovery.mdns.failed", "error", err)
		return
	}

	if err := grpcServer.Start(ctx); err != nil {
		logger.Error("start transport", "event", "transport.grpc.start_failed", "error", err)
		return
	}
	if err := websocketServer.Start(ctx); err != nil {
		logger.Error("start websocket transport", "event", "transport.websocket.start_failed", "error", err)
		return
	}
	if err := tcpServer.Start(ctx); err != nil {
		logger.Error("start tcp transport", "event", "transport.tcp.start_failed", "error", err)
		return
	}
	if err := httpControlServer.Start(ctx); err != nil {
		logger.Error("start http fallback transport", "event", "transport.http.start_failed", "error", err)
		return
	}

	// Keep this non-empty so startup validates major foundational services.
	if len(deviceManager.List()) != 0 {
		logger.Error("unexpected initial device registry state", "event", "server.bootstrap.invalid_state")
		return
	}
	if len(ioRouter.Routes()) != 0 {
		logger.Error("unexpected initial route registry state", "event", "server.bootstrap.invalid_state")
		return
	}
	if _, ok := scenarioEngine.Active("bootstrap"); ok {
		logger.Error("unexpected initial scenario state", "event", "server.bootstrap.invalid_state")
		return
	}
	if len(scheduler.List()) != 0 {
		logger.Error("unexpected initial scheduler state", "event", "server.bootstrap.invalid_state")
		return
	}
	logger.Info("control service ready", "event", "server.started", "server_id", cfg.MDNSName)
	logger.Info("websocket control ready", "event", "transport.websocket.ready", "websocket_address", websocketServer.Address(), "path", websocketServer.Path())
	logger.Info("tcp control ready", "event", "transport.tcp.ready", "tcp_address", tcpServer.Address())
	logger.Info("http fallback control ready", "event", "transport.http.ready", "http_address", httpControlServer.Address())
	if adminServer != nil {
		logger.Info("admin dashboard available", "event", "admin.http.ready", "addr", adminServer.Addr)
	}
	logger.Info("control stream handler initialized", "event", "transport.stream.ready")
	logger.Info("recording manager initialized", "event", "recording.started", "dir", cfg.RecordingDir)
	logger.Info("scenario runtime initialized", "event", "scenario.definition.registered", "builtin_scenarios", 3)
	logger.Info(
		"app runtime initialized",
		"event", "appruntime.package.loaded",
		"packages", len(appRuntime.ListPackages()),
		"definitions", len(appRuntime.Definitions()),
	)
	logger.Info(
		"housekeeping configured",
		"event", "housekeeping.configured",
		"heartbeat_timeout_seconds", cfg.HeartbeatTimeoutSeconds,
		"liveness_interval_seconds", cfg.LivenessReconcileIntervalSecs,
		"due_timer_interval_seconds", cfg.DueTimerProcessIntervalSecs,
	)
	_ = controlService
	_ = controlStream

	go runDueTimerLoop(ctx, scenarioRuntime, time.Duration(cfg.DueTimerProcessIntervalSecs)*time.Second)
	go runLivenessLoop(
		ctx,
		controlService,
		bugReports,
		time.Duration(cfg.HeartbeatTimeoutSeconds)*time.Second,
		time.Duration(cfg.LivenessReconcileIntervalSecs)*time.Second,
	)

	<-ctx.Done()
	logger.Info("terminal server shutting down", "event", "server.stopping")

	shutdownCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := httpControlServer.Stop(shutdownCtx); err != nil {
		logger.Error("stop http fallback transport", "event", "transport.http.stop_failed", "error", err)
	}
	if err := tcpServer.Stop(shutdownCtx); err != nil {
		logger.Error("stop tcp transport", "event", "transport.tcp.stop_failed", "error", err)
	}
	if err := websocketServer.Stop(shutdownCtx); err != nil {
		logger.Error("stop websocket transport", "event", "transport.websocket.stop_failed", "error", err)
	}
	if err := grpcServer.Stop(shutdownCtx); err != nil {
		logger.Error("stop transport", "event", "transport.grpc.stop_failed", "error", err)
	}
	if photoServer != nil {
		if err := photoServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("stop photo frame asset server", "event", "transport.http.photo_frame.stop_failed", "error", err)
		}
	}
	if adminServer != nil {
		if err := adminServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("stop admin dashboard", "event", "admin.http.stop_failed", "error", err)
		}
	}
	if err := mdns.Stop(shutdownCtx); err != nil {
		logger.Error("stop mDNS", "event", "discovery.mdns.stop_failed", "error", err)
	}
	if bridge, ok := telephonyBridge.(*telephony.SIPBridge); ok {
		if err := bridge.Stop(shutdownCtx); err != nil {
			logger.Error("stop telephony bridge", "event", "telephony.bridge.stop_failed", "error", err)
		}
	}
	logger.Info("terminal server stopped", "event", "server.stopped")
}

type scenarioAnalyzerRunner struct {
	Sound       scenario.SoundClassifier
	DeviceAudio scenario.DeviceAudioSubscriber
}

func (r scenarioAnalyzerRunner) StartAnalyzer(
	ctx context.Context,
	sourceDeviceID string,
	analyzer string,
	emit func(io.AnalyzerEvent),
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
				emit(io.AnalyzerEvent{
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

type worldModelAdapter struct {
	model *world.Model
}

func (w worldModelAdapter) LocateEntity(ctx context.Context, query scenario.EntityQuery) (*io.LocationEstimate, error) {
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
	bugs *bugreport.Service,
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
				for _, deviceID := range updatedIDs {
					if bugs == nil {
						continue
					}
					if _, err := bugs.FileAutodetect(ctx, deviceID, "heartbeat timeout or reconnect loop", nil); err != nil {
						logger.Error("autodetect bug filing failed",
							"event", "bug.report.autodetect.failed",
							"device_id", deviceID,
							"error", err,
						)
					}
				}
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
