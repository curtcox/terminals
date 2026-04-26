---
title: "Server-Driven UI"
kind: plan
status: superseded
owner: copilot
validation: none
last-reviewed: 2026-04-25
---

# Server-Driven UI

Status: Completed and drained on 2026-04-25.

The durable behavior and contract from this plan are now documented in:

- [`docs/server.md`](../../docs/server.md) (Text Terminal Runtime and Server-Driven UI Contract)
- [`api/terminals/ui/v1/ui.proto`](../../api/terminals/ui/v1/ui.proto) (SetUI/UpdateUI/TransitionUI and closed Node widget contract)
- [`terminal_server/internal/ui/descriptor.go`](../../terminal_server/internal/ui/descriptor.go) and [`terminal_server/internal/ui/validate.go`](../../terminal_server/internal/ui/validate.go)
- [`terminal_client/lib/main.dart`](../../terminal_client/lib/main.dart) and [`terminal_client/test/widget_test.dart`](../../terminal_client/test/widget_test.dart)

There are no remaining active tasks in this plan. Future UI capability changes
should be tracked through the protocol and architecture plans, then documented
in `docs/server.md` after implementation.
