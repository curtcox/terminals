package eventlog

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultFileName = "terminals.jsonl"
)

type Config struct {
	Dir           string
	Level         string
	MaxBytes      int64
	MaxArchives   int
	MirrorStderr  bool
	ServerID      string
	ServerVersion string
}

type Logger struct {
	logger *slog.Logger
	writer *RotatingWriter
	async  *AsyncWriter
	runID  string
	pid    int
}

func New(cfg Config) (*Logger, error) {
	dir := strings.TrimSpace(cfg.Dir)
	if dir == "" {
		dir = "logs"
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = 100 * 1024 * 1024
	}
	if cfg.MaxArchives <= 0 {
		cfg.MaxArchives = 10
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	rw, err := NewRotatingWriter(filepath.Join(dir, defaultFileName), cfg.MaxBytes, cfg.MaxArchives)
	if err != nil {
		return nil, err
	}

	sink := io.Writer(rw)
	if cfg.MirrorStderr {
		sink = io.MultiWriter(rw, os.Stderr)
	}
	async := NewAsyncWriter(sink, 4096)

	runID := makeRunID()
	pid := os.Getpid()
	async.SetWriteFailureCallback(makeWriteFailureEmitter(writeFailureEmitterConfig{
		stderr:        os.Stderr,
		runID:         runID,
		pid:           pid,
		serverID:      strings.TrimSpace(cfg.ServerID),
		serverVersion: strings.TrimSpace(cfg.ServerVersion),
	}))
	h := &enrichHandler{
		next:          newJSONHandler(async, parseLevel(cfg.Level)),
		runID:         runID,
		pid:           pid,
		serverID:      strings.TrimSpace(cfg.ServerID),
		serverVersion: strings.TrimSpace(cfg.ServerVersion),
	}
	base := slog.New(h).With(
		slog.String("component", "main"),
	)
	return &Logger{logger: base, writer: rw, async: async, runID: runID, pid: pid}, nil
}

func (l *Logger) Logger() *slog.Logger {
	if l == nil || l.logger == nil {
		return slog.Default()
	}
	return l.logger
}

func (l *Logger) Component(name string) *slog.Logger {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "main"
	}
	return l.Logger().With(slog.String("component", name))
}

func (l *Logger) StdLogAdapter(component string) io.Writer {
	c := strings.TrimSpace(component)
	if c == "" {
		c = "legacy"
	}
	return &stdLogAdapter{logger: l.Component(c)}
}

func (l *Logger) Flush() error {
	if l == nil {
		return nil
	}
	if l.async != nil {
		if dropped := l.async.DroppedSinceLast(); dropped > 0 {
			l.logger.Warn("log events dropped",
				"event", "housekeeping.log.dropped",
				"component", "housekeeping",
				"dropped", dropped,
			)
		}
		if err := l.async.Flush(); err != nil {
			return err
		}
	}
	if l.writer == nil {
		return nil
	}
	return l.writer.Sync()
}

func (l *Logger) RunID() string {
	if l == nil {
		return ""
	}
	return l.runID
}

var defaultLogger atomic.Pointer[Logger]

func SetDefault(logger *Logger) {
	defaultLogger.Store(logger)
	if logger != nil {
		slog.SetDefault(logger.Logger())
	}
}

func Default() *Logger {
	return defaultLogger.Load()
}

func Component(name string) *slog.Logger {
	if logger := Default(); logger != nil {
		return logger.Component(name)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "main"
	}
	return slog.Default().With(slog.String("component", name))
}

func Emit(ctx context.Context, event string, level slog.Level, msg string, attrs ...slog.Attr) {
	event = strings.TrimSpace(event)
	if event == "" {
		event = "unknown.event"
	}
	if strings.TrimSpace(msg) == "" {
		msg = event
	}
	all := make([]slog.Attr, 0, len(attrs)+8)
	all = append(all, slog.String("event", event))
	if span, ok := spanFromContext(ctx); ok {
		if span.TraceID != "" {
			all = append(all, slog.String("trace_id", span.TraceID))
		}
		if span.SpanID != "" {
			all = append(all, slog.String("span_id", span.SpanID))
		}
		if span.ParentSpanID != "" {
			all = append(all, slog.String("parent_span_id", span.ParentSpanID))
		}
		if span.CorrelationID != "" {
			all = append(all, slog.String("correlation_id", span.CorrelationID))
		}
	}
	all = append(all, attrsFromContext(ctx)...)
	all = append(all, attrs...)
	slog.LogAttrs(ctx, level, msg, all...)
}

type ctxAttrsKey struct{}

type spanCtxKey struct{}

type spanCtx struct {
	TraceID       string
	SpanID        string
	ParentSpanID  string
	CorrelationID string
}

func With(ctx context.Context, key string, val any) context.Context {
	return WithAttrs(ctx, slog.Any(strings.TrimSpace(key), val))
}

func WithCorrelation(ctx context.Context, correlationID string) context.Context {
	span, _ := spanFromContext(ctx)
	span.CorrelationID = strings.TrimSpace(correlationID)
	return context.WithValue(ctx, spanCtxKey{}, span)
}

func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	clean := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		if strings.TrimSpace(attr.Key) == "" {
			continue
		}
		clean = append(clean, attr)
	}
	if len(clean) == 0 {
		return ctx
	}
	prior := attrsFromContext(ctx)
	out := make([]slog.Attr, 0, len(prior)+len(clean))
	out = append(out, prior...)
	out = append(out, clean...)
	return context.WithValue(ctx, ctxAttrsKey{}, out)
}

func attrsFromContext(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return nil
	}
	attrs, _ := ctx.Value(ctxAttrsKey{}).([]slog.Attr)
	if len(attrs) == 0 {
		return nil
	}
	out := make([]slog.Attr, len(attrs))
	copy(out, attrs)
	return out
}

func WithSpan(ctx context.Context, correlationID string) (context.Context, func()) {
	var (
		traceID string
		parent  string
	)
	if current, ok := spanFromContext(ctx); ok {
		traceID = current.TraceID
		parent = current.SpanID
	}
	if traceID == "" {
		traceID = randomHex(16)
	}
	span := spanCtx{
		TraceID:       traceID,
		SpanID:        randomHex(8),
		ParentSpanID:  parent,
		CorrelationID: strings.TrimSpace(correlationID),
	}
	if span.CorrelationID == "" {
		span.CorrelationID = "trace:" + traceID
	}
	return context.WithValue(ctx, spanCtxKey{}, span), func() {}
}

func spanFromContext(ctx context.Context) (spanCtx, bool) {
	if ctx == nil {
		return spanCtx{}, false
	}
	span, ok := ctx.Value(spanCtxKey{}).(spanCtx)
	return span, ok
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func makeRunID() string {
	return time.Now().UTC().Format("2006-01-02T15-04-05Z") + "-" + randomHex(2)
}

func randomHex(bytes int) string {
	if bytes <= 0 {
		bytes = 8
	}
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

type stdLogAdapter struct {
	logger *slog.Logger
	mu     sync.Mutex
	buf    string
}

func (a *stdLogAdapter) Write(p []byte) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.buf += string(p)
	for {
		idx := strings.IndexByte(a.buf, '\n')
		if idx < 0 {
			break
		}
		line := strings.TrimSpace(a.buf[:idx])
		a.buf = a.buf[idx+1:]
		if line == "" {
			continue
		}
		a.logger.LogAttrs(context.Background(), slog.LevelInfo, line, slog.String("event", "legacy.log"))
	}
	return len(p), nil
}

type enrichHandler struct {
	next          slog.Handler
	runID         string
	serverID      string
	serverVersion string
	pid           int
	seq           atomic.Uint64
}

func (h *enrichHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *enrichHandler) Handle(ctx context.Context, r slog.Record) error {
	nr := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	hasComponent := false
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "component" {
			hasComponent = true
		}
		nr.AddAttrs(a)
		return true
	})
	if !hasComponent {
		nr.AddAttrs(slog.String("component", "main"))
	}
	nr.AddAttrs(
		slog.String("run_id", h.runID),
		slog.Int("pid", h.pid),
		slog.Uint64("seq", h.seq.Add(1)),
	)
	if h.serverID != "" {
		nr.AddAttrs(slog.String("server_id", h.serverID))
	}
	if h.serverVersion != "" {
		nr.AddAttrs(slog.String("server_version", h.serverVersion))
	}
	return h.next.Handle(ctx, nr)
}

func (h *enrichHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &enrichHandler{
		next:          h.next.WithAttrs(attrs),
		runID:         h.runID,
		serverID:      h.serverID,
		serverVersion: h.serverVersion,
		pid:           h.pid,
		seq:           h.seq,
	}
}

func (h *enrichHandler) WithGroup(name string) slog.Handler {
	return &enrichHandler{
		next:          h.next.WithGroup(name),
		runID:         h.runID,
		serverID:      h.serverID,
		serverVersion: h.serverVersion,
		pid:           h.pid,
		seq:           h.seq,
	}
}

func newJSONHandler(w io.Writer, level slog.Level) slog.Handler {
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				tm, ok := a.Value.Any().(time.Time)
				if !ok {
					return slog.String("ts", time.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00"))
				}
				return slog.String("ts", tm.UTC().Format("2006-01-02T15:04:05.000Z07:00"))
			case slog.SourceKey:
				source, ok := a.Value.Any().(*slog.Source)
				if !ok || source == nil {
					return slog.Attr{}
				}
				file := source.File
				if wd, err := os.Getwd(); err == nil {
					if rel, relErr := filepath.Rel(wd, source.File); relErr == nil {
						file = rel
					}
				}
				return slog.Group("caller",
					slog.String("file", file),
					slog.Int("line", source.Line),
					slog.String("func", shortFunc(source.Function)),
				)
			}
			return a
		},
	}
	return slog.NewJSONHandler(w, opts)
}

func shortFunc(fn string) string {
	if strings.TrimSpace(fn) == "" {
		return ""
	}
	parts := strings.Split(fn, "/")
	last := parts[len(parts)-1]
	if !strings.Contains(last, ".") {
		return last
	}
	_, tail, ok := strings.Cut(last, ".")
	if !ok {
		return last
	}
	if runtime.GOOS == "windows" {
		tail = strings.ReplaceAll(tail, "\\", "/")
	}
	return tail
}

type writeFailureEmitterConfig struct {
	stderr        io.Writer
	runID         string
	pid           int
	serverID      string
	serverVersion string
}

func makeWriteFailureEmitter(cfg writeFailureEmitterConfig) func(WriteFailure) {
	stderr := cfg.stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	return func(failure WriteFailure) {
		record := map[string]any{
			"ts":        failure.At.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
			"level":     "error",
			"event":     "housekeeping.log.write_failed",
			"msg":       "event log sink write failed",
			"component": "housekeeping",
			"run_id":    cfg.runID,
			"pid":       cfg.pid,
			"error": map[string]any{
				"type":    "write_error",
				"message": failure.Err.Error(),
			},
		}
		if cfg.serverID != "" {
			record["server_id"] = cfg.serverID
		}
		if cfg.serverVersion != "" {
			record["server_version"] = cfg.serverVersion
		}
		encoded, err := json.Marshal(record)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "eventlog write failure event encode failed: %v\n", err)
			return
		}
		_, _ = fmt.Fprintf(stderr, "%s\n", string(encoded))
	}
}
