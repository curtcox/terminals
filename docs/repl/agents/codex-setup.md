# Codex Setup

Configure Codex to talk to the Terminals MCP adapter. Read [mcp-setup.md](mcp-setup.md) first for endpoint details.

## stdio (same machine as server)

In `~/.codex/config.toml`:

```toml
[mcp_servers.terminals]
command = "/absolute/path/to/terminal_server"
args = ["mcp-stdio"]
```

The `mcp-stdio` subprocess requires a server to be already running; it proxies stdio into the running adapter.

## Streamable HTTP (laptop + remote server)

In `~/.codex/config.toml`:

```toml
[mcp_servers.terminals]
url = "http://<server-host>:50053/mcp"
```

Replace `<server-host>` with the server's hostname (mDNS `<MDNSName>.local`) or IP. Adjust port if you changed `TERMINALS_ADMIN_HTTP_PORT`.

If your Codex build lacks HTTP MCP support, fall back to stdio by running a local `terminal_server mcp-stdio` bridge that connects to the remote server — not currently shipped; file a request if you need it.

## Verify

1. Restart Codex and open a session.
2. Ask the agent to call `repl_describe` — it should return registry metadata.
3. Ask for a `read_only` action (e.g. "list devices"). It should execute without approval.
4. Ask for a `mutating` action (e.g. "stop activation act_X"). You should see an approval prompt before the tool runs.

If the mutating step does not prompt, check [troubleshooting.md](troubleshooting.md) — the client likely did not negotiate elicitation or fallback, and the session is pinned to `mutating_unavailable`.

## Recommended follow-up

Install the `terminals-mcp` skill so the agent knows how to use the catalog effectively. See [../../../.claude/skills/terminals-mcp/SKILL.md](../../../.claude/skills/terminals-mcp/SKILL.md).
