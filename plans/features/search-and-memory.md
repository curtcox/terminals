---
title: "Search and Memory Plan"
kind: plan
status: superseded
owner: copilot
validation: none
last-reviewed: 2026-04-25
---

# Search and Memory Plan

Status: Completed and drained on 2026-04-25.

The completed work from this plan is now documented in:

- `docs/repl/commands/search.md`
- `docs/repl/commands/memory.md`
- `docs/repl/api/memory-service.md`
- `docs/repl/examples/search-household-memory.md`

Server implementation and tests live in:

- `terminal_server/internal/capability/service.go`
- `terminal_server/internal/capability/service_test.go`
- `terminal_server/internal/admin/server.go`
- `terminal_server/internal/admin/server_test.go`
- `terminal_server/internal/repl/repl.go`
- `terminal_server/internal/repl/repl_test.go`

There are no remaining active tasks in this plan. Future changes to search ranking,
timeline policies, related-subject heuristics, or memory linking should be scoped in
new focused plans under `plans/features/`.
