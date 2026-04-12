package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("TERMINALS_GRPC_HOST", "")
	t.Setenv("TERMINALS_GRPC_PORT", "")
	t.Setenv("TERMINALS_MDNS_SERVICE", "")
	t.Setenv("TERMINALS_MDNS_NAME", "")
	t.Setenv("TERMINALS_VERSION", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.GRPCHost != "0.0.0.0" {
		t.Fatalf("GRPCHost = %q", cfg.GRPCHost)
	}
	if cfg.GRPCPort != 50051 {
		t.Fatalf("GRPCPort = %d", cfg.GRPCPort)
	}
}

func TestLoadInvalidPort(t *testing.T) {
	t.Setenv("TERMINALS_GRPC_PORT", "bad")
	if _, err := Load(); err == nil {
		t.Fatalf("Load() expected error for invalid port")
	}
}
