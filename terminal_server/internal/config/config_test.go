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
	t.Setenv("TERMINALS_HEARTBEAT_TIMEOUT_SECONDS", "")
	t.Setenv("TERMINALS_LIVENESS_RECONCILE_INTERVAL_SECONDS", "")
	t.Setenv("TERMINALS_DUE_TIMER_PROCESS_INTERVAL_SECONDS", "")

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
	if cfg.HeartbeatTimeoutSeconds != 120 {
		t.Fatalf("HeartbeatTimeoutSeconds = %d", cfg.HeartbeatTimeoutSeconds)
	}
	if cfg.LivenessReconcileIntervalSecs != 30 {
		t.Fatalf("LivenessReconcileIntervalSecs = %d", cfg.LivenessReconcileIntervalSecs)
	}
	if cfg.DueTimerProcessIntervalSecs != 5 {
		t.Fatalf("DueTimerProcessIntervalSecs = %d", cfg.DueTimerProcessIntervalSecs)
	}
}

func TestLoadInvalidPort(t *testing.T) {
	t.Setenv("TERMINALS_GRPC_PORT", "bad")
	if _, err := Load(); err == nil {
		t.Fatalf("Load() expected error for invalid port")
	}
}

func TestLoadIntervals(t *testing.T) {
	t.Setenv("TERMINALS_HEARTBEAT_TIMEOUT_SECONDS", "45")
	t.Setenv("TERMINALS_LIVENESS_RECONCILE_INTERVAL_SECONDS", "7")
	t.Setenv("TERMINALS_DUE_TIMER_PROCESS_INTERVAL_SECONDS", "3")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HeartbeatTimeoutSeconds != 45 {
		t.Fatalf("HeartbeatTimeoutSeconds = %d, want 45", cfg.HeartbeatTimeoutSeconds)
	}
	if cfg.LivenessReconcileIntervalSecs != 7 {
		t.Fatalf("LivenessReconcileIntervalSecs = %d, want 7", cfg.LivenessReconcileIntervalSecs)
	}
	if cfg.DueTimerProcessIntervalSecs != 3 {
		t.Fatalf("DueTimerProcessIntervalSecs = %d, want 3", cfg.DueTimerProcessIntervalSecs)
	}
}

func TestLoadInvalidInterval(t *testing.T) {
	t.Setenv("TERMINALS_HEARTBEAT_TIMEOUT_SECONDS", "abc")
	if _, err := Load(); err == nil {
		t.Fatalf("Load() expected error for invalid interval")
	}
}
