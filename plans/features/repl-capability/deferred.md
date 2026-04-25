---
title: "REPL Capability Plan — Deferred Work"
kind: plan
status: planned
owner: unowned
validation: none
last-reviewed: 2026-04-25
---

# REPL Capability Plan — Deferred Work

See [repl-capability-plan.md](plan.md) for the
umbrella this document defers from. This file catalogs concerns
surfaced during the plan's review cycles (three Codex passes) that
were *intentionally not* folded into the plan, so that future work
has a record of what was considered and why it was set aside.

Each entry names the concern, the review pass that surfaced it,
where the concern touches the plan, the rationale for deferral, and
the trigger that should cause the item to be re-opened. Entries are
not ordered by priority — pick order is a separate exercise.

## D1 — Typed handler pipelines replacing `--run <command>`

**Surfaced:** review 1, §2 generality and §3 changes-to-consider.

**Location in plan:** [repl-capability-plan.md](plan.md)
`handlers` command group (`handlers on <selector> <action>
--run <command>`), `HandlerService` substrate entry.

**Concern.** `--run <command>` is stringly. Real event choreography
wants ordered steps, conditions (guards on payload or scope),
retries, debounce, fan-out sequencing, and an explicit error
policy. The current shape is fine for demos; it is weak for
production workflows and for any scenario where multiple handlers
race on the same input.

**Why deferred.** Typed action pipelines are a real design
exercise — choosing a pipeline DSL, deciding how conditions
compose, reconciling with the existing approval-pipeline
classification (`read_only | operational | mutating`), and ensuring
MCP-origin parity. The umbrella ships with the simpler form so
that phases 3–5 can land without blocking on that design.

**Re-open when.** A production scenario hits a concrete failure
mode the string form cannot express (duplicate fires, unhandled
payload shape, missing retry on transient failure), or when two
handlers authored against the same selector race and the
resolution is not expressible as "register both, first wins."

## D2 — Cohort audience bridge to `IdentityService`

**Surfaced:** review 1, §2 generality.

**Location in plan:** `devices cohort` command group,
`CohortService`.

**Concern.** Cohort selectors are placement + `scenario=`. They
cannot express identity-driven audiences ("all devices currently
in use by a member of the `kids` group") without manual
`identity resolve` + `devices cohort put --ids …` glue. As Layer 2
lands, identity-scoped cohorts will be the common case, not the
exception.

**Why deferred.** The bridge is cheap to add once `IdentityService`
ships (phase 6), and the umbrella's phase ordering has cohorts in
phase 2 — before identity exists. Adding the audience flag upfront
would either couple phase 2 to phase 6 or define a placeholder
contract that drifts.

**Re-open when.** Phase 6 (`IdentityService`) is landing. The
concrete addition is `devices cohort put <name> --audience
<AudienceSpec>` backed by `IdentityService.ResolveAudience`, with
re-resolution on membership/device-presence changes.

## D3 — `MemoryService` merged-vs-separate decision

**Surfaced:** review 1, §2 generality and §3 nice-to-have.

**Location in plan:** L2 families table, phase 10,
[search-and-memory.md](../search-and-memory.md) (optional
`MemoryService`).

**Concern.** The umbrella and component plan both admit
`MemoryService` as "optional, may be folded into `SearchService`."
That defers a real contract decision: keep it separate and pay
surface-area cost, or merge it and risk weakening activity-stream
semantics (PL26/PL27-style "what happened while I was away").

**Why deferred.** Neither direction has surfaced a forcing
function yet. Both service shapes are plausible; the choice is
worth making when the first real consumer (household timeline,
learner progress review, activity resurfacing) is being built
end-to-end, not in the abstract.

**Re-open when.** Phase 10 begins implementation. The decision
should be made at that point, not earlier, and recorded as an
amendment to `search-and-memory.md`.

## D4 — De-hardcode session/artifact kind enums

**Surfaced:** review 1, §2 generality and §3 nice-to-have.

**Location in plan:** [collab-sessions.md](../collab-sessions.md)
session kinds (`chat_room`, `artifact_view`, …),
[shared-artifacts.md](../shared-artifacts.md) artifact kinds
(`lesson`, `quiz`, `sign`, `checklist`).

**Concern.** Enumerated kinds read as example-driven and will
age badly as new use cases arrive. Extensible kind + typed
metadata would survive PL12–PL24 template/lesson/game evolution
without a schema change per new case.

**Why deferred.** The component plans' call, not the umbrella's —
the umbrella does not enumerate kinds. Component plans are
deliberately left untouched in the current round.

**Re-open when.** Either component plan hits its first kind
addition that would otherwise require a schema bump, or when
there is evidence that apps are carrying private kind values
outside the enum.

## D5 — `sim` event injection and virtual time

**Surfaced:** review 1, §3 nice-to-have.

**Location in plan:** `sim` command group, `SimService`.

**Concern.** `sim` currently covers virtual devices, input
injection, UI capture, and assertions. It does not expose event
injection onto the bus, media simulation, or deterministic clock
control. Sensor-heavy scenarios (alert/smoke/presence regression
suites) and scheduled-behavior scenarios (timers, reminders)
cannot be made fully reproducible without those.

**Why deferred.** The sim surface shipping in phase 5 is the
minimum viable set for Phase 5's acceptance (scenarios authored
in phases 2–4 are reproducible). Sensor and clock simulation are
valuable but not required for that phase's gate.

**Re-open when.** The first sensor- or schedule-driven scenario
needs regression coverage and the existing `sim` surface cannot
express the test. Candidate additions: `sim bus emit`, `sim clock
set`, `sim clock advance`, `sim sensor ...`.

## D6 — `StoreService` CAS / atomic update

**Surfaced:** review 1, §2 generality.

**Location in plan:** `store` command group, `StoreService`.

**Concern.** Values are JSON-compatible and versioned but there
is no explicit compare-and-swap or atomic-update primitive.
Concurrent session state (multiplayer games, shared-help
co-control cursors, lesson progress) and any scenario where two
handlers may update the same key concurrently cannot be expressed
safely without it.

**Why deferred.** Phase 1 ships the store shape the existing TAL
`store` module already uses; adding CAS is a compatible
additive change. The umbrella holds it until a real concurrent
writer appears.

**Re-open when.** Phase 7 (`InteractiveSessionService`) or Phase 9
(`ArtifactService`) lands a scenario with concurrent writers. The
concrete addition is `store cas <ns> <key> <expected-version>
<value>` plus matching `StoreService.CompareAndSwap`.

## D7 — `BusService.Replay` semantics

**Surfaced:** review 1, §2 generality.

**Location in plan:** `bus` command group, `BusService.Replay`.

**Concern.** `Replay` is listed but ordering, window semantics,
and idempotence are not pinned down. Replayed events interact
with live handlers; without a policy, replay is ambiguous
(re-fire handlers? suppress? dry-run only?).

**Why deferred.** The minimum viable Phase 1 usage is `Tail`, not
`Replay`; agents and CI consume `Tail` without needing replay
semantics. Pinning down `Replay` prematurely risks a contract
that no current consumer actually needs.

**Re-open when.** The first replay consumer appears — likely a
bug-reporting diagnostic "re-run the bus state that led to this
report" flow, or a scheduled-task re-dispatch tool. The contract
should be written against that consumer.

## D8 — `SearchService` query DSL and ranking

**Surfaced:** review 1, §2 generality.

**Location in plan:** [search-and-memory.md](../search-and-memory.md)
`Query`/`Timeline`/`Related`/`Recent`/`Suggest`.

**Concern.** The search plan enumerates methods but does not
specify the query/filter DSL, ranking model, or relevance
semantics. "Service exists, behavior TBD" is a known failure mode
for search surfaces; it tends to lock in the trivial-substring
match that the first implementation ships and then resist
improvement.

**Why deferred.** The component plan's call. The umbrella
preserves the five-form rule and does not pick a search
implementation.

**Re-open when.** Phase 10 starts. Decide DSL and ranking
*before* a stub implementation ships, not after.

## D9 — `InteractiveSessionService` kinds and policy constraints

**Surfaced:** review 1, §2 generality.

**Location in plan:** [collab-sessions.md](../collab-sessions.md).

**Concern.** Kinds look hardcoded (example-driven), and policy
constraints (time gating, room gating, role gating — "kids
can't start a session after 9pm", "only caregivers can end a
help session") are underspecified. Related to D4 but orthogonal:
even with extensible kinds, policy is a separate contract.

**Why deferred.** Component plan's call.

**Re-open when.** The first session-policy use case lands
(typically a PLATO-inspired lesson with role gating, or a
help-session with role-gated takeover).

## D10 — Authored-view record lifecycle

**Surfaced:** review 3, §3.

**Location in plan:** `ui views ls/show/rm`, `UiService`
authored-view inventory.

**Concern.** The umbrella pins authored-view records as REPL-side
authoring metadata (not a change to the `server-driven-ui.md`
primitive contract). It does not specify: persistence across
server restart, scope (session-local vs. global, per-origin vs.
shared), GC on view-id collision, or visibility between MCP
origins and human REPL sessions.

**Why deferred.** The command surface and service summary land
in phase 2; the lifecycle details have not surfaced a concrete
conflict yet. Over-specifying before use pins choices
unnecessarily.

**Re-open when.** The first cross-origin authored-view conflict
appears (two REPL sessions publishing different trees under the
same view-id), or when phase 11 documentation requires a
statement on restart behavior, or when authored views need to
survive a server restart for an ops-level use case.

---

## Items intentionally *not* deferred to this document

For reference, the following concerns were raised in review but
handled inline in the plan rather than deferred here. They are
listed so that this document does not accidentally re-open them:

- bug-reporting family coverage (B1–B5): applied as L2 family +
  Phase 11.
- I1/I2/I3/I8/I9/I11 scope mismatch: applied as explicit
  out-of-scope carve-out.
- `scenarios define` lifecycle parity with TAR: applied via
  `--match event=`, `--on-event`, `--on-suspend`, `--on-resume`.
- `ui transition` omission: applied.
- authored-view inventory commands (`ui views ls/show/rm`):
  applied (lifecycle details deferred as D10).
- `timeline` taxonomy mismatch: applied (reconciled in umbrella,
  propagated into [search-and-memory.md](../search-and-memory.md)).
- ack-ownership boundary blur: applied (`IdentityService` owns
  ack; messaging delegates; actor-kind union allows non-person
  actors).
- `BugReportService.Attach` / `bug tail`: applied as proposed
  extensions to [bug-reporting.md](../bug-reporting.md) in Phase 11.

## Related

- [repl-capability-plan.md](plan.md) — the
  umbrella this document defers from.
- [identity-and-audience.md](../identity-and-audience.md),
  [collab-sessions.md](../collab-sessions.md),
  [messaging-and-boards.md](../messaging-and-boards.md),
  [shared-artifacts.md](../shared-artifacts.md),
  [search-and-memory.md](../search-and-memory.md),
  [bug-reporting.md](../bug-reporting.md) — Layer 2 component plans
  that several deferred items touch.
