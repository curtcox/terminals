# Claude Code Setup

Configure Claude Code to talk to the Terminals MCP adapter. Read [mcp-setup.md](mcp-setup.md) first for endpoint details.

## stdio (same machine as server)

```bash
claude mcp add terminals -- /absolute/path/to/terminal_server mcp-stdio
```

Or, in `~/.claude.json` (user scope) or `.mcp.json` (project scope):

```json
{
  "mcpServers": {
    "terminals": {
      "command": "/absolute/path/to/terminal_server",
      "args": ["mcp-stdio"]
    }
  }
}
```

The `mcp-stdio` subprocess requires a server to be already running; it proxies stdio into the running adapter. If no server is up, it exits with a clear error.

## Streamable HTTP (laptop + remote server)

```bash
claude mcp add --transport http terminals http://<server-host>:50053/mcp
```

Or in config:

```json
{
  "mcpServers": {
    "terminals": {
      "type": "http",
      "url": "http://<server-host>:50053/mcp"
    }
  }
}
```

Replace `<server-host>` with the server's hostname (mDNS `<MDNSName>.local`) or IP. Adjust port if you changed `TERMINALS_ADMIN_HTTP_PORT`.

## Verify

In a Claude Code session:

1. `/mcp` — confirm `terminals` is listed as connected.
2. Ask the agent to call `repl_describe` with no arguments — it should return the registry summary.
3. Ask the agent to call `repl_complete` for a command prefix (e.g. `devices `) — it should return completions.
4. Ask for a `read_only` action (e.g. "list devices"). It should execute without any approval prompt.
5. Ask for a `mutating` action (e.g. "stop activation act_X"). You should see an MCP elicitation prompt before the tool runs.

If the mutating step does not prompt, check [troubleshooting.md](troubleshooting.md) — most likely the client did not negotiate elicitation or fallback.

## Recommended follow-up

Install the `terminals-mcp` skill so the agent knows how to use the catalog effectively. See [../../../.claude/skills/terminals-mcp/SKILL.md](../../../.claude/skills/terminals-mcp/SKILL.md).
