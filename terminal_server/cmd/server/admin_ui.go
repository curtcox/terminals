package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/config"
)

func startAdminServer(cfg config.Config, handler http.Handler) (*http.Server, error) {
	if handler == nil {
		return nil, fmt.Errorf("admin handler is required")
	}
	address := net.JoinHostPort(cfg.AdminHTTPHost, fmt.Sprintf("%d", cfg.AdminHTTPPort))
	server := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	server.Addr = listener.Addr().String()

	go func() {
		log.Printf("admin dashboard listening at %s", server.Addr)
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("admin dashboard server error: %v", err)
		}
	}()
	return server, nil
}
