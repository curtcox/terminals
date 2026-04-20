# Agent Delegation Verification Issues

Date verified: 2026-04-20  
Verifier: Codex agent session via `mcp__terminals__` tools  
Source plan: [plans/agent-delegation.md](/Users/curt/me/terminals/plans/agent-delegation.md)

This file now tracks only currently outstanding issues.

## Status

Outstanding verification issues remain.

## Outstanding Issues

1. Fail-closed capability negotiation is not fully enforced as designed.
- In `terminal_server/internal/mcpadapter/server.go`, `parseClientCapabilities` defaults `supportsFallback` to true even when the client does not advertise fallback support, which makes `mutating_unavailable` effectively unreachable for server-managed sessions.
- This conflicts with the plan requirement that clients lacking both approval carriers are pinned to `mutating_unavailable`.

2. HTTP transport behavior does not meet the streamable HTTP design intent.
- `ServeHTTP` in `terminal_server/internal/mcpadapter/server.go` is implemented as a request/response JSON-RPC endpoint, while streaming chunk notifications are only emitted in stdio mode.
- The plan and docs call for streamable HTTP transport support as a first-class path.

3. The planned typed REPL streaming service contract is still not implemented as a concrete service API.
- `docs/repl/api/repl-service.md` still lists `EvalCommandStream` as planned, and there is no corresponding typed REPL service implementation surfaced in server APIs.
- Current streaming is achieved through adapter-local execution (`repl.ExecuteCommandStream`) rather than the planned typed service RPC surface.

4. Operational and approval policy knobs documented for runtime configuration are not wired through server config.
- Adapter thresholds and limits are currently defaulted in `terminal_server/internal/mcpadapter/adapter.go`.
- `terminal_server/internal/config/config.go` does not define `agent.operational.*` or approval-latency configuration fields referenced by docs.
