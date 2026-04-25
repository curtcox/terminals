---
title: "Connection Reliability Unification Plan"
kind: plan
status: planned
owner: unowned
validation: none
last-reviewed: 2026-04-25
---

# Connection Reliability Unification Plan

## Goal
Make client/server communication paths simple, shared, and consistently reliable so that if communication works in one feature, it works in all features.

## Problem Summary
Current behavior is mostly reliable for bug reporting and app launch, but not uniformly so across all feature actions.

Observed causes:
- Multiple readiness checks with overlapping semantics.
- Some actions use robust queue/replay logic, while others send directly.
- Retry/watchdog patterns are duplicated between flows.
- UI status can drift from transport readiness if state transitions diverge.

## Target Architecture

### 1) Single Connection Phase Model
Introduce a single connection phase source of truth (for example: `disconnected`, `connecting`, `connected_unregistered`, `registered`, `degraded`).

All UI labels and action gating should derive from this phase instead of combining independent booleans.

### 2) Shared Readiness Gateway
Create one helper to ensure transport and registration are ready before operation dispatch.

Example intent:
- `ensureConnectedAndRegistered(...)`
- Starts stream if needed.
- Waits/retries until register ack or timeout.
- Returns typed result (`ready`, `timeout`, `failed`).

### 3) Shared Reliable Send Primitive
Create one dispatch wrapper for outbound operations that need reliability.

Example intent:
- `sendWhenReady(...)`
- Policy options:
  - `mode: fire_and_forget`
  - `mode: queue_until_ready`
  - `mode: require_ack`
- Optional ack matcher and timeout.

This should be reused by:
- Bootstrap registration handshake.
- Bug report submit/receipt.
- App launch command.
- System/debug queries.
- User input actions that must not be lost.

### 4) One Retry/Timeout Policy Layer
Unify retry timing and timeout behavior into one policy module.

Parameters:
- Retry interval
- Max duration
- Backoff strategy (if needed)
- Logging style and severity

Both bootstrap and bug-report ack watchdogs should use this shared policy.

### 5) Shared Outbound Routing Rules
Define a small decision matrix for all outbound messages:
- Must queue when not ready?
- May be dropped?
- Requires server ack?
- Safe to replay?

Store rules in one place and enforce through the shared send wrapper.

## Migration Plan

### Phase 1: Foundation
- Add connection phase enum + mapper from current state.
- Add `ensureConnectedAndRegistered(...)` helper.
- Add shared retry policy utility.
- Keep old callsites intact behind adapters.

### Phase 2: Dispatcher Adoption
- Add `sendWhenReady(...)` with policy modes.
- Migrate bootstrap register flow.
- Migrate bug-report flow to shared primitives while preserving current behavior.

### Phase 3: Action Path Convergence
Migrate all user-facing actions to shared dispatch:
- Open application
- Runtime/device/scenario queries
- UI action events
- Key events
- Any operation currently doing direct `_outgoing.add(...)` without readiness checks

### Phase 4: UI and Telemetry Simplification
- Drive connection labels/chips from connection phase only.
- Standardize logs/messages for queued/replayed/retried actions.
- Remove duplicate status text branches.

### Phase 5: Cleanup
- Remove old per-flow readiness checks and bespoke watchdogs.
- Keep one canonical readiness and dispatch path.

## Testing Strategy

### Unit Tests
- Connection phase transitions from all state combinations.
- Retry policy timing/timeout semantics.
- `ensureConnectedAndRegistered(...)` success/failure/timeout behavior.
- `sendWhenReady(...)` queue/replay/ack behavior per mode.

### Widget/Integration Tests
Create a single matrix that runs each feature action under the same transport stress conditions:
- Delayed request stream attachment
- Missing initial register ack
- Stream interruption and reconnect
- Late ack arrival

For each action category verify:
- Not silently dropped
- Replayed as expected (if policy says so)
- Proper timeout/failure message

### Regression Set (minimum)
- Bootstrap metadata hydration on first load.
- Bug report queue + replay + positive ack.
- App launch queue-until-registered.
- Runtime/device/scenario queries behave consistently with policy.
- UI input actions follow selected reliability policy.

## Invariants
- No feature path bypasses shared readiness/dispatch for reliable operations.
- No duplicate watchdog implementation for similar retry/timeout semantics.
- UI connection indicators are phase-derived, never hardcoded.
- Bug reporting behavior and receipts remain intact.

## Risks and Mitigations
- Risk: accidental behavior changes during migration.
  - Mitigation: migrate in phases with compatibility adapters and focused regressions.
- Risk: over-queuing operations that should be immediate best-effort.
  - Mitigation: explicit per-operation policy table.
- Risk: complexity of ack matching.
  - Mitigation: typed ack predicates and strict timeout handling.

## Deliverables
- Shared readiness helper
- Shared send wrapper with policy modes
- Shared retry/timeout policy module
- Unified connection phase model
- Updated tests and stress matrix
- Removed duplicated flow-specific watchdog logic
