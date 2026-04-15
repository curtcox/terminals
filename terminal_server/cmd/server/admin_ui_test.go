package main

import (
	"net/http"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/config"
)

func TestStartAdminServerRequiresHandler(t *testing.T) {
	if _, err := startAdminServer(config.Config{AdminHTTPHost: "127.0.0.1", AdminHTTPPort: 0}, nil); err == nil {
		t.Fatalf("startAdminServer expected error when handler is nil")
	}
}

func TestStartAdminServerServesRequests(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/admin", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	server, err := startAdminServer(config.Config{AdminHTTPHost: "127.0.0.1", AdminHTTPPort: 0}, handler)
	if err != nil {
		t.Fatalf("startAdminServer() error = %v", err)
	}
	t.Cleanup(func() {
		_ = server.Close()
	})

	baseURL := "http://" + server.Addr
	resp, err := http.Get(baseURL + "/admin")
	if err != nil {
		t.Fatalf("GET /admin error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /admin status = %d, want 200", resp.StatusCode)
	}
}
