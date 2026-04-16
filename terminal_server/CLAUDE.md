# terminal_server Agent Notes

## Language and Style

- Use idiomatic Go.
- Keep packages focused and testable.
- Prefer context-aware APIs.

## Commands

```bash
go build ./...
go test ./...
golangci-lint run ./...
```

## Guardrails

- Keep scenario orchestration in server modules.
- Keep transport contract changes aligned with protobuf updates in `api/proto`.
- Keep AI integrations behind interfaces.

## Event Logging

- Server logging is structured JSONL via `internal/eventlog`.
- Default path is `logs/terminals.jsonl`, configurable with `TERMINALS_LOG_DIR`.
- Prefer `eventlog.Emit` + `slog` attributes over unstructured string logs.
- Query locally with `term logs ...` and from the dashboard at `/admin/logs`.
