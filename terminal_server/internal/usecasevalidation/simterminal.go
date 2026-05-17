package usecasevalidation

import (
	"context"
	goio "io"
	"sync"
	"time"

	"github.com/curtcox/terminals/terminal_server/internal/transport"
)

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
