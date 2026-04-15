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
	"github.com/curtcox/terminals/terminal_server/internal/audio"
	"github.com/curtcox/terminals/terminal_server/internal/config"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/discovery"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/placement"
	"github.com/curtcox/terminals/terminal_server/internal/recording"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	deviceManager := device.NewManager()
	ioRouter := io.NewRouter()
	audioHub := audio.NewHub()
	scenarioEngine := scenario.NewEngine()
	controlService := transport.NewControlService(cfg.MDNSName, deviceManager)
	store := storage.NewMemoryStore()
	scheduler := storage.NewMemoryScheduler()
	broadcaster := ui.NewMemoryBroadcaster()
	telephonyBridge, err := buildTelephonyBridge(ctx, cfg.SIP)
	if err != nil {
		log.Printf("configure telephony bridge: %v", err)
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
	if err := scenarioRuntime.RecoverActivations(ctx); err != nil {
		log.Printf("recover scenario activations: %v", err)
	}
	controlStream := transport.NewStreamHandler(controlService)
	webrtcEngine, err := transport.NewPionWebRTCSignalEngine()
	if err != nil {
		log.Printf("configure webrtc signal engine: %v", err)
		return
	}
	controlStream.SetWebRTCSignalEngine(webrtcEngine)
	photoServer, photoBaseURL, err := startPhotoFrameAssetServer(cfg)
	if err != nil {
		log.Printf("start photo frame asset server: %v", err)
		return
	}
	adminServer, err := startAdminServer(cfg, admin.NewHandler(controlService, scenarioRuntime, deviceManager, cfg))
	if err != nil {
		log.Printf("start admin dashboard: %v", err)
		return
	}
	controlService.SetRegisterMetadata(map[string]string{
		"photo_frame_asset_base_url": photoBaseURL,
	})
	configurePhotoFrame(controlStream, cfg, photoBaseURL)
	recordingManager, err := recording.NewDiskManager(cfg.RecordingDir)
	if err != nil {
		log.Printf("configure recording manager: %v", err)
		return
	}
	controlStream.SetRecordingManager(recordingManager)
	grpcServer := transport.NewServer(cfg.GRPCAddress())
	grpcServer.ConfigureControl(controlService, transport.GeneratedProtoAdapter{})
	grpcServer.ConfigureRuntime(scenarioRuntime)
	grpcServer.ConfigureDeviceAudio(audioHub)
	grpcServer.ConfigureRecording(recordingManager)
	grpcServer.ConfigureWebRTCSignalEngine(webrtcEngine)
	mdns := discovery.NewMDNSAdvertiser()

	log.Printf("terminal server starting at %s", grpcServer.Address())

	if err := mdns.Start(ctx, discovery.ServiceInfo{
		ServiceType: cfg.MDNSService,
		Name:        cfg.MDNSName,
		Port:        cfg.GRPCPort,
		Version:     cfg.Version,
	}); err != nil {
		log.Printf("start mDNS: %v", err)
		return
	}

	if err := grpcServer.Start(ctx); err != nil {
		log.Printf("start transport: %v", err)
		return
	}

	// Keep this non-empty so startup validates major foundational services.
	if len(deviceManager.List()) != 0 {
		log.Printf("unexpected initial device registry state")
		return
	}
	if len(ioRouter.Routes()) != 0 {
		log.Printf("unexpected initial route registry state")
		return
	}
	if _, ok := scenarioEngine.Active("bootstrap"); ok {
		log.Printf("unexpected initial scenario state")
		return
	}
	if len(scheduler.List()) != 0 {
		log.Printf("unexpected initial scheduler state")
		return
	}
	log.Printf("control service ready for server id %q", cfg.MDNSName)
	if adminServer != nil {
		log.Printf("admin dashboard available at http://%s/admin", adminServer.Addr)
	}
	log.Printf("control stream handler initialized")
	log.Printf("recording manager initialized dir=%s", cfg.RecordingDir)
	log.Printf("scenario runtime initialized with %d builtin scenarios", 3)
	log.Printf(
		"housekeeping configured heartbeat_timeout=%ds liveness_interval=%ds due_timer_interval=%ds",
		cfg.HeartbeatTimeoutSeconds,
		cfg.LivenessReconcileIntervalSecs,
		cfg.DueTimerProcessIntervalSecs,
	)
	_ = controlService
	_ = controlStream

	go runDueTimerLoop(ctx, scenarioRuntime, time.Duration(cfg.DueTimerProcessIntervalSecs)*time.Second)
	go runLivenessLoop(ctx, controlService, time.Duration(cfg.HeartbeatTimeoutSeconds)*time.Second, time.Duration(cfg.LivenessReconcileIntervalSecs)*time.Second)

	<-ctx.Done()
	log.Println("terminal server shutting down")

	shutdownCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := grpcServer.Stop(shutdownCtx); err != nil {
		log.Printf("stop transport: %v", err)
	}
	if photoServer != nil {
		if err := photoServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("stop photo frame asset server: %v", err)
		}
	}
	if adminServer != nil {
		if err := adminServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("stop admin dashboard: %v", err)
		}
	}
	if err := mdns.Stop(shutdownCtx); err != nil {
		log.Printf("stop mDNS: %v", err)
	}
	if bridge, ok := telephonyBridge.(*telephony.SIPBridge); ok {
		if err := bridge.Stop(shutdownCtx); err != nil {
			log.Printf("stop telephony bridge: %v", err)
		}
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

// buildTelephonyBridge returns the configured telephony bridge for the
// server runtime. When SIP is disabled a NoopBridge is returned so
// scenarios continue to function without a SIP provider.
func buildTelephonyBridge(ctx context.Context, cfg config.SIPConfig) (scenario.TelephonyBridge, error) {
	if !cfg.Enabled {
		log.Printf("telephony bridge disabled; using noop bridge")
		return telephony.NoopBridge{}, nil
	}
	bridge := telephony.NewSIPBridge(
		telephony.Registration{
			ServerURI:   cfg.ServerURI,
			Username:    cfg.Username,
			DisplayName: cfg.DisplayName,
			Password:    cfg.Password,
		},
		telephony.LogTransport{Logf: log.Printf},
		telephony.WithMediaTransport(telephony.LogMediaTransport{Logf: log.Printf}),
	)
	if err := bridge.Start(ctx); err != nil {
		return nil, err
	}
	log.Printf(
		"telephony bridge registered server=%s user=%s display=%s",
		cfg.ServerURI,
		cfg.Username,
		cfg.DisplayName,
	)
	return bridge, nil
}

func runDueTimerLoop(ctx context.Context, runtime *scenario.Runtime, interval time.Duration) {
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
				log.Printf("due timer loop error: %v", err)
				continue
			}
			if processed > 0 {
				log.Printf("due timer loop processed=%d", processed)
			}
		}
	}
}

func runLivenessLoop(ctx context.Context, control *transport.ControlService, timeout, interval time.Duration) {
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
			updated := control.ReconcileLiveness(timeout)
			if updated > 0 {
				log.Printf("liveness reconcile updated=%d", updated)
			}
		}
	}
}

func configurePhotoFrame(handler *transport.StreamHandler, cfg config.Config, baseURL string) {
	if handler == nil {
		return
	}
	interval := time.Duration(cfg.PhotoFrameIntervalSeconds) * time.Second
	slides, err := loadPhotoFrameSlides(cfg.PhotoFrameDir, baseURL)
	if err != nil {
		log.Printf("photo frame slide discovery failed dir=%q err=%v", cfg.PhotoFrameDir, err)
	}
	handler.SetPhotoFrameSettings(slides, interval)
	log.Printf(
		"photo frame configured slides=%d dir=%q interval=%ds base_url=%q",
		len(slides),
		cfg.PhotoFrameDir,
		cfg.PhotoFrameIntervalSeconds,
		baseURL,
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
		log.Printf("photo frame asset server listening at %s", address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("photo frame asset server error: %v", err)
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
