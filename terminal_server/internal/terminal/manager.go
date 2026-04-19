// Package terminal manages interactive PTY-backed terminal sessions.
package terminal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/creack/pty"
	"github.com/curtcox/terminals/terminal_server/internal/repl"
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
	// Command is the executable to launch inside the PTY.
	// When empty, Start defaults to launching the current server binary in
	// REPL mode ("repl" subcommand).
	Command string
	// Shell is retained as a deprecated alias of Command for compatibility.
	Shell string
	Args  []string
	Dir   string
	Env   []string
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
	in   io.WriteCloser
	out  io.ReadCloser
	done func()

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

// Start launches a new PTY-backed REPL session.
func (m *Manager) Start(ctx context.Context, opts StartOptions) (Session, error) {
	if opts.DeviceID == "" {
		return Session{}, ErrMissingDeviceID
	}
	command := opts.Command
	if command == "" {
		command = opts.Shell
	}
	if command == "" {
		var err error
		command, err = os.Executable()
		if err != nil {
			return Session{}, fmt.Errorf("resolve executable: %w", err)
		}
	}
	args := opts.Args
	if len(args) == 0 {
		args = []string{"repl"}
	}

	id := fmt.Sprintf("tty-%d", m.nextID.Add(1))
	meta := Session{
		ID:       id,
		DeviceID: opts.DeviceID,
		Shell:    command,
		Started:  m.now().UTC(),
	}

	if shouldUseInProcessREPL(opts) {
		live := m.startInProcessSession(ctx, meta, opts)
		m.mu.Lock()
		m.sessions[id] = live
		m.mu.Unlock()
		go m.captureOutput(id, live)
		return live.meta, nil
	}

	cmd := exec.CommandContext(ctx, command, args...)
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
		meta: meta,
		cmd:  cmd,
		in:   ptmx,
		out:  ptmx,
		done: func() {
			_ = ptmx.Close()
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			_ = cmd.Wait()
		},
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
	if _, err := live.in.Write(data); err != nil {
		return fmt.Errorf("write session input: %w", err)
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

	if live.done != nil {
		live.done()
	}
	return nil
}

// CloseAll terminates all active sessions.
func (m *Manager) CloseAll() {
	m.mu.RLock()
	ids := make([]string, 0, len(m.sessions))
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
		n, err := live.out.Read(buf)
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

func shouldUseInProcessREPL(opts StartOptions) bool {
	if strings.HasSuffix(os.Args[0], ".test") && opts.Command == "" && opts.Shell == "" {
		return true
	}
	return false
}

func (m *Manager) startInProcessSession(ctx context.Context, meta Session, opts StartOptions) *liveSession {
	inputReader, inputWriter := io.Pipe()
	outputReader, outputWriter := io.Pipe()
	childCtx, cancel := context.WithCancel(ctx)

	adminURL := ""
	for _, item := range opts.Env {
		key, value, ok := strings.Cut(item, "=")
		if ok && strings.TrimSpace(key) == "TERMINALS_REPL_ADMIN_URL" {
			adminURL = strings.TrimSpace(value)
			break
		}
	}
	go func() {
		_ = repl.Run(childCtx, inputReader, outputWriter, repl.Options{
			Prompt:       "repl> ",
			AdminBaseURL: adminURL,
		})
		_ = outputWriter.Close()
	}()

	return &liveSession{
		meta: meta,
		in:   inputWriter,
		out:  outputReader,
		done: func() {
			cancel()
			_ = inputReader.Close()
			_ = inputWriter.Close()
			_ = outputReader.Close()
			_ = outputWriter.Close()
		},
	}
}
