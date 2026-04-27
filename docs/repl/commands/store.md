# store

Typed key-value state with namespace scoping.

## Commands

- `store ns ls [--json]`
- `store put <namespace> <key> <value> [--ttl <duration>] [--json]`
- `store get <namespace> <key> [--json]`
- `store ls <namespace> [--json]`
- `store del <namespace> <key> [--json]`
- `store watch <namespace> [--prefix <p>] [--json]`
- `store bind <namespace> <key> --to <device>:<scenario> [--json]`

`--ttl` uses Go duration format (examples: `30s`, `5m`, `1h`) and expires
records lazily on read/list.
