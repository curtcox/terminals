# identity

Inspect known identities, groups, audience selectors, and acknowledgement state.

- `identity ls`
- `identity show <identity>`
- `identity groups`
- `identity resolve <audience>` where `<audience>` can be `all`, `id:<id>`, `group:<group>`, or `alias:<alias>`.
- `identity prefs <identity>`
- `identity ack ls [subject-ref]`
- `identity ack show <subject-ref>`
- `identity ack record <subject-ref> --actor <actor-ref> [--mode <mode>]`

## Examples

```text
identity ls
identity show mom
identity groups
identity resolve group:kids
identity prefs mom
identity ack ls
identity ack show message:msg-1
identity ack record message:msg-1 --actor person:mom --mode read
identity ack record alert:fire --actor device:kitchen-screen --mode dismissed
identity ack record alert:fire --actor agent:voice-bot --mode confirmed
identity ack record bulletin:door --actor anonymous:sip --mode heard
```
