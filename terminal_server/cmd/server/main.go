package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/curtcox/terminals/terminal_server/internal/config"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/discovery"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	deviceManager := device.NewManager()
	grpcServer := transport.NewServer(cfg.GRPCAddress())
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
