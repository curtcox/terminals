package main

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/config"
	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
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
		logger := eventlog.Component("admin.http")
		logger.Info("admin dashboard listening", "event", "admin.http.listener_ready", "addr", server.Addr)
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Error("admin dashboard server error", "event", "admin.http.server_error", "error", err)
		}
	}()
	return server, nil
}
