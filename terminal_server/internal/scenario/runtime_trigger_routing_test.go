package scenario

import (
	"context"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

func TestRuntimePublishesNormalizedTriggerToBus(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1"})
	engine := NewEngine()
	engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	ch, cancel := runtime.Bus.Subscribe(2)
	defer cancel()

	_, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:      TriggerManual,
		SourceID:  "d1",
		Intent:    "terminal",
		Arguments: map[string]string{"device_id": "d1"},
	})
	if err != nil {
		t.Fatalf("HandleTrigger() error = %v", err)
	}

	select {
	case got := <-ch:
		if got.IntentV2 == nil {
			t.Fatalf("expected IntentV2 on published trigger")
		}
		if got.IntentV2.Action != "terminal" {
			t.Fatalf("published intent action = %q, want terminal", got.IntentV2.Action)
		}
		if got.IntentV2.Source != SourceManual {
			t.Fatalf("published intent source = %q, want manual", got.IntentV2.Source)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timed out waiting for bus trigger")
	}
}

func TestRuntimeHandleIntentRoutesTypedIntent(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1"})
	engine := NewEngine()
	engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	name, err := runtime.HandleIntent(context.Background(), "d1", IntentRecord{
		Action: "terminal",
		Slots:  map[string]string{"device_id": "d1"},
		Source: SourceUI,
	})
	if err != nil {
		t.Fatalf("HandleIntent() error = %v", err)
	}
	if name != "terminal" {
		t.Fatalf("HandleIntent() = %q, want terminal", name)
	}
}

func TestRuntimeHandleIntentMapsSourceToTriggerKind(t *testing.T) {
	cases := []struct {
		name         string
		source       TriggerSource
		expectedKind TriggerKind
	}{
		{name: "voice", source: SourceVoice, expectedKind: TriggerVoice},
		{name: "schedule", source: SourceSchedule, expectedKind: TriggerSchedule},
		{name: "event", source: SourceEvent, expectedKind: TriggerEvent},
		{name: "cascade", source: SourceCascade, expectedKind: TriggerCascade},
		{name: "ui", source: SourceUI, expectedKind: TriggerManual},
		{name: "manual", source: SourceManual, expectedKind: TriggerManual},
		{name: "webhook", source: SourceWebhook, expectedKind: TriggerManual},
		{name: "agent", source: SourceAgent, expectedKind: TriggerManual},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			devices := device.NewManager()
			_, _ = devices.Register(device.Manifest{DeviceID: "d1"})
			engine := NewEngine()
			engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
			runtime := NewRuntime(engine, &Environment{
				Devices:   devices,
				Broadcast: ui.NewMemoryBroadcaster(),
			})
			ch, cancel := runtime.Bus.Subscribe(1)
			defer cancel()

			_, err := runtime.HandleIntent(context.Background(), "d1", IntentRecord{
				Action: "terminal",
				Slots:  map[string]string{"device_id": "d1"},
				Source: tc.source,
			})
			if err != nil {
				t.Fatalf("HandleIntent() error = %v", err)
			}

			select {
			case got := <-ch:
				if got.Kind != tc.expectedKind {
					t.Fatalf("trigger kind = %q, want %q", got.Kind, tc.expectedKind)
				}
				if got.IntentV2 == nil {
					t.Fatalf("expected IntentV2 on published trigger")
				}
				if got.IntentV2.Source != tc.source {
					t.Fatalf("intent source = %q, want %q", got.IntentV2.Source, tc.source)
				}
			case <-time.After(200 * time.Millisecond):
				t.Fatalf("timed out waiting for bus trigger")
			}
		})
	}
}

func TestRuntimeHandleEventRoutesTypedEvent(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1"})
	engine := NewEngine()
	engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: ui.NewMemoryBroadcaster(),
	})
	ch, cancel := runtime.Bus.Subscribe(1)
	defer cancel()

	name, err := runtime.HandleEvent(context.Background(), "d1", EventRecord{
		Kind:   "terminal",
		Source: SourceWebhook,
	})
	if err != nil {
		t.Fatalf("HandleEvent() error = %v", err)
	}
	if name != "terminal" {
		t.Fatalf("HandleEvent() = %q, want terminal", name)
	}

	select {
	case got := <-ch:
		if got.Kind != TriggerEvent {
			t.Fatalf("trigger kind = %q, want %q", got.Kind, TriggerEvent)
		}
		if got.EventV2 == nil {
			t.Fatalf("expected EventV2 on published trigger")
		}
		if got.EventV2.Source != SourceWebhook {
			t.Fatalf("event source = %q, want %q", got.EventV2.Source, SourceWebhook)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timed out waiting for bus trigger")
	}
}

func TestRuntimeHandleEventDispatchesToActiveEventConsumer(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1"})
	engine := NewEngine()
	app := &eventForwardingScenario{matchOn: "app.watch"}
	engine.Register(Registration{Scenario: app, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "app.watch",
	}); err != nil {
		t.Fatalf("HandleTrigger(app.watch) error = %v", err)
	}

	name, err := runtime.HandleEvent(context.Background(), "d1", EventRecord{
		Kind:       "sound.classified",
		Subject:    "d1",
		Attributes: map[string]string{"label": "dishwasher_done"},
		Source:     SourceEvent,
	})
	if err != nil {
		t.Fatalf("HandleEvent() error = %v", err)
	}
	if name != "app.watch" {
		t.Fatalf("HandleEvent() = %q, want app.watch", name)
	}
	if count := app.eventCount(); count != 1 {
		t.Fatalf("event count = %d, want 1", count)
	}
	last := app.lastEvent()
	if last.Kind != "sound.classified" {
		t.Fatalf("last event kind = %q, want sound.classified", last.Kind)
	}
	if last.Attributes["label"] != "dishwasher_done" {
		t.Fatalf("last event label = %q, want dishwasher_done", last.Attributes["label"])
	}
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

func TestRuntimeNoMatch(t *testing.T) {
	runtime := NewRuntime(NewEngine(), &Environment{})
	if _, err := runtime.HandleTrigger(context.Background(), Trigger{Intent: "unknown"}); err != ErrNoMatchingScenario {
		t.Fatalf("HandleTrigger() error = %v, want %v", err, ErrNoMatchingScenario)
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

func TestRuntimeEventTailAndWebhookAutomationIntents(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
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

	if _, err := runtime.HandleWebhookIntent(context.Background(), "d1", "terminal", nil); err != nil {
		t.Fatalf("HandleWebhookIntent() error = %v", err)
	}
	if _, err := runtime.HandleAutomationIntent(context.Background(), "d1", "terminal", nil); err != nil {
		t.Fatalf("HandleAutomationIntent() error = %v", err)
	}

	tail := runtime.EventTail(10)
	if len(tail) < 2 {
		t.Fatalf("len(EventTail) = %d, want >=2", len(tail))
	}
	if tail[len(tail)-2].IntentV2 == nil || tail[len(tail)-2].IntentV2.Source != SourceWebhook {
		t.Fatalf("expected webhook source in tail, got %+v", tail[len(tail)-2].IntentV2)
	}
	if tail[len(tail)-2].Kind != TriggerManual {
		t.Fatalf("expected webhook trigger kind manual, got %q", tail[len(tail)-2].Kind)
	}
	if tail[len(tail)-1].IntentV2 == nil || tail[len(tail)-1].IntentV2.Source != SourceAgent {
		t.Fatalf("expected agent source in tail, got %+v", tail[len(tail)-1].IntentV2)
	}
	if tail[len(tail)-1].Kind != TriggerManual {
		t.Fatalf("expected agent trigger kind manual, got %q", tail[len(tail)-1].Kind)
	}
}
