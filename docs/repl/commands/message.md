# message

See `help message` for the typed command surface.

## Commands

- `message ls [room] [--json]`
- `message post <room> <text> [--json]`
- `message dm <target> <text> [--json]`
- `message unread <identity> [room] [--json]`
- `message ack <identity> <message> [--json]`

## Notes

- `message post` is for room-scoped messages.
- `message dm` normalizes bare targets (for example `mom`) into `person:<target>`.
- `message unread` and `message ack` operate on message acknowledgement state.

## Examples

```text
message post kitchen Dinner in 10
message dm mom Come downstairs
message unread alice
message ack alice msg_42
```
