# handlers

Runtime input and event routing rules for REPL-authored scenarios.

## Commands

- `handlers ls [--json]`
- `handlers on <selector> <action> --run <command> [--json]`
- `handlers on <selector> <action> --emit <kind> <name> [payload] [--json]`
- `handlers off <handler-id> [--json]`

`<selector>` is a simple match string (for example `scenario=chat` or
`device=d1 component=alert_ack`). `--run` executes a REPL command when matched;
`--emit` records a typed bus emission target.
