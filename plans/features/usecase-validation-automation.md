---
title: "Use Case Validation Automation"
kind: plan
status: shipped
owner: curtcox
validation: automated:C1,C2,M5,T1,T2,T3,T4,AA1,AA4,UI9
last-reviewed: 2026-05-16
progress:
  - 2026-05-16: Phase 1 complete (C1 harness skeleton, evidence bundle, USECASE_ARTIFACTS flag)
  - 2026-05-16: Phase 2 complete (simulated terminal transport; UI9 reconnect + C2 multi-terminal broadcast)
  - 2026-05-16: Phase 3 complete (fake clock; T1 voice path, T2 timer reminder, M5 camera activity)
  - 2026-05-16: Phase 4 complete (morning routine monitor scenario; T3/T4 school-morning, AA1 agent trigger, AA4 timer create/cancel; all registered in usecase-validate.sh)
  - 2026-05-16: Phase 5 complete (summary.md in evidence bundle; TestReplay command; CI artifact upload; validation matrix auto-populated from usecase-validate.sh metadata; plan validation frontmatter updated)
---

# Use Case Validation Automation

## Context

The repository already treats use cases as durable product contracts. The
`usecases/INDEX.md` file is generated from `usecases/<family>.md`, and each use
case ID is intended to be stable enough for plan frontmatter to reference as
`validation: automated:<ID>`. `docs/usecase-validation-matrix.md` maps those IDs
to the current validation command and evidence, and `scripts/usecase-validate.sh`
exposes the user-facing command:

```bash
make usecase-validate USECASE=<ID>
make usecase-validate USECASE=all
```

Current coverage is useful but uneven. Some IDs are backed by narrow smoke or
transport checks, some by scenario or contract tests, and many planned IDs are
not automated at all. The missing layer is a reusable, deterministic validation
harness that can run realistic multi-user, multi-terminal stories against as
much real server code as possible while replacing slow, nondeterministic, or
physical dependencies with controlled simulation.

## Goals

- Validate use-case implementation through scenario-level tests that exercise a
  real server instance (in-process) whenever practical.
- Simulate users, terminals, clocks, sensors, media events, and external agents
  without requiring physical hardware or real elapsed days.
- Capture enough evidence on every run to prove success and to diagnose failure
  quickly.
- Make validation evidence durable enough to appear in
  `docs/usecase-validation-matrix.md` and plan frontmatter.
- Keep `make usecase-validate USECASE=<ID>` as the stable developer and CI entry
  point.
- Support both fast deterministic CI scenarios and richer local replay for
  debugging.

## Non-Goals

- Do not replace normal unit, transport, contract, Flutter, or package tests.
- Do not turn clients into scenario-specific test clients; client behavior must
  remain generic terminal behavior.
- Do not require cameras, microphones, speakers, SIP providers, LLM providers,
  wall-clock waiting, or real home automation systems in CI.
- Do not hide nondeterminism behind arbitrary sleeps or retry loops.
- Do not make simulated validation the only release gate for behavior that also
  needs occasional physical-device or manual validation.
- Do not start an out-of-process server subprocess; in-process server
  construction reusing production handlers is the correct execution model.
- Do not add YAML scenario loading before Phase 4; all scenarios in Phases 1–3
  are Go-authored.

## Design Principles

1. **Real server first** — use the same in-process server construction used by
   production, then inject deterministic seams for time, terminals, IO, and
   external services. Timing correctness and logic correctness are the goals;
   subprocess execution is not required.
2. **Generic terminals only** — simulated terminals should speak the same wire
   protocol and expose the same capabilities as real clients. Test-only helpers
   may drive a terminal, but they must not receive privileged server APIs unless
   they are explicitly modeling an admin or automation agent use case.
3. **Synthetic users as actors** — tests should describe human-like actors with
   intent, attention, location, and available inputs, then map those actions to
   voice, tap, button, REPL, API, or sensor events.
4. **Deterministic time** — every server subsystem that schedules, expires,
   escalates, dedupes, or rotates behavior must be drivable from a simulated
   clock in scenario tests.
5. **Evidence before assertions** — capture the event log, terminal transcript,
   UI snapshots, media-route graph, scheduler state, store diffs, and actor
   timeline before checking assertions so failure output explains what happened.
6. **Replayable failures** — failed scenarios should emit a compact scenario
   bundle that can be replayed locally with the same seed, clock timeline, and
   actor events.

## Proposed Architecture

### 1. Scenario Validation Harness

Add a reusable harness package at
`terminal_server/internal/usecasevalidation/`.

The harness should provide:

- `Harness.StartServer(config)` that starts a real in-process server with
  test-owned temp storage, logs, app registry, package registry, and injected
  clock.
- `Harness.ConnectTerminal(profile)` that creates simulated terminals using an
  in-process adapter that reuses the same production handlers and command
  registries as the networked server. Falls back to a loopback listener only
  when an end-to-end transport path is required by the scenario.
- `Harness.Actor(name, profile)` that can drive a terminal or API client using
  high-level steps.
- `Harness.Clock()` that exposes deterministic `Now`, `Advance`, `RunUntilIdle`,
  and `AdvanceTo` operations on a new fake clock implementation (no existing
  clock abstraction covers the server subsystems; this is greenfield).
- `Harness.Expect()` helpers for server state, terminal state, UI state, media
  routes, scheduled jobs, notifications, logs, and durable stores.
- `Harness.Evidence()` that returns the structured evidence bundle for the run.

### 2. Simulated Terminals

Create a programmable terminal model that can connect, disconnect, reconnect,
change capabilities, and emit IO like a real thin client.

Each simulated terminal should track and expose:

- terminal identity, room/location, screen geometry, privacy state, and declared
  capabilities;
- received server commands, UI layers, overlays, focus claims, prompts, and
  notifications;
- outgoing user inputs: tap, gesture, keyboard, button, voice transcript, wake
  phrase, media frame marker, audio classification marker, QR/NFC scan, and
  heartbeat;
- media routes observed or sent, without needing real audio/video payloads unless
  the use case specifically validates encoding or media transport;
- reconnect lifecycle: detach, reconnect with prior identity, restore state, and
  resume streams.

The terminal model must remain generic. A scenario can say "the kitchen tablet
hears `announce: dinner is ready`", but the simulated terminal should simply
send the same input event a real terminal would send.

### 3. Simulated Users and Agents

Add actor scripts that express intent in human terms while producing normal
terminal or API events.

Actors should support:

- voice commands with optional ambiguity, confidence, and competing nearby
  terminals;
- taps and menu navigation;
- waiting for visible or audible feedback;
- ignoring, dismissing, or escalating notifications;
- external agents calling the server API for calendar, webhook, monitoring, or
  scheduling flows;
- multi-user races such as two residents responding to the same alert.

Actor scripts should be deterministic but realistic. For example, an actor may
wait until a terminal displays an alert and then dismiss it after simulated
seconds, or may miss the first reminder and trigger an escalation path.

Simple actors that only produce tap and voice events (no time dependency) may be
introduced in Phase 3 alongside the clock work, so the two efforts can proceed
in parallel. Time-dependent actors (missed reminders, escalation paths, external
scheduling agents) are Phase 4.

### 4. Simulated Clock and Scheduled Work

Introduce a new fake clock interface for every server subsystem used by use
cases. This is a greenfield implementation; no existing clock abstraction covers
these subsystems. Before designing the interface, inventory every `time.Now()`
and `time.Sleep()` call in server code to scope the retrofit.

Subsystems that must be clock-injectable:

- timers and reminders;
- recurring school-day or business-hour schedules;
- appliance silence windows;
- wake-word dedupe windows;
- reconnect and heartbeat timeouts;
- alert escalation;
- photo-frame rotation;
- package/runtime expiry and cleanup.

Scenario tests must never sleep for behavior that can be clock-driven. A
multi-day scenario should run by advancing synthetic time and draining scheduled
work until the server reaches quiescence.

### 5. Event and Evidence Capture

Every validation run should write a scenario evidence bundle under
`artifacts/usecase-validation/<run-id>/`. The `artifacts/` directory is
`.gitignore`d and created on demand; CI artifact upload is configured separately
via the GitHub Actions `upload-artifact` step. Passing CI runs upload only the
`manifest.json` summary; failed runs upload the full bundle automatically.

Recommended bundle layout:

```text
artifacts/usecase-validation/<run-id>/
  manifest.json
  scenario.yaml
  seed.txt
  server-config.json
  actor-timeline.jsonl
  terminal-events.jsonl
  server-events.jsonl
  ui-snapshots.jsonl
  media-routes.jsonl
  scheduler-trace.jsonl
  store-diffs.jsonl
  assertions.jsonl
  summary.md
```

The manifest should include:

- use case ID and scenario name;
- git commit, server version, and test binary/package;
- random seed if any fuzzing or randomized actor timing was used;
- simulated start time and end time;
- terminal profiles and actor profiles;
- pass/fail result and failing assertion IDs.

Assertions should point to evidence by stable event IDs, not only by prose. A
failure should answer:

- what actor action was being validated;
- what the server received;
- what state changed;
- what each terminal saw or heard;
- what was expected;
- what was missing, late, duplicated, or malformed.

### 6. Scenario Specification Format

All scenarios in Phases 1–3 are Go-authored for speed, type safety, and
interface stability. YAML loading is explicitly out of scope until Phase 4,
after the core actor, terminal, clock, and evidence interfaces have settled.

Scenario testdata files live at
`terminal_server/internal/usecasevalidation/testdata/` alongside the Go test
code so that refactors stay in-tree.

When YAML loading is added in Phase 4, each step's `says` field maps to a
`VoiceAudio` proto event with the transcript set to the given text; the mapping
must be documented in the YAML schema before parsing is implemented.

Example shape (reference only; not parsed until Phase 4):

```yaml
id: T3-school-morning-escalation
usecases: [T3, T4]
clock:
  start: 2026-09-01T06:55:00-05:00
terminals:
  kitchen:
    room: kitchen
    capabilities: [display, speaker, microphone]
  child_room:
    room: bedroom
    capabilities: [camera, speaker]
actors:
  parent:
    role: parent
  child:
    role: child
steps:
  - actor: parent
    says: "monitor the morning routine at 7 AM on school days"
    at: kitchen
  - clock: advance_to
    time: 2026-09-01T07:00:00-05:00
  - terminal: child_room
    observes:
      camera_activity: absent
  - expect:
      notification:
        to: parent
        contains: "running late"
  - clock: advance
    duration: 10m
  - expect:
      terminal: child_room
      spoken: "the bus comes in 10 minutes"
```

## Validation Depth Levels

Extend the validation matrix vocabulary with stricter criteria for harness-based
coverage:

- **Smoke**: one command path or loop runs.
- **Transport**: generated/wire behavior is correct.
- **Contract**: API/package/runtime contract surfaces are correct.
- **Scenario**: trigger, routing, and server-side side effects are correct.
- **Simulation**: deterministic multi-terminal or time-driven behavior is
  correct with simulated actors or devices.
- **Full**: trigger, placement, UI, scheduling, media or sensor side effects,
  cancellation, expiry, reconnect/resume, and failure evidence are validated.

A use case should not be marked `Full` unless the evidence bundle can prove the
observable user outcome, not merely an internal function call.

## Implementation Plan

### Phase 1 — Inventory and Harness Skeleton

- Inventory every current `make usecase-validate` mapping and classify which
  tests already have reusable fixture pieces.
- Create `terminal_server/internal/usecasevalidation/harness.go` with
  `Harness.StartServer`, `Harness.Evidence`, and supporting types.
- Add `TestUseCaseC1WithEvidence` as the first harness-backed test: wrap the
  existing C1 transport coverage to emit an evidence bundle, without changing
  the underlying production behavior.
- Capture server events and assertion results into a first-version evidence
  bundle.
- Add the `USECASE_ARTIFACTS=1` environment flag to preserve bundles on demand.
- Add `artifacts/` to `.gitignore`.

Acceptance criteria:

- Existing automated IDs still pass through `make usecase-validate USECASE=all`.
- `TestUseCaseC1WithEvidence` emits a valid evidence bundle with `manifest.json`
  and `assertions.jsonl`.
- `scripts/usecase-validate.sh` is updated to register the new harness-backed
  C1 variant and any other IDs added in this phase.

### Phase 2 — Simulated Terminal Transport

- Build simulated terminals that connect through the in-process server transport
  path, reusing production handlers and command registries.
- Model registration, capabilities, heartbeat, input events, reconnect, and
  received UI/control messages.
- Implement two harness-backed scenarios as the first full Phase 2 deliverables:
  - **UI9 — Reconnect while overlay is open**: use simulated terminal reconnect
    to validate restored main and overlay layers with evidence beyond the
    existing internal assertions in `TestGeneratedSessionUI_RECON_1`.
  - **C2 — Whole-house announcement**: simulate one speaking terminal and three
    receiving terminals; verify broadcast routing, terminal observations, event
    log entries, and no duplicate delivery.
- Retain the narrower transport tests that already exist for both IDs.

Acceptance criteria:

- A two-terminal scenario can prove both the server-side route and each
  terminal's observed result.
- Failure output includes both wire-level events and terminal-level observations.
- `scripts/usecase-validate.sh` is updated to register UI9 (harness) and C2.

### Phase 3 — Simulated Clock and Time-Independent Actors

- Begin with an inventory: grep server code for `time.Now()` and `time.Sleep()`
  to identify every subsystem that needs a clock seam, and estimate retrofit
  scope before writing the interface.
- Introduce the fake clock interface and inject it into timer, reminder, dedupe,
  heartbeat, rotation, expiry, and escalation code paths.
- Replace test sleeps with `Advance`, `AdvanceTo`, and `RunUntilIdle` in
  scenario tests.
- Add simple time-independent actors (tap and voice events, no escalation or
  scheduling logic) in parallel with clock work, so actor and clock
  infrastructure can be tested together.
- Upgrade `T1` from smoke coverage toward simulation coverage.
- Add first automated coverage for `T2` using a spoken/visual reminder scenario.

Acceptance criteria:

- A scenario that represents hours or days of elapsed time completes in seconds.
- Scheduler traces show why a timer, reminder, escalation, or expiry fired.
- `scripts/usecase-validate.sh` is updated to register T1 (upgraded) and T2.

### Phase 4 — Actor Scripts, YAML Loading, and Multi-Use-Case Scenarios

- Add time-dependent actor helpers for missed notifications, dismissal,
  escalation paths, and external-agent API calls.
- Add YAML scenario loading. Before implementing the parser, document the full
  YAML schema including how `says` maps to `VoiceAudio` proto events.
- Migrate scenarios for `T3/T4` and `AA1/AA4` to YAML once the schema is
  stable; keep Go-authored tests for lower-level coverage.
- Add scenario specs for cross-use-case flows:
  - `C2` (already harness-backed from Phase 2; extend to YAML if useful);
  - `M5` camera activity monitoring;
  - `T3` and `T4` school-morning monitoring and escalation;
  - `AA1` through `AA5` automation, monitoring, LLM, scheduling, and vision
    agents.
- Ensure every scenario records actor intent, generated events, server results,
  and terminal observations.

Acceptance criteria:

- Multi-actor scenarios can model realistic user behavior without physical
  devices.
- External agent use cases use the public server API or MCP/REPL surfaces rather
  than privileged test-only hooks.
- At least one YAML scenario passes `make usecase-validate`.
- `scripts/usecase-validate.sh` is updated to register M5, T3, T4, and AA-family
  IDs added in this phase.

### Phase 5 — Evidence Review and CI Integration

- Teach `scripts/usecase-validate.sh --info` and
  `docs/usecase-validation-matrix.md` to include evidence bundle availability.
- Configure CI to upload failed scenario bundles as artifacts automatically, and
  upload `manifest.json` summaries for passing runs. Full bundles for passing
  runs are not uploaded to avoid bloating artifact storage.
- Add a local replay command:

```bash
go test ./internal/usecasevalidation -run TestReplay -args \
  -bundle artifacts/usecase-validation/<run-id>
```

- Add concise `summary.md` output that can be pasted into bug reports or plan
  review comments.
- `scripts/usecase-validate.sh` is updated to register any IDs added or promoted
  to harness coverage in this phase.

Acceptance criteria:

- A failed CI scenario gives enough evidence to reproduce and debug locally.
- Validation matrix entries distinguish narrow automated checks from
  harness-backed simulation/full coverage.
- CI uploads failed bundles and passing manifests without manual intervention.

## Initial Candidate Scenarios

Start with use cases that exercise multiple harness features and expose obvious
user-visible outcomes:

1. **C1 — Intercom (Phase 1 harness target)**
   - Wrap existing transport coverage to emit an evidence bundle.
   - First proof that the harness skeleton works end-to-end.

2. **UI9 — Reconnect while overlay is open (Phase 2 target)**
   - Use simulated terminal reconnect to validate restored main and overlay
     layers with evidence beyond internal assertions.

3. **C2 — Whole-house announcement (Phase 2 target)**
   - Simulate one speaking terminal and three receiving terminals.
   - Verify broadcast routing, terminal observations, event log entries, and no
     duplicate delivery.

4. **T2 — Reminder at a specific time (Phase 3 target)**
   - Simulate voice creation, clock advance to due time, spoken and visual
     reminder, and dismissal.
   - Verify scheduler trace and terminal UI/audio evidence.

5. **T3/T4 — Morning routine escalation (Phase 4 target)**
   - Simulate a recurring school-day schedule, no observed child activity,
     parent notification, and child-room speaker warning after synthetic time
     advances.
   - Verify escalation timing and suppression on non-school days.

6. **M5 — Camera activity watch (Phase 4 target)**
   - Simulate camera activity markers during monitored and unmonitored windows.
   - Verify alert generation only when policy says it should fire.

7. **AA1/AA4 — External automation and scheduling agent (Phase 4 target)**
   - Simulate API calls from an automation agent that creates, modifies, and
     cancels scheduled events.
   - Verify public API behavior, durable state, and terminal-visible outcomes.

## Evidence Quality Requirements

Each scenario assertion should have a stable assertion ID and should emit:

- precondition evidence;
- actor action or injected event;
- server event IDs consumed or produced;
- terminal observations;
- final state or durable store read-back;
- remediation hint when failure is a known class, such as missing route,
  duplicate route, stale clock, dropped capability, unhandled reconnect, or
  missing UI layer.

Passing runs preserve only the `manifest.json` summary by default. Failed runs
must preserve the full evidence bundle automatically.

## Acceptance Criteria

- `make usecase-validate USECASE=<ID>` remains the public validation command.
- At least five previously unautomated use cases are covered by harness-backed
  simulation tests.
- At least one scenario advances synthetic time by more than 24 hours and runs
  in seconds.
- At least one scenario uses three or more simulated terminals.
- At least one scenario uses an external automation or scheduling agent through
  the public API.
- A failing scenario produces a replayable evidence bundle with actor timeline,
  terminal events, server events, scheduler trace, store diffs, assertions, and
  a human-readable summary.
- `docs/usecase-validation-matrix.md` can identify which IDs are backed by this
  harness and link each ID to its primary evidence.

## Decisions Made

- **Harness package**: `terminal_server/internal/usecasevalidation/` — internal
  to the server module, separate from `internal/scenario` to keep validation
  infrastructure distinct from production scenario code.
- **Execution model**: in-process server construction, not a subprocess. Timing
  and logic correctness are the goals; network round-trips add noise without
  value for these tests.
- **Fake clock**: greenfield. No existing clock abstraction covers the relevant
  server subsystems. Inventory `time.Now()` / `time.Sleep()` calls before
  designing the interface.
- **YAML scenario loading**: explicitly deferred to Phase 4. All Phase 1–3
  scenarios are Go-authored.
- **Scenario testdata location**: `terminal_server/internal/usecasevalidation/testdata/`.
- **CI artifact policy**: upload failed bundles in full; upload `manifest.json`
  summaries only for passing runs.
- **First Phase 2 scenarios**: both UI9 and C2, chosen because UI9 has a clear
  existing transport boundary and C2 demonstrates multi-terminal broadcast.

## Open Questions

- Should simulated media events remain symbolic for all CI scenarios, or should
  a small golden audio/video fixture set be added for classifier-level tests?
- Should replay be implemented as a Go test mode first, or as a `term` CLI
  command that can also support manual debugging?
