---
title: "Connection Reliability Unification Plan"
kind: plan
status: superseded
owner: copilot
validation: none
last-reviewed: 2026-04-26
---

# Connection Reliability Unification Plan

Status: Completed and drained on 2026-04-26.

The durable behavior from this plan is now documented in:

- [`docs/discovery-and-connection.md`](../../docs/discovery-and-connection.md)

Primary implementation references:

- `terminal_client/lib/connection/reliability.dart`
- `terminal_client/lib/main.dart`
- `terminal_client/test/reliability_test.dart`

There are no remaining active tasks in this plan. Future connection and
transport changes should be scoped in a new plan if they materially expand the
reliability model.
