---
title: "Protocol — Progress Log"
kind: progress-log
parent: plans/features/protocol/plan.md
---

## Implementation Progress (2026-04-26)

- Added explicit capability invalidation payloads to control-plane acknowledgements in `CapabilityAck.invalidations` (`api/terminals/control/v1/control.proto`).
- Wired server transport capability ack generation to include deterministic lost-resource invalidations (resource + reason) when snapshots/deltas remove claimable resources.
- Added/updated transport regression coverage for ack invalidation content and proto adapter mapping.
- Updated durable connection docs to describe `capability_ack` invalidation behavior.
- Removed client bootstrap emission of deprecated `RegisterDevice` requests; client bootstrap now sends `hello` + `capability_snapshot` and retries snapshot delivery until acknowledgement instead of retrying register payloads.
- Removed generated-proto ingest support for deprecated `CapabilityUpdate` client payloads; generated clients must use `capability_snapshot` / `capability_delta`.
- Normalized legacy generated `register` payload ingest through capability-snapshot handling while preserving compatibility (`register_ack` remains emitted for bootstrap clients).
- Added snapshot bootstrap fallback for unknown devices: capability snapshots now synthesize identity registration when needed before applying generation-ordered capability state.
- Preserved relay registration semantics for snapshot-first sessions so cross-session route/notification fan-out behavior remains stable.
- Updated transport carrier and websocket integration tests to accept capability lifecycle bootstrap ordering (`capability_ack` may precede `register_ack`).
- Re-ran repository validation gates (`make all-check`) and promoted this plan to shipped-validated status.

Any future compatibility-window cleanup (for example fully removing deprecated proto request fields) should be tracked as a separate follow-on task, not under this completed protocol design plan.
