# StreamHandler Subsystems Progress

- 2026-05-02: Created plan. Status: proposed. No implementation started.

## 2026-05-02 — Phase 1

Status: complete

Changes:
- Added private `newStreamHandler(control, runtime)` helper in `terminal_server/internal/transport/control_stream.go` that centralizes all field defaults.
- Rewrote `NewStreamHandler` and `NewStreamHandlerWithRuntime` to delegate to the helper.
- Added field-grouping comments inside the helper (transport dispatch / metrics, terminal+REPL, UI session state, photo-frame defaults, media+route-replay+voice, diagnostics+collaborator defaults) to mark the boundaries the later phases will extract along.
- Added `control_stream_constructor_test.go` with a constructor-invariant test that runs both public constructors and asserts every map, default limit, default duration, and default collaborator (terminal manager, REPL service, no-op recording manager, UI ownership tracker, wake-word dedupe, menu-app policy) is initialized.

Validation:
- `cd terminal_server && go test ./internal/transport -run TestStreamHandlerConstructors -count=1 -v` — pass.
- `cd terminal_server && go test ./internal/transport -count=1 -race` — pass (10.2s).
- `cd terminal_server && go test ./...` — pass (all transport + adjacent suites).
- `cd terminal_server && go build ./...` — pass.

Notes:
- No behavior changes intended. The two constructors were verbatim duplicates apart from `runtime`; the helper preserves order and values.
- Field-grouping comments are intentionally lightweight markers — they will become file boundaries once Phases 2–8 land.
- Phase 0 characterization gaps (reconnect with prior UI, route replay, stale capability generation, bug-report unavailable, command validation errors) are deferred to PR 2 per the plan's PR table.
