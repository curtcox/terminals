// Package usecasevalidation provides a reusable test harness for running
// use-case validation scenarios against a real in-process server.
package usecasevalidation

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/device"
	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
	"github.com/curtcox/terminals/terminal_server/internal/scenario"
	"github.com/curtcox/terminals/terminal_server/internal/storage"
	"github.com/curtcox/terminals/terminal_server/internal/telephony"
	"github.com/curtcox/terminals/terminal_server/internal/transport"
	"github.com/curtcox/terminals/terminal_server/internal/ui"
)

// Harness is the core scaffolding for use-case validation scenarios.
// It starts a real in-process server, captures events and assertions,
// and can write a structured evidence bundle on completion.
type Harness struct {
	t     testing.TB
	runID string
	start time.Time
	clock *FakeClock

	Devices   *device.Manager
	Control   *transport.ControlService
	Runtime   *scenario.Runtime
	Broadcast *ui.MemoryBroadcaster
	Handler   *transport.StreamHandler

	sound  scenario.SoundClassifier
	llm    scenario.LLM
	vision scenario.VisionAnalyzer
	tts    *FakeTTS

	mu           sync.Mutex
	assertions   []AssertionRecord
	interactions []InteractionRecord
	frames       []FrameRecord
	audioClips   []AudioRecord
}

// New creates a Harness bound to the given test. The harness clock starts at
// the real current time; call h.Clock().SetNow to override it before StartServer.
func New(t testing.TB) *Harness {
	t.Helper()
	now := time.Now().UTC()
	return &Harness{
		t:     t,
		runID: fmt.Sprintf("%d", now.UnixNano()),
		start: now,
		clock: &FakeClock{now: now},
	}
}

// FakeClock is a deterministic clock for scenario tests. All harness helpers
// that need a "now" read from this clock. Advance synthetic time with Advance
// or AdvanceTo; never sleep in tests — drive the clock instead.
type FakeClock struct {
	mu  sync.Mutex
	now time.Time
}

// Now returns the current synthetic time.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// SetNow sets the fake clock to an absolute time.
func (c *FakeClock) SetNow(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t.UTC()
}

// Advance moves synthetic time forward by d.
func (c *FakeClock) Advance(d time.Duration) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	return c.now
}

// AdvanceTo moves synthetic time to t (no-op if t is before current time).
func (c *FakeClock) AdvanceTo(t time.Time) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t.UTC().After(c.now) {
		c.now = t.UTC()
	}
	return c.now
}

// StartServer initializes a real in-process server with isolated, test-owned
// dependencies: memory storage, memory scheduler, noop telephony, and a fresh
// IO router. All dependencies are replaced with test doubles; no external
// services or subprocesses are started.
func (h *Harness) StartServer() {
	h.Devices = device.NewManager()
	h.Control = transport.NewControlService("srv-test", h.Devices)
	h.Broadcast = ui.NewMemoryBroadcaster()
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	h.tts = &FakeTTS{}
	h.Runtime = scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   h.Devices,
		IO:        iorouter.NewRouter(),
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: h.Broadcast,
		Sound:     h.sound,
		LLM:       h.llm,
		Vision:    h.vision,
		TTS:       h.tts,
		UI:        ui.NewMemoryHost(),
	})
	h.Handler = transport.NewStreamHandlerWithRuntime(h.Control, h.Runtime)
}

// SetSound configures a SoundClassifier to inject into the scenario runtime.
// Must be called before StartServer.
func (h *Harness) SetSound(sc scenario.SoundClassifier) {
	h.sound = sc
}

// SetLLM configures an LLM to inject into the scenario runtime.
// Must be called before StartServer.
func (h *Harness) SetLLM(llm scenario.LLM) {
	h.llm = llm
}

// SetVision configures a VisionAnalyzer to inject into the scenario runtime.
// Must be called before StartServer.
func (h *Harness) SetVision(v scenario.VisionAnalyzer) {
	h.vision = v
}

// Clock returns the harness's deterministic fake clock. Scenario tests should
// use h.Clock().Advance or h.Clock().AdvanceTo to move synthetic time forward,
// then call h.ProcessDueTimers to fire any scheduled work that became due.
func (h *Harness) Clock() *FakeClock {
	return h.clock
}

// ProcessDueTimers drives the scenario runtime's timer loop at the current
// synthetic clock time. Returns the number of timers processed.
func (h *Harness) ProcessDueTimers(ctx context.Context) (int, error) {
	h.t.Helper()
	return h.Runtime.ProcessDueTimers(ctx, h.clock.Now())
}

// ConnectTerminal starts a simulated terminal session in a background goroutine.
// The initial message (typically a Register or CapabilitySnapshot) is sent once
// the session starts. Callers should call WaitForAny on the returned SimTerminal
// before sending subsequent messages, to ensure the session is established.
func (h *Harness) ConnectTerminal(deviceID string, initial transport.ProtoClientEnvelope) *SimTerminal {
	sendCh := make(chan transport.ProtoClientEnvelope, 16)
	outCh := make(chan transport.ProtoServerEnvelope, 64)
	newMsg := make(chan struct{}, 1)
	doneCh := make(chan struct{})

	st := &SimTerminal{
		DeviceID: deviceID,
		h:        h,
		sendCh:   sendCh,
		outCh:    outCh,
		newMsg:   newMsg,
		doneCh:   doneCh,
	}

	stream := &asyncStream{
		ctx:    context.Background(),
		sendCh: sendCh,
		outCh:  outCh,
	}

	go func() {
		defer close(doneCh)
		st.err = transport.RunProtoSession(h.Handler, h.Control, stream, transport.GeneratedProtoAdapter{})
	}()
	go st.collect()

	sendCh <- initial
	return st
}

// NewStreamHandler returns the shared StreamHandler for this harness.
// The same handler must be used across reconnect sessions so that
// per-device state (route replay, UI session state) is preserved.
func (h *Harness) NewStreamHandler() *transport.StreamHandler {
	return h.Handler
}

// Assert records an assertion result and fails the test if pass is false.
func (h *Harness) Assert(id, description string, pass bool, detail string) {
	h.t.Helper()
	h.mu.Lock()
	h.assertions = append(h.assertions, AssertionRecord{
		ID:          id,
		Description: description,
		Pass:        pass,
		Detail:      detail,
		Timestamp:   time.Now().UTC(),
	})
	h.mu.Unlock()
	if !pass {
		h.t.Errorf("assertion %s failed: %s — %s", id, description, detail)
	}
}

// RecordInteraction appends one user-facing action to the evidence timeline.
// Validation docs use this as the "How to use it" source, so keep summaries
// phrased from the actor's point of view rather than as low-level protocol.
func (h *Harness) RecordInteraction(kind, summary, terminal string) {
	h.mu.Lock()
	h.interactions = append(h.interactions, InteractionRecord{
		Kind:      kind,
		Summary:   summary,
		Terminal:  terminal,
		Timestamp: time.Now().UTC(),
	})
	h.mu.Unlock()
}
