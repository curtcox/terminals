package scenario

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/audio"
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

type testLLM struct {
	text    string
	queries [][]LLMMessage
}

func (l *testLLM) Query(_ context.Context, messages []LLMMessage, _ LLMOptions) (*LLMResponse, error) {
	copyMsgs := make([]LLMMessage, len(messages))
	copy(copyMsgs, messages)
	l.queries = append(l.queries, copyMsgs)
	return &LLMResponse{Text: l.text, FinishReason: "stop"}, nil
}

type testTTS struct {
	calls []string
}

func (t *testTTS) Synthesize(_ context.Context, text string, _ TTSOptions) (AudioPlayback, error) {
	t.calls = append(t.calls, text)
	return bytes.NewReader(nil), nil
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
