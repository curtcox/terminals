// Package usecasevalidation provides a reusable test harness for running
// use-case validation scenarios against a real in-process server.
package usecasevalidation

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	goio "io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
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

	mu           sync.Mutex
	assertions   []AssertionRecord
	interactions []InteractionRecord
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

// SimTerminal is an in-process simulated terminal running an async ProtoSession
// in a background goroutine. It captures all server messages for inspection.
type SimTerminal struct {
	DeviceID string

	h      *Harness
	sendCh chan transport.ProtoClientEnvelope
	outCh  chan transport.ProtoServerEnvelope
	newMsg chan struct{}
	doneCh chan struct{}
	err    error

	mu       sync.Mutex
	received []transport.ProtoServerEnvelope
}

func (st *SimTerminal) collect() {
	for env := range st.outCh {
		st.mu.Lock()
		st.received = append(st.received, env)
		st.mu.Unlock()
		select {
		case st.newMsg <- struct{}{}:
		default:
		}
	}
}

// Send delivers a message from this terminal to the server.
func (st *SimTerminal) Send(msg transport.ProtoClientEnvelope) {
	st.sendCh <- msg
}

// Disconnect closes the terminal's send channel, causing the session to end,
// then waits for the session goroutine to finish.
func (st *SimTerminal) Disconnect() error {
	close(st.sendCh)
	<-st.doneCh
	return st.err
}

// Received returns a copy of all server messages received so far.
func (st *SimTerminal) Received() []transport.ProtoServerEnvelope {
	st.mu.Lock()
	defer st.mu.Unlock()
	out := make([]transport.ProtoServerEnvelope, len(st.received))
	copy(out, st.received)
	return out
}

// WaitFor blocks until a received server message satisfies pred, or the
// timeout expires. Returns (matched message, true) on success.
func (st *SimTerminal) WaitFor(pred func(transport.ProtoServerEnvelope) bool, timeout time.Duration) (transport.ProtoServerEnvelope, bool) {
	deadline := time.Now().Add(timeout)
	for {
		st.mu.Lock()
		for _, env := range st.received {
			if pred(env) {
				st.mu.Unlock()
				return env, true
			}
		}
		st.mu.Unlock()

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, false
		}
		select {
		case <-st.newMsg:
		case <-time.After(remaining):
			return nil, false
		}
	}
}

// WaitForAny blocks until at least one server message arrives, or the timeout
// expires. Use this to confirm a session is established before sending commands.
func (st *SimTerminal) WaitForAny(timeout time.Duration) bool {
	_, ok := st.WaitFor(func(transport.ProtoServerEnvelope) bool { return true }, timeout)
	return ok
}

// asyncStream implements transport.ProtoStream using channels.
// sendCh carries messages from the test to the server (RecvProto reads it).
// outCh carries messages from the server to the test (SendProto writes to it).
type asyncStream struct {
	ctx    context.Context
	sendCh chan transport.ProtoClientEnvelope
	outCh  chan transport.ProtoServerEnvelope
}

func (a *asyncStream) RecvProto() (transport.ProtoClientEnvelope, error) {
	env, ok := <-a.sendCh
	if !ok {
		return nil, goio.EOF
	}
	return env, nil
}

func (a *asyncStream) SendProto(env transport.ProtoServerEnvelope) error {
	select {
	case a.outCh <- env:
		return nil
	case <-a.ctx.Done():
		return a.ctx.Err()
	}
}

func (a *asyncStream) Context() context.Context { return a.ctx }

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

// Evidence writes the evidence bundle for this run and returns a summary.
// The bundle is always written under artifacts/usecase-validation/<run-id>/.
// The full bundle (including assertions.jsonl) is written when any assertion
// failed or USECASE_ARTIFACTS=1 is set. Otherwise only manifest.json is written.
func (h *Harness) Evidence(usecaseID string) *EvidenceBundle {
	h.t.Helper()
	h.mu.Lock()
	assertions := make([]AssertionRecord, len(h.assertions))
	copy(assertions, h.assertions)
	interactions := make([]InteractionRecord, len(h.interactions))
	copy(interactions, h.interactions)
	h.mu.Unlock()

	end := time.Now().UTC()
	pass := true
	var failingIDs []string
	for _, a := range assertions {
		if !a.Pass {
			pass = false
			failingIDs = append(failingIDs, a.ID)
		}
	}

	bundle := &EvidenceBundle{
		Manifest: Manifest{
			RunID:             h.runID,
			UseCaseID:         usecaseID,
			ScenarioName:      h.t.Name(),
			GitCommit:         gitCommit(),
			TimestampStart:    h.start,
			TimestampEnd:      end,
			Pass:              pass,
			FailingAssertions: failingIDs,
			InteractionTrace:  interactions,
		},
		Assertions:   assertions,
		Interactions: interactions,
	}

	writeArtifacts := os.Getenv("USECASE_ARTIFACTS") == "1" || !pass
	dir := filepath.Join(artifactsRoot(), "usecase-validation", h.runID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		h.t.Logf("usecasevalidation: could not create artifacts dir %s: %v", dir, err)
		return bundle
	}

	if err := writeJSON(filepath.Join(dir, "manifest.json"), bundle.Manifest); err != nil {
		h.t.Logf("usecasevalidation: could not write manifest.json: %v", err)
	}
	resultDir := filepath.Join(artifactsRoot(), "usecases", usecaseID)
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		h.t.Logf("usecasevalidation: could not create result dir %s: %v", resultDir, err)
	} else if err := writeJSON(filepath.Join(resultDir, "result.json"), bundle.Manifest); err != nil {
		h.t.Logf("usecasevalidation: could not write result.json: %v", err)
	}

	if writeArtifacts {
		if err := writeJSONL(filepath.Join(dir, "assertions.jsonl"), assertionsToAny(assertions)); err != nil {
			h.t.Logf("usecasevalidation: could not write assertions.jsonl: %v", err)
		}
		if err := writeJSONL(filepath.Join(dir, "interaction_trace.jsonl"), interactionsToAny(interactions)); err != nil {
			h.t.Logf("usecasevalidation: could not write interaction_trace.jsonl: %v", err)
		}
		if err := writeSummaryMD(filepath.Join(dir, "summary.md"), bundle); err != nil {
			h.t.Logf("usecasevalidation: could not write summary.md: %v", err)
		}
		h.t.Logf("usecasevalidation: full evidence bundle at %s", dir)
	} else {
		h.t.Logf("usecasevalidation: manifest at %s/manifest.json (set USECASE_ARTIFACTS=1 for full bundle)", dir)
	}

	return bundle
}

func writeSummaryMD(path string, b *EvidenceBundle) error {
	m := b.Manifest
	result := "PASS"
	if !m.Pass {
		result = "FAIL"
	}

	passed := 0
	failed := 0
	for _, a := range b.Assertions {
		if a.Pass {
			passed++
		} else {
			failed++
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Use Case %s — Validation Summary\n\n", m.UseCaseID)
	fmt.Fprintf(&sb, "**Run ID:** %s  \n", m.RunID)
	fmt.Fprintf(&sb, "**Scenario:** %s  \n", m.ScenarioName)
	fmt.Fprintf(&sb, "**Result:** %s  \n", result)
	fmt.Fprintf(&sb, "**Start:** %s  \n", m.TimestampStart.Format(time.RFC3339))
	fmt.Fprintf(&sb, "**End:** %s  \n", m.TimestampEnd.Format(time.RFC3339))
	if m.GitCommit != "" {
		fmt.Fprintf(&sb, "**Git commit:** %s  \n", m.GitCommit)
	}
	fmt.Fprintf(&sb, "\n## Assertions (%d passed, %d failed)\n\n", passed, failed)
	fmt.Fprintf(&sb, "| ID | Description | Result | Detail |\n")
	fmt.Fprintf(&sb, "|---|---|---|---|\n")
	for _, a := range b.Assertions {
		mark := "✓ PASS"
		if !a.Pass {
			mark = "✗ FAIL"
		}
		fmt.Fprintf(&sb, "| %s | %s | %s | %s |\n", a.ID, a.Description, mark, a.Detail)
	}
	if len(m.FailingAssertions) > 0 {
		fmt.Fprintf(&sb, "\n**Failing assertions:** %s\n", strings.Join(m.FailingAssertions, ", "))
	}
	if len(b.Interactions) > 0 {
		fmt.Fprintf(&sb, "\n## Interaction trace\n\n")
		for i, interaction := range b.Interactions {
			fmt.Fprintf(&sb, "%d. %s\n", i+1, interaction.Summary)
		}
	}
	fmt.Fprintf(&sb, "\n## Replay\n\n```bash\ngo test ./internal/usecasevalidation -run TestReplay -args -bundle %s\n```\n", filepath.Dir(path))
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

// AssertionRecord captures the result of a single named assertion.
type AssertionRecord struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Pass        bool      `json:"pass"`
	Detail      string    `json:"detail,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// InteractionRecord captures a user-facing action injected by a scenario.
type InteractionRecord struct {
	Kind      string    `json:"kind"`
	Summary   string    `json:"summary"`
	Terminal  string    `json:"terminal,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Manifest is the top-level summary written to manifest.json.
type Manifest struct {
	RunID             string              `json:"run_id"`
	UseCaseID         string              `json:"usecase_id"`
	ScenarioName      string              `json:"scenario_name"`
	GitCommit         string              `json:"git_commit,omitempty"`
	TimestampStart    time.Time           `json:"timestamp_start"`
	TimestampEnd      time.Time           `json:"timestamp_end"`
	Pass              bool                `json:"pass"`
	FailingAssertions []string            `json:"failing_assertions,omitempty"`
	InteractionTrace  []InteractionRecord `json:"interaction_trace,omitempty"`
}

// EvidenceBundle holds the full set of captured evidence for a scenario run.
type EvidenceBundle struct {
	Manifest     Manifest
	Assertions   []AssertionRecord
	Interactions []InteractionRecord
}

// MemStream is an in-process implementation of transport.ProtoStream.
// It drains recvQueue for incoming messages and appends to Sent for outgoing ones.
type MemStream struct {
	ctx       context.Context
	recvQueue []transport.ProtoClientEnvelope
	mu        sync.Mutex
	pos       int
	Sent      []transport.ProtoServerEnvelope
}

// NewMemStream creates a MemStream that will deliver msgs in order then return EOF.
func NewMemStream(ctx context.Context, msgs []transport.ProtoClientEnvelope) *MemStream {
	return &MemStream{ctx: ctx, recvQueue: msgs}
}

// RecvProto delivers the next queued message or EOF when the queue is exhausted.
func (s *MemStream) RecvProto() (transport.ProtoClientEnvelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pos >= len(s.recvQueue) {
		return nil, goio.EOF
	}
	msg := s.recvQueue[s.pos]
	s.pos++
	return msg, nil
}

// SendProto appends a server-to-client message to Sent.
func (s *MemStream) SendProto(env transport.ProtoServerEnvelope) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Sent = append(s.Sent, env)
	return nil
}

// Context returns the stream's context.
func (s *MemStream) Context() context.Context { return s.ctx }

func artifactsRoot() string {
	// Walk up from the test binary's working directory to find the repo root.
	// Fall back to the current directory if not found.
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
			return filepath.Join(dir, "artifacts")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "artifacts"
}

func gitCommit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return s.Value
		}
	}
	return ""
}

func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	encErr := enc.Encode(v)
	closeErr := f.Close()
	if encErr != nil {
		return encErr
	}
	return closeErr
}

func writeJSONL(path string, records []any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	for _, r := range records {
		if err := enc.Encode(r); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := w.Flush(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func assertionsToAny(assertions []AssertionRecord) []any {
	out := make([]any, len(assertions))
	for i, a := range assertions {
		out[i] = a
	}
	return out
}

func interactionsToAny(interactions []InteractionRecord) []any {
	out := make([]any, len(interactions))
	for i, interaction := range interactions {
		out[i] = interaction
	}
	return out
}
