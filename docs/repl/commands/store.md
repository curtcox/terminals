# store

Typed key-value state with namespace scoping.

## Commands

- `store put <namespace> <key> <value> [--ttl <duration>] [--json]`
- `store get <namespace> <key> [--json]`
- `store ls <namespace> [--json]`

`--ttl` uses Go duration format (examples: `30s`, `5m`, `1h`) and expires
records lazily on read/list.
