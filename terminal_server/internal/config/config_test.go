package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("TERMINALS_GRPC_HOST", "")
	t.Setenv("TERMINALS_GRPC_PORT", "")
	t.Setenv("TERMINALS_CONTROL_WS_HOST", "")
	t.Setenv("TERMINALS_CONTROL_WS_PORT", "")
	t.Setenv("TERMINALS_CONTROL_TCP_HOST", "")
	t.Setenv("TERMINALS_CONTROL_TCP_PORT", "")
	t.Setenv("TERMINALS_CONTROL_HTTP_HOST", "")
	t.Setenv("TERMINALS_CONTROL_HTTP_PORT", "")
	t.Setenv("TERMINALS_CONTROL_WS_ALLOWED_ORIGINS", "")
	t.Setenv("TERMINALS_ADMIN_HTTP_HOST", "")
	t.Setenv("TERMINALS_ADMIN_HTTP_PORT", "")
	t.Setenv("TERMINALS_LOG_DIR", "")
	t.Setenv("TERMINALS_LOG_LEVEL", "")
	t.Setenv("TERMINALS_LOG_MAX_BYTES", "")
	t.Setenv("TERMINALS_LOG_MAX_ARCHIVES", "")
	t.Setenv("TERMINALS_LOG_STDERR", "")
	t.Setenv("TERMINALS_PHOTO_FRAME_HTTP_HOST", "")
	t.Setenv("TERMINALS_PHOTO_FRAME_HTTP_PORT", "")
	t.Setenv("TERMINALS_PHOTO_FRAME_PUBLIC_BASE_URL", "")
	t.Setenv("TERMINALS_MDNS_SERVICE", "")
	t.Setenv("TERMINALS_MDNS_NAME", "")
	t.Setenv("TERMINALS_VERSION", "")
	t.Setenv("TERMINALS_WAKE_WORD_PREFIXES", "")
	t.Setenv("TERMINALS_HEARTBEAT_TIMEOUT_SECONDS", "")
	t.Setenv("TERMINALS_LIVENESS_RECONCILE_INTERVAL_SECONDS", "")
	t.Setenv("TERMINALS_DUE_TIMER_PROCESS_INTERVAL_SECONDS", "")
	t.Setenv("TERMINALS_PHOTO_FRAME_DIR", "")
	t.Setenv("TERMINALS_PHOTO_FRAME_INTERVAL_SECONDS", "")
	t.Setenv("TERMINALS_AGENT_OPERATIONAL_MAX_STREAMS", "")
	t.Setenv("TERMINALS_AGENT_OPERATIONAL_STREAM_TTL_SECONDS", "")
	t.Setenv("TERMINALS_AGENT_APPROVAL_MIN_HUMAN_LATENCY_MS", "")
	t.Setenv("TERMINALS_AGENT_APPROVAL_CONFIRMATION_TTL_SECONDS", "")

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
	if cfg.ControlWSHost != "0.0.0.0" {
		t.Fatalf("ControlWSHost = %q, want 0.0.0.0", cfg.ControlWSHost)
	}
	if cfg.ControlWSPort != 50054 {
		t.Fatalf("ControlWSPort = %d, want 50054", cfg.ControlWSPort)
	}
	if cfg.ControlTCPHost != "0.0.0.0" {
		t.Fatalf("ControlTCPHost = %q, want 0.0.0.0", cfg.ControlTCPHost)
	}
	if cfg.ControlTCPPort != 50055 {
		t.Fatalf("ControlTCPPort = %d, want 50055", cfg.ControlTCPPort)
	}
	if cfg.ControlHTTPHost != "0.0.0.0" {
		t.Fatalf("ControlHTTPHost = %q, want 0.0.0.0", cfg.ControlHTTPHost)
	}
	if cfg.ControlHTTPPort != 50056 {
		t.Fatalf("ControlHTTPPort = %d, want 50056", cfg.ControlHTTPPort)
	}
	if len(cfg.ControlWSAllowedOrigins) != 0 {
		t.Fatalf("ControlWSAllowedOrigins = %+v, want empty", cfg.ControlWSAllowedOrigins)
	}
	if cfg.AdminHTTPHost != "0.0.0.0" {
		t.Fatalf("AdminHTTPHost = %q, want 0.0.0.0", cfg.AdminHTTPHost)
	}
	if cfg.AdminHTTPPort != 50053 {
		t.Fatalf("AdminHTTPPort = %d, want 50053", cfg.AdminHTTPPort)
	}
	if cfg.LogDir != "logs" {
		t.Fatalf("LogDir = %q, want logs", cfg.LogDir)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.LogMaxBytes != 104857600 {
		t.Fatalf("LogMaxBytes = %d, want 104857600", cfg.LogMaxBytes)
	}
	if cfg.LogMaxArchives != 10 {
		t.Fatalf("LogMaxArchives = %d, want 10", cfg.LogMaxArchives)
	}
	if !cfg.LogStderr {
		t.Fatalf("LogStderr = false, want true")
	}
	if cfg.PhotoFrameHTTPHost != "0.0.0.0" {
		t.Fatalf("PhotoFrameHTTPHost = %q, want 0.0.0.0", cfg.PhotoFrameHTTPHost)
	}
	if cfg.PhotoFrameHTTPPort != 50052 {
		t.Fatalf("PhotoFrameHTTPPort = %d, want 50052", cfg.PhotoFrameHTTPPort)
	}
	if cfg.PhotoFramePublicBaseURL != "" {
		t.Fatalf("PhotoFramePublicBaseURL = %q, want empty", cfg.PhotoFramePublicBaseURL)
	}
	if cfg.RecordingDir != "recordings" {
		t.Fatalf("RecordingDir = %q, want recordings", cfg.RecordingDir)
	}
	if cfg.PhotoFrameDir != "" {
		t.Fatalf("PhotoFrameDir = %q, want empty", cfg.PhotoFrameDir)
	}
	if cfg.PhotoFrameIntervalSeconds != 12 {
		t.Fatalf("PhotoFrameIntervalSeconds = %d, want 12", cfg.PhotoFrameIntervalSeconds)
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
	if len(cfg.WakeWordPrefixes) != 2 || cfg.WakeWordPrefixes[0] != "assistant" || cfg.WakeWordPrefixes[1] != "hey terminal" {
		t.Fatalf("WakeWordPrefixes = %+v, want default assistant/hey terminal", cfg.WakeWordPrefixes)
	}
	if cfg.Agent.Operational.MaxStreams != 3 {
		t.Fatalf("Agent.Operational.MaxStreams = %d, want 3", cfg.Agent.Operational.MaxStreams)
	}
	if cfg.Agent.Operational.StreamTTLSeconds != 120 {
		t.Fatalf("Agent.Operational.StreamTTLSeconds = %d, want 120", cfg.Agent.Operational.StreamTTLSeconds)
	}
	if cfg.Agent.Approval.MinHumanLatencyMS != 500 {
		t.Fatalf("Agent.Approval.MinHumanLatencyMS = %d, want 500", cfg.Agent.Approval.MinHumanLatencyMS)
	}
	if cfg.Agent.Approval.ConfirmationTTLSeconds != 120 {
		t.Fatalf("Agent.Approval.ConfirmationTTLSeconds = %d, want 120", cfg.Agent.Approval.ConfirmationTTLSeconds)
	}
}

func TestLoadAgentMCPPolicyConfigFromEnv(t *testing.T) {
	t.Setenv("TERMINALS_AGENT_OPERATIONAL_MAX_STREAMS", "7")
	t.Setenv("TERMINALS_AGENT_OPERATIONAL_STREAM_TTL_SECONDS", "45")
	t.Setenv("TERMINALS_AGENT_APPROVAL_MIN_HUMAN_LATENCY_MS", "850")
	t.Setenv("TERMINALS_AGENT_APPROVAL_CONFIRMATION_TTL_SECONDS", "180")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Agent.Operational.MaxStreams != 7 {
		t.Fatalf("Agent.Operational.MaxStreams = %d, want 7", cfg.Agent.Operational.MaxStreams)
	}
	if cfg.Agent.Operational.StreamTTLSeconds != 45 {
		t.Fatalf("Agent.Operational.StreamTTLSeconds = %d, want 45", cfg.Agent.Operational.StreamTTLSeconds)
	}
	if cfg.Agent.Approval.MinHumanLatencyMS != 850 {
		t.Fatalf("Agent.Approval.MinHumanLatencyMS = %d, want 850", cfg.Agent.Approval.MinHumanLatencyMS)
	}
	if cfg.Agent.Approval.ConfirmationTTLSeconds != 180 {
		t.Fatalf("Agent.Approval.ConfirmationTTLSeconds = %d, want 180", cfg.Agent.Approval.ConfirmationTTLSeconds)
	}
}

func TestLoadWakeWordPrefixesFromEnv(t *testing.T) {
	t.Setenv("TERMINALS_WAKE_WORD_PREFIXES", "jarvis, computer ,   ,hey house")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.WakeWordPrefixes) != 3 {
		t.Fatalf("WakeWordPrefixes len = %d, want 3", len(cfg.WakeWordPrefixes))
	}
	if cfg.WakeWordPrefixes[0] != "jarvis" ||
		cfg.WakeWordPrefixes[1] != "computer" ||
		cfg.WakeWordPrefixes[2] != "hey house" {
		t.Fatalf("WakeWordPrefixes = %+v, want [jarvis computer hey house]", cfg.WakeWordPrefixes)
	}
}

func TestLoadRecordingDirFromEnv(t *testing.T) {
	t.Setenv("TERMINALS_RECORDING_DIR", "/tmp/terminals-recordings")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.RecordingDir != "/tmp/terminals-recordings" {
		t.Fatalf("RecordingDir = %q, want /tmp/terminals-recordings", cfg.RecordingDir)
	}
}

func TestLoadPhotoFrameConfigFromEnv(t *testing.T) {
	t.Setenv("TERMINALS_CONTROL_WS_HOST", "127.0.0.1")
	t.Setenv("TERMINALS_CONTROL_WS_PORT", "7002")
	t.Setenv("TERMINALS_CONTROL_TCP_HOST", "127.0.0.1")
	t.Setenv("TERMINALS_CONTROL_TCP_PORT", "7003")
	t.Setenv("TERMINALS_CONTROL_HTTP_HOST", "127.0.0.1")
	t.Setenv("TERMINALS_CONTROL_HTTP_PORT", "7004")
	t.Setenv("TERMINALS_CONTROL_WS_ALLOWED_ORIGINS", "http://localhost:60739,https://example.test")
	t.Setenv("TERMINALS_PHOTO_FRAME_DIR", "/tmp/terminals-photos")
	t.Setenv("TERMINALS_PHOTO_FRAME_INTERVAL_SECONDS", "30")
	t.Setenv("TERMINALS_ADMIN_HTTP_HOST", "127.0.0.1")
	t.Setenv("TERMINALS_ADMIN_HTTP_PORT", "7000")
	t.Setenv("TERMINALS_PHOTO_FRAME_HTTP_HOST", "127.0.0.1")
	t.Setenv("TERMINALS_PHOTO_FRAME_HTTP_PORT", "7001")
	t.Setenv("TERMINALS_PHOTO_FRAME_PUBLIC_BASE_URL", "https://photos.example.test/slides")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.PhotoFrameDir != "/tmp/terminals-photos" {
		t.Fatalf("PhotoFrameDir = %q, want /tmp/terminals-photos", cfg.PhotoFrameDir)
	}
	if cfg.ControlWSHost != "127.0.0.1" {
		t.Fatalf("ControlWSHost = %q, want 127.0.0.1", cfg.ControlWSHost)
	}
	if cfg.ControlWSPort != 7002 {
		t.Fatalf("ControlWSPort = %d, want 7002", cfg.ControlWSPort)
	}
	if cfg.ControlTCPHost != "127.0.0.1" {
		t.Fatalf("ControlTCPHost = %q, want 127.0.0.1", cfg.ControlTCPHost)
	}
	if cfg.ControlTCPPort != 7003 {
		t.Fatalf("ControlTCPPort = %d, want 7003", cfg.ControlTCPPort)
	}
	if cfg.ControlHTTPHost != "127.0.0.1" {
		t.Fatalf("ControlHTTPHost = %q, want 127.0.0.1", cfg.ControlHTTPHost)
	}
	if cfg.ControlHTTPPort != 7004 {
		t.Fatalf("ControlHTTPPort = %d, want 7004", cfg.ControlHTTPPort)
	}
	if len(cfg.ControlWSAllowedOrigins) != 2 ||
		cfg.ControlWSAllowedOrigins[0] != "http://localhost:60739" ||
		cfg.ControlWSAllowedOrigins[1] != "https://example.test" {
		t.Fatalf("ControlWSAllowedOrigins = %+v, want configured list", cfg.ControlWSAllowedOrigins)
	}
	if cfg.AdminHTTPHost != "127.0.0.1" {
		t.Fatalf("AdminHTTPHost = %q, want 127.0.0.1", cfg.AdminHTTPHost)
	}
	if cfg.AdminHTTPPort != 7000 {
		t.Fatalf("AdminHTTPPort = %d, want 7000", cfg.AdminHTTPPort)
	}
	if cfg.PhotoFrameIntervalSeconds != 30 {
		t.Fatalf("PhotoFrameIntervalSeconds = %d, want 30", cfg.PhotoFrameIntervalSeconds)
	}
	if cfg.PhotoFrameHTTPHost != "127.0.0.1" {
		t.Fatalf("PhotoFrameHTTPHost = %q, want 127.0.0.1", cfg.PhotoFrameHTTPHost)
	}
	if cfg.PhotoFrameHTTPPort != 7001 {
		t.Fatalf("PhotoFrameHTTPPort = %d, want 7001", cfg.PhotoFrameHTTPPort)
	}
	if cfg.PhotoFramePublicBaseURL != "https://photos.example.test/slides" {
		t.Fatalf("PhotoFramePublicBaseURL = %q, want configured URL", cfg.PhotoFramePublicBaseURL)
	}
}

func TestLoadEventLogConfigFromEnv(t *testing.T) {
	t.Setenv("TERMINALS_LOG_DIR", "/tmp/terminals-logs")
	t.Setenv("TERMINALS_LOG_LEVEL", "debug")
	t.Setenv("TERMINALS_LOG_MAX_BYTES", "1234")
	t.Setenv("TERMINALS_LOG_MAX_ARCHIVES", "7")
	t.Setenv("TERMINALS_LOG_STDERR", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.LogDir != "/tmp/terminals-logs" {
		t.Fatalf("LogDir = %q, want /tmp/terminals-logs", cfg.LogDir)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q, want debug", cfg.LogLevel)
	}
	if cfg.LogMaxBytes != 1234 {
		t.Fatalf("LogMaxBytes = %d, want 1234", cfg.LogMaxBytes)
	}
	if cfg.LogMaxArchives != 7 {
		t.Fatalf("LogMaxArchives = %d, want 7", cfg.LogMaxArchives)
	}
	if cfg.LogStderr {
		t.Fatalf("LogStderr = true, want false")
	}
}

func TestLoadInvalidPort(t *testing.T) {
	t.Setenv("TERMINALS_GRPC_PORT", "bad")
	if _, err := Load(); err == nil {
		t.Fatalf("Load() expected error for invalid port")
	}
}

func TestLoadInvalidControlWSPort(t *testing.T) {
	t.Setenv("TERMINALS_CONTROL_WS_PORT", "bad")
	if _, err := Load(); err == nil {
		t.Fatalf("Load() expected error for invalid control websocket port")
	}
}

func TestLoadInvalidControlTCPPort(t *testing.T) {
	t.Setenv("TERMINALS_CONTROL_TCP_PORT", "bad")
	if _, err := Load(); err == nil {
		t.Fatalf("Load() expected error for invalid control tcp port")
	}
}

func TestLoadInvalidControlHTTPPort(t *testing.T) {
	t.Setenv("TERMINALS_CONTROL_HTTP_PORT", "bad")
	if _, err := Load(); err == nil {
		t.Fatalf("Load() expected error for invalid control http port")
	}
}

func TestLoadRejectsWildcardControlWSAllowedOrigin(t *testing.T) {
	t.Setenv("TERMINALS_CONTROL_WS_ALLOWED_ORIGINS", "*")
	if _, err := Load(); err == nil {
		t.Fatalf("Load() expected error for wildcard control websocket origin")
	}
}

func TestLoadRejectsInvalidControlWSAllowedOrigin(t *testing.T) {
	t.Setenv("TERMINALS_CONTROL_WS_ALLOWED_ORIGINS", "not-an-origin")
	if _, err := Load(); err == nil {
		t.Fatalf("Load() expected error for malformed control websocket origin")
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

func TestLoadInvalidPhotoFrameInterval(t *testing.T) {
	t.Setenv("TERMINALS_PHOTO_FRAME_INTERVAL_SECONDS", "nope")
	if _, err := Load(); err == nil {
		t.Fatalf("Load() expected error for invalid photo frame interval")
	}
}

func TestLoadInvalidPhotoFrameHTTPPort(t *testing.T) {
	t.Setenv("TERMINALS_PHOTO_FRAME_HTTP_PORT", "wat")
	if _, err := Load(); err == nil {
		t.Fatalf("Load() expected error for invalid photo frame HTTP port")
	}
}

func TestLoadInvalidAdminHTTPPort(t *testing.T) {
	t.Setenv("TERMINALS_ADMIN_HTTP_PORT", "wat")
	if _, err := Load(); err == nil {
		t.Fatalf("Load() expected error for invalid admin HTTP port")
	}
}

func TestLoadSIPDisabledByDefault(t *testing.T) {
	t.Setenv("TERMINALS_SIP_ENABLED", "")
	t.Setenv("TERMINALS_SIP_SERVER_URI", "")
	t.Setenv("TERMINALS_SIP_USERNAME", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.SIP.Enabled {
		t.Fatalf("SIP.Enabled = true, want false")
	}
}

func TestLoadSIPEnabledRequiresServer(t *testing.T) {
	t.Setenv("TERMINALS_SIP_ENABLED", "true")
	t.Setenv("TERMINALS_SIP_SERVER_URI", "")
	t.Setenv("TERMINALS_SIP_USERNAME", "alice")

	if _, err := Load(); err == nil {
		t.Fatalf("Load() expected error for missing SIP server")
	}
}

func TestLoadSIPEnabledRequiresUsername(t *testing.T) {
	t.Setenv("TERMINALS_SIP_ENABLED", "true")
	t.Setenv("TERMINALS_SIP_SERVER_URI", "sip:home.example")
	t.Setenv("TERMINALS_SIP_USERNAME", "")

	if _, err := Load(); err == nil {
		t.Fatalf("Load() expected error for missing SIP username")
	}
}

func TestLoadSIPEnabledPopulatesConfig(t *testing.T) {
	t.Setenv("TERMINALS_SIP_ENABLED", "1")
	t.Setenv("TERMINALS_SIP_SERVER_URI", "sip:home.example")
	t.Setenv("TERMINALS_SIP_USERNAME", "alice")
	t.Setenv("TERMINALS_SIP_DISPLAY_NAME", "Alice")
	t.Setenv("TERMINALS_SIP_PASSWORD", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.SIP.Enabled {
		t.Fatalf("SIP.Enabled = false, want true")
	}
	if cfg.SIP.ServerURI != "sip:home.example" {
		t.Fatalf("SIP.ServerURI = %q", cfg.SIP.ServerURI)
	}
	if cfg.SIP.Username != "alice" {
		t.Fatalf("SIP.Username = %q", cfg.SIP.Username)
	}
	if cfg.SIP.DisplayName != "Alice" {
		t.Fatalf("SIP.DisplayName = %q", cfg.SIP.DisplayName)
	}
	if cfg.SIP.Password != "secret" {
		t.Fatalf("SIP.Password = %q", cfg.SIP.Password)
	}
}

func TestLoadSIPInvalidEnabledValue(t *testing.T) {
	t.Setenv("TERMINALS_SIP_ENABLED", "bogus")
	if _, err := Load(); err == nil {
		t.Fatalf("Load() expected error for invalid SIP enabled flag")
	}
}
