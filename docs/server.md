# Server — Build & Run

The Go server lives in `terminal_server/`.

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

1. Listen for gRPC connections on port 50051 (default).
2. Advertise itself via mDNS so clients can auto-discover it.
3. Start the admin dashboard at `http://localhost:50053/admin`.
4. Optionally start the photo-frame asset server on port 50052.

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

## Monitoring Support Tiers

Device capability declarations include monitoring tier operators under
`edge.operators`.

Clients declare these operators only when they have explicit platform-backed
evidence. Unknown tiers are omitted rather than inferred from platform type.

- `monitor.tier.foreground_only`: the client can run monitor workloads only
  while active in the foreground.
- `monitor.tier.background_capable`: the client can sustain monitor workloads
  when backgrounded.

Placement treats `background_monitor` roles as requiring
`monitor.background_capable=true` after capability flattening. Clients that
declare only `monitor.tier.foreground_only` are filtered out from background
monitor assignments.

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
