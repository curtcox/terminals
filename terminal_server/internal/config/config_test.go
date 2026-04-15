package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("TERMINALS_GRPC_HOST", "")
	t.Setenv("TERMINALS_GRPC_PORT", "")
	t.Setenv("TERMINALS_ADMIN_HTTP_HOST", "")
	t.Setenv("TERMINALS_ADMIN_HTTP_PORT", "")
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
	if cfg.AdminHTTPHost != "0.0.0.0" {
		t.Fatalf("AdminHTTPHost = %q, want 0.0.0.0", cfg.AdminHTTPHost)
	}
	if cfg.AdminHTTPPort != 50053 {
		t.Fatalf("AdminHTTPPort = %d, want 50053", cfg.AdminHTTPPort)
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
