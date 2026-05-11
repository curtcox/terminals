package scenario

import (
	"context"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestRuntimeHandleVoiceTimer(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()
	scheduler := storage.NewMemoryScheduler()

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: &TimerReminderScenario{},
		Priority: PriorityNormal,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Scheduler: scheduler,
		Broadcast: broadcaster,
	})

	_, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"set a timer for 5 minutes",
		time.Date(2026, 4, 11, 21, 30, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("HandleVoiceText() error = %v", err)
	}

	scheduled := scheduler.List()
	if len(scheduled) != 1 {
		t.Fatalf("len(scheduled) = %d, want 1", len(scheduled))
	}
	records := scheduler.DueRecords(time.Date(2026, 4, 11, 21, 36, 0, 0, time.UTC).UnixMilli())
	if len(records) != 1 {
		t.Fatalf("len(DueRecords()) = %d, want 1", len(records))
	}
	if records[0].Kind != "timer" || records[0].DeviceID != "d1" || records[0].Subject != "timer" || records[0].Payload["duration_seconds"] != "300" {
		t.Fatalf("timer record = %+v, want structured timer metadata", records[0])
	}
	events := broadcaster.Events()
	if len(events) != 1 || events[0].Message != "Timer set" {
		t.Fatalf("unexpected broadcast events: %+v", events)
	}
}

func TestRuntimeHandleTimerRendersCountdownOnPlacedDisplay(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	scheduler := storage.NewMemoryScheduler()
	uiHost := ui.NewMemoryHost()
	placement := &fakePlacement{nearestRef: DeviceRef{DeviceID: "kitchen-screen"}}

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: &TimerReminderScenario{},
		Priority: PriorityNormal,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Scheduler: scheduler,
		UI:        uiHost,
		Placement: placement,
	})

	fireUnixMS := time.Now().Add(60 * time.Second).UnixMilli()
	_, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerVoice,
		SourceID: "d1",
		Intent:   "set timer",
		Arguments: map[string]string{
			"duration_seconds": "60",
			"fire_unix_ms":     strconv.FormatInt(fireUnixMS, 10),
			"label":            "pasta",
		},
	})
	if err != nil {
		t.Fatalf("HandleTrigger() error = %v", err)
	}
	events := uiHost.Events()
	if len(events) != 1 || events[0].Kind != "set" || events[0].DeviceID != "kitchen-screen" {
		t.Fatalf("UI events = %+v, want countdown set on kitchen-screen", events)
	}
	if value := findDescriptorProp(events[0].Node, "remaining", "value"); value != "01:00" {
		t.Fatalf("remaining value = %q, want 01:00", value)
	}
	records := scheduler.DueRecords(fireUnixMS)
	if len(records) != 2 {
		t.Fatalf("len(DueRecords) = %d, want expiry and tick", len(records))
	}
	if placement.lastNearestCap != "screen" {
		t.Fatalf("placement cap = %q, want screen", placement.lastNearestCap)
	}
}

func TestRuntimeProcessDueTimerTickPatchesRemaining(t *testing.T) {
	scheduler := storage.NewMemoryScheduler()
	uiHost := ui.NewMemoryHost()
	runtime := NewRuntime(NewEngine(), &Environment{
		Scheduler: scheduler,
		UI:        uiHost,
	})

	now := time.Date(2026, 4, 12, 9, 0, 10, 0, time.UTC)
	expiry := now.Add(50 * time.Second)
	_ = scheduler.ScheduleRecord(context.Background(), storage.ScheduleRecord{
		Key:      timerTickScheduleKey("d1", expiry.UnixMilli(), 60, "pasta", now.UnixMilli()-1000),
		Kind:     "timer.tick",
		Subject:  "pasta",
		DeviceID: "d1",
		UnixMS:   now.UnixMilli() - 1000,
		Payload: map[string]string{
			"duration_seconds": "60",
			"expiry_unix_ms":   strconv.FormatInt(expiry.UnixMilli(), 10),
			"target_device_id": "kitchen-screen",
		},
	})

	processed, err := runtime.ProcessDueTimers(context.Background(), now)
	if err != nil {
		t.Fatalf("ProcessDueTimers() error = %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
	events := uiHost.Events()
	if len(events) != 1 || events[0].Kind != "patch" || events[0].ComponentID != "remaining" {
		t.Fatalf("UI events = %+v, want remaining patch", events)
	}
	if events[0].Node.Props["value"] != "00:50" {
		t.Fatalf("remaining patch = %+v, want 00:50", events[0].Node.Props)
	}
	if len(scheduler.DueRecords(now.Add(time.Second).UnixMilli())) != 1 {
		t.Fatalf("expected next tick to be scheduled")
	}
}

func TestRuntimeCancelTimerRemovesSchedulesAndClearsUI(t *testing.T) {
	scheduler := storage.NewMemoryScheduler()
	uiHost := ui.NewMemoryHost()
	broadcaster := ui.NewMemoryBroadcaster()
	runtime := NewRuntime(NewEngine(), &Environment{
		Scheduler: scheduler,
		UI:        uiHost,
		Broadcast: broadcaster,
	})

	now := time.Date(2026, 4, 12, 9, 0, 0, 0, time.UTC)
	expiry := now.Add(time.Minute)
	_ = scheduler.ScheduleRecord(context.Background(), storage.ScheduleRecord{
		Key:      timerScheduleKey("d1", expiry.UnixMilli(), 60, "pasta"),
		Kind:     "timer",
		Subject:  "pasta",
		DeviceID: "d1",
		UnixMS:   expiry.UnixMilli(),
		Payload: map[string]string{
			"duration_seconds": "60",
			"target_device_id": "kitchen-screen",
		},
	})
	_ = scheduler.ScheduleRecord(context.Background(), storage.ScheduleRecord{
		Key:      timerTickScheduleKey("d1", expiry.UnixMilli(), 60, "pasta", now.Add(time.Second).UnixMilli()),
		Kind:     "timer.tick",
		Subject:  "pasta",
		DeviceID: "d1",
		UnixMS:   now.Add(time.Second).UnixMilli(),
		Payload: map[string]string{
			"duration_seconds": "60",
			"expiry_unix_ms":   strconv.FormatInt(expiry.UnixMilli(), 10),
			"target_device_id": "kitchen-screen",
		},
	})

	scenario := &TimerReminderScenario{}
	if !scenario.Match(Trigger{Intent: "cancel timer", SourceID: "d1"}) {
		t.Fatalf("cancel timer did not match")
	}
	result, err := scenario.StartResult(context.Background(), runtime.Env)
	if err != nil {
		t.Fatalf("StartResult(cancel) error = %v", err)
	}
	if err := ExecuteOperations(context.Background(), runtime.Env, result.Ops, now); err != nil {
		t.Fatalf("ExecuteOperations(cancel) error = %v", err)
	}
	if due := scheduler.Due(math.MaxInt64); len(due) != 0 {
		t.Fatalf("scheduled records after cancel = %+v, want none", due)
	}
	events := uiHost.Events()
	if len(events) != 1 || events[0].Kind != "clear" || events[0].DeviceID != "kitchen-screen" {
		t.Fatalf("UI events = %+v, want clear on kitchen-screen", events)
	}
	if broadcasts := broadcaster.Events(); len(broadcasts) != 1 || broadcasts[0].Message != "Timer cancelled" {
		t.Fatalf("broadcasts = %+v, want Timer cancelled", broadcasts)
	}
}

func TestRuntimeProcessDueTimers(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	scheduler := storage.NewMemoryScheduler()
	broadcaster := ui.NewMemoryBroadcaster()
	tts := &testTTS{}
	runtime := NewRuntime(NewEngine(), &Environment{
		Devices:   devices,
		Scheduler: scheduler,
		Broadcast: broadcaster,
		TTS:       tts,
	})
	busEvents, cancel := runtime.Bus.Subscribe(1)
	defer cancel()

	now := time.Date(2026, 4, 12, 9, 0, 0, 0, time.UTC)
	_ = scheduler.Schedule(context.Background(), timerScheduleKey("d1", now.UnixMilli()-1000, 600, "pasta"), now.UnixMilli()-1000)
	_ = scheduler.Schedule(context.Background(), "timer:d1:200", now.UnixMilli()+60_000)

	processed, err := runtime.ProcessDueTimers(context.Background(), now)
	if err != nil {
		t.Fatalf("ProcessDueTimers() error = %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
	events := broadcaster.Events()
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if events[0].Message != "Timer done!" {
		t.Fatalf("message = %q, want Timer done!", events[0].Message)
	}
	if len(tts.calls) != 1 || tts.calls[0] != "Your pasta is ready." {
		t.Fatalf("TTS calls = %+v, want Your pasta is ready.", tts.calls)
	}
	select {
	case event := <-busEvents:
		if event.EventV2 == nil || event.EventV2.Kind != "timer.expired" || event.EventV2.Subject != "pasta" {
			t.Fatalf("bus event = %+v, want timer.expired pasta", event)
		}
		if event.EventV2.Attributes["duration_seconds"] != "600" {
			t.Fatalf("duration_seconds = %q, want 600", event.EventV2.Attributes["duration_seconds"])
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected timer.expired bus event")
	}
	if len(scheduler.Due(now.UnixMilli())) != 0 {
		t.Fatalf("expected due timers to be removed")
	}

	// Not-yet-due timer should still exist.
	laterDue := scheduler.Due(now.Add(2 * time.Minute).UnixMilli())
	if len(laterDue) != 1 || laterDue[0] != "timer:d1:200" {
		t.Fatalf("later due = %+v, want timer:d1:200", laterDue)
	}
}

func TestRuntimeProcessDueTimersUsesStructuredRecordMetadata(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	scheduler := storage.NewMemoryScheduler()
	broadcaster := ui.NewMemoryBroadcaster()
	tts := &testTTS{}
	runtime := NewRuntime(NewEngine(), &Environment{
		Devices:   devices,
		Scheduler: scheduler,
		Broadcast: broadcaster,
		TTS:       tts,
	})
	busEvents, cancel := runtime.Bus.Subscribe(1)
	defer cancel()

	now := time.Date(2026, 4, 12, 9, 0, 0, 0, time.UTC)
	_ = scheduler.ScheduleRecord(context.Background(), storage.ScheduleRecord{
		Key:      "timer-record-1",
		Kind:     "timer",
		Subject:  "pasta",
		DeviceID: "d1",
		UnixMS:   now.UnixMilli() - 1000,
		Payload:  map[string]string{"duration_seconds": "600"},
	})

	processed, err := runtime.ProcessDueTimers(context.Background(), now)
	if err != nil {
		t.Fatalf("ProcessDueTimers() error = %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
	if len(tts.calls) != 1 || tts.calls[0] != "Your pasta is ready." {
		t.Fatalf("TTS calls = %+v, want Your pasta is ready.", tts.calls)
	}
	select {
	case event := <-busEvents:
		if event.EventV2 == nil || event.EventV2.Subject != "pasta" {
			t.Fatalf("bus event = %+v, want pasta subject", event)
		}
		if event.EventV2.Attributes["duration_seconds"] != "600" {
			t.Fatalf("duration_seconds = %q, want 600", event.EventV2.Attributes["duration_seconds"])
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected timer.expired bus event")
	}
	events := broadcaster.Events()
	if len(events) != 1 || len(events[0].DeviceIDs) != 1 || events[0].DeviceIDs[0] != "d1" {
		t.Fatalf("broadcast events = %+v, want d1 timer done", events)
	}
}
