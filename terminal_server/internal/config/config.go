package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config captures runtime server settings.
type Config struct {
	GRPCHost    string
	GRPCPort    int
	MDNSService string
	MDNSName    string
	Version     string
}

// Load reads config from environment with sane defaults for local development.
func Load() (Config, error) {
	cfg := Config{
		GRPCHost:    getenv("TERMINALS_GRPC_HOST", "0.0.0.0"),
		GRPCPort:    50051,
		MDNSService: getenv("TERMINALS_MDNS_SERVICE", "_terminals._tcp.local."),
		MDNSName:    getenv("TERMINALS_MDNS_NAME", "HomeServer"),
		Version:     getenv("TERMINALS_VERSION", "1"),
	}

	if rawPort := os.Getenv("TERMINALS_GRPC_PORT"); rawPort != "" {
		parsed, err := strconv.Atoi(rawPort)
		if err != nil {
			return Config{}, fmt.Errorf("parse TERMINALS_GRPC_PORT: %w", err)
		}
		cfg.GRPCPort = parsed
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
