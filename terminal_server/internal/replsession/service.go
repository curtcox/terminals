package replsession

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/terminal"
)

var (
	// ErrMissingSessionID indicates a request did not include a session id.
	ErrMissingSessionID = errors.New("missing session id")
	// ErrMissingDeviceID indicates a request did not include a device id.
	ErrMissingDeviceID = errors.New("missing device id")
	// ErrSessionNotFound indicates a session id is unknown.
	ErrSessionNotFound = errors.New("repl session not found")
	// ErrDeviceNotAttached indicates the requesting device is not attached.
	ErrDeviceNotAttached = errors.New("device is not attached to session")
)

// ReplStateSnapshot captures persisted session state needed for reconnect.
type ReplStateSnapshot struct {
	History           []string
	PinnedContext     []string
	LLMThread         string
	ApprovalPolicy    string
	SelectedProvider  string
	SelectedModel     string
	PendingToolCallID string
}

// ReplSession is a typed metadata snapshot for a REPL session.
type ReplSession struct {
	ID                string
	OwnerActivationID string
	PTYSessionID      string
	AttachedDevices   []string
	CreatedAt         time.Time
	LastAttachAt      time.Time
	Idle              bool
	TTYWidth          int
	TTYHeight         int
	State             ReplStateSnapshot
}

// CreateSessionRequest creates a new REPL session.
type CreateSessionRequest struct {
	DeviceID          string
	OwnerActivationID string
	ReplAdminURL      string
}

// CreateSessionResponse returns the created session snapshot.
type CreateSessionResponse struct {
	Session ReplSession
}

// AttachSessionRequest attaches a device to an existing session.
type AttachSessionRequest struct {
	SessionID string
	DeviceID  string
}

// AttachSessionResponse returns the updated session snapshot.
type AttachSessionResponse struct {
	Session ReplSession
}

// DetachSessionRequest detaches a device from an existing session.
type DetachSessionRequest struct {
	SessionID string
	DeviceID  string
}

// DetachSessionResponse returns the updated session snapshot.
type DetachSessionResponse struct {
	Session ReplSession
}

// ResizeSessionRequest applies terminal dimensions to a session.
type ResizeSessionRequest struct {
	SessionID string
	TTYWidth  int
	TTYHeight int
}

// ResizeSessionResponse returns the updated session snapshot.
type ResizeSessionResponse struct {
	Session ReplSession
}

// SendInputRequest writes input bytes into the REPL PTY.
type SendInputRequest struct {
	SessionID string
	DeviceID  string
	Input     string
}

// SendInputResponse acknowledges accepted input.
type SendInputResponse struct{}

// TerminateSessionRequest terminates a REPL session.
type TerminateSessionRequest struct {
	SessionID string
}

// TerminateSessionResponse reports the terminated session id.
type TerminateSessionResponse struct {
	SessionID string
}

// ListSessionsRequest lists current REPL sessions.
type ListSessionsRequest struct{}

// ListSessionsResponse returns all known REPL sessions.
type ListSessionsResponse struct {
	Sessions []ReplSession
}

// GetSessionRequest fetches one REPL session.
type GetSessionRequest struct {
	SessionID string
}

// GetSessionResponse returns one REPL session.
type GetSessionResponse struct {
	Session ReplSession
}

// Service provides typed REPL session lifecycle APIs.
type Service struct {
	mu              sync.RWMutex
	now             func() time.Time
	nextID          atomic.Uint64
	terminalManager *terminal.Manager
	sessions        map[string]*liveSession
	deviceToSession map[string]string
}

type liveSession struct {
	meta          ReplSession
	attached      map[string]struct{}
	output        string
	draftByDevice map[string]string
	outputDirty   bool
	lastUIFlush   time.Time
}

// NewService constructs a typed in-memory REPL session service.
func NewService(terminals *terminal.Manager) *Service {
	if terminals == nil {
		terminals = terminal.NewManager()
	}
	return &Service{
		now:             time.Now,
		terminalManager: terminals,
		sessions:        map[string]*liveSession{},
		deviceToSession: map[string]string{},
	}
}

// CreateSession creates a new REPL session backed by a PTY.
func (s *Service) CreateSession(ctx context.Context, req CreateSessionRequest) (*CreateSessionResponse, error) {
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return nil, ErrMissingDeviceID
	}

	if existing, ok := s.SessionIDForDevice(deviceID); ok {
		session, err := s.GetSession(ctx, GetSessionRequest{SessionID: existing})
		if err != nil {
			return nil, err
		}
		return &CreateSessionResponse{Session: session.Session}, nil
	}

	metaID := fmt.Sprintf("repl-%d", s.nextID.Add(1))
	started, err := s.terminalManager.Start(ctx, terminal.StartOptions{
		DeviceID: deviceID,
		Env: []string{
			"TERMINALS_REPL_ADMIN_URL=" + strings.TrimSpace(req.ReplAdminURL),
		},
	})
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	live := &liveSession{
		meta: ReplSession{
			ID:                metaID,
			OwnerActivationID: strings.TrimSpace(req.OwnerActivationID),
			PTYSessionID:      started.ID,
			AttachedDevices:   []string{deviceID},
			CreatedAt:         now,
			LastAttachAt:      now,
			Idle:              false,
			State: ReplStateSnapshot{
				History:       []string{},
				PinnedContext: []string{},
			},
		},
		attached:      map[string]struct{}{deviceID: {}},
		draftByDevice: map[string]string{},
	}
	if live.meta.OwnerActivationID == "" {
		live.meta.OwnerActivationID = "terminal"
	}

	s.mu.Lock()
	s.sessions[live.meta.ID] = live
	s.deviceToSession[deviceID] = live.meta.ID
	s.mu.Unlock()

	return &CreateSessionResponse{Session: s.snapshot(live)}, nil
}

// AttachSession attaches a device to an existing REPL session.
func (s *Service) AttachSession(_ context.Context, req AttachSessionRequest) (*AttachSessionResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	deviceID := strings.TrimSpace(req.DeviceID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}
	if deviceID == "" {
		return nil, ErrMissingDeviceID
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	live, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	live.attached[deviceID] = struct{}{}
	live.meta.LastAttachAt = s.now().UTC()
	live.meta.Idle = false
	s.deviceToSession[deviceID] = sessionID
	live.meta.AttachedDevices = sortedKeys(live.attached)
	return &AttachSessionResponse{Session: s.snapshot(live)}, nil
}

// DetachSession detaches a device from an existing REPL session.
func (s *Service) DetachSession(_ context.Context, req DetachSessionRequest) (*DetachSessionResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	deviceID := strings.TrimSpace(req.DeviceID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}
	if deviceID == "" {
		return nil, ErrMissingDeviceID
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	live, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	delete(live.attached, deviceID)
	delete(s.deviceToSession, deviceID)
	delete(live.draftByDevice, deviceID)
	live.meta.AttachedDevices = sortedKeys(live.attached)
	live.meta.Idle = len(live.attached) == 0
	return &DetachSessionResponse{Session: s.snapshot(live)}, nil
}

// ResizeSession updates known terminal dimensions for a session.
func (s *Service) ResizeSession(_ context.Context, req ResizeSessionRequest) (*ResizeSessionResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	live, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	if req.TTYWidth > 0 {
		live.meta.TTYWidth = req.TTYWidth
	}
	if req.TTYHeight > 0 {
		live.meta.TTYHeight = req.TTYHeight
	}
	return &ResizeSessionResponse{Session: s.snapshot(live)}, nil
}

// SendInput writes input bytes into the backing PTY.
func (s *Service) SendInput(_ context.Context, req SendInputRequest) (*SendInputResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	deviceID := strings.TrimSpace(req.DeviceID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}
	if deviceID == "" {
		return nil, ErrMissingDeviceID
	}

	s.mu.RLock()
	live, ok := s.sessions[sessionID]
	if !ok {
		s.mu.RUnlock()
		return nil, ErrSessionNotFound
	}
	if _, attached := live.attached[deviceID]; !attached {
		s.mu.RUnlock()
		return nil, ErrDeviceNotAttached
	}
	ptySessionID := live.meta.PTYSessionID
	s.mu.RUnlock()

	if err := s.terminalManager.Write(ptySessionID, []byte(req.Input)); err != nil {
		return nil, err
	}
	return &SendInputResponse{}, nil
}

// TerminateSession closes and removes a REPL session.
func (s *Service) TerminateSession(_ context.Context, req TerminateSessionRequest) (*TerminateSessionResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}

	s.mu.Lock()
	live, ok := s.sessions[sessionID]
	if ok {
		delete(s.sessions, sessionID)
		for deviceID, mapped := range s.deviceToSession {
			if mapped == sessionID {
				delete(s.deviceToSession, deviceID)
			}
		}
	}
	s.mu.Unlock()
	if !ok {
		return nil, ErrSessionNotFound
	}

	if err := s.terminalManager.Close(live.meta.PTYSessionID); err != nil && !errors.Is(err, terminal.ErrSessionNotFound) {
		return nil, err
	}
	return &TerminateSessionResponse{SessionID: sessionID}, nil
}

// ListSessions returns all current REPL sessions.
func (s *Service) ListSessions(_ context.Context, _ ListSessionsRequest) (*ListSessionsResponse, error) {
	s.mu.RLock()
	out := make([]ReplSession, 0, len(s.sessions))
	for _, live := range s.sessions {
		out = append(out, s.snapshot(live))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return &ListSessionsResponse{Sessions: out}, nil
}

// GetSession returns one session by id.
func (s *Service) GetSession(_ context.Context, req GetSessionRequest) (*GetSessionResponse, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}
	s.mu.RLock()
	live, ok := s.sessions[sessionID]
	if !ok {
		s.mu.RUnlock()
		return nil, ErrSessionNotFound
	}
	snap := s.snapshot(live)
	s.mu.RUnlock()
	return &GetSessionResponse{Session: snap}, nil
}

// SessionIDForDevice returns the attached session id for a device.
func (s *Service) SessionIDForDevice(deviceID string) (string, bool) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return "", false
	}
	s.mu.RLock()
	sessionID, ok := s.deviceToSession[deviceID]
	s.mu.RUnlock()
	return sessionID, ok
}

// AppendOutput appends output text and returns the bounded output buffer.
func (s *Service) AppendOutput(sessionID, chunk string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", ErrMissingSessionID
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	live, ok := s.sessions[sessionID]
	if !ok {
		return "", ErrSessionNotFound
	}
	if chunk != "" {
		live.output += chunk
		live.outputDirty = true
	}
	if len(live.output) > 12000 {
		live.output = live.output[len(live.output)-12000:]
	}
	return live.output, nil
}

// ReadAvailable reads available bytes from the PTY and appends them to output.
func (s *Service) ReadAvailable(sessionID string, maxBytes int) ([]byte, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, ErrMissingSessionID
	}
	s.mu.RLock()
	live, ok := s.sessions[sessionID]
	if !ok {
		s.mu.RUnlock()
		return nil, ErrSessionNotFound
	}
	ptySessionID := live.meta.PTYSessionID
	s.mu.RUnlock()

	chunk, err := s.terminalManager.ReadAvailable(ptySessionID, maxBytes)
	if err != nil {
		return nil, err
	}
	if len(chunk) == 0 {
		return nil, nil
	}
	if _, err := s.AppendOutput(sessionID, string(chunk)); err != nil {
		return nil, err
	}
	return chunk, nil
}

// ShouldEmitUpdate returns whether a UI refresh should be emitted.
func (s *Service) ShouldEmitUpdate(sessionID string, force bool, now time.Time, interval time.Duration) (bool, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false, ErrMissingSessionID
	}
	s.mu.RLock()
	live, ok := s.sessions[sessionID]
	if !ok {
		s.mu.RUnlock()
		return false, ErrSessionNotFound
	}
	dirty := live.outputDirty
	last := live.lastUIFlush
	s.mu.RUnlock()

	if !dirty {
		return false, nil
	}
	if force {
		return true, nil
	}
	if interval <= 0 {
		interval = 800 * time.Millisecond
	}
	if last.IsZero() {
		return true, nil
	}
	return now.Sub(last) >= interval, nil
}

// MarkFlushed clears output dirty state and returns the current output.
func (s *Service) MarkFlushed(sessionID string, now time.Time) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", ErrMissingSessionID
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	live, ok := s.sessions[sessionID]
	if !ok {
		return "", ErrSessionNotFound
	}
	live.outputDirty = false
	live.lastUIFlush = now.UTC()
	return live.output, nil
}

// Output returns the current buffered output.
func (s *Service) Output(sessionID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", ErrMissingSessionID
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	live, ok := s.sessions[sessionID]
	if !ok {
		return "", ErrSessionNotFound
	}
	return live.output, nil
}

// SetDraft tracks a pending input draft for a device attached to a session.
func (s *Service) SetDraft(sessionID, deviceID, draft string) error {
	sessionID = strings.TrimSpace(sessionID)
	deviceID = strings.TrimSpace(deviceID)
	if sessionID == "" {
		return ErrMissingSessionID
	}
	if deviceID == "" {
		return ErrMissingDeviceID
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	live, ok := s.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}
	if _, attached := live.attached[deviceID]; !attached {
		return ErrDeviceNotAttached
	}
	live.draftByDevice[deviceID] = draft
	return nil
}

// Draft returns the pending input draft for an attached device.
func (s *Service) Draft(sessionID, deviceID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	deviceID = strings.TrimSpace(deviceID)
	if sessionID == "" {
		return "", ErrMissingSessionID
	}
	if deviceID == "" {
		return "", ErrMissingDeviceID
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	live, ok := s.sessions[sessionID]
	if !ok {
		return "", ErrSessionNotFound
	}
	if _, attached := live.attached[deviceID]; !attached {
		return "", ErrDeviceNotAttached
	}
	return live.draftByDevice[deviceID], nil
}

// ClearDraft removes any pending draft for an attached device.
func (s *Service) ClearDraft(sessionID, deviceID string) error {
	sessionID = strings.TrimSpace(sessionID)
	deviceID = strings.TrimSpace(deviceID)
	if sessionID == "" {
		return ErrMissingSessionID
	}
	if deviceID == "" {
		return ErrMissingDeviceID
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	live, ok := s.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}
	if _, attached := live.attached[deviceID]; !attached {
		return ErrDeviceNotAttached
	}
	delete(live.draftByDevice, deviceID)
	return nil
}

func (s *Service) snapshot(live *liveSession) ReplSession {
	out := live.meta
	out.AttachedDevices = sortedKeys(live.attached)
	out.State.History = append([]string(nil), live.meta.State.History...)
	out.State.PinnedContext = append([]string(nil), live.meta.State.PinnedContext...)
	return out
}

func sortedKeys(in map[string]struct{}) []string {
	out := make([]string, 0, len(in))
	for key := range in {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
