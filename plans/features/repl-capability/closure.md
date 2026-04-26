---
title: "REPL Capability Closure Plan"
kind: plan
status: superseded
owner: copilot
validation: none
last-reviewed: 2026-04-26
---

# REPL Capability Closure Plan

Status: Completed and drained on 2026-04-26.

The capability-closure scope from this umbrella plan is now covered by shipped
component plans and durable docs:

- `plans/features/identity-and-audience.md`
- `plans/features/collab-sessions.md`
- `plans/features/messaging-and-boards.md`
- `plans/features/shared-artifacts.md`
- `plans/features/search-and-memory.md`
- `docs/repl/commands/identity.md`
- `docs/repl/commands/session.md`
- `docs/repl/commands/message.md`
- `docs/repl/commands/board.md`
- `docs/repl/commands/artifact.md`
- `docs/repl/commands/canvas.md`
- `docs/repl/commands/search.md`
- `docs/repl/commands/memory.md`
- `docs/repl/commands/placement.md`
- `docs/repl/commands/recent.md`
- `docs/repl/commands/store.md`
- `docs/repl/commands/bus.md`
- `docs/repl/examples/start-room-chat.md`
- `docs/repl/examples/send-direct-message.md`
- `docs/repl/examples/pin-family-bulletin.md`
- `docs/repl/examples/remote-help-session.md`
- `docs/repl/examples/shared-lesson-session.md`
- `docs/repl/examples/annotate-shared-canvas.md`
- `docs/repl/examples/search-household-memory.md`
- `docs/repl/examples/review-learner-progress.md`
- `docs/repl/examples/resume-multiplayer-session.md`

Validation evidence lives in the server test suite and use-case gate mappings,
including:

- `terminal_server/internal/capability/service_test.go`
- `terminal_server/internal/repl/repl_test.go`
- `terminal_server/internal/admin/server_test.go`
- `docs/usecase-validation-matrix.md` (`PL1`, `PL8`, and `PL20`)

There are no remaining active tasks in this umbrella closure plan. Future
cross-capability deltas should be scoped in targeted plans under
`plans/features/`.
