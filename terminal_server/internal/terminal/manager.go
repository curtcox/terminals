// Package terminal manages interactive PTY-backed terminal sessions.
package terminal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/creack/pty"
)

var (
	// ErrMissingDeviceID indicates session creation was requested without a device id.
	ErrMissingDeviceID = errors.New("missing device id")
	// ErrSessionNotFound indicates a requested session id does not exist.
	ErrSessionNotFound = errors.New("session not found")
)

// StartOptions controls how a terminal session process is spawned.
type StartOptions struct {
	DeviceID string
	Shell    string
	Args     []string
	Dir      string
	Env      []string
}

// Session is an immutable metadata snapshot of an active terminal session.
type Session struct {
	ID       string
	DeviceID string
	Shell    string
	Started  time.Time
}

type liveSession struct {
	meta Session
	cmd  *exec.Cmd
	pty  *os.File

	bufferMu sync.Mutex
	buffer   bytes.Buffer
}

// Manager owns the lifecycle of all PTY sessions.
type Manager struct {
	mu       sync.RWMutex
	now      func() time.Time
	nextID   atomic.Uint64
	sessions map[string]*liveSession
}

// NewManager creates an empty terminal session manager.
func NewManager() *Manager {
	return &Manager{
		now:      time.Now,
		sessions: map[string]*liveSession{},
	}
}

// Start launches a new PTY-backed shell session.
func (m *Manager) Start(ctx context.Context, opts StartOptions) (Session, error) {
	if opts.DeviceID == "" {
		return Session{}, ErrMissingDeviceID
	}
	shell := opts.Shell
	if shell == "" {
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
	}
	args := opts.Args
	if len(args) == 0 {
		args = []string{"-i"}
	}

	id := fmt.Sprintf("tty-%d", m.nextID.Add(1))
	cmd := exec.CommandContext(ctx, shell, args...)
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}
	if len(opts.Env) > 0 {
		cmd.Env = append(os.Environ(), opts.Env...)
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return Session{}, fmt.Errorf("start pty: %w", err)
	}

	live := &liveSession{
		meta: Session{
			ID:       id,
			DeviceID: opts.DeviceID,
			Shell:    shell,
			Started:  m.now().UTC(),
		},
		cmd: cmd,
		pty: ptmx,
	}

	m.mu.Lock()
	m.sessions[id] = live
	m.mu.Unlock()

	go m.captureOutput(id, live)

	return live.meta, nil
}

// Write sends bytes into a running terminal session.
func (m *Manager) Write(sessionID string, data []byte) error {
	live, err := m.getSession(sessionID)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	if _, err := live.pty.Write(data); err != nil {
		return fmt.Errorf("write pty: %w", err)
	}
	return nil
}

// ReadAvailable returns and drains buffered output bytes for a session.
func (m *Manager) ReadAvailable(sessionID string, maxBytes int) ([]byte, error) {
	live, err := m.getSession(sessionID)
	if err != nil {
		return nil, err
	}
	if maxBytes <= 0 {
		maxBytes = 4096
	}

	live.bufferMu.Lock()
	defer live.bufferMu.Unlock()
	if live.buffer.Len() == 0 {
		return nil, nil
	}
	if live.buffer.Len() <= maxBytes {
		out := append([]byte(nil), live.buffer.Bytes()...)
		live.buffer.Reset()
		return out, nil
	}

	out := make([]byte, maxBytes)
	_, _ = live.buffer.Read(out)
	return out, nil
}

// Close terminates and removes a session.
func (m *Manager) Close(sessionID string) error {
	m.mu.Lock()
	live, ok := m.sessions[sessionID]
	if ok {
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()
	if !ok {
		return ErrSessionNotFound
	}

	_ = live.pty.Close()
	if live.cmd.Process != nil {
		_ = live.cmd.Process.Kill()
	}
	_ = live.cmd.Wait()
	return nil
}

// CloseAll terminates all active sessions.
func (m *Manager) CloseAll() {
	ids := []string{}
	m.mu.RLock()
	for id := range m.sessions {
		ids = append(ids, id)
	}
	m.mu.RUnlock()

	for _, id := range ids {
		_ = m.Close(id)
	}
}

// List returns active session snapshots sorted by session id.
func (m *Manager) List() []Session {
	m.mu.RLock()
	out := make([]Session, 0, len(m.sessions))
	for _, live := range m.sessions {
		out = append(out, live.meta)
	}
	m.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (m *Manager) getSession(sessionID string) (*liveSession, error) {
	m.mu.RLock()
	live, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrSessionNotFound
	}
	return live, nil
}

func (m *Manager) captureOutput(sessionID string, live *liveSession) {
	buf := make([]byte, 2048)
	for {
		n, err := live.pty.Read(buf)
		if n > 0 {
			live.bufferMu.Lock()
			_, _ = live.buffer.Write(buf[:n])
			live.bufferMu.Unlock()
		}
		if err != nil {
			// Process exit or pty close ends capture and cleans stale state.
			m.mu.Lock()
			delete(m.sessions, sessionID)
			m.mu.Unlock()
			return
		}
	}
}
