# memory

Typed memory operations backed by indexed capability content.

## Commands

- `memory remember <scope> <text> [--json]`
- `memory recall <text> [--json]`
- `memory stream [scope] [--json]`

## Notes

- `memory remember` appends a durable memory entry under a named scope.
- `memory recall` matches text and scope names.
- `memory stream` lists memory entries in insertion order and optionally filters
	by scope or filter term.

## Examples

```text
memory remember kitchen buy milk
memory recall milk
memory stream
memory stream kitchen
```
