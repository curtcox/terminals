# Agent Delegation Verification Issues

Date verified: 2026-04-20
Verifier: Claude Code session
Source plan: [plans/agent-delegation.md](agent-delegation.md)

## Outstanding Issues

1. Mutating tool calls are unreachable from stock Claude Code / Codex clients over stdio.
   - Symptom: any `mutating` tool call (e.g. `sessions_terminate`, `app_reload`, `activations_stop`) returns `status=error`, `error_code=unsupported_client`, `"mutating tools require elicitation or confirmation_id fallback support"`. The session is pinned to `mutating_capability=mutating_unavailable` at `initialize`, even when the client advertises `capabilities.elicitation`.
   - Root cause: `runMCPStdio` / `proxyMCPStdio` in [terminal_server/cmd/server/main.go:391](../terminal_server/cmd/server/main.go:391) implement the `mcp-stdio` subcommand as a unary stdin→`POST /mcp`→stdout proxy. There is no channel for the server to push a server-initiated `elicitation/create` request back through the subprocess. The in-process `Server.ServeStdio` implementation at [terminal_server/internal/mcpadapter/server.go:165](../terminal_server/internal/mcpadapter/server.go:165) does support bidirectional stdio (and is exercised by `server_test.go`), but it is never wired to the `mcp-stdio` subprocess.
   - Compounding: the HTTP transport path in `parseClientCapabilities` at [terminal_server/internal/mcpadapter/server.go:589](../terminal_server/internal/mcpadapter/server.go:589) forcibly sets `supportsElicitation = false`, so even a direct HTTP client that advertises elicitation is downgraded.
   - Consequence: the "stop activation → MCP elicitation prompt" verification step in [docs/repl/agents/claude-code-setup.md:55](../docs/repl/agents/claude-code-setup.md:55) cannot succeed with a stock client, and the plan's headline user-experience flow (`activations_stop act_51` with approval) cannot run end-to-end.
   - Fallback still works only with a non-standard capability advertisement: clients that send `capabilities.terminals_fallback_confirmation: true` receive a proper `confirmation_required` response carrying a `confirmation_id`. Neither Claude Code nor Codex advertises this Terminals-specific capability, so the fallback is effectively unreachable from real clients.
   - Suggested fix direction: have `mcp-stdio` host a direct in-process MCP connection against the shared adapter (e.g. by calling `ServeStdio` on a per-subprocess connection bound to the running adapter), or upgrade the subprocess↔server link to a bidirectional transport (SSE-backed Streamable HTTP, WebSocket, or gRPC bidi stream) so the server can originate `elicitation/create`. Either path lets `mutating_via_elicitation` apply for Claude Code / Codex as written in the plan.

## Verified Working

- `claude mcp add terminals -- .bin/terminal_server mcp-stdio` registers and `claude mcp list` reports `✓ Connected`.
- Tool catalog is registry-derived (~30 tools) with `read_only | operational | mutating` classification in every description and `discouraged_for_agents` set on the `ai_*` group.
- No tool schema exposes `confirm` or `force`.
- `repl_complete` and `repl_describe` (including the no-argument registry summary) are callable.
- Read-only calls dispatch and return structured results (`devices_ls` returns the `ID  ZONE  CAPS  STATE` header).
- The server `initialize` response advertises `registry_version` and assigns a per-session `session_id`.
