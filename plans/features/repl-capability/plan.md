---
title: "REPL Capability Plan"
kind: plan
status: building
owner: copilot
validation: automated:AA6
last-reviewed: 2026-04-27
---

# REPL Capability Plan

See [masterplan.md](../../../masterplan.md) for overall system context.
Extends [repl-and-shell.md](../repl-and-shell.md) (base REPL) and
[application-runtime.md](../application-runtime.md) (TAR/TAL runtime).
Supersedes the earlier `repl-authoring-capabilities.md` and
`Capability-plans.md` by merging their content into a single plan.

This document is the umbrella plan for closing the REPL and
runtime's capability surface against
[usecases.md](../../../usecases.md) and
[plato_inspired_usecases.md](../plato-inspired-usecases.md).
Detailed designs live in the per-family plans listed under
**Component Plans** below; this document owns the overall shape,
layering, acceptance rules, and authoring substrate.

## Progress (2026-04-26)

- Phase 1 slice shipped in code: `store put` now accepts optional
  `--ttl <duration>` (REPL + admin API + capability service), and
  expired records are pruned lazily on `store get` / `store ls`.
- Added focused coverage in capability, admin, and REPL tests.
- Completed the remaining Phase 1 scope from this plan:
  `store ns ls`, `store del`, `store watch`, and `store bind`
  (REPL + admin API + capability service), plus filtered
  `bus tail` and windowed `bus replay`.
- Phase 2 cohort slice shipped in code: named `cohort` CRUD
  (`cohort ls|show|put|del`) is now wired through REPL, admin API,
  and capability service, with dynamic member resolution against live
  device metadata selectors.
- Added focused coverage for capability service cohort semantics,
  admin cohort endpoints, and REPL command routing.
- Landed the first Phase 2 UI authoring slice in code: authored
  view inventory is now REPL-visible via `ui views ls/show/rm`
  (capability service + admin API + REPL command surface + docs).
- Landed the active Phase 2 UI operations slice in code:
  `ui push`, `ui patch`, `ui transition`, `ui broadcast`,
  `ui subscribe`, and `ui snapshot` are now wired through
  capability service + admin API + REPL command surface + docs.
- Landed the Phase 3 routing slice in code: `handlers ls`,
  `handlers on` (`--run` and `--emit` targets), and `handlers off`
  are now wired through capability service + admin API + REPL
  command surface + docs, with focused tests.
- Landed the Phase 4 inline authoring slice in code:
  `scenarios ls`, `scenarios show`, `scenarios define`, and
  `scenarios undefine` are now wired through capability service +
  admin API + REPL command surface + docs, with focused tests.
- Landed the first Phase 5 simulation/scripting slice in code:
  `sim device new|rm`, `sim input`, `sim ui`, and
  `scripts dry-run` are now wired through capability service +
  admin API + REPL command surface + docs, with focused tests.
- Landed the remaining Phase 5 assertion/execution slice in code:
  `sim expect`, `sim record`, and `scripts run` are now wired
  through capability service + admin API + REPL command surface +
  docs, with focused tests.
- Closed the remaining substrate example-topic documentation gap for
  acceptance: added `red-alert-broadcast`, `timer-and-reminder`, and
  `presence-query` example guides under `docs/repl/examples/`, and
  tightened REPL tests so `docs examples` must continue surfacing all
  required capability-closure topics (including `sim-only-assertion`).
- Closed the remaining Layer 2 required-example placeholder gap for
  acceptance: replaced stub docs for `start-room-chat`,
  `send-direct-message`, `pin-family-bulletin`,
  `remote-help-session`, `shared-lesson-session`,
  `annotate-shared-canvas`, `review-learner-progress`, and
  `resume-multiplayer-session` with concrete typed REPL walkthroughs.
- Landed the first Phase 11 bug-reporting control-plane slice in code:
  REPL `bug ls|show|file|confirm` now routes through the existing
  bug-reporting admin APIs, with command docs and focused REPL/MCP
  coverage to preserve catalog safety constraints.
- Landed the first Phase 12 cross-use-case validation slice in code:
  `scripts run` now executes a deterministic simulation fixture over
  `sim`/`store`/`ui`/`bus`, with AA6 wired into
  `make usecase-validate USECASE=AA6` and reflected in the validation
  matrix.
- Expanded the Phase 12 AA6 fixture to include a Layer 2 domain-family
  command (`message rooms`) so `scripts run` now validates both
  substrate (`sim`/`store`/`ui`/`bus`) and Layer 2 command routing in
  one deterministic scripted path.
- Extended the Phase 12 AA6 fixture with a mutating Layer 2 command
  path (`message post phase12-room fixture-layer2-mutating`) plus
  durable read-back (`message ls phase12-room` and admin assertion)
  so `scripts run` now verifies persisted Layer 2 side effects in
  addition to command routing.
- Extended the Phase 12 AA6 fixture with a second mutating Layer 2
  command path outside `message` (`board post phase12-board
  fixture-board-mutating`) plus deterministic read-back (`board ls`
  in-script and admin `/board` assertion) so `scripts run` now covers
  mutating validation across multiple Layer 2 domain families.
- Extended the Phase 12 AA6 fixture with an additional mutating Layer 2
  path in the artifact domain (`artifact create lesson
  fixture-artifact-mutating`) plus deterministic read-back
  (`artifact history art-1` in-script and admin `/artifact/history`
  assertion) so `scripts run` now validates durable artifact mutation
  in the same cross-use-case scripted path.
- Extended the Phase 12 AA6 fixture with an additional mutating Layer 2
  path in the canvas domain (`canvas annotate phase12-canvas
  fixture-canvas-mutating`) plus deterministic read-back
  (`canvas ls phase12-canvas` in-script and admin `/canvas` assertion)
  so `scripts run` now validates durable canvas mutation alongside the
  existing message/board/artifact checks.
- Extended the Phase 12 AA6 fixture with an additional mutating Layer 2
  path in the session domain (`session create lesson phase12-session`
  and `session join latest fixture-session-member`) plus deterministic
  read-back (`session members latest` in-script and admin
  `/session/members` assertion) so `scripts run` now validates durable
  session membership mutation alongside the existing
  message/board/artifact/canvas checks.
- Extended the Phase 12 AA6 fixture with an additional mutating Layer 2
  path in the identity domain (`identity ack record
  phase12-identity-subject --actor person:fixture-identity --mode
  confirmed`) plus deterministic read-back (`identity ack show
  phase12-identity-subject` in-script and admin `/identity/ack`
  assertion) so `scripts run` now validates durable identity
  acknowledgement mutation alongside the existing
  message/board/artifact/canvas/session checks.
- Extended the Phase 12 AA6 fixture with an additional mutating Layer 2
  path in the memory domain (`memory remember phase12-memory
  fixture-memory-mutating`) plus deterministic read-back
  (`memory recall fixture-memory-mutating` in-script and admin
  `/memory` assertion) so `scripts run` now validates durable memory
  mutation alongside the existing
  message/board/artifact/canvas/session/identity checks.
- Landed the next Phase 11 bug-reporting control-plane extension in
  code: REPL `bug tail` now tails `/admin/logs.jsonl` with a
  bug-report event filter prefix (`bug.report`), with focused REPL
  coverage and command docs updates.

## Problem

The existing runtime and REPL plans cover orchestration-heavy
scenarios well: media routing, observation, scheduling, device
placement, telephony, PTY-backed terminal access, and AI-assisted
operator workflows. Two gaps remain.

**Gap A — authoring substrate.** Building the chat use case required
Go edits across seven files in five packages (`chat/room.go`,
`scenario/chat.go`, `scenario/register.go`, `ui/descriptor.go`,
`transport/chat_wire.go`, `transport/control_stream.go`,
`admin/server.go`). None of those edits carry intrinsically
chat-specific architecture. They are generic capabilities — keyed
TTL store, UI composition, input routing, broadcast, out-of-band
trigger — that every new scenario currently re-implements in Go.

**Gap B — domain capabilities.** Collaborative, durable-content,
identity-aware, and searchable-memory use cases (room chat, DMs,
boards, lessons, shared canvases, household memory) do not map to
any existing typed service. Each such use case currently has to
invent its own schema in `store` plus custom UI, which is how
Gap A metastasizes.

The two gaps reinforce each other: without substrate, every domain
capability is re-authored per scenario; without domain capabilities,
the substrate is asked to do things that should be owned by a typed
service.

## Design Principle

Every application capability required by a supported use case must
exist in five aligned forms:

1. a TAL host module,
2. a typed control-plane service,
3. a REPL command surface,
4. in-REPL documentation and examples,
5. simulation and test support.

No use case is considered supported unless a human or agent can
exercise the required capability through the REPL using typed
requests and responses. This means:

- no hidden-only capability available to apps but not visible in REPL,
- no app-only service that bypasses the typed command surface,
- no REPL command that shells out to host state instead of
  authoritative kernel services.

The REPL remains a typed control-plane shell. This plan does **not**
turn it into a Unix shell or expose arbitrary host capabilities.

## Layering

The capability surface is organized into two layers.

**Layer 1 — Authoring Substrate.** Generic primitives every scenario
reuses: keyed TTL state, typed bus, UI view composition, input
routing, device cohorts, inline scenarios, simulation, scripted
execution.

**Layer 2 — Domain Capabilities.** Typed services for substrates
that a use case family needs in durable, first-class form:
identity/audience, collaborative sessions, messaging/boards,
shared artifacts/canvases, search, memory, bug reporting.

Layer 2 services are implemented *using* Layer 1 primitives
internally, but they expose their own typed APIs and are the
canonical entry point whenever a use case fits their domain. A use
case that has no Layer 2 fit (red-alert broadcast, timer/reminder,
presence-query, PA announcement) composes Layer 1 primitives
directly.

```
┌────────────────────────────────────────────────────────────┐
│  Layer 2 — Domain Capabilities                             │
│    identity │ session │ message │ board │ artifact │       │
│    canvas   │ search  │ memory  │ bug                       │
├────────────────────────────────────────────────────────────┤
│  Layer 1 — Authoring Substrate                             │
│    store │ bus │ ui (authoring) │ devices cohort │         │
│    handlers │ scenarios (inline) │ sim │ scripts           │
├────────────────────────────────────────────────────────────┤
│  Existing kernel services                                  │
│    placement │ claims │ flow │ observe │ recent │ presence │
│    world │ scheduler │ telephony │ pty │ ai │ http │ log   │
└────────────────────────────────────────────────────────────┘
```

`store`, `bus`, `placement`, and `recent` already exist partially in
the runtime but are implicit or internal; this plan promotes them to
first-class typed services with explicit REPL groups.

---

## Layer 1 — Authoring Substrate

### Capability Gap Analysis

Each row names a capability the chat implementation needed, the
broad set of use cases that require the same thing, and the nearest
existing surface.

| # | Capability | Use cases needing it | Existing surface | Gap |
|---|------------|----------------------|------------------|-----|
| G1  | Keyed in-memory state with TTL | M1, M2, AH13, AH14, AH15, T1, T2, chat | ad-hoc Go maps (e.g. `chat.Room`) | No REPL surface to create, read, write, or expire structured keyed state without a Go type. |
| G2  | Per-device, per-scenario mutable state | P2, AH12, AH5, chat | scenario-specific Go structs | No REPL surface for "bind a value to (device, scenario) and retrieve it later." |
| G3  | Compose and publish a UI view | every UI-bearing scenario; esp. S1, D1, C5, chat | hand-written `ui.Descriptor` builders | No REPL surface to declare a view, bind it to a root id, and push it to a device. |
| G4  | Apply a targeted UI patch | M3, M4, D2, C1, chat | `UpdateUI{ComponentID, Node}` wrapped in transport helpers | No REPL command to emit an `UpdateUI` to one or many devices from a typed tree expression. |
| G5  | Register an input/event handler | every interactive scenario; chat | `handleInput` switch in `control_stream.go` | No REPL surface to register "when device X's component Y fires action Z, run command W." |
| G6  | Broadcast a UI message to a device set | C2, C3, M3, S1, AO3, chat | `RelayToDeviceID` + `globalSessionRelayRegistry` | No REPL command to fan out an arbitrary server message to a named device cohort. |
| G7  | Out-of-band trigger emission | AA1, AA2, AA4, B3, chat (admin post) | ad-hoc `/admin/...` HTTP handlers | No REPL command that emits a typed `Intent`/`Event` onto the bus as if it came from voice/UI/webhook. |
| G8  | Virtual/simulated devices and inputs | AA6, B3, I8, test harness | integration-test helpers in Go | No REPL surface to register a fake device, deliver synthetic input, or observe outbound messages to it. |
| G9  | Define a scenario without a TAR package | I10, chat, all prototyping | full TAR app on disk; `engine.Register` in Go | No REPL surface for an inline scenario with `match` intents and `on start/stop/input/event` actions. |
| G10 | Inspect and unregister runtime-defined artifacts | ops discipline | nothing — Go-defined things are immortal in process | No REPL surface that lists REPL-authored scenarios/views/handlers/stores and safely removes them. |
| G11 | Assertions and scripted verification | I8, AA6, every end-to-end test | Go test files | No REPL command surface for "expect device X to receive Y within N ms" usable both interactively and from a script. |
| G12 | Cross-session attachment to scenario UI | P2, C6, chat history to late joiner | ad-hoc broadcast wiring | No general "subscribe this session/device to that scenario's UI stream" command. |

G1–G8 are the capabilities that get rebuilt *every* time a new
scenario is added. G9–G12 are the capabilities that make the first
set reachable and testable from inside the REPL rather than through Go.

### Substrate Command Groups

Each command classifies as `read_only | operational | mutating` for
the approval pipeline described in the base REPL plan.

#### `store` — keyed state with TTL (G1, G2)

```text
store ns ls                                      read_only
store put <ns> <key> <value> [--ttl 24h]         mutating
store get <ns> <key>                             read_only
store del <ns> <key>                             mutating
store ls <ns> [--prefix <p>]                     read_only
store watch <ns> [--prefix <p>]                  operational
store bind <ns> <key> --to <device>:<scenario>   mutating
```

Values are structured (JSON-compatible) and versioned. TTL is
per-record and enforced by the store, not by callers.

#### `ui` — view and patch authoring (G3, G4, G6, G10, G12)

```text
ui show <device> [--root <id>]                              read_only
ui push <device> <descriptor-expr> [--root <id>]            mutating
ui patch <device> <component-id> <descriptor-expr>          mutating
ui transition <device> <component-id> <transition-expr>     mutating
ui broadcast <cohort> <descriptor-expr> [--patch <id>]      mutating
ui subscribe <device> --to <activation|cohort>              mutating
ui snapshot <device> [--format json]                        read_only
ui views ls                                                 read_only
ui views show <view-id>                                     read_only
ui views rm <view-id>                                       mutating
```

A `descriptor-expr` is a small literal form over the closed UI
primitive set — no new primitives, only a typed way to compose them
from text. `ui transition` is the authoring form of the existing
`TransitionUI` primitive from
[server-driven-ui.md](../server-driven-ui.md). Authored views are
first-class records: `ui push` either publishes inline or binds to
a named view-id, and `ui views ls/show/rm` lets operators inventory
and remove REPL-authored views (satisfies the G10 listability
promise for UI artifacts). The REPL holds a single reusable
descriptor builder that the TAL `ui` module, `ui push`, and
internal scenarios all share.

#### `devices cohort` — named device sets (G6)

```text
devices cohort ls                                           read_only
devices cohort show <name>                                  read_only
devices cohort put <name> --where <selector>                mutating
devices cohort put <name> --ids <id>,<id>,...               mutating
devices cohort del <name>                                   mutating
```

`<selector>` reuses the placement selector grammar plus a
`scenario=<name>` predicate, so cohorts like "everyone currently
in chat" are expressible without code.

#### `bus` — typed intent/event emission (G7)

```text
bus emit intent <action> [slots...] [--scope device=<id>]   mutating
bus emit event <kind> [attrs...] [--subject <s>]            mutating
bus tail [--kind <k>] [--source <s>]                        operational
bus replay <from> <to> [--filter ...]                       operational
```

`bus emit` replaces one-off `/admin/api/...` endpoints with a single
typed path.

#### `handlers` — input/event routing (G5)

```text
handlers ls                                                 read_only
handlers on <selector> <action> --run <command>             mutating
handlers on <selector> <action> --emit intent <action> ...  mutating
handlers off <handler-id>                                   mutating
```

`<selector>` matches on `scenario=<name>`, `component=<id>`, and/or
`device=<id>`. Handlers are first-class records with IDs, listable,
disableable, and logged when they fire.

#### `scenarios` — inline definition (G9, G10)

```text
scenarios ls                                                read_only
scenarios show <name>                                       read_only
scenarios define <name> --match <intent-list>
                        [--match event=<kind>]...
                        [--priority normal]
                         [--on-start <command>]
                         [--on-input <handler-spec>]
                         [--on-event <kind> <command>]...
                         [--on-suspend <command>]
                         [--on-resume <command>]
                         [--on-stop <command>]              mutating
scenarios undefine <name>                                   mutating
```

Inline scenarios reach lifecycle parity with the TAR path against
the actual scenario-engine interface: `ScenarioDefinition.Match`
(driven by `--match` intent and event predicates),
`ScenarioDefinition.NewActivation` (driven by the triggering
`ActivationRequest`), and `ScenarioActivation.Start/Stop/Suspend/
Resume` (driven by the corresponding `--on-*` hooks). `--on-input`
and `--on-event` bind incoming user input and bus events to
command fragments scoped to a live activation; they are not new
interface methods, they are inline-scenario routing over the
existing trigger model. Not TAL — it composes existing REPL
commands into lifecycle hooks. A real app still graduates to a
TAR package on disk; inline scenarios exist for prototyping,
demos, and ops-level one-offs. The scenario engine supervisor
treats Go-defined, TAR-package, and inline scenarios identically.

#### `sim` — virtual devices and scripted verification (G8, G11)

```text
sim device new <id> [--caps display,keyboard]               mutating
sim device rm <id>                                          mutating
sim input <id> <component-id> <action> [<value>]            mutating
sim ui <id>                                                 read_only
sim expect <id> ui contains <expr> [--within 2s]            operational
sim expect <id> message <selector> [--within 2s]            operational
sim record <id> [--duration 30s]                            operational
sim script <path>                                           operational
```

`sim` devices participate in the full registry, placement, claims,
and transport path; they differ from real devices only in that
their IO is captured instead of forwarded.

#### `scripts` — non-interactive REPL execution

```text
scripts run <path> [--json]                                 operational
scripts dry-run <path>                                      read_only
```

Runs newline-delimited REPL commands with ordinary classification
semantics. CI invokes the REPL this way for use-case validation.

### Substrate Typed Services

- `StoreService` — TTL, namespaces, watch, scoped binding.
- `UiService` — `Push`, `Patch`, `Transition`, `Broadcast`,
  `Subscribe`, `Snapshot`, plus authored-view inventory
  (`ListViews`, `GetView`, `RemoveView`). Authored-view records
  are REPL-side authoring metadata (who published what, under
  which view-id), not a change to the `server-driven-ui.md`
  primitive contract.
- `BusService` — `Emit`, `Tail`, `Replay`.
- `HandlerService` — `Register`, `Unregister`, `List`, `Trigger`.
- `CohortService` — device set CRUD plus live membership evaluation.
- `ScenarioAuthoringService` — inline scenario CRUD, layered above
  the existing scenario engine `Register`.
- `SimService` — virtual device registration, input injection,
  output capture, assertion evaluation.

All of these sit under the existing REPL command-registry pipeline
(classification, approval, streaming dispatch) and are equally
available to MCP origins per
[agent-delegation.md](../agent-delegation.md).

### Substrate Worked Example — Red-Alert Broadcast Without Go Changes

A red-alert-like scenario (M3, M4) is the canonical primitives-only
demo because it has no Layer 2 domain fit — it is pure fan-out plus
acknowledgement, not durable messaging.

```text
devices cohort put all_screens --where capability=display
scenarios define red_alert --match intent="red alert",intent="all hands" \
    --on-start 'ui broadcast all_screens $(alert_banner $reason)' \
    --on-stop  'ui broadcast all_screens $(clear_banner)'
handlers on scenario=red_alert component=alert_ack submit \
    --run 'store put alert.ack $device true --ttl 1h; \
           ui patch $device alert_banner $(ack_confirmation)'
```

No file under `internal/transport/` or `internal/scenario/`.
This is the pattern AH8 (smoke alarm), M3 (red alert), AO3 (PA
announcement), and the timer/reminder family all need.

---

## Layer 2 — Domain Capabilities

Each domain family is specified in its own plan; this document owns
the overall shape and the invariant that every family conforms to
the five-form rule (TAL module, typed service, REPL group, docs,
sim).

### Capability Families and Component Plans

| Family | Component plan | TAL module | Service | REPL groups |
|---|---|---|---|---|
| Identity, groups, audiences, ack state | [identity-and-audience.md](../identity-and-audience.md) | `identity` | `IdentityService` | `identity` |
| Generalized interactive sessions | [collab-sessions.md](../collab-sessions.md) | `session` | `InteractiveSessionService` | `session` |
| Rooms, DMs, boards, bulletins, threads | [messaging-and-boards.md](../messaging-and-boards.md) | `message` | `MessagingService` | `message`, `board` |
| Durable shared artifacts, canvases, annotations | [shared-artifacts.md](../shared-artifacts.md) | `artifact` | `ArtifactService` | `artifact`, `canvas` |
| Unified search, timeline, household memory | [search-and-memory.md](../search-and-memory.md) | `search`, `memory` | `SearchService`, `MemoryService` | `search`, `memory` |
| Bug reporting and diagnostics | [bug-reporting.md](../bug-reporting.md) | `bug` | `BugReportService` | `bug` |

#### Acknowledgement ownership

`IdentityService` is the canonical owner of acknowledgement state.
`Acknowledgement` records live on `IdentityService` keyed by
subject reference (message id, bulletin id, alert id, artifact id,
…) and actor. Other L2 services that expose ack operations —
`MessagingService.AcknowledgeSubject`, `MessagingService.ListUnread`,
bulletin-pin ack in `ArtifactService`, alert ack in the monitoring
flows — are thin helpers that delegate to
`IdentityService.RecordAcknowledgement` /
`IdentityService.GetAcknowledgements` and expose convenience
filters on top. Ack semantics (modes `seen`/`read`/`heard`/
`dismissed`/`confirmed`, audience resolution, durability) are
defined once in [identity-and-audience.md](../identity-and-audience.md).
Actor references are a discriminated union over
`person`/`device`/`agent`/`anonymous` kinds (full definition in
[identity-and-audience.md](../identity-and-audience.md)); this plan
relies on that union so that kiosk taps, automated-agent acks, and
off-device SIP/webhook acks all land through the same typed path.
This resolves the boundary blur between identity and messaging.

#### Search taxonomy

`timeline`, `related`, and `recent` are subcommands of `search`
(e.g., `search timeline --since 24h`), not top-level REPL groups.
If [search-and-memory.md](../search-and-memory.md) examples imply a
top-level `timeline` group, read them as `search timeline` for the
purposes of this umbrella.

Revised base documents that Layer 2 touches directly:

- [application-runtime.md](../application-runtime.md) — adds the
  `identity`, `session`, `message`, `artifact`, `search`, `memory`
  TAL host modules and their permission model.
- [repl-and-shell.md](../repl-and-shell.md) — adds the corresponding
  REPL groups alongside the existing
  `devices`/`activations`/`claims`/`ui`/`flow`/`observe`/`presence`/
  `world`/`scheduler`/`app`/`logs`/`telephony`/`ai`/`docs` set.

### Use-Case Coverage Matrix

| Use-case family | TAL modules | REPL groups | Status target |
|---|---|---|---|
| Intercom / PA / calling | placement, claims, flow, telephony, ui, ai | devices, flow, telephony, activations | already covered |
| Voice assistant | ai, ui, placement, bus | ai, activations, logs | already covered |
| Monitoring / alerts | flow, observe, recent, scheduler, ui, bus | observe, recent, scheduler, logs | already covered |
| Broadcast fan-out (red alert, PA, smoke) | ui, bus, store, handlers, cohorts | ui, bus, handlers, devices | Layer 1 only |
| Messaging / boards | identity, message, search | identity, message, board, search | Layer 2 (new) |
| Shared help / co-control | session, identity, ui | session, identity, activations | Layer 2 (new) |
| Lessons / guided practice | session, artifact, scheduler, identity, search, ai | session, artifact, scheduler, identity, search, ai | Layer 2 (new) |
| Shared canvas / symbols | artifact, session, ui | artifact, canvas, session | Layer 2 (new) |
| Multiplayer games | session, identity, artifact or store | session, identity, artifact | Layer 2 (new) |
| Household knowledge / memory | search, memory, message, artifact | search, memory, board, artifact | Layer 2 (new) |
| Bug reporting and diagnostics (B1–B5) | bug, identity, observe | bug, identity, observe | Layer 2 (new, via [bug-reporting.md](../bug-reporting.md)) |

#### Out-of-scope use cases

The following use-case families are *not* closed by this plan
because they are not application/scenario capabilities. They are
infrastructure, protocol, or developer-tooling concerns whose
authoritative home is elsewhere in the masterplan:

- **I1, I2, I3, I11** — device discovery, connection lifecycle,
  capability handshake, reconnect/restore. Covered by
  [discovery.md](../discovery.md), [protocol.md](../protocol.md), and
  [transport-multiplexing.md](../transport-multiplexing.md), not by
  typed scenario capabilities.
- **I8** — CI / `make all-check` build-quality gate. Covered by
  [ci.md](../ci.md); the REPL `scripts` + `sim` surfaces are
  *consumed* by CI but do not replace the repo-wide quality gate.
- **I9** — developer tooling and documentation conventions.
  Covered by [../CLAUDE.md](../../../CLAUDE.md) (project-wide agent and
  contributor rules) and [agent-config.md](../agent-config.md)
  (agent-facing configuration conventions). No typed runtime
  service maps to it.

These remain in scope for the project overall; they are excluded
from the REPL-closure acceptance rule below.

### Canonical Chat Recipe

Chat is built on `MessagingService`, not on raw Layer 1 primitives.
A room is a durable `MessageRoom`; messages live in typed
`Message` records with threading, unread state, acknowledgement,
and timeline/search integration provided by the service. The UI
scenario reduces to: pick a room, subscribe to its message stream,
render, and route input back to `message.post`.

Why not compose chat from `store` + `handlers` + `cohorts`: doing
so skips durable room objects, threading, unread state, ack state,
audience policies, retention, and search integration — exactly the
substrate `MessagingService` exists to own. Re-implementing that
per scenario is the Gap B failure mode this plan exists to close.

The authoring substrate is the right tool when no Layer 2 fit
exists (red-alert broadcast, timer + reminder, presence-query,
PA announcement), *or* as the implementation path inside a Layer 2
service itself. It is not the right tool for building chat above
the typed service.

---

## Interactions With Existing Plans

- **[scenario-engine.md](../scenario-engine.md)** — `scenarios define`
  is an additional factory path into the existing engine.
  Runtime-defined scenarios use the same
  `ScenarioDefinition`/`ScenarioActivation` interfaces; they are
  not a parallel lifecycle. `Start/Stop/Suspend/Resume` still go
  through the engine's supervisor.
- **[application-runtime.md](../application-runtime.md)** — TAR/TAL
  remains the authoring path for durable applications. Inline
  `scenarios define` is the cheap prototyping path; graduation to
  a TAR package is a copy-out, not a rewrite, because both targets
  use the same typed service surface.
- **[server-driven-ui.md](../server-driven-ui.md)** — `ui push/patch/
  broadcast` adds no new primitives. The closed UI contract is
  unchanged; what changes is who can compose primitives (now: the
  REPL, not only hand-written Go).
- **[repl-and-shell.md](../repl-and-shell.md)** — this document adds
  command groups alongside the existing set. Classification
  metadata, approval pipeline, streaming dispatch, and AI tool-use
  mediation all apply to the new groups exactly as to the existing
  ones.
- **[agent-delegation.md](../agent-delegation.md)** — every new
  command is usable from MCP origins with the same approval model.

## Acceptance Criteria

- A user can implement a basic multi-device UI scenario — identity,
  input, log, fan-out, retention — using only REPL commands,
  without Go recompilation, and without new UI primitives.
- Every **application/scenario** use case in
  [usecases.md](../../../usecases.md) and
  [plato_inspired_usecases.md](../plato-inspired-usecases.md) maps
  cleanly to REPL-visible typed capabilities. Infrastructure,
  protocol, and developer-tooling use cases (see
  §"Out-of-scope use cases" above) are explicitly excluded from
  this clause; their closure lives in the plans named there.
- No required use case depends on an app-only service that lacks
  REPL visibility.
- REPL and TAL expose the same capability families at the same
  conceptual granularity.
- Every new command group is classified
  (`read_only | operational | mutating`) and honored by the
  existing approval pipeline.
- All REPL-authored artifacts (stores, cohorts, handlers, views,
  scenarios, sim devices) are listable, inspectable, and removable
  via their group's `ls`/`show`/`rm`/`undefine` commands.
- `sim` produces reproducible runs: the same script yields the
  same captured output on a clean server, and `sim expect` exits
  non-zero on violation.
- An inline scenario authored with `scenarios define` and a
  scenario authored as a TAR package are indistinguishable to the
  scenario engine's supervisor (same lifecycle, same claim
  behavior, same suspend/resume semantics).
- Every new command is reachable from MCP origins with the same
  approval model as human REPL input.
- Documentation (`docs open api/UiService`, `docs open
  api/MessagingService`, …) is generated from service metadata for
  each new service, plus hand-authored usage guides under
  `docs/repl/commands/` and examples under `docs/repl/examples/`.
- Required new example topics: `start-room-chat`,
  `send-direct-message`, `pin-family-bulletin`,
  `remote-help-session`, `shared-lesson-session`,
  `annotate-shared-canvas`, `search-household-memory`,
  `review-learner-progress`, `resume-multiplayer-session`,
  plus the substrate-only examples: `red-alert-broadcast`,
  `timer-and-reminder`, `presence-query`, `sim-only-assertion`.
- Each new capability is usable in simulation: create and join
  sessions, post/read/ack messages, create and patch artifacts,
  run search queries against seeded data, validate audience
  resolution against fixture identities.

## Implementation Order

Layer 1 lands first because Layer 2 services are built on top of it.

### Phase 1 — `store` and `bus`

Typed TTL store service exposed to both TAL and REPL. Typed bus
service with `Emit`/`Tail`. Lowest-level gaps (G1, G7); unblock the
rest.

### Phase 2 — `ui` authoring and `devices cohort`

`UiService` with `Push/Patch/Transition/Broadcast/Subscribe/Snapshot`
plus authored-view inventory (`ListViews/GetView/RemoveView`).
Cohort CRUD backed by live selector evaluation. At the end of this
phase, a REPL session can drive a screen without Go changes as
long as inputs are ignored.

### Phase 3 — `handlers`

Input and event routing with classification-aware `--run` execution.
First point at which interactive scenarios are buildable from the
REPL.

### Phase 4 — `scenarios define` / `undefine`

Inline scenario authoring. Reuses Phase 1–3 primitives for lifecycle
hooks; no parallel engine. Includes an explicit rollback path for
failed definitions.

### Phase 5 — `sim` and `scripts`

Virtual devices, input injection, output capture, assertions,
scripted execution. Ties prior phases to CI (I8) and to the
agent-driven test surface (AA6).

### Phase 6 — Identity and audience

`IdentityService`, TAL `identity` module, REPL `identity` group.
See [identity-and-audience.md](../identity-and-audience.md).

### Phase 7 — Generalized sessions

`InteractiveSessionService`, TAL `session` module, REPL `session`
group. See [collab-sessions.md](../collab-sessions.md).

### Phase 8 — Messaging and boards

`MessagingService`, TAL `message` module, REPL `message` and `board`
groups. Canonical chat recipe ships here. See
[messaging-and-boards.md](../messaging-and-boards.md).

### Phase 9 — Shared artifacts and canvases

`ArtifactService`, TAL `artifact` module, REPL `artifact` and
`canvas` groups. See [shared-artifacts.md](../shared-artifacts.md).

### Phase 10 — Search and memory

`SearchService` and (optional) `MemoryService`, TAL `search`/`memory`
modules, REPL `search`/`memory` groups. See
[search-and-memory.md](../search-and-memory.md).

### Phase 11 — Bug reporting

Promote [bug-reporting.md](../bug-reporting.md) to the five-form rule.
**Depends on all five rollout phases of `bug-reporting.md`** —
core pipeline, client context capture, on-device entry points,
third-party reporting, and autodetection/dead-device fallback —
because B1–B5 coverage is only complete once the SIP bug line,
email-in adapter, NFC tag, and autodetect paths from that plan's
phase 5 are in. This phase does not rebuild that pipeline, it
grafts a typed control-plane surface onto it. Adds the typed
`BugReportService` (`File`, `Get`, `List`, `Confirm`), TAL `bug`
host module, and REPL `bug` group (`bug file`, `bug ls`,
`bug show`, `bug confirm`) so that humans and agents can exercise
B1–B5 end-to-end through the same typed control-plane surface.
Acknowledgement on bug reports delegates to `IdentityService` per
the ack-ownership rule.

**Proposed extensions to `bug-reporting.md`**, added here rather
than in the component plan so the contract lives with intake:
`BugReportService.Attach(report_id, blob)` to add
screenshots/audio/logs after initial filing (distinct from the
inline `screenshot_png`/`audio_wav` fields already in the wire
contract); and a `bug tail` operational command / `BusService`
subscription filtered to `bug.report.*` events, equivalent to the
`/admin/logs` live-tail the bug-reporting plan names as an open
question. Both graft cleanly onto the existing durable store and
event-log pipeline.

### Phase 12 — Documentation, examples, and cross-use-case validation

Generated docs for each new service. Worked examples for each
use-case family. Cross-use-case validation via `scripts run` over a
seeded simulation fixture.

## Related Plans

- [repl-and-shell.md](../repl-and-shell.md) — base REPL: sessions,
  classification, approval pipeline, AI assistance.
- [application-runtime.md](../application-runtime.md) — TAR/TAL
  durable authoring; uses the same typed services this plan
  extends.
- [scenario-engine.md](../scenario-engine.md) — supervisor for both
  Go-defined and REPL-defined scenarios.
- [server-driven-ui.md](../server-driven-ui.md) — closed UI primitive
  contract; unchanged.
- [agent-delegation.md](../agent-delegation.md) — MCP exposure of the
  REPL command surface.
- [identity-and-audience.md](../identity-and-audience.md),
  [collab-sessions.md](../collab-sessions.md),
  [messaging-and-boards.md](../messaging-and-boards.md),
  [shared-artifacts.md](../shared-artifacts.md),
  [search-and-memory.md](../search-and-memory.md),
  [bug-reporting.md](../bug-reporting.md) — Layer 2 domain
  plans.
- [discovery.md](../discovery.md), [protocol.md](../protocol.md),
  [transport-multiplexing.md](../transport-multiplexing.md),
  [ci.md](../ci.md) — homes for the infrastructure/tooling use cases
  explicitly excluded from the REPL-closure clause.
- [usecases.md](../../../usecases.md) and
  [plato_inspired_usecases.md](../plato-inspired-usecases.md) — the
  user stories this plan moves out of Go and into the REPL.
