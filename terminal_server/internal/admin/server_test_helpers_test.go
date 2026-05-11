package admin

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/curtcox/terminals/terminal_server/internal/config"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
	"github.com/curtcox/terminals/terminal_server/internal/world"
)

func testHandler(t *testing.T, cfgOverride ...config.Config) http.Handler {
	t.Helper()
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	control := transport.NewControlService("HomeServer", devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	worldModel := world.NewModel()
	worldModel.UpsertGeometry(context.Background(), world.DeviceGeometry{DeviceID: "d1", Zone: "kitchen"})

	cfg := config.Config{
		GRPCHost:      "0.0.0.0",
		GRPCPort:      50051,
		MDNSService:   "_terminals._tcp.local.",
		MDNSName:      "HomeServer",
		Version:       "1",
		AdminHTTPHost: "127.0.0.1",
		AdminHTTPPort: 50053,
		LogDir:        filepath.Join(t.TempDir(), "logs"),
	}
	if len(cfgOverride) > 0 {
		override := cfgOverride[0]
		if strings.TrimSpace(override.MDNSName) != "" {
			cfg.MDNSName = override.MDNSName
		}
		if strings.TrimSpace(override.LogDir) != "" {
			cfg.LogDir = override.LogDir
		}
	}
	return NewHandler(control, runtime, nil, nil, nil, nil, devices, cfg, worldModel, nil)
}

func createTestAppPackage(t *testing.T, name, version string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll(app) error = %v", err)
	}
	manifest := "name = \"" + name + "\"\nversion = \"" + version + "\"\nlanguage = \"tal/1\"\nexports = [\"watch\"]\n"
	if err := os.WriteFile(filepath.Join(root, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.tal"), []byte("def on_start(): pass\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main.tal) error = %v", err)
	}
	return root
}
