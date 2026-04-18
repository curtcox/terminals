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
	ControlWSHost                 string
	ControlWSPort                 int
	ControlTCPHost                string
	ControlTCPPort                int
	ControlHTTPHost               string
	ControlHTTPPort               int
	ControlWSAllowedOrigins       []string
	AdminHTTPHost                 string
	AdminHTTPPort                 int
	LogDir                        string
	LogLevel                      string
	LogMaxBytes                   int64
	LogMaxArchives                int
	LogStderr                     bool
	PhotoFrameHTTPHost            string
	PhotoFrameHTTPPort            int
	PhotoFramePublicBaseURL       string
	RecordingDir                  string
	PhotoFrameDir                 string
	PhotoFrameIntervalSeconds     int
	MDNSService                   string
	MDNSName                      string
	Version                       string
	WakeWordPrefixes              []string
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
		ControlWSHost:                 getenv("TERMINALS_CONTROL_WS_HOST", "0.0.0.0"),
		ControlWSPort:                 50054,
		ControlTCPHost:                getenv("TERMINALS_CONTROL_TCP_HOST", "0.0.0.0"),
		ControlTCPPort:                50055,
		ControlHTTPHost:               getenv("TERMINALS_CONTROL_HTTP_HOST", "0.0.0.0"),
		ControlHTTPPort:               50056,
		ControlWSAllowedOrigins:       []string{},
		AdminHTTPHost:                 getenv("TERMINALS_ADMIN_HTTP_HOST", "0.0.0.0"),
		AdminHTTPPort:                 50053,
		LogDir:                        getenv("TERMINALS_LOG_DIR", "logs"),
		LogLevel:                      getenv("TERMINALS_LOG_LEVEL", "info"),
		LogMaxBytes:                   104857600,
		LogMaxArchives:                10,
		LogStderr:                     true,
		PhotoFrameHTTPHost:            getenv("TERMINALS_PHOTO_FRAME_HTTP_HOST", "0.0.0.0"),
		PhotoFrameHTTPPort:            50052,
		PhotoFramePublicBaseURL:       strings.TrimSpace(os.Getenv("TERMINALS_PHOTO_FRAME_PUBLIC_BASE_URL")),
		RecordingDir:                  getenv("TERMINALS_RECORDING_DIR", "recordings"),
		PhotoFrameDir:                 getenv("TERMINALS_PHOTO_FRAME_DIR", ""),
		PhotoFrameIntervalSeconds:     12,
		MDNSService:                   getenv("TERMINALS_MDNS_SERVICE", "_terminals._tcp.local."),
		MDNSName:                      getenv("TERMINALS_MDNS_NAME", "HomeServer"),
		Version:                       getenv("TERMINALS_VERSION", "1"),
		WakeWordPrefixes:              []string{"assistant", "hey terminal"},
		HeartbeatTimeoutSeconds:       120,
		LivenessReconcileIntervalSecs: 30,
		DueTimerProcessIntervalSecs:   5,
	}

	if prefixes := parseCSVStrings(os.Getenv("TERMINALS_WAKE_WORD_PREFIXES")); len(prefixes) > 0 {
		cfg.WakeWordPrefixes = prefixes
	}

	if rawPort := os.Getenv("TERMINALS_GRPC_PORT"); rawPort != "" {
		parsed, err := strconv.Atoi(rawPort)
		if err != nil {
			return Config{}, fmt.Errorf("parse TERMINALS_GRPC_PORT: %w", err)
		}
		cfg.GRPCPort = parsed
	}
	if v, ok, err := parseOptionalInt("TERMINALS_CONTROL_WS_PORT"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.ControlWSPort = v
	}
	if v, ok, err := parseOptionalInt("TERMINALS_CONTROL_TCP_PORT"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.ControlTCPPort = v
	}
	if v, ok, err := parseOptionalInt("TERMINALS_CONTROL_HTTP_PORT"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.ControlHTTPPort = v
	}
	if origins := parseCSVStrings(os.Getenv("TERMINALS_CONTROL_WS_ALLOWED_ORIGINS")); len(origins) > 0 {
		cfg.ControlWSAllowedOrigins = origins
	}
	if v, ok, err := parseOptionalInt("TERMINALS_PHOTO_FRAME_HTTP_PORT"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.PhotoFrameHTTPPort = v
	}
	if v, ok, err := parseOptionalInt("TERMINALS_ADMIN_HTTP_PORT"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.AdminHTTPPort = v
	}
	if v, ok, err := parseOptionalInt("TERMINALS_LOG_MAX_BYTES"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.LogMaxBytes = int64(v)
	}
	if v, ok, err := parseOptionalInt("TERMINALS_LOG_MAX_ARCHIVES"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.LogMaxArchives = v
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
	if v, ok, err := parseOptionalInt("TERMINALS_PHOTO_FRAME_INTERVAL_SECONDS"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.PhotoFrameIntervalSeconds = v
	}
	if v, err := parseOptionalBool("TERMINALS_LOG_STDERR"); err != nil {
		return Config{}, err
	} else if strings.TrimSpace(os.Getenv("TERMINALS_LOG_STDERR")) != "" {
		cfg.LogStderr = v
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

// ControlWSAddress returns host:port for the websocket control listener.
func (c Config) ControlWSAddress() string {
	return fmt.Sprintf("%s:%d", c.ControlWSHost, c.ControlWSPort)
}

// ControlTCPAddress returns host:port for the TCP control listener.
func (c Config) ControlTCPAddress() string {
	return fmt.Sprintf("%s:%d", c.ControlTCPHost, c.ControlTCPPort)
}

// ControlHTTPAddress returns host:port for the HTTP fallback control listener.
func (c Config) ControlHTTPAddress() string {
	return fmt.Sprintf("%s:%d", c.ControlHTTPHost, c.ControlHTTPPort)
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

func parseCSVStrings(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
