---
title: "Client Architecture"
kind: plan
status: superseded
owner: copilot
validation: none
last-reviewed: 2026-04-26
---

# Client Architecture

Status: Completed and drained on 2026-04-26.

The durable architecture and behavior from this plan now live in:

- [`docs/client-architecture.md`](../../docs/client-architecture.md)
- [`docs/discovery-and-connection.md`](../../docs/discovery-and-connection.md)
- [`docs/server.md`](../../docs/server.md)

Primary implementation references:

- [`terminal_client/lib/main.dart`](../../terminal_client/lib/main.dart)
- [`terminal_client/lib/capabilities/probe.dart`](../../terminal_client/lib/capabilities/probe.dart)
- [`terminal_client/test/widget_test.dart`](../../terminal_client/test/widget_test.dart)

There are no remaining active tasks in this plan. Future client architecture
changes should be tracked in focused feature plans and reflected in
`docs/client-architecture.md` after implementation.
