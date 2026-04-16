// Package main tests server process lifecycle helpers.
package main

import (
	"context"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/diagnostics/bugreport"
	"github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestRunDueTimerLoopProcessesTimers(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	scheduler := storage.NewMemoryScheduler()
	broadcaster := ui.NewMemoryBroadcaster()
	runtime := scenario.NewRuntime(scenario.NewEngine(), &scenario.Environment{
		Devices:   devices,
		IO:        io.NewRouter(),
		Scheduler: scheduler,
		Broadcast: broadcaster,
	})

	now := time.Now().UTC()
	_ = scheduler.Schedule(context.Background(), "timer:d1:test", now.Add(-1*time.Second).UnixMilli())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runDueTimerLoop(ctx, runtime, 10*time.Millisecond)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if len(broadcaster.Events()) > 0 {
			cancel()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected due timer loop to emit at least one timer notification")
}

func TestRunLivenessLoopMarksStaleDevices(t *testing.T) {
	devices := device.NewManager()
	control := transport.NewControlService("srv-1", devices)

	base := time.Now().UTC()
	control.SetNowForTest(func() time.Time { return base.Add(-1 * time.Minute) })
	_, _ = control.Register(context.Background(), transport.RegisterRequest{
		DeviceID:   "stale-1",
		DeviceName: "Hall Tablet",
	})
	_ = control.Heartbeat(context.Background(), "stale-1")
	control.SetNowForTest(func() time.Time { return base })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runLivenessLoop(ctx, control, bugreport.NewService(t.TempDir(), devices, nil), 10*time.Second, 10*time.Millisecond)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		dev, ok := devices.Get("stale-1")
		if ok && dev.State == device.StateDisconnected {
			cancel()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected liveness loop to mark stale device disconnected")
}
