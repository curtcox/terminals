# sim

Virtual-device simulation helpers for REPL-driven validation.

## Commands

- `sim device new <id> [--caps <cap[,cap...]>] [--json]`
- `sim device rm <id> [--json]`
- `sim input <id> <component-id> <action> [<value>] [--json]`
- `sim ui <id> [--json]`

Use `sim device new` before injecting inputs. `sim ui` returns both the
captured authored UI snapshot and the buffered synthetic inputs for that
virtual device.
