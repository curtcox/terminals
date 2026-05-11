package scenario

import (
	"context"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

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
		t.Fatalf("StopTrigger() = %q, want audio_monitor", stopped)
	}

	if !waitFor(func() bool { return deviceAudio.subscriberCount("d1") == 0 }, 200*time.Millisecond) {
		t.Fatalf("expected DeviceAudio subscription released after stop, count = %d", deviceAudio.subscriberCount("d1"))
	}
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
		t.Fatalf("expected audio subscription released during red alert preemption")
	}

	if _, err := runtime.StopTrigger(context.Background(), Trigger{
		Kind:     TriggerManual,
		SourceID: "d1",
		Intent:   "red_alert",
	}); err != nil {
		t.Fatalf("StopTrigger(red_alert) error = %v", err)
	}

	if !waitFor(func() bool { return deviceAudio.subscriberCount("d1") == 1 }, 300*time.Millisecond) {
		t.Fatalf("expected audio subscription restored after red alert stop, count = %d", deviceAudio.subscriberCount("d1"))
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
