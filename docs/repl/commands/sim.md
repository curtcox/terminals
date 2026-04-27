# sim

Virtual-device simulation helpers for REPL-driven validation.

## Commands

- `sim device new <id> [--caps <cap[,cap...]>] [--json]`
- `sim device rm <id> [--json]`
- `sim input <id> <component-id> <action> [<value>] [--json]`
- `sim ui <id> [--json]`
- `sim expect <id> <ui|message> <selector> [--within <duration>] [--json]`
- `sim record <id> [--duration <duration>] [--json]`

Use `sim device new` before injecting inputs. `sim ui` returns both the
captured authored UI snapshot and the buffered synthetic inputs for that
virtual device.

`sim expect` returns non-zero when the selector does not match captured output.
Use `ui` kind to assert on captured UI payload and `message` kind to assert on
captured bus messages. `sim record` returns a bounded capture payload (snapshot,
input history, and message tail) for deterministic inspection.
