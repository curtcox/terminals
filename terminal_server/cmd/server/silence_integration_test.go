package main

import (
	"context"
	"encoding/binary"
	"math"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/ai"
	"github.com/curtcox/terminals/terminal_server/internal/audio"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

// TestSilenceClassifierThroughAudioHubNotifiesAudioMonitor wires the real
// SilenceClassifier, scenarioSoundClassifier adapter, scenarioDeviceAudio
// adapter, AudioMonitorScenario, and audio.Hub together end-to-end. PCM
// audio published to the hub should flow through the classifier, trigger a
// loud-to-quiet event, and cause the scenario to broadcast a detection
// notification to the source device.
func TestSilenceClassifierThroughAudioHubNotifiesAudioMonitor(t *testing.T) {
	devices := device.NewManager()
	_, _ = devices.Register(device.Manifest{DeviceID: "d1", DeviceName: "Kitchen"})
	broadcaster := ui.NewMemoryBroadcaster()
	hub := audio.NewHub()

	cfg := ai.SilenceClassifierConfig{
		SampleRate:     8000,
		Channels:       1,
		WindowDuration: 10 * time.Millisecond,
		LoudThreshold:  0.3,
		QuietThreshold: 0.05,
		HoldDuration:   60 * time.Millisecond,
	}
	classifier := ai.NewSilenceClassifier(cfg)

	engine := scenario.NewEngine()
	engine.Register(scenario.Registration{Scenario: &scenario.AudioMonitorScenario{}, Priority: scenario.PriorityNormal})
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:     devices,
		Broadcast:   broadcaster,
		Sound:       scenarioSoundClassifier{backend: classifier},
		DeviceAudio: scenarioDeviceAudio{hub: hub},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Target "sound" matches the classifier's "silence_after_sound" label
	// via substring containment in AudioMonitorScenario.
	if _, err := runtime.HandleTrigger(ctx, scenario.Trigger{
		Kind:      scenario.TriggerManual,
		SourceID:  "d1",
		Intent:    "audio_monitor",
		Arguments: map[string]string{"target": "sound"},
	}); err != nil {
		t.Fatalf("HandleTrigger(audio_monitor) error = %v", err)
	}

	// Wait for the scenario to subscribe to the hub before we publish.
	if !waitForSub(hub, "d1", 1, 500*time.Millisecond) {
		t.Fatalf("expected DeviceAudio subscriber count for d1 = 1, got %d", hub.SubscriberCount("d1"))
	}

	// Publish loud PCM chunks to establish the loud state.
	hub.Publish("d1", pcmSamples(cfg.SampleRate, 0.8, 200*time.Millisecond))
	// Then publish sustained quiet to trigger the loud-to-quiet event.
	hub.Publish("d1", pcmSamples(cfg.SampleRate, 0, 400*time.Millisecond))

	// Wait for the detection broadcast.
	deadline := time.Now().Add(2 * time.Second)
	var detected bool
	for time.Now().Before(deadline) {
		for _, ev := range broadcaster.Events() {
			if ev.Message == "Audio monitor detected: "+ai.SilenceAfterSoundLabel {
				if len(ev.DeviceIDs) != 1 || ev.DeviceIDs[0] != "d1" {
					t.Fatalf("detection device IDs = %+v, want [d1]", ev.DeviceIDs)
				}
				detected = true
				break
			}
		}
		if detected {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !detected {
		t.Fatalf("expected %q broadcast, got events = %+v",
			"Audio monitor detected: "+ai.SilenceAfterSoundLabel,
			broadcaster.Events())
	}
}

func waitForSub(hub *audio.Hub, deviceID string, want int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if hub.SubscriberCount(deviceID) == want {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return hub.SubscriberCount(deviceID) == want
}

// pcmSamples returns a 16-bit little-endian PCM buffer of the given
// duration. amplitude is the normalized [0, 1] peak amplitude of an
// alternating +/- square wave, so amplitude=0 produces a silent buffer.
func pcmSamples(sampleRate int, amplitude float64, d time.Duration) []byte {
	samples := int(int64(sampleRate) * int64(d) / int64(time.Second))
	if samples < 0 {
		samples = 0
	}
	buf := make([]byte, samples*2)
	sample := int16(math.Round(amplitude * float64(math.MaxInt16)))
	for i := 0; i < samples; i++ {
		v := sample
		if i%2 == 1 {
			v = -sample
		}
		binary.LittleEndian.PutUint16(buf[i*2:i*2+2], uint16(v))
	}
	return buf
}
