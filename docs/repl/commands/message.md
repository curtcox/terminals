# message

See `help message` for the typed command surface.

## Commands

- `message rooms [--json]`
- `message room new <name> [--json]`
- `message room show <room> [--json]`
- `message ls [room] [--json]`
- `message get <message> [--json]`
- `message post <room> <text> [--json]`
- `message dm <target> <text> [--json]`
- `message thread <root-message> <text> [--json]`
- `message unread <identity> [room] [--json]`
- `message ack <identity> <message> [--json]`

## Notes

- `message post` is for room-scoped messages.
- `message rooms` and `message room ...` manage durable room records.
- `message dm` normalizes bare targets (for example `mom`) into `person:<target>`.
- `message thread` records `thread_root_ref` and `thread_parent_ref` on replies.
- `message unread` and `message ack` operate on message acknowledgement state.

## Examples

```text
message rooms
message room new family
message room show kitchen
message post kitchen Dinner in 10
message get msg_42
message thread msg_42 Bring plates too
message dm mom Come downstairs
message unread alice
message ack alice msg_42
```
