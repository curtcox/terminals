# Terminals

[![Server CI](https://github.com/curtcox/terminals/actions/workflows/server-ci.yml/badge.svg)](https://github.com/curtcox/terminals/actions/workflows/server-ci.yml)
[![Client CI](https://github.com/curtcox/terminals/actions/workflows/client-ci.yml/badge.svg)](https://github.com/curtcox/terminals/actions/workflows/client-ci.yml)
[![Proto CI](https://github.com/curtcox/terminals/actions/workflows/proto-ci.yml/badge.svg)](https://github.com/curtcox/terminals/actions/workflows/proto-ci.yml)

Terminals is a client/server home system where thin clients act as generic IO surfaces and the server owns all behavior.

## Repository Layout

- `terminal_server/`: Go server (control plane, routing, scenarios)
- `terminal_client/`: Flutter client (discovery, connection, rendering, IO bridge)
- `api/`: protobuf contracts and codegen configuration

## Prerequisites

- Go `1.26+`
- Flutter SDK (for client build/test)
- Buf CLI (for proto lint/codegen)
- `golangci-lint`

## Quick Start

```bash
make server-build
make server-test
```

When Flutter and Buf are installed:

```bash
make client-build
make proto-lint
make all-check
make usecase-validate USECASE=C1
```

Run the server locally:

```bash
make run-server
```

Run server + local client with bootstrap checks (works from a fresh clone or existing checkout):

```bash
make run-local
```

The script defaults to a web client. Use a macOS client instead:

```bash
./scripts/run-local.sh --client macos
```

Validate the local launcher behavior (bootstrap, port selection, and startup checks):

```bash
make run-local-test
```

Run an opt-in smoke test against real local tools (Go + Flutter):

```bash
make run-local-smoke-test
```

## Architecture Rule

Add behavior on the server, not the client. The client remains a generic terminal.

## Agent Delegation (Claude Code / Codex)

The server exposes the REPL command registry to Claude Code and Codex over MCP so an LLM agent can drive the system with exactly the access a REPL user has — no more, no less. See [`plans/agent-delegation.md`](plans/agent-delegation.md) for the design, and the setup guides to enable it:

- [`docs/repl/agents/mcp-setup.md`](docs/repl/agents/mcp-setup.md) — endpoints, transports, auth
- [`docs/repl/agents/claude-code-setup.md`](docs/repl/agents/claude-code-setup.md)
- [`docs/repl/agents/codex-setup.md`](docs/repl/agents/codex-setup.md)
- [`docs/repl/agents/approval-contract.md`](docs/repl/agents/approval-contract.md) — mutating calls require out-of-band user approval
- [`docs/repl/agents/troubleshooting.md`](docs/repl/agents/troubleshooting.md)

Connected agents benefit from the [`terminals-mcp`](.claude/skills/terminals-mcp/SKILL.md) skill, which teaches effective use of the tool catalog.

## Event Logging

The server writes structured JSONL events to `TERMINALS_LOG_DIR` (default `logs/`) with size-based rotation (`TERMINALS_LOG_MAX_BYTES`, `TERMINALS_LOG_MAX_ARCHIVES`). Use `term logs ...` for local querying and `/admin/logs` for browser-based filtering.

Event naming reference: [`docs/event-taxonomy.md`](/Users/curt/me/terminals/docs/event-taxonomy.md).

## Photo Frame Configuration

The photo-frame scenario is configured entirely on the server.

Environment variables:

- `TERMINALS_PHOTO_FRAME_DIR`: Directory containing photo assets (`.jpg`, `.jpeg`, `.png`, `.webp`, `.gif`).
- `TERMINALS_PHOTO_FRAME_INTERVAL_SECONDS`: Slide rotation interval (default `12`).
- `TERMINALS_PHOTO_FRAME_HTTP_HOST`: Bind host for the built-in asset server (default `0.0.0.0`).
- `TERMINALS_PHOTO_FRAME_HTTP_PORT`: Bind port for the built-in asset server (default `50052`).
- `TERMINALS_PHOTO_FRAME_PUBLIC_BASE_URL`: Optional externally hosted base URL. When set, the built-in photo asset server is not started.

Metadata contract:

- On connect, the server sends `RegisterAck.metadata["photo_frame_asset_base_url"]`.
- Clients treat this value as the canonical base URL for photo-frame slide assets.
- If `TERMINALS_PHOTO_FRAME_PUBLIC_BASE_URL` is set, that exact value (trailing slash trimmed) is published in metadata.
- Otherwise the server publishes `http://<mdns_name>.local:<photo_frame_http_port>/photo-frame`.

Example env config is in [`terminal_server/configs/server.env.example`](/Users/curtcox/me/terminals/terminal_server/configs/server.env.example).
