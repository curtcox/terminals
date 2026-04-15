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
