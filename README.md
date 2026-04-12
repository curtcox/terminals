# Terminals

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
```

## Architecture Rule

Add behavior on the server, not the client. The client remains a generic terminal.

