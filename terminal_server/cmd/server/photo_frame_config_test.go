package main

import (
	"context"
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

	slides, err := loadPhotoFrameSlides(dir)
	if err != nil {
		t.Fatalf("loadPhotoFrameSlides() error = %v", err)
	}
	if len(slides) != 2 {
		t.Fatalf("len(slides) = %d, want 2", len(slides))
	}
	if !strings.HasSuffix(slides[0], "/a.jpg") {
		t.Fatalf("slides[0] = %q, want a.jpg", slides[0])
	}
	if !strings.HasSuffix(slides[1], "/z.png") {
		t.Fatalf("slides[1] = %q, want z.png", slides[1])
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
	})

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
	if !strings.HasSuffix(firstURL, "/a.jpg") {
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
	if !strings.HasSuffix(secondURL, "/b.jpg") {
		t.Fatalf("second photo url = %q, want b.jpg from configured directory", secondURL)
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
