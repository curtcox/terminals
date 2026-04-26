---
title: "Server Event Logging"
kind: plan
status: superseded
owner: copilot
validation: none
last-reviewed: 2026-04-26
---

# Server Event Logging

Status: Completed and drained on 2026-04-26.

The completed event-logging work from this plan is now documented in:

- [`README.md`](../../README.md)
- [`docs/server.md`](../../docs/server.md)
- [`docs/event-taxonomy.md`](../../docs/event-taxonomy.md)
- [`terminal_server/CLAUDE.md`](../../terminal_server/CLAUDE.md)

Implementation lives in the server runtime and supporting packages:

- `terminal_server/internal/eventlog/` (writer, slog integration, context helpers)
- `terminal_server/internal/eventlog/query/` (log filtering and query helpers)
- `terminal_server/cmd/server/main.go` (logger initialization and process wiring)
- `terminal_server/cmd/term/main.go` (`term logs` command surface)
- `terminal_server/internal/admin/server.go` (`/admin/logs*` endpoints)

There are no remaining active tasks in this plan. Future event logging
enhancements should be tracked in a new focused plan (for example: remote
shipping, retention policy changes, or schema-versioned exports).
