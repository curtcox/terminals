package scenario

import (
	"bytes"
	"context"
	"math"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/audio"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

type testAIBackend struct {
	lastInput string
	response  string
}

type testLLM struct {
	text    string
	queries [][]LLMMessage
}

type testTTS struct {
	calls []string
}

func (t *testTTS) Synthesize(_ context.Context, text string, _ TTSOptions) (AudioPlayback, error) {
	t.calls = append(t.calls, text)
	return bytes.NewReader(nil), nil
}

type eventForwardingScenario struct {
	mu      sync.Mutex
	events  []EventRecord
	matchOn string
}

func (s *eventForwardingScenario) Name() string {
	return "app.watch"
}

func (s *eventForwardingScenario) Match(trigger Trigger) bool {
	return trigger.Intent == s.matchOn
}

func (s *eventForwardingScenario) Start(ctx context.Context, env *Environment) error {
	_ = ctx
	_ = env
	return nil
}

func (s *eventForwardingScenario) Stop() error {
	return nil
}

func (s *eventForwardingScenario) HandleEvent(ctx context.Context, env *Environment, event EventRecord) error {
	_ = ctx
	_ = env
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *eventForwardingScenario) eventCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.events)
}

func (s *eventForwardingScenario) lastEvent() EventRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.events) == 0 {
		return EventRecord{}
	}
	return s.events[len(s.events)-1]
}

func (l *testLLM) Query(_ context.Context, messages []LLMMessage, _ LLMOptions) (*LLMResponse, error) {
	copyMsgs := make([]LLMMessage, len(messages))
	copy(copyMsgs, messages)
	l.queries = append(l.queries, copyMsgs)
	return &LLMResponse{Text: l.text, FinishReason: "stop"}, nil
}

func TestRuntimeAudioMonitorNotifiesWhenTargetDetected(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()
	classifier := &testSoundClassifier{events: []SoundEvent{{Label: "dishwasher_stopped", Confidence: 0.92, AtMS: 101}}}

	engine := NewEngine()
	engine.Register(Registration{Scenario: &AudioMonitorScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
		Sound:     classifier,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "audio_monitor",
		Arguments: map[string]string{
			"target": "dishwasher",
		},
	}); err != nil {
		t.Fatalf("HandleTrigger(audio_monitor) error = %v", err)
	}

	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		events := broadcaster.Events()
		if len(events) >= 2 {
			if events[0].Message != "Audio monitor armed: dishwasher" {
				t.Fatalf("event0 message = %q, want Audio monitor armed: dishwasher", events[0].Message)
			}
			if events[1].Message != "Audio monitor detected: dishwasher_stopped" {
				t.Fatalf("event1 message = %q, want detection message", events[1].Message)
			}
			if len(events[1].DeviceIDs) != 1 || events[1].DeviceIDs[0] != "d1" {
				t.Fatalf("event1 device IDs = %+v, want [d1]", events[1].DeviceIDs)
			}
			if got := classifier.captured(); len(got) != 0 {
				t.Fatalf("expected silence source to immediately EOF, got bytes = %d", len(got))
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected audio monitor detection notification; events = %+v", broadcaster.Events())
}

func TestRuntimeAudioMonitorNotifiesWhenDryerBeeps(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Laundry"})
	broadcaster := ui.NewMemoryBroadcaster()
	classifier := &testSoundClassifier{events: []SoundEvent{{Label: "dryer_beep", Confidence: 0.88, AtMS: 202}}}

	engine := NewEngine()
	engine.Register(Registration{Scenario: &AudioMonitorScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
		Sound:     classifier,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "audio_monitor",
		Arguments: map[string]string{
			"target": "dryer",
		},
	}); err != nil {
		t.Fatalf("HandleTrigger(audio_monitor) error = %v", err)
	}

	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		events := broadcaster.Events()
		if len(events) >= 2 {
			if events[0].Message != "Audio monitor armed: dryer" {
				t.Fatalf("event0 message = %q, want Audio monitor armed: dryer", events[0].Message)
			}
			if events[1].Message != "Audio monitor detected: dryer_beep" {
				t.Fatalf("event1 message = %q, want Audio monitor detected: dryer_beep", events[1].Message)
			}
			if len(events[1].DeviceIDs) != 1 || events[1].DeviceIDs[0] != "d1" {
				t.Fatalf("event1 device IDs = %+v, want [d1]", events[1].DeviceIDs)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected audio monitor dryer detection notification; events = %+v", broadcaster.Events())
}

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

func TestRuntimeHandleVoiceTextUsesLLMIntentResolutionForAmbiguousInput(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1"})
	engine := NewEngine()
	engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
	llm := &testLLM{
		text: `{"action":"terminal","object":"","slots":{"device_id":"d1"},"scope":{}}`,
	}
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: ui.NewMemoryBroadcaster(),
		LLM:       llm,
	})

	name, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"could you open a terminal for me",
		time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("HandleVoiceText() error = %v", err)
	}
	if name != "terminal" {
		t.Fatalf("resolved scenario = %q, want terminal", name)
	}
	if len(llm.queries) != 1 {
		t.Fatalf("LLM query count = %d, want 1", len(llm.queries))
	}
}

func TestRuntimeHandleVoiceTextSkipsLLMForKnownIntent(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1"})
	engine := NewEngine()
	engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
	llm := &testLLM{
		text: `{"action":"red_alert","slots":{},"scope":{}}`,
	}
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: ui.NewMemoryBroadcaster(),
		LLM:       llm,
	})

	name, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"open terminal",
		time.Date(2026, 4, 15, 10, 1, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("HandleVoiceText() error = %v", err)
	}
	if name != "terminal" {
		t.Fatalf("scenario = %q, want terminal", name)
	}
	if len(llm.queries) != 0 {
		t.Fatalf("expected no LLM query for known intent, got %d", len(llm.queries))
	}
}

func TestRuntimeAudioMonitorConsumesLiveDeviceAudio(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()
	deviceAudio := newFakeDeviceAudio()
	classifier := &testSoundClassifier{}

	engine := NewEngine()
	engine.Register(Registration{Scenario: &AudioMonitorScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:     devices,
		Broadcast:   broadcaster,
		Sound:       classifier,
		DeviceAudio: deviceAudio,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "audio_monitor",
		Arguments: map[string]string{
			"target": "dishwasher",
		},
	}); err != nil {
		t.Fatalf("HandleTrigger(audio_monitor) error = %v", err)
	}

	// Wait for the scenario to register a live subscription.
	if !waitFor(func() bool { return deviceAudio.subscriberCount("d1") == 1 }, 200*time.Millisecond) {
		t.Fatalf("expected DeviceAudio subscriber count for d1 = 1, got %d", deviceAudio.subscriberCount("d1"))
	}

	// Simulate live mic audio arriving from the transport layer. The
	// scenario should forward it into the sound classifier.
	deviceAudio.publish("d1", []byte("dishwasher-audio-chunk"))

	if !waitFor(func() bool { return len(classifier.captured()) >= len("dishwasher-audio-chunk") }, 300*time.Millisecond) {
		t.Fatalf("classifier never received live audio; captured = %q", string(classifier.captured()))
	}
	if got := string(classifier.captured()); got != "dishwasher-audio-chunk" {
		t.Fatalf("classifier captured = %q, want dishwasher-audio-chunk", got)
	}

	// Emit a matching event and confirm the scenario notifies the source device.
	classifier.emit(SoundEvent{Label: "dishwasher_stopped", Confidence: 0.9, AtMS: 101})

	if !waitFor(func() bool { return len(broadcaster.Events()) >= 2 }, 300*time.Millisecond) {
		t.Fatalf("expected detection broadcast, got events = %+v", broadcaster.Events())
	}
	events := broadcaster.Events()
	if events[1].Message != "Audio monitor detected: dishwasher_stopped" {
		t.Fatalf("detection message = %q, want Audio monitor detected: dishwasher_stopped", events[1].Message)
	}

	// After detection, the scenario closes the subscription.
	classifier.close()
	if !waitFor(func() bool { return deviceAudio.subscriberCount("d1") == 0 }, 200*time.Millisecond) {
		t.Fatalf("expected subscription to be closed after detection, count = %d", deviceAudio.subscriberCount("d1"))
	}
}

func TestRuntimeAudioMonitorIgnoresNonMatchingEvents(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()
	classifier := &testSoundClassifier{events: []SoundEvent{{Label: "microwave_beep", Confidence: 0.9, AtMS: 101}}}

	engine := NewEngine()
	engine.Register(Registration{Scenario: &AudioMonitorScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
		Sound:     classifier,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "audio_monitor",
		Arguments: map[string]string{
			"target": "dishwasher",
		},
	}); err != nil {
		t.Fatalf("HandleTrigger(audio_monitor) error = %v", err)
	}

	time.Sleep(40 * time.Millisecond)
	events := broadcaster.Events()
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1 (armed only)", len(events))
	}
	if events[0].Message != "Audio monitor armed: dishwasher" {
		t.Fatalf("event0 message = %q, want Audio monitor armed: dishwasher", events[0].Message)
	}
}

// TestRuntimeAudioMonitorStopReleasesSubscription verifies that an explicit
// StopTrigger for the audio_monitor scenario cancels the classifier
// goroutine and releases the live DeviceAudio subscription immediately,
// without waiting for a matching sound event.
func TestRuntimeAudioMonitorStopReleasesSubscription(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()
	deviceAudio := newFakeDeviceAudio()
	classifier := &testSoundClassifier{}

	engine := NewEngine()
	engine.Register(Registration{Scenario: &AudioMonitorScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:     devices,
		Broadcast:   broadcaster,
		Sound:       classifier,
		DeviceAudio: deviceAudio,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "audio_monitor",
		Arguments: map[string]string{
			"target": "dishwasher",
		},
	}); err != nil {
		t.Fatalf("HandleTrigger(audio_monitor) error = %v", err)
	}

	if !waitFor(func() bool { return deviceAudio.subscriberCount("d1") == 1 }, 200*time.Millisecond) {
		t.Fatalf("expected 1 DeviceAudio subscriber for d1, got %d", deviceAudio.subscriberCount("d1"))
	}

	// Explicit stop should cancel the classifier goroutine without needing a
	// matching sound event.
	stopped, err := runtime.StopTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "audio_monitor",
	})
	if err != nil {
		t.Fatalf("StopTrigger(audio_monitor) error = %v", err)
	}
	if stopped != "audio_monitor" {
		t.Fatalf("stopped scenario = %q, want audio_monitor", stopped)
	}

	if !waitFor(func() bool { return deviceAudio.subscriberCount("d1") == 0 }, 200*time.Millisecond) {
		t.Fatalf("expected subscription to be released after stop, count = %d", deviceAudio.subscriberCount("d1"))
	}
}

func TestRuntimeTargetDevicesUsesPlacementZoneScope(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen mic"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Kitchen display"})
	engine := NewEngine()
	engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
	placement := &fakePlacement{
		findRefs: []DeviceRef{{DeviceID: "d2"}},
	}
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: ui.NewMemoryBroadcaster(),
		Placement: placement,
	})

	matched, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "terminal",
		Arguments: map[string]string{
			"zone": "kitchen",
		},
	})
	if err != nil {
		t.Fatalf("HandleTrigger(terminal+zone) error = %v", err)
	}
	if matched != "terminal" {
		t.Fatalf("matched scenario = %q, want terminal", matched)
	}

	if active, ok := engine.Active("d2"); !ok || active != "terminal" {
		t.Fatalf("active(d2) = (%q, %v), want (terminal, true)", active, ok)
	}
	if _, ok := engine.Active("d1"); ok {
		t.Fatalf("d1 should remain inactive when placement scopes to d2")
	}
	if placement.findCalls != 1 {
		t.Fatalf("placement.Find calls = %d, want 1", placement.findCalls)
	}
	if placement.lastQuery.Scope.Zone != "kitchen" {
		t.Fatalf("placement query zone = %q, want kitchen", placement.lastQuery.Scope.Zone)
	}
}

func TestRuntimeTargetDevicesUsesPlacementNearest(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Origin"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Nearest screen"})
	engine := NewEngine()
	engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
	placement := &fakePlacement{
		nearestRef: DeviceRef{DeviceID: "d3"},
	}
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: ui.NewMemoryBroadcaster(),
		Placement: placement,
	})

	_, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "terminal",
		Arguments: map[string]string{
			"nearest":            "true",
			"nearest_capability": "screen",
		},
	})
	if err != nil {
		t.Fatalf("HandleTrigger(terminal+nearest) error = %v", err)
	}

	if active, ok := engine.Active("d3"); !ok || active != "terminal" {
		t.Fatalf("active(d3) = (%q, %v), want (terminal, true)", active, ok)
	}
	if placement.nearestCalls != 1 {
		t.Fatalf("placement.NearestWith calls = %d, want 1", placement.nearestCalls)
	}
	if placement.lastNearestCap != "screen" {
		t.Fatalf("placement nearest capability = %q, want screen", placement.lastNearestCap)
	}
}

func TestRuntimeTargetDevicesTrimsExplicitDeviceID(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Device 1"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Device 2"})

	engine := NewEngine()
	engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: ui.NewMemoryBroadcaster(),
	})

	matched, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "terminal",
		Arguments: map[string]string{
			"device_id": " d2 ",
		},
	})
	if err != nil {
		t.Fatalf("HandleTrigger(terminal+device_id) error = %v", err)
	}
	if matched != "terminal" {
		t.Fatalf("matched scenario = %q, want terminal", matched)
	}

	if active, ok := engine.Active("d2"); !ok || active != "terminal" {
		t.Fatalf("active(d2) = (%q, %v), want (terminal, true)", active, ok)
	}
	if _, ok := engine.Active("d1"); ok {
		t.Fatalf("d1 should remain inactive when explicit device_id targets d2")
	}
}

type fakePlacement struct {
	findRefs []DeviceRef
	findErr  error

	nearestRef DeviceRef
	nearestErr error

	findCalls      int
	nearestCalls   int
	lastQuery      PlacementQuery
	lastNearestCap string
}

func (f *fakePlacement) Find(_ context.Context, q PlacementQuery) ([]DeviceRef, error) {
	f.findCalls++
	f.lastQuery = q
	if f.findErr != nil {
		return nil, f.findErr
	}
	if len(f.findRefs) == 0 {
		return nil, nil
	}
	out := make([]DeviceRef, 0, len(f.findRefs))
	out = append(out, f.findRefs...)
	return out, nil
}

func (f *fakePlacement) NearestWith(_ context.Context, _ DeviceRef, capability string) (DeviceRef, error) {
	f.nearestCalls++
	f.lastNearestCap = capability
	if f.nearestErr != nil {
		return DeviceRef{}, f.nearestErr
	}
	return f.nearestRef, nil
}

func (f *fakePlacement) DevicesInZone(_ context.Context, _ string) ([]DeviceRef, error) {
	return nil, nil
}

func (f *fakePlacement) DevicesWithRole(_ context.Context, _ string) ([]DeviceRef, error) {
	return nil, nil
}

func TestRuntimeAudioMonitorPreemptedByRedAlertSuspendsAndResumes(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()
	deviceAudio := newFakeDeviceAudio()
	classifier := &testSoundClassifier{}

	engine := NewEngine()
	engine.Register(Registration{Scenario: &AudioMonitorScenario{}, Priority: PriorityNormal})
	engine.Register(Registration{Scenario: AlertScenario{}, Priority: PriorityCritical})
	runtime := NewRuntime(engine, &Environment{
		Devices:     devices,
		Broadcast:   broadcaster,
		Sound:       classifier,
		DeviceAudio: deviceAudio,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "audio_monitor",
		Arguments: map[string]string{
			"target": "dishwasher",
		},
	}); err != nil {
		t.Fatalf("HandleTrigger(audio_monitor) error = %v", err)
	}

	if !waitFor(func() bool { return deviceAudio.subscriberCount("d1") == 1 }, 200*time.Millisecond) {
		t.Fatalf("expected DeviceAudio subscriber count for d1 = 1, got %d", deviceAudio.subscriberCount("d1"))
	}

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "red_alert",
	}); err != nil {
		t.Fatalf("HandleTrigger(red_alert) error = %v", err)
	}

	if !waitFor(func() bool { return deviceAudio.subscriberCount("d1") == 0 }, 200*time.Millisecond) {
		t.Fatalf("expected preemption to close audio monitor subscription, got %d", deviceAudio.subscriberCount("d1"))
	}

	if _, err := runtime.StopTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "red_alert",
	}); err != nil {
		t.Fatalf("StopTrigger(red_alert) error = %v", err)
	}

	if !waitFor(func() bool { return deviceAudio.subscriberCount("d1") == 1 }, 200*time.Millisecond) {
		t.Fatalf("expected resume to re-open audio monitor subscription, got %d", deviceAudio.subscriberCount("d1"))
	}

	before := len(classifier.captured())
	deviceAudio.publish("d1", []byte("post-resume-audio"))
	if !waitFor(func() bool { return len(classifier.captured()) > before }, 300*time.Millisecond) {
		t.Fatalf("classifier did not receive post-resume audio; captured = %q", string(classifier.captured()))
	}
}

func TestRuntimeStartScenarioTargetsSpecificDevices(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
	})

	started, err := runtime.StartScenario(context.Background(), "terminal", []string{"d2"})
	if err != nil {
		t.Fatalf("StartScenario() error = %v", err)
	}
	if started != "terminal" {
		t.Fatalf("started = %q, want terminal", started)
	}
	if active, ok := engine.Active("d2"); !ok || active != "terminal" {
		t.Fatalf("active(d2) = (%q, %v), want (terminal, true)", active, ok)
	}
	if _, ok := engine.Active("d1"); ok {
		t.Fatalf("d1 should remain inactive when d2 is explicitly targeted")
	}
}

func TestRuntimeStopScenarioStopsTargetedDevice(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &TerminalScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
	})

	if _, err := runtime.StartScenario(context.Background(), "terminal", []string{"d1"}); err != nil {
		t.Fatalf("StartScenario() error = %v", err)
	}
	stopped, err := runtime.StopScenario(context.Background(), "terminal", []string{"d1"})
	if err != nil {
		t.Fatalf("StopScenario() error = %v", err)
	}
	if stopped != "terminal" {
		t.Fatalf("stopped = %q, want terminal", stopped)
	}
	if _, ok := engine.Active("d1"); ok {
		t.Fatalf("d1 should be inactive after StopScenario")
	}
}

func TestRuntimeIntercomPreemptedByRedAlertSuspendsAndResumesRoutes(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
	broadcaster := ui.NewMemoryBroadcaster()
	router := iorouter.NewRouter()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &IntercomScenario{}, Priority: PriorityHigh})
	engine.Register(Registration{Scenario: AlertScenario{}, Priority: PriorityCritical})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		IO:        router,
		Broadcast: broadcaster,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "intercom",
	}); err != nil {
		t.Fatalf("HandleTrigger(intercom) error = %v", err)
	}
	if got := router.RouteCount(); got != 4 {
		t.Fatalf("route count after intercom start = %d, want 4", got)
	}

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "red_alert",
	}); err != nil {
		t.Fatalf("HandleTrigger(red_alert) error = %v", err)
	}
	if got := router.RouteCount(); got != 0 {
		t.Fatalf("route count after red_alert preemption = %d, want 0", got)
	}

	if _, err := runtime.StopTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "red_alert",
	}); err != nil {
		t.Fatalf("StopTrigger(red_alert) error = %v", err)
	}
	if got := router.RouteCount(); got != 4 {
		t.Fatalf("route count after red_alert stop resume = %d, want 4", got)
	}
}

func TestRuntimePAPreemptedByRedAlertSuspendsAndResumesRoutes(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
	broadcaster := ui.NewMemoryBroadcaster()
	router := iorouter.NewRouter()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &PASystemScenario{}, Priority: PriorityHigh})
	engine.Register(Registration{Scenario: AlertScenario{}, Priority: PriorityCritical})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		IO:        router,
		Broadcast: broadcaster,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "pa_system",
	}); err != nil {
		t.Fatalf("HandleTrigger(pa_system) error = %v", err)
	}
	if got := router.RouteCount(); got != 2 {
		t.Fatalf("route count after pa start = %d, want 2", got)
	}

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "red_alert",
	}); err != nil {
		t.Fatalf("HandleTrigger(red_alert) error = %v", err)
	}
	if got := router.RouteCount(); got != 0 {
		t.Fatalf("route count after red_alert preemption = %d, want 0", got)
	}

	if _, err := runtime.StopTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "red_alert",
	}); err != nil {
		t.Fatalf("StopTrigger(red_alert) error = %v", err)
	}
	if got := router.RouteCount(); got != 2 {
		t.Fatalf("route count after red_alert stop resume = %d, want 2", got)
	}
}

func TestRuntimePAClaimsEndpointScopedResourcesWhenAvailable(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{
		DeviceID:   "d1",
		DeviceName: "Kitchen",
		Capabilities: device.CapabilitySet{
			"microphone.endpoint_count": "1",
			"microphone.endpoint.0.id":  "Mic Main",
		},
	})
	_, _ = devices.Register(device.Manifest{
		DeviceID:   "d2",
		DeviceName: "Hall",
		Capabilities: device.CapabilitySet{
			"speakers.endpoint_count": "1",
			"speakers.endpoint.0.id":  "Hall Speaker",
		},
	})
	router := iorouter.NewRouter()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &PASystemScenario{}, Priority: PriorityHigh})
	runtime := NewRuntime(engine, &Environment{
		Devices: devices,
		IO:      router,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "pa_system",
	}); err != nil {
		t.Fatalf("HandleTrigger(pa_system) error = %v", err)
	}

	sourceClaims := router.Claims().Snapshot("d1")
	if len(sourceClaims) != 1 {
		t.Fatalf("source claims len = %d, want 1", len(sourceClaims))
	}
	if sourceClaims[0].Resource != "audio_in.mic-main.capture" {
		t.Fatalf("source claim resource = %q, want audio_in.mic-main.capture", sourceClaims[0].Resource)
	}
	targetClaims := router.Claims().Snapshot("d2")
	if len(targetClaims) != 1 {
		t.Fatalf("target claims len = %d, want 1", len(targetClaims))
	}
	if targetClaims[0].Resource != "audio_out.hall-speaker" {
		t.Fatalf("target claim resource = %q, want audio_out.hall-speaker", targetClaims[0].Resource)
	}
}

// TestRuntimeAudioMonitorVoiceTriggerArmsWithParsedTarget verifies that the
// Phase-6 milestone phrasing ("tell me when the dishwasher stops") is parsed
// by ParseVoiceTrigger and routed through the runtime to AudioMonitorScenario
// with the parsed target surfaced in the arming broadcast.
func TestRuntimeAudioMonitorVoiceTriggerArmsWithParsedTarget(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()
	classifier := &testSoundClassifier{}

	engine := NewEngine()
	engine.Register(Registration{Scenario: &AudioMonitorScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
		Sound:     classifier,
	})

	now := time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
	started, err := runtime.HandleVoiceText(
		context.Background(),
		"d1",
		"tell me when the dishwasher stops",
		now,
	)
	if err != nil {
		t.Fatalf("HandleVoiceText error = %v", err)
	}
	if started != "audio_monitor" {
		t.Fatalf("started scenario = %q, want audio_monitor", started)
	}

	if !waitFor(func() bool { return len(broadcaster.Events()) >= 1 }, 200*time.Millisecond) {
		t.Fatalf("expected arming broadcast, got events = %+v", broadcaster.Events())
	}
	events := broadcaster.Events()
	if events[0].Message != "Audio monitor armed: dishwasher" {
		t.Fatalf("arming message = %q, want Audio monitor armed: dishwasher", events[0].Message)
	}
	if len(events[0].DeviceIDs) != 1 || events[0].DeviceIDs[0] != "d1" {
		t.Fatalf("arming device IDs = %+v, want [d1]", events[0].DeviceIDs)
	}
}

func TestRuntimeScheduleMonitorSensorHookNotifiesOnMotion(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &ScheduleMonitorScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "schedule_monitor",
	}); err != nil {
		t.Fatalf("HandleTrigger(schedule_monitor) error = %v", err)
	}

	err := runtime.ProcessSensorReading(context.Background(), SensorReading{
		DeviceID: "d1",
		UnixMS:   1713000000000,
		Values: map[string]float64{
			"accelerometer.x": 0.9,
			"accelerometer.y": 0.9,
			"accelerometer.z": 0.9,
		},
	})
	if err != nil {
		t.Fatalf("ProcessSensorReading() error = %v", err)
	}

	events := broadcaster.Events()
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2", len(events))
	}
	if events[0].Message != "Schedule monitor active" {
		t.Fatalf("arming message = %q, want Schedule monitor active", events[0].Message)
	}
	if events[1].Message != "Schedule monitor activity detected: magnitude=1.56" {
		t.Fatalf("activity message = %q, want motion detection", events[1].Message)
	}
	if len(events[1].DeviceIDs) != 1 || events[1].DeviceIDs[0] != "d1" {
		t.Fatalf("activity device IDs = %+v, want [d1]", events[1].DeviceIDs)
	}
}

func TestRuntimeScheduleMonitorSensorHookRespectsCooldown(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &ScheduleMonitorScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Broadcast: broadcaster,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "schedule_monitor",
		Arguments: map[string]string{
			"cooldown_ms": "60000",
		},
	}); err != nil {
		t.Fatalf("HandleTrigger(schedule_monitor) error = %v", err)
	}

	first := SensorReading{
		DeviceID: "d1",
		UnixMS:   1713000000000,
		Values: map[string]float64{
			"motion.magnitude": 2.0,
		},
	}
	second := SensorReading{
		DeviceID: "d1",
		UnixMS:   1713000005000, // 5s later, still within cooldown.
		Values: map[string]float64{
			"motion.magnitude": 3.0,
		},
	}

	if err := runtime.ProcessSensorReading(context.Background(), first); err != nil {
		t.Fatalf("ProcessSensorReading(first) error = %v", err)
	}
	if err := runtime.ProcessSensorReading(context.Background(), second); err != nil {
		t.Fatalf("ProcessSensorReading(second) error = %v", err)
	}

	events := broadcaster.Events()
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2 (armed + first detection)", len(events))
	}
	if events[1].Message != "Schedule monitor activity detected: magnitude=2.00" {
		t.Fatalf("detection message = %q, want first detection magnitude", events[1].Message)
	}
}

func TestRuntimeBluetoothPassthroughDispatchAndEventHook(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()
	passthrough := &testPassthroughBridge{}

	engine := NewEngine()
	engine.Register(Registration{Scenario: &BluetoothPassthroughScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:     devices,
		Broadcast:   broadcaster,
		Passthrough: passthrough,
	})

	started, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "bluetooth_passthrough",
		Arguments: map[string]string{
			"action":    "connect",
			"target_id": "AA:BB:CC:DD",
			"profile":   "a2dp",
		},
	})
	if err != nil {
		t.Fatalf("HandleTrigger(bluetooth_passthrough) error = %v", err)
	}
	if started != "bluetooth_passthrough" {
		t.Fatalf("started scenario = %q, want bluetooth_passthrough", started)
	}

	if len(passthrough.bluetooth) != 1 {
		t.Fatalf("len(bluetooth commands) = %d, want 1", len(passthrough.bluetooth))
	}
	cmd := passthrough.bluetooth[0]
	if cmd.DeviceID != "d1" || cmd.Action != "connect" || cmd.TargetID != "AA:BB:CC:DD" {
		t.Fatalf("bluetooth command = %+v", cmd)
	}
	if cmd.Parameters["profile"] != "a2dp" {
		t.Fatalf("profile = %q, want a2dp", cmd.Parameters["profile"])
	}

	if err := runtime.ProcessBluetoothEvent(context.Background(), BluetoothEvent{
		DeviceID: "d1",
		Event:    "scan_result",
		Data:     map[string]string{"target_id": "AA:BB:CC:DD"},
	}); err != nil {
		t.Fatalf("ProcessBluetoothEvent() error = %v", err)
	}

	events := broadcaster.Events()
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2", len(events))
	}
	if events[0].Message != "Bluetooth passthrough requested: connect" {
		t.Fatalf("start message = %q", events[0].Message)
	}
	if events[1].Message != "Bluetooth event: scan_result" {
		t.Fatalf("event message = %q", events[1].Message)
	}
}

func TestRuntimeUSBPassthroughDispatchAndEventHook(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()
	passthrough := &testPassthroughBridge{}

	engine := NewEngine()
	engine.Register(Registration{Scenario: &USBPassthroughScenario{}, Priority: PriorityNormal})
	runtime := NewRuntime(engine, &Environment{
		Devices:     devices,
		Broadcast:   broadcaster,
		Passthrough: passthrough,
	})

	started, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "usb_passthrough",
		Arguments: map[string]string{
			"action":     "claim",
			"vendor_id":  "1a2b",
			"product_id": "3c4d",
			"mode":       "raw",
		},
	})
	if err != nil {
		t.Fatalf("HandleTrigger(usb_passthrough) error = %v", err)
	}
	if started != "usb_passthrough" {
		t.Fatalf("started scenario = %q, want usb_passthrough", started)
	}

	if len(passthrough.usb) != 1 {
		t.Fatalf("len(usb commands) = %d, want 1", len(passthrough.usb))
	}
	cmd := passthrough.usb[0]
	if cmd.DeviceID != "d1" || cmd.Action != "claim" || cmd.VendorID != "1a2b" || cmd.ProductID != "3c4d" {
		t.Fatalf("usb command = %+v", cmd)
	}
	if cmd.Parameters["mode"] != "raw" {
		t.Fatalf("mode = %q, want raw", cmd.Parameters["mode"])
	}

	if err := runtime.ProcessUSBEvent(context.Background(), USBEvent{
		DeviceID: "d1",
		Event:    "device_claimed",
		Data:     map[string]string{"vendor_id": "1a2b", "product_id": "3c4d"},
	}); err != nil {
		t.Fatalf("ProcessUSBEvent() error = %v", err)
	}

	events := broadcaster.Events()
	if len(events) != 2 {
		t.Fatalf("event count = %d, want 2", len(events))
	}
	if events[0].Message != "USB passthrough requested: claim" {
		t.Fatalf("start message = %q", events[0].Message)
	}
	if events[1].Message != "USB event: device_claimed" {
		t.Fatalf("event message = %q", events[1].Message)
	}
}

func TestRuntimeDispatchPassthroughCommandsUsesBridge(t *testing.T) {
	passthrough := &testPassthroughBridge{}
	runtime := NewRuntime(NewEngine(), &Environment{Passthrough: passthrough})

	err := runtime.DispatchBluetoothCommand(context.Background(), BluetoothCommand{
		DeviceID: " d1 ",
		Action:   " scan ",
		Parameters: map[string]string{
			"profile": "le",
		},
	})
	if err != nil {
		t.Fatalf("DispatchBluetoothCommand() error = %v", err)
	}
	err = runtime.DispatchUSBCommand(context.Background(), USBCommand{
		DeviceID:  " d1 ",
		Action:    " enumerate ",
		VendorID:  " 1a2b ",
		ProductID: " 3c4d ",
	})
	if err != nil {
		t.Fatalf("DispatchUSBCommand() error = %v", err)
	}

	if len(passthrough.bluetooth) != 1 {
		t.Fatalf("len(bluetooth commands) = %d, want 1", len(passthrough.bluetooth))
	}
	if passthrough.bluetooth[0].DeviceID != "d1" || passthrough.bluetooth[0].Action != "scan" {
		t.Fatalf("unexpected bluetooth command: %+v", passthrough.bluetooth[0])
	}
	if len(passthrough.usb) != 1 {
		t.Fatalf("len(usb commands) = %d, want 1", len(passthrough.usb))
	}
	if passthrough.usb[0].DeviceID != "d1" || passthrough.usb[0].Action != "enumerate" {
		t.Fatalf("unexpected usb command: %+v", passthrough.usb[0])
	}
	if passthrough.usb[0].VendorID != "1a2b" || passthrough.usb[0].ProductID != "3c4d" {
		t.Fatalf("unexpected usb vid/pid: %+v", passthrough.usb[0])
	}
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

type testPassthroughBridge struct {
	mu sync.Mutex

	bluetooth []BluetoothCommand
	usb       []USBCommand
}

func (t *testPassthroughBridge) DispatchBluetoothCommand(_ context.Context, cmd BluetoothCommand) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.bluetooth = append(t.bluetooth, cmd)
	return nil
}

func (t *testPassthroughBridge) DispatchUSBCommand(_ context.Context, cmd USBCommand) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.usb = append(t.usb, cmd)
	return nil
}

type testSoundClassifier struct {
	events []SoundEvent

	mu     sync.Mutex
	buf    []byte
	out    chan SoundEvent
	closed bool
}

func (t *testSoundClassifier) Classify(_ context.Context, audioSrc AudioSource) (SoundEventStream, error) {
	t.mu.Lock()
	out := make(chan SoundEvent, len(t.events)+8)
	for _, event := range t.events {
		out <- event
	}
	autoClose := len(t.events) > 0
	t.out = out
	if autoClose {
		close(out)
		t.closed = true
	}
	t.mu.Unlock()

	if audioSrc != nil {
		go func() {
			buf := make([]byte, 256)
			for {
				n, err := audioSrc.Read(buf)
				if n > 0 {
					t.mu.Lock()
					t.buf = append(t.buf, buf[:n]...)
					t.mu.Unlock()
				}
				if err != nil {
					return
				}
			}
		}()
	}

	return out, nil
}

// captured returns a snapshot of bytes read from the audio source.
func (t *testSoundClassifier) captured() []byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]byte(nil), t.buf...)
}

// emit pushes a runtime event onto an open event stream. No-op if closed.
func (t *testSoundClassifier) emit(event SoundEvent) {
	t.mu.Lock()
	out := t.out
	closed := t.closed
	t.mu.Unlock()
	if closed || out == nil {
		return
	}
	out <- event
}

// close shuts down the event stream so range-reading consumers can exit.
func (t *testSoundClassifier) close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed || t.out == nil {
		return
	}
	close(t.out)
	t.closed = true
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

func TestRuntimeHandleTriggerTargetsExplicitDeviceIDs(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: PhotoFrameScenario{},
		Priority: PriorityLow,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices: devices,
	})

	name, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "photo frame",
		Arguments: map[string]string{
			"device_ids": "d1,d3",
		},
	})
	if err != nil {
		t.Fatalf("HandleTrigger() error = %v", err)
	}
	if name != "photo_frame" {
		t.Fatalf("name = %q, want photo_frame", name)
	}

	if active, ok := engine.Active("d1"); !ok || active != "photo_frame" {
		t.Fatalf("d1 active scenario = %q (ok=%t), want photo_frame", active, ok)
	}
	if _, ok := engine.Active("d2"); ok {
		t.Fatalf("expected d2 to remain inactive")
	}
	if active, ok := engine.Active("d3"); !ok || active != "photo_frame" {
		t.Fatalf("d3 active scenario = %q (ok=%t), want photo_frame", active, ok)
	}
}

func TestRuntimeIntercomTargetsExplicitDeviceIDs(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
	router := iorouter.NewRouter()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &IntercomScenario{}, Priority: PriorityHigh})
	runtime := NewRuntime(engine, &Environment{
		Devices: devices,
		IO:      router,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "intercom",
		Arguments: map[string]string{
			"device_ids": "d1,d3",
		},
	}); err != nil {
		t.Fatalf("HandleTrigger(intercom targeted) error = %v", err)
	}

	if got := router.RouteCount(); got != 2 {
		t.Fatalf("route count = %d, want 2", got)
	}
	routes := router.Routes()
	for _, route := range routes {
		if route.SourceID == "d2" || route.TargetID == "d2" {
			t.Fatalf("unexpected route involving d2: %+v", route)
		}
	}
}

func TestRuntimePATargetsExplicitDeviceIDs(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d3", DeviceName: "Office"})
	router := iorouter.NewRouter()

	engine := NewEngine()
	engine.Register(Registration{Scenario: &PASystemScenario{}, Priority: PriorityHigh})
	runtime := NewRuntime(engine, &Environment{
		Devices: devices,
		IO:      router,
	})

	if _, err := runtime.HandleTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "pa_system",
		Arguments: map[string]string{
			"device_ids": "d1,d3",
		},
	}); err != nil {
		t.Fatalf("HandleTrigger(pa targeted) error = %v", err)
	}

	if got := router.RouteCount(); got != 1 {
		t.Fatalf("route count = %d, want 1", got)
	}
	routes := router.Routes()
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1", len(routes))
	}
	if routes[0].SourceID != "d1" || routes[0].TargetID != "d3" || routes[0].StreamKind != "pa_audio" {
		t.Fatalf("route = %+v, want d1->d3 pa_audio", routes[0])
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

func TestRuntimeRecoverActivations(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	store := storage.NewMemoryStore()
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: &TerminalScenario{},
		Priority: PriorityNormal,
	})
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		Storage:   store,
		Broadcast: broadcaster,
	})

	if _, err := runtime.StartScenario(context.Background(), "terminal", []string{"d1"}); err != nil {
		t.Fatalf("StartScenario() error = %v", err)
	}

	recoveredEngine := NewEngine()
	recoveredEngine.Register(Registration{
		Scenario: &TerminalScenario{},
		Priority: PriorityNormal,
	})
	recovered := NewRuntime(recoveredEngine, &Environment{
		Devices:   devices,
		Storage:   store,
		Broadcast: broadcaster,
	})
	if err := recovered.RecoverActivations(context.Background()); err != nil {
		t.Fatalf("RecoverActivations() error = %v", err)
	}
	if got, ok := recovered.Engine.Active("d1"); !ok || got != "terminal" {
		t.Fatalf("Active(d1) = (%q,%v), want (terminal,true)", got, ok)
	}
}

func TestRuntimeNestedPreemptionSoak(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	router := iorouter.NewRouter()
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	RegisterBuiltins(engine)
	runtime := NewRuntime(engine, &Environment{
		Devices:   devices,
		IO:        router,
		Broadcast: broadcaster,
	})

	for i := 0; i < 25; i++ {
		if _, err := runtime.HandleTrigger(context.Background(), Trigger{
			Kind:     TriggerManual,
			SourceID: "d1",
			Intent:   "photo frame",
		}); err != nil {
			t.Fatalf("iteration %d photo_frame error = %v", i, err)
		}
		if _, err := runtime.HandleTrigger(context.Background(), Trigger{
			Kind:     TriggerManual,
			SourceID: "d1",
			Intent:   "voice_assistant",
		}); err != nil {
			t.Fatalf("iteration %d voice_assistant error = %v", i, err)
		}
		if _, err := runtime.HandleTrigger(context.Background(), Trigger{
			Kind:     TriggerManual,
			SourceID: "d1",
			Intent:   "pa_system",
		}); err != nil {
			t.Fatalf("iteration %d pa_system error = %v", i, err)
		}
		if _, err := runtime.HandleTrigger(context.Background(), Trigger{
			Kind:     TriggerManual,
			SourceID: "d1",
			Intent:   "red_alert",
		}); err != nil {
			t.Fatalf("iteration %d red_alert start error = %v", i, err)
		}
		if _, err := runtime.StopTrigger(context.Background(), Trigger{
			Kind:     TriggerManual,
			SourceID: "d1",
			Intent:   "red_alert",
		}); err != nil {
			t.Fatalf("iteration %d red_alert stop error = %v", i, err)
		}
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

func TestRuntimeHandleVoiceInternalVideoCallCreatesBidirectionalAVRoutes(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	_, _ = devices.Register(device.Manifest{DeviceID: "d2", DeviceName: "Hall"})
	router := iorouter.NewRouter()
	broadcaster := ui.NewMemoryBroadcaster()

	engine := NewEngine()
	engine.Register(Registration{
		Scenario: &InternalVideoCallScenario{},
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
		"video call d2",
		time.Date(2026, 4, 12, 0, 7, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("HandleVoiceText(video call) error = %v", err)
	}
	if name != "internal_video_call" {
		t.Fatalf("scenario name = %q, want internal_video_call", name)
	}
	if router.RouteCount() != 4 {
		t.Fatalf("route count = %d, want 4", router.RouteCount())
	}

	events := broadcaster.Events()
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].Message != "Video call active: d2" {
		t.Fatalf("source event message = %q, want Video call active: d2", events[0].Message)
	}
	if len(events[0].DeviceIDs) != 1 || events[0].DeviceIDs[0] != "d1" {
		t.Fatalf("source event device IDs = %+v, want [d1]", events[0].DeviceIDs)
	}
	if events[1].Message != "Incoming video call: d1" {
		t.Fatalf("target event message = %q, want Incoming video call: d1", events[1].Message)
	}
	if len(events[1].DeviceIDs) != 1 || events[1].DeviceIDs[0] != "d2" {
		t.Fatalf("target event device IDs = %+v, want [d2]", events[1].DeviceIDs)
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

// fakeDeviceAudio wraps an audio.Hub so scenario tests can exercise the
// DeviceAudioSubscriber interface end-to-end.
type fakeDeviceAudio struct {
	hub *audio.Hub
}

func newFakeDeviceAudio() *fakeDeviceAudio {
	return &fakeDeviceAudio{hub: audio.NewHub()}
}

func (f *fakeDeviceAudio) SubscribeAudio(ctx context.Context, deviceID string) (AudioSubscription, error) {
	return f.hub.Subscribe(ctx, deviceID), nil
}

func (f *fakeDeviceAudio) publish(deviceID string, chunk []byte) {
	f.hub.Publish(deviceID, chunk)
}

func (f *fakeDeviceAudio) subscriberCount(deviceID string) int {
	return f.hub.SubscriberCount(deviceID)
}

// waitFor polls condition until it returns true or timeout elapses.
func waitFor(condition func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return condition()
}

func findDescriptorProp(node ui.Descriptor, id, prop string) string {
	if node.Props["id"] == id {
		return node.Props[prop]
	}
	for _, child := range node.Children {
		if value := findDescriptorProp(child, id, prop); value != "" {
			return value
		}
	}
	return ""
}
