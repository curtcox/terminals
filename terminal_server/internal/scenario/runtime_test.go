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

type testAIBackend struct {
	lastInput string
	response  string
}

func (t *testAIBackend) Query(_ context.Context, input string) (string, error) {
	t.lastInput = input
	return t.response, nil
}

type testTelephonyBridge struct {
	lastTarget string
}

func (t *testTelephonyBridge) Call(_ context.Context, target string) error {
	t.lastTarget = target
	return nil
}

func (t *testTelephonyBridge) Hangup(context.Context, string) error {
	return nil
}

func TestRuntimeHandleTrigger(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: AlertScenario{},
		Priority: PriorityCritical,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
	})

	name, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:   TriggerVoice,
		Intent: "red alert",
	})
	if err != nil {
		t.Fatalf("HandleTrigger() error = %v", err)
	}
	if name != "red_alert" {
		t.Fatalf("scenario name = %q, want red_alert", name)
	}

	events := broadcaster.Events()
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].Message != "RED ALERT" {
		t.Fatalf("event message = %q, want RED ALERT", events[0].Message)
	}
}

func TestRuntimeStopVoiceTextStandDownStopsRedAlert(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: AlertScenario{},
		Priority: PriorityCritical,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
	})

	if _, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"red alert",
		time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("HandleVoiceText(red alert) error = %v", err)
	}

	stopped, err := runtime.StopVoiceText(
		context.Background(),
		"d1",
		"stand down",
		time.Date(2026, 4, 11, 21, 1, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("StopVoiceText(stand down) error = %v", err)
	}
	if stopped != "red_alert" {
		t.Fatalf("stopped scenario = %q, want red_alert", stopped)
	}
	if _, ok := engine.Active("d1"); ok {
		t.Fatalf("expected no active scenario after stand down stop")
	}
}

func TestRuntimeStopVoiceTextEndPAStopsPASystem(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	broadcaster := ui.NewMemoryBroadcaster()
	router := iorouter.NewRouter()

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: &PASystemScenario{},
		Priority: PriorityHigh,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		IO:        router,
		Broadcast: broadcaster,
	})

	if _, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"pa mode",
		time.Date(2026, 4, 11, 21, 0, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("HandleVoiceText(pa mode) error = %v", err)
	}
	if router.RouteCount() != 1 {
		t.Fatalf("route count after pa start = %d, want 1", router.RouteCount())
	}

	stopped, err := runtime.StopVoiceText(
		context.Background(),
		"d1",
		"end pa",
		time.Date(2026, 4, 11, 21, 1, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("StopVoiceText(end pa) error = %v", err)
	}
	if stopped != "pa_system" {
		t.Fatalf("stopped scenario = %q, want pa_system", stopped)
	}
	if _, ok := engine.Active("d1"); ok {
		t.Fatalf("expected no active scenario after end pa stop")
	}
}

func TestRuntimeNoMatch(t *testing.T) {
	runtime := NewRuntime(NewEngine(), &Environment{})
	if _, err := runtime.HandleTrigger(context.Background(), Trigger{Intent: "unknown"}); err != ErrNoMatchingScenario {
		t.Fatalf("HandleTrigger() error = %v, want %v", err, ErrNoMatchingScenario)
	}
}

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
	events := broadcaster.Events()
	if len(events) != 1 || events[0].Message != "Timer set" {
		t.Fatalf("unexpected broadcast events: %+v", events)
	}
}

func TestRuntimeStopTrigger(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	engine := NewEngine()
	engine.Register(Registration{
		Scenario: PhotoFrameScenario{},
		Priority: PriorityLow,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices: devices,
	})

	name, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:   TriggerManual,
		Intent: "photo frame",
	})
	if err != nil {
		t.Fatalf("HandleTrigger() error = %v", err)
	}
	if name != "photo_frame" {
		t.Fatalf("name = %q, want photo_frame", name)
	}

	stopped, err := runtime.StopTrigger(context.Background(), Trigger{
		Kind:   TriggerManual,
		Intent: "photo frame",
	})
	if err != nil {
		t.Fatalf("StopTrigger() error = %v", err)
	}
	if stopped != "photo_frame" {
		t.Fatalf("stopped = %q, want photo_frame", stopped)
	}
	if _, ok := engine.Active("d1"); ok {
		t.Fatalf("expected no active scenario after stop")
	}
}

func TestRuntimeHandleVoiceTerminalTargetsSourceDevice(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: &TerminalScenario{},
		Priority: PriorityNormal,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
	})

	name, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"open terminal",
		time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("HandleVoiceText() error = %v", err)
	}
	if name != "terminal" {
		t.Fatalf("scenario name = %q, want terminal", name)
	}

	events := broadcaster.Events()
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if events[0].Message != "Terminal active" {
		t.Fatalf("event message = %q, want Terminal active", events[0].Message)
	}
	if len(events[0].DeviceIDs) != 1 || events[0].DeviceIDs[0] != "d1" {
		t.Fatalf("event device IDs = %+v, want [d1]", events[0].DeviceIDs)
	}
}

func TestRuntimeStatusData(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	routes := iorouter.NewRouter()
	engine := NewEngine()
	engine.Register(Registration{
		Scenario: PhotoFrameScenario{},
		Priority: PriorityLow,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		IO:        routes,
		Scheduler: storage.NewMemoryScheduler(),
	})

	_, _ = runtime.HandleTrigger(context.Background(), Trigger{
		Kind:   TriggerManual,
		Intent: "photo frame",
	})
	_ = routes.Connect("d1", "d2", "audio")

	status := runtime.StatusData()
	if status["active_scenarios"] != "1" {
		t.Fatalf("active_scenarios = %q, want 1", status["active_scenarios"])
	}
	if status["active_routes"] != "1" {
		t.Fatalf("active_routes = %q, want 1", status["active_routes"])
	}
	if status["registered_scenarios"] != "1" {
		t.Fatalf("registered_scenarios = %q, want 1", status["registered_scenarios"])
	}
	if status["pending_timers"] != "0" {
		t.Fatalf("pending_timers = %q, want 0", status["pending_timers"])
	}
}

func TestRuntimeProcessDueTimers(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	scheduler := storage.NewMemoryScheduler()
	broadcaster := ui.NewMemoryBroadcaster()
	runtime := NewRuntime(NewEngine(), &Environment{
		Devices:   devices,
		Scheduler: scheduler,
		Broadcast: broadcaster,
	})

	now := time.Date(2026, 4, 12, 9, 0, 0, 0, time.UTC)
	_ = scheduler.Schedule(context.Background(), "timer:d1:100", now.UnixMilli()-1000)
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
	if events[0].Message != "Timer complete" {
		t.Fatalf("message = %q, want Timer complete", events[0].Message)
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

func TestRuntimeHandleVoiceIntercomCreatesAudioRoutes(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
	router := iorouter.NewRouter()
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: &IntercomScenario{},
		Priority: PriorityHigh,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		IO:        router,
		Broadcast: broadcaster,
	})

	name, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"intercom",
		time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("HandleVoiceText() error = %v", err)
	}
	if name != "intercom" {
		t.Fatalf("scenario name = %q, want intercom", name)
	}
	if router.RouteCount() != 4 {
		t.Fatalf("route count = %d, want 4", router.RouteCount())
	}
	events := broadcaster.Events()
	if len(events) != 1 || events[0].Message != "Intercom active" {
		t.Fatalf("unexpected broadcast events: %+v", events)
	}
	if len(events[0].DeviceIDs) != 1 || events[0].DeviceIDs[0] != "d1" {
		t.Fatalf("broadcast target = %+v, want [d1]", events[0].DeviceIDs)
	}
}

func TestRuntimeHandleVoiceAssistantAndPhoneCall(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()
	aiBackend := &testAIBackend{response: "assistant response"}
	telephony := &testTelephonyBridge{}

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: &VoiceAssistantScenario{},
		Priority: PriorityNormal,
	})
	engine.Register(Registration{
		Scenario: &PhoneCallScenario{},
		Priority: PriorityHigh,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
		AI:        aiBackend,
		Telephony: telephony,
	})

	assistantName, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"assistant what is on my calendar",
		time.Date(2026, 4, 12, 0, 5, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("HandleVoiceText(assistant) error = %v", err)
	}
	if assistantName != "voice_assistant" {
		t.Fatalf("assistant scenario name = %q, want voice_assistant", assistantName)
	}
	if aiBackend.lastInput != "what is on my calendar" {
		t.Fatalf("assistant query = %q, want what is on my calendar", aiBackend.lastInput)
	}

	callName, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"call 5551212",
		time.Date(2026, 4, 12, 0, 6, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("HandleVoiceText(call) error = %v", err)
	}
	if callName != "phone_call" {
		t.Fatalf("call scenario name = %q, want phone_call", callName)
	}
	if telephony.lastTarget != "5551212" {
		t.Fatalf("last telephony target = %q, want 5551212", telephony.lastTarget)
	}

	events := broadcaster.Events()
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].Message != "assistant response" {
		t.Fatalf("assistant event message = %q, want assistant response", events[0].Message)
	}
	if events[1].Message != "Calling 5551212" {
		t.Fatalf("call event message = %q, want Calling 5551212", events[1].Message)
	}
}

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

func TestRuntimeManualAliasIntentsForPAAndMultiWindow(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
	router := iorouter.NewRouter()
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &PASystemScenario{}, Priority: PriorityHigh})
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

	if router.RouteCount() != 6 {
		t.Fatalf("route count = %d, want 6", router.RouteCount())
	}
	events := broadcaster.Events()
	if len(events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(events))
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
