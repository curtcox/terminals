// Package config loads runtime server configuration.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config captures runtime server settings.
type Config struct {
	GRPCHost                      string
	GRPCPort                      int
	MDNSService                   string
	MDNSName                      string
	Version                       string
	HeartbeatTimeoutSeconds       int
	LivenessReconcileIntervalSecs int
	DueTimerProcessIntervalSecs   int
}

// Load reads config from environment with sane defaults for local development.
func Load() (Config, error) {
	cfg := Config{
		GRPCHost:                      getenv("TERMINALS_GRPC_HOST", "0.0.0.0"),
		GRPCPort:                      50051,
		MDNSService:                   getenv("TERMINALS_MDNS_SERVICE", "_terminals._tcp.local."),
		MDNSName:                      getenv("TERMINALS_MDNS_NAME", "HomeServer"),
		Version:                       getenv("TERMINALS_VERSION", "1"),
		HeartbeatTimeoutSeconds:       120,
		LivenessReconcileIntervalSecs: 30,
		DueTimerProcessIntervalSecs:   5,
	}

	if rawPort := os.Getenv("TERMINALS_GRPC_PORT"); rawPort != "" {
		parsed, err := strconv.Atoi(rawPort)
		if err != nil {
			return Config{}, fmt.Errorf("parse TERMINALS_GRPC_PORT: %w", err)
		}
		cfg.GRPCPort = parsed
	}
	if v, ok, err := parseOptionalInt("TERMINALS_HEARTBEAT_TIMEOUT_SECONDS"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.HeartbeatTimeoutSeconds = v
	}
	if v, ok, err := parseOptionalInt("TERMINALS_LIVENESS_RECONCILE_INTERVAL_SECONDS"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.LivenessReconcileIntervalSecs = v
	}
	if v, ok, err := parseOptionalInt("TERMINALS_DUE_TIMER_PROCESS_INTERVAL_SECONDS"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.DueTimerProcessIntervalSecs = v
	}

	return cfg, nil
}

// GRPCAddress returns host:port.
func (c Config) GRPCAddress() string {
	return fmt.Sprintf("%s:%d", c.GRPCHost, c.GRPCPort)
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func parseOptionalInt(env string) (int, bool, error) {
	raw := os.Getenv(env)
	if raw == "" {
		return 0, false, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, true, fmt.Errorf("parse %s: %w", env, err)
	}
	return v, true, nil
}
