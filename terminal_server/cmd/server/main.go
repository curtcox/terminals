// Package main is the entry point for the terminal server binary.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/ai"
	"github.com/curtcox/terminals/terminal_server/internal/config"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/discovery"
	"github.com/curtcox/terminals/terminal_server/internal/io"
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
	environment := &scenario.Environment{
		Devices:   deviceManager,
		IO:        ioRouter,
		AI:        ai.LLMQueryAdapter{LLM: aiBackends.LLM},
		LLM:       scenarioLLM{backend: aiBackends.LLM},
		STT:       scenarioSTT{backend: aiBackends.STT},
		WakeWord:  scenario.PrefixWakeWordDetector{Prefixes: cfg.WakeWordPrefixes},
		TTS:       scenarioTTS{backend: aiBackends.TTS},
		Telephony: telephonyBridge,
		Storage:   store,
		Scheduler: scheduler,
		Broadcast: broadcaster,
	}
	scenario.RegisterBuiltins(scenarioEngine)
	scenarioRuntime := scenario.NewRuntime(scenarioEngine, environment)
	controlStream := transport.NewStreamHandler(controlService)
	grpcServer := transport.NewServer(cfg.GRPCAddress())
	grpcServer.ConfigureControl(controlService, transport.GeneratedProtoAdapter{})
	grpcServer.ConfigureRuntime(scenarioRuntime)
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
	log.Printf("control stream handler initialized")
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
	if err := mdns.Stop(shutdownCtx); err != nil {
		log.Printf("stop mDNS: %v", err)
	}
	if bridge, ok := telephonyBridge.(*telephony.SIPBridge); ok {
		if err := bridge.Stop(shutdownCtx); err != nil {
			log.Printf("stop telephony bridge: %v", err)
		}
	}
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
