---
title: "Server Event Logging"
kind: plan
status: planned
owner: unowned
validation: none
last-reviewed: 2026-04-25
---

# Server Event Logging

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context. This plan introduces a structured, filesystem-resident event log that captures everything the server does while running, so future feature work can be grounded in real traces rather than guesswork.

## Goals

- Persist every runtime event the server produces to local disk in a machine-readable form.
- Preserve enough context per event to reconstruct causal chains — who triggered what, on which device, via which scenario or flow, and what the result was.
- Make the log trivially filterable and searchable both from the command line and from the admin UI.
- Treat logging as a first-class observability surface: feature developers should be able to say "show me every `scenario.activation.started` for device X in the last hour" without instrumentation work.

## Non-Goals

- Shipping logs off-box. The log is strictly local-filesystem in this phase; remote aggregation (Loki, ELK, OTLP) is a later concern and must not shape the on-disk format.
- Replacing the observation store, recording manager, or scenario activation persistence. Those remain authoritative for their domains; the event log is an append-only trace.
- Defining client-side logging. Only the Go server is in scope.

## Design Decisions

| Concern | Decision |
| --- | --- |
| Scope | All server events — lifecycle, transport, scenarios, observations, flows, admin, recording, telephony, errors. |
| Format | JSON Lines (`*.jsonl`), one event per line. |
| Rotation | Size-based, configurable max size and max archive count. |
| Location | `./logs/` under the server working directory by default, overridable via `TERMINALS_LOG_DIR`. |
| Redaction | None in this phase. Raw values go to disk; local filesystem is the trust boundary. |
| Reader surfaces | `term logs` CLI subcommand **and** an admin HTTP endpoint. |
| Instrumentation strategy | Migrate existing `log.Printf` callsites to `log/slog` with a JSON handler pointing at the rotating file, plus named domain-event emission at high-value seams. |

## Event Record Schema

Each line in a `*.jsonl` file is a single JSON object with stable top-level keys. Unknown or event-specific data goes under `attrs`.

```json
{
  "ts": "2026-04-16T14:03:22.184Z",
  "level": "info",
  "event": "scenario.activation.started",
  "msg": "scenario activation started",
  "component": "scenario.runtime",
  "server_id": "HomeServer",
  "server_version": "1",
  "pid": 48231,
  "run_id": "2026-04-16T14-03-18Z-8f2a",
  "seq": 1423,

  "trace_id": "01HW8J4Z7K...",
  "span_id": "9a1b...",
  "parent_span_id": "2c77...",
  "correlation_id": "trigger:intent:9a1b...",

  "activation_id": "act-0f91",
  "scenario": "bootstrap",
  "trigger": {
    "kind": "intent",
    "source": "device:kitchen-display",
    "intent": "play_music",
    "arguments": {"artist": "Miles Davis"}
  },
  "device": {"id": "kitchen-display"},
  "flow_id": "",
  "observation_id": "",

  "attrs": {
    "targets": ["kitchen-display"],
    "claims": ["audio.out:kitchen-display"]
  },

  "error": null,
  "caller": {"file": "scenario/runtime.go", "line": 214, "func": "Runtime.handleTrigger"}
}
```

### Top-level fields

- `ts` — RFC 3339 UTC timestamp with millisecond precision.
- `level` — `debug`, `info`, `warn`, `error`.
- `event` — dotted, stable event name (see [Event Taxonomy](#event-taxonomy)).
- `msg` — short human-readable message.
- `component` — subsystem that emitted the event (`transport.control`, `scenario.runtime`, `io.router`, `admin.http`, `recording.disk`, `telephony.sip`, `appruntime`, `discovery.mdns`, `world.model`, `observation.store`, `main`).
- `server_id`, `server_version` — from `config.Config`.
- `pid`, `run_id`, `seq` — disambiguate events across restarts and within a run. `run_id` is generated once per process start; `seq` is a monotonic per-run counter.

### Correlation fields

- `trace_id` / `span_id` / `parent_span_id` — lightweight tree-of-spans identifiers emitted whenever a request or activation creates logical children. Not OpenTelemetry yet; a plain string is sufficient for filtering.
- `correlation_id` — a human-meaningful anchor (e.g. `trigger:intent:...`, `grpc:RegisterDevice:...`) so lines that share no span can still be joined.

### Domain fields

Optional per-event but always rendered in a fixed location when present so queries are uniform:

- `activation_id`, `scenario`, `trigger`
- `flow_id`, `flow_node_id`
- `observation_id`, `artifact_id`
- `device` ( `id`, `kind`, `zone` )
- `http` ( `method`, `path`, `status`, `duration_ms` )
- `grpc` ( `method`, `peer`, `status`, `duration_ms` )
- `error` — `{ "type": "...", "message": "...", "stack": "..." }` (stack only at `error` level)

### Caller metadata

Captured by the `slog` handler for every record so grep-by-callsite works without decorating callsites manually.

## Event Taxonomy

Canonical event names, grouped by component. Components emit more over time; the names listed here are the minimum set the plan requires.

### Lifecycle (`main`)
- `server.starting`, `server.started`, `server.stopping`, `server.stopped`
- `config.loaded`
- `run.id_assigned`

### Transport (`transport.*`)
- `transport.grpc.listener_ready`, `transport.grpc.stopped`
- `transport.grpc.request.started`, `transport.grpc.request.finished`
- `transport.webrtc.signal.offer`, `transport.webrtc.signal.answer`, `transport.webrtc.signal.ice`
- `transport.stream.opened`, `transport.stream.closed`

### Discovery (`discovery.mdns`)
- `discovery.mdns.started`, `discovery.mdns.stopped`, `discovery.mdns.advert_renewed`

### Devices (`device`)
- `device.registered`, `device.heartbeat`, `device.liveness_changed`, `device.unregistered`

### Scenario engine (`scenario.*`)
- `scenario.definition.registered`
- `scenario.trigger.received`, `scenario.trigger.matched`, `scenario.trigger.unmatched`
- `scenario.activation.started`, `scenario.activation.suspended`, `scenario.activation.resumed`, `scenario.activation.stopped`
- `scenario.activation.failed`
- `scenario.recovery.started`, `scenario.recovery.finished`
- `scenario.timer.due`, `scenario.timer.processed`

### App runtime (`appruntime`)
- `appruntime.package.loaded`, `appruntime.package.skipped`
- `appruntime.definition.registered`
- `appruntime.op.emitted`

### IO / flow (`io.router`, `io.flow`)
- `io.route.applied`, `io.route.torn_down`
- `io.flow.started`, `io.flow.patched`, `io.flow.stopped`, `io.flow.stats`
- `io.analyzer.started`, `io.analyzer.event`, `io.analyzer.stopped`

### Observations (`observation.store`)
- `observation.emitted`, `observation.artifact.requested`, `observation.artifact.materialized`

### Recording (`recording.disk`)
- `recording.started`, `recording.segment_flushed`, `recording.stopped`, `recording.gc_ran`

### Telephony (`telephony.sip`)
- `telephony.bridge.registered`, `telephony.call.incoming`, `telephony.call.answered`, `telephony.call.ended`
- `telephony.media.rtp_in`, `telephony.media.rtp_out` (counters at `debug`)

### Admin (`admin.http`)
- `admin.http.request`, `admin.action.applied`

### Housekeeping (`housekeeping`)
- `housekeeping.liveness.reconciled`, `housekeeping.due_timers.processed`

### AI backends (`ai.*`)
- `ai.llm.request`, `ai.llm.response`, `ai.vision.request`, `ai.sound.classified`, `ai.stt.transcribed`, `ai.tts.synthesized`, `ai.wakeword.detected`

This list is load-bearing: every seam that currently calls `log.Printf` in `cmd/server` and the internal packages maps to one of these events as part of the migration (see [Deliverables](#deliverables)).

## File Layout, Rotation, Retention

### Directory

```
logs/
  terminals.jsonl             # current log
  terminals.jsonl.1           # most recent rotated archive
  terminals.jsonl.2
  ...
  terminals.jsonl.N           # oldest retained archive
```

The active file is always `terminals.jsonl` so `tail -F` works across rotations. Archives are renumbered on rotation (oldest removed when retention is exceeded).

### Rotation triggers

- Size: rotate when `terminals.jsonl` reaches `TERMINALS_LOG_MAX_BYTES` (default **100 MiB**).
- Boot: on startup, if the current file is non-empty and from a previous run, emit a `run.id_assigned` separator line (same file — no rotation on restart).

### Retention

- Keep at most `TERMINALS_LOG_MAX_ARCHIVES` rotated files (default **10**).
- Total on-disk budget defaults to ~1 GiB and can be tuned via the two env vars above.
- No time-based pruning in this phase; size bounds are sufficient and predictable.

### Config additions

New fields on `config.Config`:

| Field | Env var | Default |
| --- | --- | --- |
| `LogDir` | `TERMINALS_LOG_DIR` | `logs` |
| `LogLevel` | `TERMINALS_LOG_LEVEL` | `info` |
| `LogMaxBytes` | `TERMINALS_LOG_MAX_BYTES` | `104857600` |
| `LogMaxArchives` | `TERMINALS_LOG_MAX_ARCHIVES` | `10` |
| `LogStderr` | `TERMINALS_LOG_STDERR` | `true` (mirror to stderr during dev) |

`configs/server.env.example` and `configs/server.yaml` are updated to document each.

## Logger Implementation

### Core package: `internal/eventlog`

New package. Owns:

- The rotating file writer (size-based, implemented directly — no new third-party dependency; the logic is small and already has a natural test surface).
- A configured `*slog.Logger` whose handler writes JSON to the rotating writer.
- A secondary `io.MultiWriter` sink for stderr when `LogStderr` is true.
- A `RunID` and atomic `seq` counter included in every record via a `slog.Handler` wrapper.
- Context helpers: `eventlog.WithSpan(ctx, name) (ctx, end)` creates `span_id`/`parent_span_id` so nested operations share a trace. `eventlog.With(ctx, key, val)` attaches a structured attribute for the remainder of the context.
- An `Emit(ctx, event, level, msg, attrs...)` convenience for domain-event emission that enforces the `event` name is non-empty.

### Global wiring

`cmd/server/main.go` builds the logger from `config.Config` before any other subsystem, then:

- Installs it as `slog.Default()`.
- Redirects the standard `log` package output via `log.SetOutput(eventlog.StdLogAdapter())` and `log.SetFlags(0)` so any residual `log.Printf` call lands in the JSONL file under a `component:"legacy"` tag until migrated.
- Passes the logger (or a component-scoped child) into constructors that already accept an options struct. New subsystems take a `*slog.Logger` via constructor injection; leaf helpers may fetch a child via `slog.Default().With(...)`.

### Migration of existing `log.Printf` callsites

Each `log.Printf` in the server becomes a structured call. Mapping rules:

- Startup lines → `server.*` / `config.loaded` / subsystem `.ready` events at `info`.
- Failure lines (`"configure X: %v"`) → `error` level with `error` field, and the matching `server.starting.failed` / `*.failed` event name.
- Loop heartbeats (`"due timer loop processed=%d"`) → `housekeeping.*` events at `debug` when the count is zero, `info` when non-zero.
- Telephony transport/media log lines (`telephony.LogTransport{Logf: log.Printf}`) get replaced with a `telephony.LogTransport{Logger: eventlog.Component("telephony.sip")}` adapter that maps each entry to a typed event.

The migration is mechanical and is tracked as a deliverable; it does not require semantic refactors of the surrounding code.

### Domain-event seams

Beyond the mechanical migration, these seams get first-class emission (they currently produce no `log.Printf`):

- `scenario.Runtime` — `scenario.trigger.*`, `scenario.activation.*`, `scenario.timer.*`.
- `observation.Store.AddObservation` — `observation.emitted` with full `Observation` fields flattened into `attrs`.
- `io.MediaPlanner` analyzer and observation sinks — `io.analyzer.event`, `io.flow.*`.
- `transport.ControlService` request entry/exit — `transport.grpc.request.*` with `grpc.method`, `peer`, `status`, `duration_ms`.
- `admin.Handler` — `admin.http.request` and `admin.action.applied` for every mutation.
- `recording.DiskManager` — `recording.segment_flushed` on each flush.
- `appruntime.Runtime` — `appruntime.op.emitted` whenever a TAL result commits ops.

All of these already have the data in-scope; the work is wiring, not modeling.

## Correlation Model

- Every inbound unit of work gets a `trace_id` at the outermost seam (gRPC interceptor, admin HTTP middleware, mDNS advert, scheduler tick). The trace_id is attached to the context via `eventlog.WithSpan` and propagates automatically.
- Scenario activations inherit the trace_id of the trigger that produced them and stamp `activation_id` on every subsequent record.
- Observation records carry the `flow_id` + `flow_node_id` from their `Provenance` into event logs so flow graphs can be reconstructed from the log alone.
- `correlation_id` is a human-stable string chosen at the seam (for example `intent:<intent>:<source>:<ts>`) that makes grep-for-cause searches easy when a trace_id is not known.

## Reader Surfaces

### CLI: `term logs`

Extend the existing `cmd/term` binary with a `logs` subcommand family that reads the same JSONL files. This is strictly read-only and works against a local `logs/` directory (path overridable via `--dir` / `TERMINALS_LOG_DIR`).

Subcommands:

- `term logs tail [--follow] [--since DURATION] [-n N]`
  Print the last N events; optionally follow new events. Output defaults to colorized human-readable text but `--json` emits the raw JSONL.
- `term logs search [filters...]`
  Structured filter language over fields:
  `event=scenario.activation.started`, `activation_id=act-0f91`, `device.id=kitchen-display`, `level>=warn`, `since=1h`, `until=2026-04-16T13:00Z`, `trace=<trace_id>`, `component=scenario.runtime`, `free-text`.
  Multiple filters AND together; repeated keys OR.
- `term logs trace <trace_id>`
  Reconstruct and print the full span tree for a trace, indented and time-ordered.
- `term logs activation <activation_id>`
  Shortcut for `search activation_id=<id>` sorted by `seq`.
- `term logs stats [--by=event|component|level] [--since=1h]`
  Quick histogram for ad-hoc triage.

The CLI consumes archives and the current file transparently, in time order, handling rotation boundaries.

### Admin HTTP endpoint

New endpoint registered by `admin.Handler`:

- `GET /admin/logs` — HTML view with filter form. Renders a paginated table of recent events, highlights errors, and links traces/activations to filtered views.
- `GET /admin/logs.jsonl?...filters...` — NDJSON stream of matching events. Same filter vocabulary as the CLI (same server-side parser).
- `GET /admin/logs/trace/{trace_id}` — rendered trace tree.
- `GET /admin/logs/activation/{activation_id}` — rendered activation timeline.

Both the CLI and the admin endpoint share a single Go package (`internal/eventlog/query`) that parses filters and iterates rotated files. The admin HTTP view is a thin presentation layer on top.

## Self-Observability

- The logger must never block the server. The rotating writer uses a buffered channel; on overflow it drops the oldest events and increments a `log_events_dropped` counter, which itself is emitted as a `housekeeping.log.dropped` event on the next flush.
- Disk-full on the log directory must not crash the server. On write failure, fall back to stderr and emit a single `housekeeping.log.write_failed` event per minute until healthy.
- The logger package exposes a `Flush()` called from `main.go` during shutdown so the final `server.stopped` event is durable.

## Deliverables

- [ ] `internal/eventlog/` package: rotating writer, slog handler, context helpers, run-id/seq injector, stderr mirror.
- [ ] `internal/eventlog/query/` package: filter parser, file iterator, trace/activation aggregators.
- [ ] `config.Config` extended with `LogDir`, `LogLevel`, `LogMaxBytes`, `LogMaxArchives`, `LogStderr`, plus matching env parsing and example files.
- [ ] `cmd/server/main.go` initializes the logger first and routes the standard library `log` through it.
- [ ] Every existing `log.Printf` / `log.Fatalf` / `log.Println` in `terminal_server/` migrated to `slog` with a mapped `event` name (see [Event Taxonomy](#event-taxonomy)).
- [ ] gRPC server / stream handler: unary and stream interceptors emit `transport.grpc.request.*` events and attach a `trace_id`.
- [ ] Admin HTTP: middleware emits `admin.http.request` events and attaches a `trace_id`.
- [ ] Scenario runtime, observation store, IO router, recording manager, telephony bridge, app runtime, housekeeping loops instrumented with the domain events listed above.
- [ ] `cmd/term logs …` subcommand family implemented against `internal/eventlog/query`.
- [ ] `admin.Handler` registers `/admin/logs`, `/admin/logs.jsonl`, `/admin/logs/trace/{id}`, `/admin/logs/activation/{id}` and renders the HTML view.
- [ ] Docs: `terminal_server/CLAUDE.md` gains a "Event logging" section; `README.md` gets a short pointer; `docs/` gains an event-taxonomy reference generated from (or checked against) the list in this plan.

## Testing

- **Unit**
  - Rotating writer: rotation at configured size, archive renumbering, retention cap, tolerates interleaved fsync failures.
  - Handler: every record has `ts`/`run_id`/`seq`/`component`, `seq` is monotonic, context span ids propagate.
  - Filter parser: covers every supported operator and precedence edge case.
- **Integration**
  - Boot the server with a temp `LogDir`, drive a scripted scenario (trigger → activation → observation → stop), assert the resulting JSONL contains exactly the expected events in order with a single shared `trace_id`.
  - Simulate a rotation mid-test and assert the CLI reader stitches archives and the live file correctly.
  - Disk-full simulation: writes fail, server stays up, `housekeeping.log.write_failed` is emitted on stderr, recovery resumes writing.
- **CLI**
  - `term logs search` golden tests on fixture JSONL files.
- **Admin**
  - HTTP handler tests for filter round-trip, trace reconstruction, and NDJSON streaming.

`make server-test` must cover these; `make all-check` remains green.

## Rollout Steps

1. Land `internal/eventlog` and its query package with tests; no callsite changes yet.
2. Wire the logger in `cmd/server/main.go` behind a feature flag (`TERMINALS_LOG_ENABLED=true` default) and route stdlib `log` through it. Ship with both the JSONL file and stderr mirror active.
3. Migrate `log.Printf` callsites file by file; each PR maps a single component to its typed events.
4. Add the domain-event seams in scenario, observation, IO, recording, telephony, appruntime, admin.
5. Land `term logs` CLI.
6. Land admin HTTP log views.
7. Flip `TERMINALS_LOG_STDERR` default to `false` for non-dev deployments; update docs.

Each step is independently mergeable and keeps the server runnable.

## Related Plans

- [observation-plane.md](observation-plane.md) — Observation records feed directly into `observation.emitted` events.
- [scenario-engine.md](scenario-engine.md) — Scenario activations and triggers are primary event sources.
- [architecture-server.md](architecture-server.md) — Server component layout the `component` field mirrors.
- [protocol.md](protocol.md) — Transport layer whose interceptors anchor trace ids.
- [application-runtime.md](application-runtime.md) — App runtime emits `appruntime.*` events.
- [phase-6-monitoring.md](../phases/phase-6-monitoring.md) — Later phase that will build on this foundation for external observability.
