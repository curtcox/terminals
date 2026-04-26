# Server — Build & Run

The Go server lives in `terminal_server/`.

## Architecture Overview

The server is the system orchestrator. Scenario logic, routing, and state live
in Go on the server; clients remain generic terminals.

### Module layout

```text
terminal_server/
|- cmd/server/                    # Server entrypoint and process wiring
|- internal/discovery/            # mDNS discovery advertisement
|- internal/transport/            # gRPC/WebSocket/TCP/HTTP control carriers
|- internal/device/               # Device registry + lifecycle state
|- internal/placement/            # Semantic target resolution
|- internal/io/                   # Media plans, claims, routing
|- internal/intent/               # Typed intent/event ingress and dispatch
|- internal/scenario/             # Scenario definitions and activation engine
|- internal/ai/                   # Provider interfaces and adapters
|- internal/telephony/            # SIP bridge
|- internal/ui/                   # Server-driven UI descriptor validation
|- internal/storage/              # Config, scheduler, and persisted state
|- configs/                       # Runtime configuration defaults/examples
|- gen/                           # Generated server-side bindings
```

### Responsibilities

- Device manager: tracks connected terminals and runtime state.
- Placement engine: resolves semantic scopes (zones/roles) into concrete
  device targets.
- IO router + claim manager: compiles media plans, applies routing, and
  arbitrates resource preemption.
- Intent/event bus: normalizes triggers from voice, UI actions, schedules,
  analyzers, and automation.
- Scenario engine: matches triggers, starts/supervises activations, and
  coordinates claim lifecycle.
- AI backends: keep STT/TTS/LLM/vision/classification behind interfaces.
- Storage: persists scheduler/config/runtime records needed for resilient
  operation and recovery.

### Contract boundaries

- Client/server message contracts are protobuf under `api/terminals/`.
- Scenario-specific behavior belongs in server packages, not the Flutter
  client.
- UI is server-driven using shared primitives and validated descriptors.

## Prerequisites

| Tool | Minimum version | Install |
|------|-----------------|---------|
| Go | 1.26+ | <https://go.dev/dl/> |
| golangci-lint | latest | `brew install golangci-lint` or see <https://golangci-lint.run/> |
| buf | v1 | `brew install bufbuild/buf/buf` (only needed for proto work) |

## Configuration

The server reads configuration from environment variables. Defaults are tuned for local development — no configuration is required to get started.

Copy the example env file and adjust as needed:

```bash
cp terminal_server/configs/server.env.example .env
source .env
```

### Core settings

| Variable | Default | Description |
|----------|---------|-------------|
| `TERMINALS_GRPC_HOST` | `0.0.0.0` | gRPC listen address |
| `TERMINALS_GRPC_PORT` | `50051` | gRPC listen port |
| `TERMINALS_CONTROL_WS_HOST` | `0.0.0.0` | WebSocket control listener address |
| `TERMINALS_CONTROL_WS_PORT` | `50054` | WebSocket control listener port |
| `TERMINALS_CONTROL_WS_ALLOWED_ORIGINS` | *(empty)* | Comma-separated explicit origins allowed for cross-origin browser websocket upgrades. Wildcard `*` is rejected; same-origin and loopback-origin development flows are allowed without this setting. |
| `TERMINALS_ADMIN_HTTP_HOST` | `0.0.0.0` | Admin dashboard listen address |
| `TERMINALS_ADMIN_HTTP_PORT` | `50053` | Admin dashboard listen port |
| `TERMINALS_MDNS_SERVICE` | `_terminals._tcp.local.` | mDNS service type for discovery |
| `TERMINALS_MDNS_NAME` | `HomeServer` | Server identity advertised over mDNS |
| `TERMINALS_VERSION` | `1` | Protocol version |

### Runtime tuning

| Variable | Default | Description |
|----------|---------|-------------|
| `TERMINALS_HEARTBEAT_TIMEOUT_SECONDS` | `120` | Consider a device dead after this many seconds without a heartbeat |
| `TERMINALS_LIVENESS_RECONCILE_INTERVAL_SECONDS` | `30` | How often to sweep for dead devices |
| `TERMINALS_DUE_TIMER_PROCESS_INTERVAL_SECONDS` | `5` | How often to fire scheduled timers |
| `TERMINALS_WAKE_WORD_PREFIXES` | `assistant,hey terminal` | Comma-separated wake-word prefixes |

### Photo frame

| Variable | Default | Description |
|----------|---------|-------------|
| `TERMINALS_PHOTO_FRAME_DIR` | *(empty — disabled)* | Directory containing photo assets |
| `TERMINALS_PHOTO_FRAME_INTERVAL_SECONDS` | `12` | Rotation interval |
| `TERMINALS_PHOTO_FRAME_HTTP_HOST` | `0.0.0.0` | Photo asset HTTP server address |
| `TERMINALS_PHOTO_FRAME_HTTP_PORT` | `50052` | Photo asset HTTP server port |
| `TERMINALS_PHOTO_FRAME_PUBLIC_BASE_URL` | *(empty)* | Override the photo base URL (e.g. CDN) |

### Storage

| Variable | Default | Description |
|----------|---------|-------------|
| `TERMINALS_RECORDING_DIR` | `recordings` | Where audio recordings are stored on disk |

### SIP telephony (optional)

| Variable | Default | Description |
|----------|---------|-------------|
| `TERMINALS_SIP_ENABLED` | `false` | Enable the SIP bridge |
| `TERMINALS_SIP_SERVER_URI` | — | SIP server URI (required when enabled) |
| `TERMINALS_SIP_USERNAME` | — | SIP account username (required when enabled) |
| `TERMINALS_SIP_DISPLAY_NAME` | — | Caller display name |
| `TERMINALS_SIP_PASSWORD` | — | SIP account password |

## Build

```bash
# From the repo root:
make server-build

# Or directly:
cd terminal_server && go build ./...
```

## Run

```bash
# From the repo root:
make run-server

# Or directly:
cd terminal_server && go run ./cmd/server
```

On startup the server will:

1. Listen for gRPC control connections on port 50051 (default).
2. Listen for WebSocket control connections on port 50054 (default).
3. Listen for TCP control connections on port 50055 (default).
4. Listen for HTTP fallback control connections on port 50056 (default).
5. Advertise service and carrier metadata via mDNS so clients can auto-discover it.
6. Start the admin dashboard at `http://localhost:50053/admin`.
7. Optionally start the photo-frame asset server on port 50052.

## Control Transport Carriers

The control plane is carrier-neutral and supports the same logical session
semantics across these listeners:

- gRPC: preferred transport when HTTP/2 and long-lived streams are available.
- WebSocket: browser-friendly fallback using binary protobuf envelopes.
- TCP: length-framed socket fallback for constrained or non-HTTP environments.
- HTTP: correctness-first fallback for restrictive request/response-only paths.

mDNS TXT metadata advertises per-carrier endpoints (`grpc`, `ws`, `tcp`,
`http`) and an ordered `priority` list. Clients use this advertised order,
filtered by runtime support, when selecting or failing over carriers.

## Test

```bash
make server-test          # run all tests
make server-coverage      # run tests with coverage report
```

Some server tests intentionally exercise real local networking. In restricted
agent sandboxes, those tests can fail before server code runs because the
sandbox blocks listener creation or host-interface inspection. Typical errors
include:

- `listen tcp 127.0.0.1:0: bind: operation not permitted`
- `httptest: failed to listen on a port: listen tcp6 [::1]:0: bind: operation
  not permitted`
- mDNS setup errors when host IP addresses cannot be determined

When that happens, rerun the affected server tests in an environment that
allows loopback listeners and network-interface reads. Keep the Go build cache
inside the workspace or `/tmp` if the default user cache is also restricted:

```bash
cd terminal_server
GOCACHE=/tmp/terminals-go-build go test ./...
```

The long-term remediation plan is tracked in
[`plans/server-test-sandbox-network.md`](../plans/features/server-test-sandbox-network.md).

## Scheduler

The in-memory scheduler stores structured records in
`terminal_server/internal/storage.ScheduleRecord`. A record includes the stable
key, kind, subject, device ID, trigger time, optional string payload, and
creation timestamp.

Legacy key-only callers remain supported through `Schedule(ctx, key, unixMS)`
and `Due(unixMS)`. Those calls are stored internally as records, and known key
prefixes such as `timer:` infer the schedule kind. New server code should prefer
`ScheduleRecord` and `DueRecords` when it needs typed metadata such as a timer
label or duration.

Timer due processing prefers structured records and falls back to parsing
legacy timer keys, so old scheduled entries still fire and are removed.

## Scenario Operations

Scenarios may opt in to the result-returning path by implementing
`ResultScenario`. Instead of directly performing every side effect in `Start`,
they return a `ScenarioResult` containing typed operations and emitted
triggers. The engine validates all operations before committing any side
effects, then executes the operations in order.

Currently executable operation kinds are `ui.set`, `ui.patch`, `ui.clear`,
`scheduler.after`, `scheduler.cancel`, `broadcast.notify`, `ai.tts`, and
`bus.emit`. The shared model also defines transition and flow operation names
for the TAL/TAR contract, but those are rejected until the corresponding
executors exist.

`TimerReminderScenario` is the first built-in scenario on this path. It returns
UI, scheduler, and confirmation operations; due-timer processing applies tick
and expiry side effects from structured scheduler records. Legacy scenarios
continue to use their existing `Start` methods.

## Text Terminal Runtime

The server provides a generic text-terminal runtime used by terminal scenarios.
Client code remains scenario-agnostic: it renders server-provided UI descriptors
and forwards input events.

- Terminal sessions are activation-scoped and keyed by device/session identity.
- Terminal UI is delivered as server-driven descriptors with output patches,
  refresh actions, and enter/restore transitions.
- Input handling normalizes keyboard text and routes submit/interactive actions
  into the active terminal session.
- Heartbeats can flush/coalesce terminal output updates to avoid stale UI and
  reduce unnecessary patch traffic.

Automated validation coverage for this runtime is mapped to use case `P1`:

```bash
make usecase-validate USECASE=P1
```

Primary transport evidence lives in generated and wire integration tests under
`terminal_server/internal/transport`:

- `TestGeneratedSessionTerminalTransitions`
- `TestWireSessionTerminalTransitions`

## Server-Driven UI Contract

Server-driven UI is implemented as a transport-level contract between server
and client.

- `SetUI` replaces the full UI tree for a target terminal.
- `UpdateUI` patches a component subtree by ID.
- `TransitionUI` communicates transition hints and duration for client-side
  presentation.

The canonical protobuf messages are defined in `api/terminals/ui/v1/ui.proto`
and carried on the control stream in `ConnectResponse`.

The widget contract is intentionally closed and validated. Server descriptors
must use one of the supported primitives in `Node.widget`:

- Layout: `stack`, `row`, `grid`, `scroll`, `padding`, `center`, `expand`
- Content: `text`, `image`, `video_surface`, `audio_visualizer`, `canvas`
- Input: `text_input`, `button`, `slider`, `toggle`, `dropdown`,
  `gesture_area`
- Overlay/system: `overlay`, `progress`, `fullscreen`, `keep_awake`,
  `brightness`

Validation and adaptation paths live in `terminal_server/internal/ui` and
`terminal_server/internal/transport`, with client rendering and patch handling
in `terminal_client/lib/main.dart`.

Representative test coverage includes:

- Server descriptor and transport tests under
  `terminal_server/internal/ui/descriptor_test.go` and
  `terminal_server/internal/transport/*_test.go`
- Client renderer tests in `terminal_client/test/widget_test.dart`
  (including SetUI, UpdateUI, and TransitionUI handling)

## Monitoring Support Tiers

Device capability declarations include monitoring tier operators under
`edge.operators`.

Clients declare these operators only when they have explicit platform-backed
evidence. Unknown tiers are omitted rather than inferred from platform type.
Capability flattening also omits default-false/default-empty sensor,
connectivity, and edge list fields (including `edge.runtimes` and
`edge.operators`) unless values are explicitly true or non-empty.

- `monitor.tier.foreground_only`: the client can run monitor workloads only
  while active in the foreground.
- `monitor.tier.background_capable`: the client can sustain monitor workloads
  when backgrounded.

Placement treats `background_monitor` roles as requiring
`monitor.background_capable=true` after capability flattening. Clients that
declare only `monitor.tier.foreground_only` are filtered out from background
monitor assignments.

Placement `ExcludeBusy` filtering also consults active IO claims. A device is
treated as busy when it currently holds one or more active claims in the
shared claim manager, even if no explicit `liveness=busy` capability key is
present.

## Lint

```bash
make server-lint
```

## Protobuf

If you modify `.proto` files under `api/terminals/`:

```bash
make proto-generate   # regenerate Go + Dart stubs
make proto-lint        # lint proto files
make proto-breaking    # check for breaking changes vs main
```

Requires `buf` to be installed.
