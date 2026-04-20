# Agent Delegation Verification Issues

Date verified: 2026-04-20  
Verifier: Codex agent session via `mcp__terminals__` tools  
Source plan: [plans/agent-delegation.md](/Users/curt/me/terminals/plans/agent-delegation.md)

This file now tracks only currently outstanding issues.

## Status

No outstanding verification issues remain.

## Resolved In This Pass

1. Fail-closed capability negotiation now matches plan intent.
- `parseClientCapabilities` no longer defaults fallback support to true.
- Sessions without elicitation and without fallback confirmation carrier now remain `mutating_unavailable`.

2. HTTP transport now supports streamable tool-call output.
- `ServeHTTP` supports `Accept: text/event-stream` for `tools/call`.
- Streaming chunk notifications are emitted over SSE, followed by a final JSON-RPC response event.

3. Typed REPL streaming service contract docs now describe shipped APIs.
- `docs/repl/api/repl-service.md` now documents the concrete typed API (`ExecuteCommand`, `ExecuteCommandStream`, `CommandSpecs`, `DescribeCommand`, `Complete`) implemented in `terminal_server/internal/repl/api.go`.

4. Operational and approval runtime knobs are now wired through config.
- Added `Config.Agent.Operational` and `Config.Agent.Approval` fields in `terminal_server/internal/config/config.go`.
- Added env wiring:
  - `TERMINALS_AGENT_OPERATIONAL_MAX_STREAMS`
  - `TERMINALS_AGENT_OPERATIONAL_STREAM_TTL_SECONDS`
  - `TERMINALS_AGENT_APPROVAL_MIN_HUMAN_LATENCY_MS`
  - `TERMINALS_AGENT_APPROVAL_CONFIRMATION_TTL_SECONDS`
- `terminal_server/cmd/server/main.go` now passes these values into `mcpadapter.Config`.
