# scripts

Script inspection helpers for reproducible REPL workflows.

## Commands

- `scripts dry-run <path> [--json]`

`dry-run` reads a script file on the server, strips blank/comment lines, and
returns a deterministic command summary without executing commands.
