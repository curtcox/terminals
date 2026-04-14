package transport

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/audio"
	"github.com/curtcox/terminals/terminal_server/internal/device"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

// TestControlStreamAudioMonitorStreamsLiveMicAudio drives the audio monitor
// pipeline through the control stream end-to-end:
//
//   - Manual command activates AudioMonitorScenario on a registered device.
//   - Subsequent VoiceAudio chunks are fanned out through the audio hub to
//     the scenario-owned SoundClassifier subscription.
//   - When the classifier emits a matching event, the scenario notifies the
//     source device via Broadcaster, which surfaces as a broadcast event.
//
// This is the Phase-6 "Tell me when the dishwasher stops" milestone as seen
// from the control-plane seam.
func TestControlStreamAudioMonitorStreamsLiveMicAudio(t *testing.T) {
	classifier := newLiveSoundClassifier()
	devices := device.NewManager()
	control := NewControlService("srv-1", devices)
	broadcaster := ui.NewMemoryBroadcaster()
	hub := audio.NewHub()

	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	runtime := scenario.NewRuntime(engine, &scenario.Environment{
		Devices:     devices,
		IO:          iorouter.NewRouter(),
		Sound:       classifier,
		Telephony:   telephony.NoopBridge{},
		Storage:     storage.NewMemoryStore(),
		Scheduler:   storage.NewMemoryScheduler(),
		Broadcast:   broadcaster,
		DeviceAudio: hubSubscriberAdapter{hub: hub},
	})

	handler := NewStreamHandlerWithRuntime(control, runtime)
	handler.SetDeviceAudioPublisher(hub)

	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Register: &RegisterRequest{
			DeviceID:   "device-1",
			DeviceName: "Kitchen Chromebook",
		},
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}

	// Activate the audio monitor scenario via a manual command.
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		Command: &CommandRequest{
			RequestID: "cmd-armed",
			DeviceID:  "device-1",
			Kind:      CommandKindManual,
			Intent:    "audio_monitor",
		},
	}); err != nil {
		t.Fatalf("activate audio_monitor error = %v", err)
	}

	// Wait for AudioMonitorScenario to register its live-audio subscription.
	if !waitForCondition(func() bool { return hub.SubscriberCount("device-1") == 1 }, 200*time.Millisecond) {
		t.Fatalf("expected 1 audio subscriber for device-1, got %d", hub.SubscriberCount("device-1"))
	}

	// Simulate live mic audio chunks streaming in from the device. Use
	// IsFinal=false throughout so the handler does not trigger the voice
	// command STT pipeline — the goal here is continuous monitoring.
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		VoiceAudio: &VoiceAudioRequest{
			DeviceID:   "device-1",
			Audio:      []byte("dishwasher-"),
			SampleRate: 16000,
			IsFinal:    false,
		},
	}); err != nil {
		t.Fatalf("VoiceAudio chunk error = %v", err)
	}
	if _, err := handler.HandleMessage(context.Background(), ClientMessage{
		VoiceAudio: &VoiceAudioRequest{
			DeviceID:   "device-1",
			Audio:      []byte("stopping"),
			SampleRate: 16000,
			IsFinal:    false,
		},
	}); err != nil {
		t.Fatalf("VoiceAudio chunk error = %v", err)
	}

	wantCaptured := "dishwasher-stopping"
	if !waitForCondition(func() bool { return string(classifier.captured()) == wantCaptured }, 500*time.Millisecond) {
		t.Fatalf("classifier captured = %q, want %q", string(classifier.captured()), wantCaptured)
	}

	// Emit a matching sound event; the scenario should notify the source
	// device through the broadcaster. The default target "sound" matches
	// every label, which is the behavior we exercise here.
	classifier.emit(scenario.SoundEvent{Label: "dishwasher_stopped", Confidence: 0.9, AtMS: 101})

	if !waitForCondition(func() bool {
		for _, e := range broadcaster.Events() {
			if e.Message == "Audio monitor detected: dishwasher_stopped" {
				return true
			}
		}
		return false
	}, 500*time.Millisecond) {
		t.Fatalf("expected detection broadcast, got events = %+v", broadcaster.Events())
	}

	// Once the scenario processes the matching event, it closes its
	// subscription and detaches from the hub.
	classifier.close()
	if !waitForCondition(func() bool { return hub.SubscriberCount("device-1") == 0 }, 200*time.Millisecond) {
		t.Fatalf("expected subscription to be released, got %d", hub.SubscriberCount("device-1"))
	}
}

// hubSubscriberAdapter wraps *audio.Hub so the scenario runtime can use it
// as a DeviceAudioSubscriber without introducing a dependency on the audio
// package from the scenario package.
type hubSubscriberAdapter struct {
	hub *audio.Hub
}

func (a hubSubscriberAdapter) SubscribeAudio(
	ctx context.Context,
	deviceID string,
) (scenario.AudioSubscription, error) {
	return a.hub.Subscribe(ctx, deviceID), nil
}

// liveSoundClassifier is a scenario.SoundClassifier that reads audio in a
// background goroutine and allows tests to emit events on demand.
type liveSoundClassifier struct {
	mu     sync.Mutex
	buf    []byte
	out    chan scenario.SoundEvent
	closed bool
}

func newLiveSoundClassifier() *liveSoundClassifier {
	return &liveSoundClassifier{}
}

func (c *liveSoundClassifier) Classify(
	_ context.Context,
	src scenario.AudioSource,
) (scenario.SoundEventStream, error) {
	c.mu.Lock()
	out := make(chan scenario.SoundEvent, 8)
	c.out = out
	c.mu.Unlock()

	if src != nil {
		go func() {
			buf := make([]byte, 256)
			for {
				n, err := src.Read(buf)
				if n > 0 {
					c.mu.Lock()
					c.buf = append(c.buf, buf[:n]...)
					c.mu.Unlock()
				}
				if err != nil {
					return
				}
			}
		}()
	}

	return out, nil
}

func (c *liveSoundClassifier) captured() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.buf...)
}

func (c *liveSoundClassifier) emit(event scenario.SoundEvent) {
	c.mu.Lock()
	out := c.out
	closed := c.closed
	c.mu.Unlock()
	if closed || out == nil {
		return
	}
	out <- event
}

func (c *liveSoundClassifier) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.out == nil {
		return
	}
	close(c.out)
	c.closed = true
}

func waitForCondition(condition func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return condition()
}
