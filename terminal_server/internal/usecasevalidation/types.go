package usecasevalidation

import (
	"context"
	goio "io"
	"sync"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/transport"
)

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

// AudioRecord describes a captured audio artifact from a PlayAudio server message.
type AudioRecord struct {
	Label     string    `json:"label"`
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
	// PCM holds the raw pcm16 bytes — populated only in memory, not serialised.
	PCM []byte `json:"-"`
}

// MediaManifest groups doc-site media artifacts emitted by validation.
type MediaManifest struct {
	Frames []FrameRecord `json:"frames,omitempty"`
	Videos []VideoRecord `json:"videos,omitempty"`
	Audio  []AudioRecord `json:"audio,omitempty"`
}

// VideoRecord describes a generated validation video artifact.
type VideoRecord struct {
	Label string `json:"label"`
	Path  string `json:"path"`
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
	Videos       []VideoRecord
	Audio        []AudioRecord
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
