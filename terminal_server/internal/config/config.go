// Package config loads runtime server configuration.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
	SIP                           SIPConfig
}

// SIPConfig captures the subset of server configuration relevant to the
// telephony (SIP) bridge.
type SIPConfig struct {
	Enabled     bool
	ServerURI   string
	Username    string
	DisplayName string
	Password    string
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

	sip, err := loadSIPConfig()
	if err != nil {
		return Config{}, err
	}
	cfg.SIP = sip

	return cfg, nil
}

func loadSIPConfig() (SIPConfig, error) {
	cfg := SIPConfig{
		ServerURI:   os.Getenv("TERMINALS_SIP_SERVER_URI"),
		Username:    os.Getenv("TERMINALS_SIP_USERNAME"),
		DisplayName: os.Getenv("TERMINALS_SIP_DISPLAY_NAME"),
		Password:    os.Getenv("TERMINALS_SIP_PASSWORD"),
	}

	enabled, err := parseOptionalBool("TERMINALS_SIP_ENABLED")
	if err != nil {
		return SIPConfig{}, err
	}
	cfg.Enabled = enabled

	if cfg.Enabled && cfg.ServerURI == "" {
		return SIPConfig{}, fmt.Errorf("TERMINALS_SIP_SERVER_URI is required when TERMINALS_SIP_ENABLED is set")
	}
	if cfg.Enabled && cfg.Username == "" {
		return SIPConfig{}, fmt.Errorf("TERMINALS_SIP_USERNAME is required when TERMINALS_SIP_ENABLED is set")
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

func parseOptionalBool(env string) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(env))
	if raw == "" {
		return false, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", env, err)
	}
	return v, nil
}
