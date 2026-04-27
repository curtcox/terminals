# scripts

Script inspection helpers for reproducible REPL workflows.

## Commands

- `scripts dry-run <path> [--json]`
- `scripts run <path> [--json]`

`dry-run` reads a script file on the server, strips blank/comment lines, and
returns a deterministic command summary without executing commands.

`run` reads the same script format and executes it through the server-side
scripts runtime, returning deterministic execution counters (`command_count`,
`executed_count`, `failed_count`).
