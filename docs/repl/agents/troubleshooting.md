# Troubleshooting

## `unsupported_client` on a mutating tool call

Your client negotiated neither elicitation nor the fallback carrier at connect time; the session is pinned to `mutating_unavailable`. Fixes:

- Update the MCP client to a version that supports elicitation (spec 2025-06-18 or later).
- If you're using a Streamable HTTP client behind a proxy or middleware, verify the `Mcp-Confirmation-Id` request header isn't stripped.
- If you're using a typed MCP client, verify `_meta` on `tools/call` requests isn't dropped on the way out.

Check `sessions show <id>` — it shows the negotiated capability state.

## Mutating tool ran with no visible prompt

Either:

- The desktop app's own confirmation UX is answering the adapter's elicitation without surfacing it. The mutation is audited server-side regardless.
- The approval returned faster than `agent.approval.min_human_latency_ms` (default 500 ms), which emits an `unsafe_confirmation_protocol` audit event. Query with `logs query 'kind == "unsafe_confirmation_protocol"'`.

Humans virtually never approve in under 500 ms. If this fires regularly from a given client, treat it as a bypass and investigate that client's behavior.

## Tool missing after server update

Reconnect your MCP client so it re-reads the catalog. The adapter advertises its version and the registry version at connect; desktop clients refresh on change.

## `mcp-stdio` exits immediately

The `mcp-stdio` subcommand requires a server to already be running. Start the server with `make run-server` first, then relaunch your desktop client so the subprocess starts fresh.

## Streaming tool never returns

Streaming tools (`logs tail`, `observe tail`, `app logs -f`) don't return — they emit incrementally until cancelled. Use the client's MCP cancel operation. The adapter maps cancel to the REPL's cancellation path.

Also: sessions have a concurrent-stream cap and a stream-TTL budget. If you've hit either, new stream starts return a structured rate-limit error.

## Docs output is hard to parse

`docs open` and `docs search` called via MCP return plain Markdown, not paged terminal output. If you're seeing pager control codes, you're probably reaching the REPL directly rather than through the MCP adapter.

## MCP-origin session keeps accumulating

Client disconnects **detach**, they don't terminate. Terminate explicitly with `sessions terminate <id>`, or configure an idle timeout at the server.

## Where to look in the logs

- Every MCP call is a structured-logged REPL session event: `logs query 'session_origin == "mcp"'`.
- Elicitation outcomes (approved / rejected / timed-out) are fields on the log record.
- `sessions ls` shows the MCP-origin sessions.
- `sessions show <id>` shows capability state, agent-self-reported identity, and recent activity.
