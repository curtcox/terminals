# bug

Typed bug-reporting operations exposed through the REPL.

## Commands

- `bug ls [--json]`
- `bug show <report-id> [--json]`
- `bug file <reporter-device-id> <subject-device-id> <description> [--source <source>] [--tags <tag[,tag...]>] [--json]`
- `bug confirm <report-id> [--json]`
- `bug tail [<query>]`

## Notes

- `bug ls` reads the same list view as `/admin/api/bugs`.
- `bug file` submits through `/bug/intake` with typed JSON payload fields.
- `--source` accepts either short forms like `admin`, `sip`, `webhook` or full enum values such as `BUG_REPORT_SOURCE_SIP`.
- `--tags` accepts comma-separated tags (for example `ui_glitch,lost_connection`).
- `bug tail` queries `/admin/logs.jsonl` with an automatic `bug.report` filter prefix, so `bug tail severity:error` becomes a log query for `bug.report severity:error`.

## Examples

```text
bug ls
bug show bug_1234
bug file kitchen-display hallway-panel "screen frozen" --source sip --tags ui_glitch,lost_connection
bug confirm bug_1234
bug tail
bug tail severity:error
```
