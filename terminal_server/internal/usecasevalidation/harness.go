// Package usecasevalidation provides a reusable test harness for running
// use-case validation scenarios against a real in-process server.
package usecasevalidation

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	goio "io"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	uiv1 "github.com/curtcox/terminals/terminal_server/gen/go/ui/v1"
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
	frames       []FrameRecord
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

// CaptureFrame records a deterministic server-side visual snapshot for docs.
func (h *Harness) CaptureFrame(stepID, terminal string, messages []transport.ProtoServerEnvelope) {
	summary := summarizeLatestUI(messages)
	h.mu.Lock()
	h.frames = append(h.frames, FrameRecord{
		StepID:    stepID,
		Terminal:  terminal,
		Label:     stepID,
		Summary:   summary,
		Path:      filepath.ToSlash(filepath.Join("frames", safeArtifactName(stepID)+".png")),
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
	frames := make([]FrameRecord, len(h.frames))
	copy(frames, h.frames)
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
			Media: MediaManifest{
				Frames: frames,
			},
		},
		Assertions:   assertions,
		Interactions: interactions,
		Frames:       frames,
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
	if len(frames) > 0 {
		if err := writeFramePNGs(resultDir, frames); err != nil {
			h.t.Logf("usecasevalidation: could not write frame artifacts: %v", err)
		}
	}

	if writeArtifacts {
		if err := writeJSONL(filepath.Join(dir, "assertions.jsonl"), assertionsToAny(assertions)); err != nil {
			h.t.Logf("usecasevalidation: could not write assertions.jsonl: %v", err)
		}
		if err := writeJSONL(filepath.Join(dir, "interaction_trace.jsonl"), interactionsToAny(interactions)); err != nil {
			h.t.Logf("usecasevalidation: could not write interaction_trace.jsonl: %v", err)
		}
		if len(frames) > 0 {
			if err := writeFramePNGs(dir, frames); err != nil {
				h.t.Logf("usecasevalidation: could not write evidence frame artifacts: %v", err)
			}
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
	if len(b.Frames) > 0 {
		fmt.Fprintf(&sb, "\n## Visual frames\n\n")
		for _, frame := range b.Frames {
			fmt.Fprintf(&sb, "- [%s](%s): %s\n", frame.Label, frame.Path, frame.Summary)
		}
	}
	fmt.Fprintf(&sb, "\n## Replay\n\n```bash\ngo test ./internal/usecasevalidation -run TestReplay -args -bundle %s\n```\n", filepath.Dir(path))
	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

func writeFramePNGs(baseDir string, frames []FrameRecord) error {
	for _, frame := range frames {
		path := filepath.Join(baseDir, filepath.FromSlash(frame.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		err = png.Encode(file, renderFrameImage(frame))
		closeErr := file.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func renderFrameImage(frame FrameRecord) image.Image {
	const width = 960
	const height = 540
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	fillRect(img, img.Bounds(), color.RGBA{R: 16, G: 20, B: 24, A: 255})
	fillRect(img, image.Rect(0, 0, width, 72), color.RGBA{R: 37, G: 90, B: 99, A: 255})
	drawText(img, 32, 28, "Terminals validation frame", color.RGBA{R: 238, G: 246, B: 247, A: 255})
	drawText(img, 32, 94, "Step: "+frame.StepID, color.RGBA{R: 226, G: 232, B: 234, A: 255})
	if frame.Terminal != "" {
		drawText(img, 32, 122, "Terminal: "+frame.Terminal, color.RGBA{R: 190, G: 203, B: 207, A: 255})
	}
	y := 166
	for _, line := range wrapText(frame.Summary, 82) {
		drawText(img, 32, y, line, color.RGBA{R: 238, G: 238, B: 232, A: 255})
		y += 28
		if y > height-36 {
			break
		}
	}
	return img
}

func summarizeLatestUI(messages []transport.ProtoServerEnvelope) string {
	for i := len(messages) - 1; i >= 0; i-- {
		resp, ok := messages[i].(*controlv1.ConnectResponse)
		if !ok {
			continue
		}
		if setUI := resp.GetSetUi(); setUI != nil {
			return "Set UI: " + summarizeNode(setUI.GetRoot())
		}
		if updateUI := resp.GetUpdateUi(); updateUI != nil {
			return "Update UI " + updateUI.GetComponentId() + ": " + summarizeNode(updateUI.GetNode())
		}
		if notification := resp.GetNotification(); notification != nil {
			return "Notification: " + notification.GetTitle() + " - " + notification.GetBody()
		}
	}
	return "No server-driven UI message was observed before this assertion."
}

func summarizeNode(node *uiv1.Node) string {
	if node == nil {
		return "(empty)"
	}
	parts := []string{nodeWidgetName(node)}
	if node.GetId() != "" {
		parts = append(parts, "#"+node.GetId())
	}
	if text := node.GetText(); text != nil && text.GetValue() != "" {
		parts = append(parts, quoteSummary(text.GetValue()))
	}
	if button := node.GetButton(); button != nil && button.GetLabel() != "" {
		parts = append(parts, "button "+quoteSummary(button.GetLabel()))
	}
	props := node.GetProps()
	if len(props) > 0 {
		keys := make([]string, 0, len(props))
		for key := range props {
			if key == "id" || key == "draw_ops_json" {
				continue
			}
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if props[key] != "" {
				parts = append(parts, key+"="+quoteSummary(props[key]))
			}
		}
	}
	children := node.GetChildren()
	if len(children) > 0 {
		childSummaries := make([]string, 0, minInt(len(children), 4))
		for _, child := range children {
			if len(childSummaries) == 4 {
				break
			}
			childSummaries = append(childSummaries, summarizeNode(child))
		}
		if len(children) > 4 {
			childSummaries = append(childSummaries, fmt.Sprintf("%d more", len(children)-4))
		}
		parts = append(parts, "children=["+strings.Join(childSummaries, "; ")+"]")
	}
	return strings.Join(parts, " ")
}

func nodeWidgetName(node *uiv1.Node) string {
	switch {
	case node.GetStack() != nil:
		return "stack"
	case node.GetRow() != nil:
		return "row"
	case node.GetGrid() != nil:
		return "grid"
	case node.GetScroll() != nil:
		return "scroll"
	case node.GetPadding() != nil:
		return "padding"
	case node.GetCenter() != nil:
		return "center"
	case node.GetExpand() != nil:
		return "expand"
	case node.GetText() != nil:
		return "text"
	case node.GetImage() != nil:
		return "image"
	case node.GetVideoSurface() != nil:
		return "video_surface"
	case node.GetAudioVisualizer() != nil:
		return "audio_visualizer"
	case node.GetCanvas() != nil:
		return "canvas"
	case node.GetTextInput() != nil:
		return "text_input"
	case node.GetButton() != nil:
		return "button"
	case node.GetSlider() != nil:
		return "slider"
	case node.GetToggle() != nil:
		return "toggle"
	case node.GetDropdown() != nil:
		return "dropdown"
	case node.GetGestureArea() != nil:
		return "gesture_area"
	case node.GetOverlay() != nil:
		return "overlay"
	case node.GetProgress() != nil:
		return "progress"
	case node.GetFullscreen() != nil:
		return "fullscreen"
	case node.GetKeepAwake() != nil:
		return "keep_awake"
	case node.GetBrightness() != nil:
		return "brightness"
	default:
		return "node"
	}
}

func quoteSummary(value string) string {
	if len(value) > 80 {
		value = value[:77] + "..."
	}
	return fmt.Sprintf("%q", value)
}

func safeArtifactName(value string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if b.Len() > 0 && !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "frame"
	}
	return out
}

func fillRect(img *image.RGBA, rect image.Rectangle, c color.RGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func drawText(img *image.RGBA, x, y int, text string, c color.RGBA) {
	for i, r := range text {
		if r < 32 || r > 126 {
			r = '?'
		}
		drawGlyph(img, x+i*8, y, byte(r), c)
	}
}

func drawGlyph(img *image.RGBA, x, y int, ch byte, c color.RGBA) {
	for row := 0; row < 7; row++ {
		for col := 0; col < 5; col++ {
			if ((int(ch) >> uint((row+col)%7)) & 1) == 0 {
				continue
			}
			fillRect(img, image.Rect(x+col, y+row*2, x+col+1, y+row*2+2), c)
		}
	}
}

func wrapText(text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{"(no UI summary)"}
	}
	var lines []string
	line := ""
	for _, word := range words {
		if len(line)+len(word)+1 > width && line != "" {
			lines = append(lines, line)
			line = word
			continue
		}
		if line == "" {
			line = word
		} else {
			line += " " + word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
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

// FrameRecord describes a deterministic visual artifact captured from server primitives.
type FrameRecord struct {
	StepID    string    `json:"step_id"`
	Terminal  string    `json:"terminal,omitempty"`
	Label     string    `json:"label"`
	Path      string    `json:"path"`
	Summary   string    `json:"summary,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// MediaManifest groups doc-site media artifacts emitted by validation.
type MediaManifest struct {
	Frames []FrameRecord `json:"frames,omitempty"`
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
	Media             MediaManifest       `json:"media,omitempty"`
}

// EvidenceBundle holds the full set of captured evidence for a scenario run.
type EvidenceBundle struct {
	Manifest     Manifest
	Assertions   []AssertionRecord
	Interactions []InteractionRecord
	Frames       []FrameRecord
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
