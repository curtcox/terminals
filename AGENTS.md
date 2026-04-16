# Terminals Agent Guide

This file mirrors `CLAUDE.md` for Codex compatibility.

## Project Overview

Terminals is a thin-client system:

- Flutter clients are generic terminals
- Go server owns scenarios, routing, and orchestration
- Protobuf is the canonical contract for client/server IO

## Repo Layout

- `terminal_server/`: Go server
- `terminal_client/`: Flutter client
- `api/terminals/`: protobuf definitions

## Core Rules

1. Never add scenario-specific behavior to the client.
2. Define all client/server messages in protobuf, not ad-hoc JSON.
3. Keep AI providers behind interfaces in server code.
4. Build UIs from shared server-driven primitives.

## Build and Check Commands

```bash
make server-build
make server-test
make server-lint
make client-build
make client-test
make client-lint
make proto-lint
make proto-generate
make all-check
```

## Local Development

1. Start server: `make run-server`
2. Start web client: `make run-client-web`
3. Connect via discovery/manual connect screen

## Source of Truth

Architectural details live in `masterplan.md`.
