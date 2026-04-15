package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/config"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestLoadPhotoFrameSlidesSortedAndFiltered(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"z.png", "a.jpg", "note.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", name, err)
		}
	}

	slides, err := loadPhotoFrameSlides(dir, "http://HomeServer.local:50052/photo-frame")
	if err != nil {
		t.Fatalf("loadPhotoFrameSlides() error = %v", err)
	}
	if len(slides) != 2 {
		t.Fatalf("len(slides) = %d, want 2", len(slides))
	}
	if slides[0] != "http://HomeServer.local:50052/photo-frame/a.jpg" {
		t.Fatalf("slides[0] = %q, want HTTP URL ending in a.jpg", slides[0])
	}
	if slides[1] != "http://HomeServer.local:50052/photo-frame/z.png" {
		t.Fatalf("slides[1] = %q, want HTTP URL ending in z.png", slides[1])
	}
}

func TestConfigurePhotoFrameUsesDirectorySlidesAndInterval(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.jpg", "b.jpg"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", name, err)
		}
	}

	devices := device.NewManager()
	control := transport.NewControlService("srv-1", devices)
	broadcaster := ui.NewMemoryBroadcaster()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Telephony: telephony.NoopBridge{},
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Broadcast: broadcaster,
	})
	now := time.Date(2026, 4, 14, 18, 0, 0, 0, time.UTC)
	control.SetNowForTest(func() time.Time { return now })

	handler := transport.NewStreamHandlerWithRuntime(control, runtime)
	configurePhotoFrame(handler, config.Config{
		PhotoFrameDir:             dir,
		PhotoFrameIntervalSeconds: 1,
	}, "https://photos.example.test/frame")

	_, _ = handler.HandleMessage(context.Background(), transport.ClientMessage{
		Register: &transport.RegisterRequest{DeviceID: "device-1", DeviceName: "Kitchen Chromebook"},
	})
	startOut, err := handler.HandleMessage(context.Background(), transport.ClientMessage{
		Command: &transport.CommandRequest{
			RequestID: "cmd-photo-start",
			DeviceID:  "device-1",
			Kind:      "manual",
			Intent:    "photo frame",
		},
	})
	if err != nil {
		t.Fatalf("photo frame start error = %v", err)
	}
	if len(startOut) < 2 || startOut[1].SetUI == nil {
		t.Fatalf("expected initial photo frame SetUI, got %+v", startOut)
	}
	firstURL := findNodePropValue(*startOut[1].SetUI, "photo_frame_image", "url")
	if !strings.HasSuffix(firstURL, "/frame/a.jpg") {
		t.Fatalf("first photo url = %q, want a.jpg from configured directory", firstURL)
	}

	now = now.Add(2 * time.Second)
	heartbeatOut, err := handler.HandleMessage(context.Background(), transport.ClientMessage{
		Heartbeat: &transport.HeartbeatRequest{DeviceID: "device-1"},
	})
	if err != nil {
		t.Fatalf("heartbeat error = %v", err)
	}
	if len(heartbeatOut) != 1 || heartbeatOut[0].SetUI == nil {
		t.Fatalf("expected rotated photo frame SetUI, got %+v", heartbeatOut)
	}
	secondURL := findNodePropValue(*heartbeatOut[0].SetUI, "photo_frame_image", "url")
	if !strings.HasSuffix(secondURL, "/frame/b.jpg") {
		t.Fatalf("second photo url = %q, want b.jpg from configured directory", secondURL)
	}
}

func TestPhotoFrameAssetBaseURLFromConfig(t *testing.T) {
	cfg := config.Config{
		MDNSName:                "HomeServer",
		PhotoFrameHTTPPort:      7002,
		PhotoFramePublicBaseURL: " https://cdn.example.test/photos/ ",
	}
	if got := photoFrameAssetBaseURL(cfg); got != "https://cdn.example.test/photos" {
		t.Fatalf("photoFrameAssetBaseURL() = %q, want explicit configured URL", got)
	}

	cfg.PhotoFramePublicBaseURL = ""
	if got := photoFrameAssetBaseURL(cfg); got != "http://HomeServer.local:7002/photo-frame" {
		t.Fatalf("photoFrameAssetBaseURL() = %q, want mDNS-derived local URL", got)
	}
}

func TestPhotoFrameAssetHandlerServesFilesAndBlocksDirectories(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.jpg"), []byte("abc"), 0o644); err != nil {
		t.Fatalf("WriteFile(a.jpg) error = %v", err)
	}

	server := httptest.NewServer(newPhotoFrameAssetHandler(dir))
	defer server.Close()

	fileRes, err := http.Get(server.URL + "/photo-frame/a.jpg")
	if err != nil {
		t.Fatalf("GET file error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := fileRes.Body.Close(); closeErr != nil {
			t.Fatalf("fileRes.Body.Close() error = %v", closeErr)
		}
	})
	if fileRes.StatusCode != http.StatusOK {
		t.Fatalf("GET file status = %d, want 200", fileRes.StatusCode)
	}
	if got := fileRes.Header.Get("Cache-Control"); got != "public, max-age=60" {
		t.Fatalf("Cache-Control = %q, want public, max-age=60", got)
	}

	dirRes, err := http.Get(server.URL + "/photo-frame/")
	if err != nil {
		t.Fatalf("GET dir error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := dirRes.Body.Close(); closeErr != nil {
			t.Fatalf("dirRes.Body.Close() error = %v", closeErr)
		}
	})
	if dirRes.StatusCode != http.StatusNotFound {
		t.Fatalf("GET dir status = %d, want 404", dirRes.StatusCode)
	}
}

func findNodePropValue(root ui.Descriptor, nodeID, key string) string {
	if root.Props["id"] == nodeID {
		return root.Props[key]
	}
	for _, child := range root.Children {
		if value := findNodePropValue(child, nodeID, key); value != "" {
			return value
		}
	}
	return ""
}
