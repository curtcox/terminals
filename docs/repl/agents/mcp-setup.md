# MCP Setup

The Terminals MCP adapter exposes the same command surface as the REPL command registry.

## Stdio

Use the server binary with the `mcp-stdio` subcommand in your desktop client MCP config.

## Streamable HTTP

Point your desktop client to:

`http://<server-host>:<port>/mcp`

## Approval

- `mutating` commands require out-of-band approval.
- `read_only` and `operational` commands execute directly.
- Tool schemas do not include `confirm` or `force` fields.
