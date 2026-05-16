---
title: "Use Case Validation Automation"
kind: plan
status: planned
owner: curtcox
validation: none
last-reviewed: 2026-05-16
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
  real running server instance whenever practical.
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

## Design Principles

1. **Real server first** — start the actual server binary or the same in-process
   server construction used by production, then inject deterministic seams for
   time, terminals, IO, and external services.
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

Add a reusable harness under `terminal_server/internal/usecasevalidation` or an
adjacent package name chosen during implementation.

The harness should provide:

- `Harness.StartServer(config)` that starts a real server with test-owned temp
  storage, logs, app registry, package registry, and network/listener strategy.
- `Harness.ConnectTerminal(profile)` that creates simulated terminals over the
  real transport path when possible.
- `Harness.Actor(name, profile)` that can drive a terminal or API client using
  high-level steps.
- `Harness.Clock()` that exposes deterministic `Now`, `Advance`, `RunUntilIdle`,
  and `AdvanceTo` operations.
- `Harness.Expect()` helpers for server state, terminal state, UI state, media
  routes, scheduled jobs, notifications, logs, and durable stores.
- `Harness.Evidence()` that returns the structured evidence bundle for the run.

The preferred execution mode should be a real server instance and real protocol
messages. In-process adapters are acceptable only when they reuse the same
production handlers and command registries as the networked server.

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

### 4. Simulated Clock and Scheduled Work

Introduce or standardize a clock interface for every server subsystem used by
use cases:

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

Every validation run should write a scenario evidence bundle. Keep it small for
passing CI runs and detailed for failures.

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

Support a declarative scenario format for larger use cases, with Go helpers for
lower-level coverage. YAML is a good default because use cases are already
Markdown-based and the repo has generated indexes.

Example shape:

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

The implementation should begin with Go-authored scenarios for speed and type
safety, then add YAML loading once the core actor, terminal, clock, and evidence
interfaces settle.

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
- Add a minimal harness package that can start the server with temp storage,
  structured logging, and deterministic configuration.
- Capture server events and assertion results into a first-version evidence
  bundle.
- Add a `make usecase-validate-artifacts USECASE=<ID>` or environment flag such
  as `USECASE_ARTIFACTS=1` to preserve bundles on demand.

Acceptance criteria:

- Existing automated IDs still pass through `make usecase-validate USECASE=all`.
- At least one existing scenario test emits a useful evidence bundle without
  changing production behavior.

### Phase 2 — Simulated Terminal Transport

- Build simulated terminals that connect through the real server transport path
  where possible.
- Model registration, capabilities, heartbeat, input events, reconnect, and
  received UI/control messages.
- Convert one existing transport-heavy use case, such as `C1`, `M3`, or `UI9`,
  into a harness-backed scenario while retaining the narrower transport tests.

Acceptance criteria:

- A two-terminal scenario can prove both the server-side route and each
  terminal's observed result.
- Failure output includes both wire-level events and terminal-level observations.

### Phase 3 — Simulated Clock

- Standardize a clock abstraction for timers, reminders, dedupe, heartbeat,
  rotation, expiry, and escalation code paths.
- Replace test sleeps with `Advance`, `AdvanceTo`, and `RunUntilIdle` in
  scenario tests.
- Upgrade `T1` from smoke coverage toward simulation coverage.
- Add first automated coverage for `T2` using a spoken/visual reminder scenario.

Acceptance criteria:

- A scenario that represents hours or days of elapsed time completes in seconds.
- Scheduler traces show why a timer, reminder, escalation, or expiry fired.

### Phase 4 — Actor Scripts and Multi-Use-Case Scenarios

- Add actor helpers for voice, taps, menu choices, missed notifications,
  dismissal, and external-agent API calls.
- Add scenario specs for cross-use-case flows, especially:
  - `C2` whole-house announcement;
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

### Phase 5 — Evidence Review and CI Integration

- Teach `scripts/usecase-validate.sh --info` and
  `docs/usecase-validation-matrix.md` to include evidence bundle availability.
- Publish failed scenario bundles as CI artifacts.
- Add a local replay command, for example:

```bash
go test ./internal/usecasevalidation -run TestReplay -args \
  -bundle artifacts/usecase-validation/<run-id>
```

- Add concise `summary.md` output that can be pasted into bug reports or plan
  review comments.

Acceptance criteria:

- A failed CI scenario gives enough evidence to reproduce and debug locally.
- Validation matrix entries distinguish narrow automated checks from
  harness-backed simulation/full coverage.

## Initial Candidate Scenarios

Start with use cases that exercise multiple harness features and expose obvious
user-visible outcomes:

1. **C2 — Whole-house announcement**
   - Simulate one speaking terminal and three receiving terminals.
   - Verify broadcast routing, terminal observations, event log entries, and no
     duplicate delivery.

2. **T2 — Reminder at a specific time**
   - Simulate voice creation, clock advance to due time, spoken and visual
     reminder, and dismissal.
   - Verify scheduler trace and terminal UI/audio evidence.

3. **T3/T4 — Morning routine escalation**
   - Simulate a recurring school-day schedule, no observed child activity,
     parent notification, and child-room speaker warning after synthetic time
     advances.
   - Verify escalation timing and suppression on non-school days.

4. **M5 — Camera activity watch**
   - Simulate camera activity markers during monitored and unmonitored windows.
   - Verify alert generation only when policy says it should fire.

5. **AA1/AA4 — External automation and scheduling agent**
   - Simulate API calls from an automation agent that creates, modifies, and
     cancels scheduled events.
   - Verify public API behavior, durable state, and terminal-visible outcomes.

6. **UI9 — Reconnect while overlay is open**
   - Use simulated terminal reconnect to validate restored main and overlay
     layers with evidence beyond internal assertions.

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

Passing runs may keep only summary evidence by default. Failed runs must preserve
full evidence automatically.

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

## Open Questions

- Should scenario specs live beside use case files (`usecases/scenarios/`) or in
  server testdata (`terminal_server/internal/usecasevalidation/testdata/`)?
- Should CI upload passing evidence bundles for all scenarios, or only failed
  bundles plus sampled passing summaries?
- Should simulated media events remain symbolic for all CI scenarios, or should
  a small golden audio/video fixture set be added for classifier-level tests?
- Should replay be implemented as a Go test mode first, or as a `term` CLI
  command that can also support manual debugging?
