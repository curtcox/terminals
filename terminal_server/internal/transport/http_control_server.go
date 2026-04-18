package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	controlv1 "github.com/curtcox/terminals/terminal_server/gen/go/control/v1"
	"github.com/curtcox/terminals/terminal_server/internal/eventlog"
	"google.golang.org/protobuf/proto"
)

const defaultHTTPPollWait = 20 * time.Second

type httpSession struct {
	id     string
	ctx    context.Context
	cancel context.CancelFunc
	inbox  chan *controlv1.WireEnvelope
	outbox chan *controlv1.WireEnvelope
}

type httpSessionEnvelopeStream struct {
	session *httpSession
}

func (s *httpSessionEnvelopeStream) ReadEnvelope() (*controlv1.WireEnvelope, error) {
	select {
	case <-s.session.ctx.Done():
		return nil, io.EOF
	case env, ok := <-s.session.inbox:
		if !ok {
			return nil, io.EOF
		}
		return env, nil
	}
}

func (s *httpSessionEnvelopeStream) WriteEnvelope(env *controlv1.WireEnvelope) error {
	select {
	case <-s.session.ctx.Done():
		return io.EOF
	case s.session.outbox <- env:
		return nil
	}
}

func (s *httpSessionEnvelopeStream) Context() context.Context {
	return s.session.ctx
}

func (s *httpSessionEnvelopeStream) Carrier() controlv1.CarrierKind {
	return controlv1.CarrierKind_CARRIER_KIND_HTTP
}

// HTTPControlServer hosts HTTP fallback control endpoints.
type HTTPControlServer struct {
	addr      string
	transport *Server

	running   atomic.Bool
	boundAddr string
	mu        sync.Mutex
	listener  net.Listener
	http      *http.Server
	sessions  map[string]*httpSession
}

// NewHTTPControlServer returns an HTTP fallback server.
func NewHTTPControlServer(addr string, transport *Server) *HTTPControlServer {
	return &HTTPControlServer{
		addr:      addr,
		transport: transport,
		sessions:  map[string]*httpSession{},
	}
}

// Address returns bound address or configured address.
func (s *HTTPControlServer) Address() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.boundAddr != "" {
		return s.boundAddr
	}
	return s.addr
}

// Running reports lifecycle state.
func (s *HTTPControlServer) Running() bool {
	return s.running.Load()
}

// Start binds and serves fallback HTTP endpoints.
func (s *HTTPControlServer) Start(context.Context) error {
	if s.transport == nil {
		return errors.New("http transport server not configured")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running.Load() {
		return nil
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/control/poll/", s.handlePoll)
	mux.HandleFunc("/v1/control/stream/", s.handleStream)

	httpServer := &http.Server{Handler: mux}
	s.listener = listener
	s.http = httpServer
	s.boundAddr = listener.Addr().String()
	s.running.Store(true)

	go func(listener net.Listener, server *http.Server) {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.mu.Lock()
			running := s.running.Load()
			address := s.boundAddr
			s.mu.Unlock()
			if running {
				eventlog.Emit(context.Background(), "transport.http.serve_failed", slog.LevelError, "http fallback serve failed",
					slog.String("component", "transport.http"),
					slog.String("address", address),
					slog.Any("error", err),
				)
			}
		}
	}(listener, httpServer)

	eventlog.Emit(context.Background(), "transport.http.listener_ready", slog.LevelInfo, "http fallback listener ready",
		slog.String("component", "transport.http"),
		slog.String("address", s.boundAddr),
	)
	return nil
}

// Stop gracefully shuts down fallback HTTP endpoints.
func (s *HTTPControlServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.running.Load() {
		s.mu.Unlock()
		return nil
	}
	httpServer := s.http
	address := s.boundAddr
	sessions := s.sessions
	s.sessions = map[string]*httpSession{}
	s.mu.Unlock()

	for _, session := range sessions {
		session.cancel()
	}

	err := httpServer.Shutdown(ctx)

	s.mu.Lock()
	s.running.Store(false)
	s.listener = nil
	s.http = nil
	s.boundAddr = ""
	s.mu.Unlock()

	eventlog.Emit(context.Background(), "transport.http.stopped", slog.LevelInfo, "http fallback server stopped",
		slog.String("component", "transport.http"),
		slog.String("address", address),
	)
	return err
}

func (s *HTTPControlServer) handlePoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID := sessionIDFromPath(r.URL.Path, "/v1/control/poll/")
	if sessionID == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	env := &controlv1.WireEnvelope{}
	if err := proto.Unmarshal(payload, env); err != nil {
		http.Error(w, "decode envelope", http.StatusBadRequest)
		return
	}

	session := s.getOrCreateSession(sessionID)
	select {
	case <-session.ctx.Done():
		http.Error(w, "session closed", http.StatusGone)
		return
	case session.inbox <- env:
		w.WriteHeader(http.StatusAccepted)
	}
}

func (s *HTTPControlServer) handleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID := sessionIDFromPath(r.URL.Path, "/v1/control/stream/")
	if sessionID == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	session, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	wait := defaultHTTPPollWait
	if raw := strings.TrimSpace(r.URL.Query().Get("wait_ms")); raw != "" {
		if parsed, err := time.ParseDuration(raw + "ms"); err == nil && parsed > 0 {
			wait = parsed
		}
	}

	var timer <-chan time.Time
	if wait > 0 {
		timer = time.After(wait)
	}

	select {
	case <-session.ctx.Done():
		http.Error(w, "session closed", http.StatusGone)
	case env := <-session.outbox:
		if env == nil {
			http.Error(w, "session closed", http.StatusGone)
			return
		}
		payload, err := proto.Marshal(env)
		if err != nil {
			http.Error(w, "encode envelope", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	case <-timer:
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *HTTPControlServer) getOrCreateSession(id string) *httpSession {
	s.mu.Lock()
	if session, ok := s.sessions[id]; ok {
		s.mu.Unlock()
		return session
	}
	ctx, cancel := context.WithCancel(context.Background())
	session := &httpSession{
		id:     id,
		ctx:    ctx,
		cancel: cancel,
		inbox:  make(chan *controlv1.WireEnvelope, 64),
		outbox: make(chan *controlv1.WireEnvelope, 64),
	}
	s.sessions[id] = session
	s.mu.Unlock()

	go func(session *httpSession) {
		defer func() {
			s.mu.Lock()
			delete(s.sessions, session.id)
			s.mu.Unlock()
			session.cancel()
			close(session.inbox)
			close(session.outbox)
		}()
		if err := s.transport.ConnectEnvelope(&httpSessionEnvelopeStream{session: session}); err != nil && !errors.Is(err, io.EOF) {
			eventlog.Emit(context.Background(), "transport.http.session.error", slog.LevelError, "http fallback control session ended with error",
				slog.String("component", "transport.http"),
				slog.String("session_id", session.id),
				slog.Any("error", err),
			)
		}
	}(session)

	return session
}

func sessionIDFromPath(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	id := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	id = strings.Trim(id, "/")
	if id == "" {
		return ""
	}
	return id
}

func httpSessionURL(baseAddr, sessionID, endpoint string) string {
	return fmt.Sprintf("http://%s%s/%s", baseAddr, endpoint, sessionID)
}
