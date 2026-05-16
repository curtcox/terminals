// Package usecasevalidation provides a reusable test harness for running
// use-case validation scenarios against a real in-process server.
package usecasevalidation

import (
	"bufio"
	"context"
	goio "io"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
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

	Devices *device.Manager
	Control *transport.ControlService
	Runtime *scenario.Runtime

	mu         sync.Mutex
	assertions []AssertionRecord
}

// New creates a Harness bound to the given test.
func New(t testing.TB) *Harness {
	t.Helper()
	return &Harness{
		t:     t,
		runID: fmt.Sprintf("%d", time.Now().UnixNano()),
		start: time.Now().UTC(),
	}
}

// StartServer initializes a real in-process server with isolated, test-owned
// dependencies: memory storage, memory scheduler, noop telephony, and a fresh
// IO router. All dependencies are replaced with test doubles; no external
// services or subprocesses are started.
func (h *Harness) StartServer() {
	h.Devices = device.NewManager()
	h.Control = transport.NewControlService("srv-test", h.Devices)
	engine := scenario.NewEngine()
	scenario.RegisterBuiltins(engine)
	h.Runtime = scenario.NewRuntime(engine, &scenario.Environment{
		Devices:   h.Devices,
		IO:        iorouter.NewRouter(),
		Storage:   storage.NewMemoryStore(),
		Scheduler: storage.NewMemoryScheduler(),
		Telephony: telephony.NoopBridge{},
		Broadcast: ui.NewMemoryBroadcaster(),
	})
}

// NewStreamHandler creates a StreamHandler wired to the harness runtime.
func (h *Harness) NewStreamHandler() *transport.StreamHandler {
	return transport.NewStreamHandlerWithRuntime(h.Control, h.Runtime)
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

// Evidence writes the evidence bundle for this run and returns a summary.
// The bundle is always written under artifacts/usecase-validation/<run-id>/.
// The full bundle (including assertions.jsonl) is written when any assertion
// failed or USECASE_ARTIFACTS=1 is set. Otherwise only manifest.json is written.
func (h *Harness) Evidence(usecaseID string) *EvidenceBundle {
	h.t.Helper()
	h.mu.Lock()
	assertions := make([]AssertionRecord, len(h.assertions))
	copy(assertions, h.assertions)
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
		},
		Assertions: assertions,
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

	if writeArtifacts {
		if err := writeJSONL(filepath.Join(dir, "assertions.jsonl"), assertionsToAny(assertions)); err != nil {
			h.t.Logf("usecasevalidation: could not write assertions.jsonl: %v", err)
		}
		h.t.Logf("usecasevalidation: full evidence bundle at %s", dir)
	} else {
		h.t.Logf("usecasevalidation: manifest at %s/manifest.json (set USECASE_ARTIFACTS=1 for full bundle)", dir)
	}

	return bundle
}

// AssertionRecord captures the result of a single named assertion.
type AssertionRecord struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Pass        bool      `json:"pass"`
	Detail      string    `json:"detail,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// Manifest is the top-level summary written to manifest.json.
type Manifest struct {
	RunID             string    `json:"run_id"`
	UseCaseID         string    `json:"usecase_id"`
	ScenarioName      string    `json:"scenario_name"`
	GitCommit         string    `json:"git_commit,omitempty"`
	TimestampStart    time.Time `json:"timestamp_start"`
	TimestampEnd      time.Time `json:"timestamp_end"`
	Pass              bool      `json:"pass"`
	FailingAssertions []string  `json:"failing_assertions,omitempty"`
}

// EvidenceBundle holds the full set of captured evidence for a scenario run.
type EvidenceBundle struct {
	Manifest   Manifest
	Assertions []AssertionRecord
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

func (s *MemStream) SendProto(env transport.ProtoServerEnvelope) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Sent = append(s.Sent, env)
	return nil
}

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
	return filepath.Join("artifacts")
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
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeJSONL(path string, records []any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	for _, r := range records {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	return w.Flush()
}

func assertionsToAny(assertions []AssertionRecord) []any {
	out := make([]any, len(assertions))
	for i, a := range assertions {
		out[i] = a
	}
	return out
}
