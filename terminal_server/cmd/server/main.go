package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

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
	environment := &scenario.Environment{
		Devices:   deviceManager,
		IO:        ioRouter,
		AI:        ai.NoopBackend{},
		Telephony: telephony.NoopBridge{},
		Storage:   store,
		Scheduler: scheduler,
		Broadcast: broadcaster,
	}
	scenario.RegisterBuiltins(scenarioEngine)
	scenarioRuntime := scenario.NewRuntime(scenarioEngine, environment)
	controlStream := transport.NewStreamHandler(controlService)
	grpcServer := transport.NewServer(cfg.GRPCAddress())
	grpcServer.ConfigureControl(controlService, transport.PassthroughProtoAdapter{})
	grpcServer.ConfigureRuntime(scenarioRuntime)
	mdns := discovery.NoopAdvertiser{}

	log.Printf("terminal server starting at %s", grpcServer.Address())

	if err := mdns.Start(ctx, discovery.ServiceInfo{
		ServiceType: cfg.MDNSService,
		Name:        cfg.MDNSName,
		Port:        cfg.GRPCPort,
		Version:     cfg.Version,
	}); err != nil {
		log.Fatalf("start mDNS: %v", err)
	}

	if err := grpcServer.Start(ctx); err != nil {
		log.Fatalf("start transport: %v", err)
	}

	// Keep this non-empty so startup validates major foundational services.
	if len(deviceManager.List()) != 0 {
		log.Fatalf("unexpected initial device registry state")
	}
	if len(ioRouter.Routes()) != 0 {
		log.Fatalf("unexpected initial route registry state")
	}
	if _, ok := scenarioEngine.Active("bootstrap"); ok {
		log.Fatalf("unexpected initial scenario state")
	}
	if len(scheduler.List()) != 0 {
		log.Fatalf("unexpected initial scheduler state")
	}
	log.Printf("control service ready for server id %q", cfg.MDNSName)
	log.Printf("control stream handler initialized")
	log.Printf("scenario runtime initialized with %d builtin scenarios", 3)
	_ = controlService
	_ = controlStream
	_ = scenarioRuntime

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
}
