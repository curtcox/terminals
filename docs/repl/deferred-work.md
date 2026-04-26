# REPL Capability Deferred Work Ledger

This document preserves deferred concerns from the REPL capability planning
cycle. It is the durable backlog reference for items that were intentionally
not folded into immediate implementation plans.

Source: drained from `plans/features/repl-capability/deferred.md` on
2026-04-26.

## D1 — Typed handler pipelines replacing `--run <command>`

**Surfaced:** review 1, section 2 generality and section 3 changes-to-consider.

**Touches:** REPL `handlers` group (`handlers on <selector> <action> --run <command>`), `HandlerService`.

**Concern.** `--run <command>` is stringly. Real choreography needs ordered
steps, guards on payload/scope, retries, debounce, fan-out sequencing, and
explicit error policy.

**Deferred because.** Typed action pipelines require DSL and policy design,
approval-pipeline classification alignment, and MCP-origin parity.

**Re-open trigger.** Production failures not expressible in string form, or
competing handlers on the same selector that cannot be resolved safely.

## D2 — Cohort audience bridge to `IdentityService`

**Surfaced:** review 1, section 2 generality.

**Touches:** `devices cohort` command group, `CohortService`.

**Concern.** Cohort selectors are placement + `scenario=` and cannot express
identity-driven audiences without manual glue.

**Deferred because.** The bridge is straightforward once `IdentityService`
ships (phase 6). Adding it earlier couples phase ordering.

**Re-open trigger.** Phase 6 implementation. Add
`devices cohort put <name> --audience <AudienceSpec>` backed by
`IdentityService.ResolveAudience` with re-resolution on membership/presence
changes.

## D3 — `MemoryService` merged-vs-separate decision

**Surfaced:** review 1, section 2 generality and section 3 nice-to-have.

**Touches:** Layer 2 family table, phase 10, search/memory component plan.

**Concern.** Contract decision is deferred: keep `MemoryService` separate or
fold into `SearchService`.

**Deferred because.** No forcing consumer exists yet.

**Re-open trigger.** Phase 10 start; decide with first end-to-end consumer
(timeline, learner review, or activity resurfacing).

## D4 — De-hardcode session/artifact kind enums

**Surfaced:** review 1, section 2 generality and section 3 nice-to-have.

**Touches:** collab sessions and shared artifacts component plans.

**Concern.** Hardcoded kind enums are example-driven and may age poorly.

**Deferred because.** This belongs to component-plan evolution, not umbrella
plan scope.

**Re-open trigger.** First kind addition requiring schema bump, or evidence of
private out-of-band kind values.

## D5 — `sim` event injection and virtual time

**Surfaced:** review 1, section 3 nice-to-have.

**Touches:** `sim` command group, `SimService`.

**Concern.** Current sim omits bus event injection, media simulation, and
deterministic clock control.

**Deferred because.** Phase 5 acceptance does not require those additions.

**Re-open trigger.** Sensor- or schedule-driven scenario cannot be expressed by
existing sim surface. Candidate additions: `sim bus emit`, `sim clock set`,
`sim clock advance`, `sim sensor ...`.

## D6 — `StoreService` CAS / atomic update

**Surfaced:** review 1, section 2 generality.

**Touches:** `store` command group, `StoreService`.

**Concern.** No explicit compare-and-swap/atomic-update primitive for
concurrent writes.

**Deferred because.** Phase 1 mirrors existing store shape; CAS can be additive
later.

**Re-open trigger.** Phase 7 or phase 9 lands concurrent-writer scenarios.
Candidate addition: `store cas <ns> <key> <expected-version> <value>` and
`StoreService.CompareAndSwap`.

## D7 — `BusService.Replay` semantics

**Surfaced:** review 1, section 2 generality.

**Touches:** `bus` command group, `BusService.Replay`.

**Concern.** Ordering/window/idempotence and interaction with live handlers are
unspecified.

**Deferred because.** Current consumers only need `Tail`.

**Re-open trigger.** First replay consumer (for example bug-report diagnostics
or scheduled-task redispatch tooling).

## D8 — `SearchService` query DSL and ranking

**Surfaced:** review 1, section 2 generality.

**Touches:** search/memory component plan.

**Concern.** Methods exist but query/filter DSL and ranking semantics are not
specified.

**Deferred because.** Component-plan decision, not umbrella decision.

**Re-open trigger.** Phase 10 start; choose DSL/ranking before shipping a stub.

## D9 — `InteractiveSessionService` kinds and policy constraints

**Surfaced:** review 1, section 2 generality.

**Touches:** collab sessions component plan.

**Concern.** Session kinds look hardcoded and policy constraints are
underspecified (time, room, and role gating).

**Deferred because.** Component-plan decision.

**Re-open trigger.** First session-policy use case lands.

## D10 — Authored-view record lifecycle

**Surfaced:** review 3, section 3.

**Touches:** `ui views ls/show/rm`, `UiService` authored-view inventory.

**Concern.** Lifecycle rules are not pinned down: persistence across restart,
scope and visibility, collision handling, and cross-origin behavior.

**Deferred because.** Surface landed before lifecycle conflicts appeared.

**Re-open trigger.** Cross-origin authored-view conflict, phase 11 docs
hardening, or operational requirement to survive server restart.
