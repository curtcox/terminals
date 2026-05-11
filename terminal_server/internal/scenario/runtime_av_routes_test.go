package scenario

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestRuntimeManualAudioSchedulePAAndMultiWindow(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
	router := iorouter.NewRouter()
	store := storage.NewMemoryStore()
	scheduler := storage.NewMemoryScheduler()
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &AudioMonitorScenario{}, Priority: PriorityNormal})
	engine.Register(Registration{Scenario: &ScheduleMonitorScenario{}, Priority: PriorityNormal})
	engine.Register(Registration{Scenario: &PASystemScenario{}, Priority: PriorityHigh})
	engine.Register(Registration{Scenario: &MultiWindowScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		IO:        router,
		Storage:   store,
		Scheduler: scheduler,
		Broadcast: broadcaster,
	})

	_, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "audio_monitor",
		Arguments: map[string]string{
			"target": "dishwasher",
		},
	})
	if err != nil {
		t.Fatalf("HandleTrigger(audio_monitor) error = %v", err)
	}
	soundTarget, err := store.Get(context.Background(), "audio_monitor:d1")
	if err != nil {
		t.Fatalf("store.Get(audio monitor) error = %v", err)
	}
	if soundTarget != "dishwasher" {
		t.Fatalf("audio monitor target = %q, want dishwasher", soundTarget)
	}

	checkTime := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC).UnixMilli()
	_, err = runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "schedule_monitor",
		Arguments: map[string]string{
			"check_unix_ms": strconv.FormatInt(checkTime, 10),
		},
	})
	if err != nil {
		t.Fatalf("HandleTrigger(schedule_monitor) error = %v", err)
	}
	if len(scheduler.Due(checkTime)) != 1 || scheduler.Due(checkTime)[0] != "schedule_monitor:d1" {
		t.Fatalf("schedule monitor due keys = %+v, want [schedule_monitor:d1]", scheduler.Due(checkTime))
	}

	_, err = runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "pa_system",
	})
	if err != nil {
		t.Fatalf("HandleTrigger(pa_system) error = %v", err)
	}
	_, err = runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "multi_window",
	})
	if err != nil {
		t.Fatalf("HandleTrigger(multi_window) error = %v", err)
	}

	if router.RouteCount() != 6 {
		t.Fatalf("route count = %d, want 6", router.RouteCount())
	}
	events := broadcaster.Events()
	if len(events) != 5 {
		t.Fatalf("len(events) = %d, want 5", len(events))
	}
	if events[0].Message != "Audio monitor armed: dishwasher" {
		t.Fatalf("event0 message = %q", events[0].Message)
	}
	if events[1].Message != "Schedule monitor active" {
		t.Fatalf("event1 message = %q", events[1].Message)
	}
	if events[2].Message != "PA system active" {
		t.Fatalf("event2 message = %q", events[2].Message)
	}
	if events[3].Message != "PA from d1" {
		t.Fatalf("event3 message = %q, want PA from d1", events[3].Message)
	}
	if len(events[3].DeviceIDs) != 2 || events[3].DeviceIDs[0] != "d2" || events[3].DeviceIDs[1] != "d3" {
		t.Fatalf("event3 device IDs = %+v, want [d2 d3]", events[3].DeviceIDs)
	}
	if events[4].Message != "Multi-window active" {
		t.Fatalf("event4 message = %q", events[4].Message)
	}
}

func TestRuntimeManualAliasIntentsForPAAnnouncementAndMultiWindow(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
	router := iorouter.NewRouter()
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &PASystemScenario{}, Priority: PriorityHigh})
	engine.Register(Registration{Scenario: &AnnouncementScenario{}, Priority: PriorityHigh})
	engine.Register(Registration{Scenario: &MultiWindowScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		IO:        router,
		Broadcast: broadcaster,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "pa mode",
	}); err != nil {
		t.Fatalf("HandleTrigger(pa mode) error = %v", err)
	}

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "show all cameras",
	}); err != nil {
		t.Fatalf("HandleTrigger(show all cameras) error = %v", err)
	}

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "announce",
	}); err != nil {
		t.Fatalf("HandleTrigger(announce) error = %v", err)
	}

	if router.RouteCount() != 8 {
		t.Fatalf("route count = %d, want 8", router.RouteCount())
	}
	events := broadcaster.Events()
	if len(events) != 5 {
		t.Fatalf("len(events) = %d, want 5", len(events))
	}
	if events[0].Message != "PA system active" {
		t.Fatalf("event0 message = %q, want PA system active", events[0].Message)
	}
	if events[1].Message != "PA from d1" {
		t.Fatalf("event1 message = %q, want PA from d1", events[1].Message)
	}
	if len(events[1].DeviceIDs) != 2 || events[1].DeviceIDs[0] != "d2" || events[1].DeviceIDs[1] != "d3" {
		t.Fatalf("event1 device IDs = %+v, want [d2 d3]", events[1].DeviceIDs)
	}
	if events[2].Message != "Multi-window active" {
		t.Fatalf("event2 message = %q, want Multi-window active", events[2].Message)
	}
	if events[3].Message != "Announcement active" {
		t.Fatalf("event3 message = %q, want Announcement active", events[3].Message)
	}
	if events[4].Message != "Announcement from d1" {
		t.Fatalf("event4 message = %q, want Announcement from d1", events[4].Message)
	}
	if len(events[4].DeviceIDs) != 2 || events[4].DeviceIDs[0] != "d2" || events[4].DeviceIDs[1] != "d3" {
		t.Fatalf("event4 device IDs = %+v, want [d2 d3]", events[4].DeviceIDs)
	}
}

func TestRuntimeMultiWindowFocusRoutesSingleAudioSource(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
	router := iorouter.NewRouter()
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &MultiWindowScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		IO:        router,
		Broadcast: broadcaster,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "multi_window",
		Arguments: map[string]string{
			"audio_focus_device_id": "d2",
		},
	}); err != nil {
		t.Fatalf("HandleTrigger(multi_window focus) error = %v", err)
	}

	if router.RouteCount() != 3 {
		t.Fatalf("route count = %d, want 3", router.RouteCount())
	}

	routes := router.RoutesForDevice("d1")
	hasFocusedAudio := false
	hasAudioMix := false
	for _, route := range routes {
		if route.SourceID == "d2" && route.TargetID == "d1" && route.StreamKind == "audio" {
			hasFocusedAudio = true
		}
		if route.StreamKind == "audio_mix" {
			hasAudioMix = true
		}
	}
	if !hasFocusedAudio {
		t.Fatalf("expected focused audio route d2->d1 audio")
	}
	if hasAudioMix {
		t.Fatalf("did not expect audio_mix routes when focus is selected")
	}
}

func TestRuntimeMultiWindowTargetsExplicitDeviceIDs(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
	router := iorouter.NewRouter()
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &MultiWindowScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		IO:        router,
		Broadcast: broadcaster,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "multi_window",
		Arguments: map[string]string{
			"device_ids": "d1,d3",
		},
	}); err != nil {
		t.Fatalf("HandleTrigger(multi_window targeted) error = %v", err)
	}

	if got := router.RouteCount(); got != 2 {
		t.Fatalf("route count = %d, want 2", got)
	}
	routes := router.RoutesForDevice("d1")
	for _, route := range routes {
		if route.SourceID == "d2" || route.TargetID == "d2" {
			t.Fatalf("unexpected route involving d2: %+v", route)
		}
	}
}
