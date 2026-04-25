---
title: "Agent Delegation Verification Issues"
kind: audit
status: open
owner: unowned
validation: none
last-reviewed: 2026-04-25
---

# Agent Delegation Verification Issues

Date verified: 2026-04-20
Verifier: Claude Code session
Source plan: [plans/agent-delegation.md](../features/agent-delegation.md)

## Resolved Issues

1. Mutating tool calls are reachable from stock Claude Code / Codex clients over `mcp-stdio`.
   - `proxyMCPStdio` now bridges fallback confirmation into a standard MCP `elicitation/create` prompt:
     1) initialize requests that advertise elicitation are augmented with `capabilities.terminals_fallback_confirmation=true` for the HTTP backend.
     2) when HTTP returns `confirmation_required`, the proxy emits `elicitation/create` to the stdio client and waits for approval.
     3) approved requests are replayed with `Mcp-Confirmation-Id`, returning the final tool result to the client.
   - `initialize` responses are normalized to report `mutating_capability=mutating_via_elicitation` when the client supports elicitation, matching the user-facing behavior.
   - Coverage added in `terminal_server/cmd/server/main_test.go` (`TestProxyMCPStdioBridgesFallbackConfirmationThroughElicitation`).

2. HTTP capability parsing no longer downgrades advertised elicitation support.
   - `parseClientCapabilities` now preserves explicit elicitation capability for HTTP clients.
   - Coverage updated in `terminal_server/internal/mcpadapter/server_test.go` (`TestParseClientCapabilitiesFailClosedFallback`).

## Verified Working

- `claude mcp add terminals -- .bin/terminal_server mcp-stdio` registers and `claude mcp list` reports `✓ Connected`.
- Tool catalog is registry-derived (~30 tools) with `read_only | operational | mutating` classification in every description and `discouraged_for_agents` set on the `ai_*` group.
- No tool schema exposes `confirm` or `force`.
- `repl_complete` and `repl_describe` (including the no-argument registry summary) are callable.
- Read-only calls dispatch and return structured results (`devices_ls` returns the `ID  ZONE  CAPS  STATE` header).
- The server `initialize` response advertises `registry_version` and assigns a per-session `session_id`.
