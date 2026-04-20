# MCP Setup

The Terminals MCP adapter exposes the REPL command registry as MCP tools. See [agent-delegation.md](../../../plans/agent-delegation.md) for the design; this page tells you how to turn it on.

The adapter is in-process in the Go server. Start the server (`make run-server`) and the adapter is running. Nothing else to install.

## Endpoints

- **Streamable HTTP:** `http://<host>:<port>/mcp`
  - Default port is `50053` (`TERMINALS_ADMIN_HTTP_PORT`).
  - Default host is `0.0.0.0` (`TERMINALS_ADMIN_HTTP_HOST`).
  - Use this when Claude Code / Codex runs on a different machine than the server.
- **stdio:** `terminal_server mcp-stdio`
  - The subprocess attaches stdio to the already-running server's MCP endpoint. It does not start a second server.
  - Use this when Claude Code / Codex runs on the same machine as the server.

Discovery: the HTTP endpoint is advertised via mDNS alongside the REPL endpoint (see [discovery.md](../../../plans/discovery.md)).

## Transport choice

| You run Claude Code / Codex… | Pick |
|---|---|
| On the same Mac as the server | stdio |
| On a laptop, with the server on a Mac mini on the LAN | Streamable HTTP |

Both transports go through the same adapter and expose the same tool catalog.

## Approval model

- `read_only` — runs immediately.
- `operational` — runs immediately, but each session has a concurrent-stream cap and stream-TTL budget.
- `mutating` — requires out-of-band user approval per call.
  - Primary: MCP elicitation. The client pops a confirmation dialog carrying the rendered command.
  - Fallback: two-call `confirmation_id` protocol, for clients without elicitation support.
    - Streamable HTTP carrier: `Mcp-Confirmation-Id` header.
    - stdio carrier: `_meta.terminals_confirmation_id` field.
  - Clients that support neither are pinned to `mutating_unavailable` and cannot call mutating tools.

No tool schema contains a `confirm` or `force` argument. See [approval-contract.md](approval-contract.md) for the full contract.

## Auth

None on the LAN. Trusted-network assumption, same as the rest of the system. Do not expose the adapter beyond a trusted network.

## Verifying the endpoint

From the REPL (or any shell):

```bash
curl -sS http://<host>:50053/mcp -X POST \
  -H 'content-type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"curl","version":"0"}}}'
```

A JSON response with the adapter's `serverInfo` confirms the endpoint is live.

## Next

- [claude-code-setup.md](claude-code-setup.md)
- [codex-setup.md](codex-setup.md)
- [tool-catalog.md](tool-catalog.md)
- [approval-contract.md](approval-contract.md)
- [troubleshooting.md](troubleshooting.md)
