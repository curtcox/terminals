# terminal_server

Go server — the system orchestrator. All scenario logic, routing, and state live here; clients are generic terminals.

## Quick commands

```bash
# From repo root
make server-build
make server-test
make server-lint
make run-server
```

Single-package test (fast):
```bash
go test ./internal/scenario/... -run '^TestFoo$' -count=1 -v
```

## Module layout

```
cmd/
  server/         # entrypoint and process wiring
  term/           # local CLI for querying logs, REPL, etc.
  proto-contract-generate/ # codegen entrypoint
internal/
  transport/      # gRPC / WebSocket / TCP / HTTP control carriers
  device/         # device registry and lifecycle state
  scenario/       # scenario definitions, activation engine, trigger bus
  placement/      # semantic target resolution (zone/role → device IDs)
  io/             # media plans, resource claims, routing
  world/          # calibrated world model and observation history
  discovery/      # mDNS advertisement
  ai/             # provider interfaces (STT/TTS/LLM/vision/classification)
  repl/           # control-plane REPL implementation
  replai/         # sticky AI model selection per REPL session
  replsession/    # REPL session lifecycle and attachment state
  admin/          # web dashboard and JSON admin APIs
  eventlog/       # structured JSONL logging
  apppackage/     # .tap application archive build and validation
  appruntime/     # app definition and migration runtime
  audio/          # device-scoped audio pub/sub hub
  capability/     # typed in-memory REPL capability services
  chat/           # in-memory chat room
  config/         # runtime configuration loading
  contracttest/   # shared test helpers for protocol contract tests
  diagnostics/    # bug-report storage and retrieval
  mcpadapter/     # MCP tool-call → REPL capability bridge
  observation/    # recent observation storage for sensing scenarios
  protocolcontract/ # wire-level protocol contract test fixtures
  recording/      # stream recording lifecycle hooks and disk management
  storage/        # in-memory key/value persistence primitives
  telephony/      # SIP bridge abstractions
  terminal/       # PTY-backed interactive terminal sessions
  trust/          # distribution policy vetting
  ui/             # server-driven UI descriptor validation and broadcast
  usecasevalidation/ # in-process test harness for use-case validation
apps/
  kitchen_timer/  # example built-in app (timer scenario)
configs/          # server.env.example with all env-var defaults
gen/              # generated Go protobuf bindings (do not edit — see gen/README.md)
```

## Detailed docs

See [docs/server.md](../docs/server.md) for full configuration reference, transport details, scenario engine internals, and env-var table.

## Key rules

- Scenario-specific behavior lives in server packages, not in the Flutter client.
- All client/server messages are protobuf (defined in `api/terminals/`).
- AI backends stay behind interfaces in `internal/ai`.
