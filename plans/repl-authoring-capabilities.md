# REPL Authoring Capabilities

See [masterplan.md](../masterplan.md) for overall system context. See
[repl-and-shell.md](repl-and-shell.md) for the existing REPL plan this
document extends, and [application-runtime.md](application-runtime.md) for
the TAR/TAL runtime that shares its typed service surface.

## Problem

The existing [REPL plan](repl-and-shell.md) is deep on **introspection** —
`devices ls`, `activations ls`, `claims tree`, `logs tail`, `app reload`,
`ai ask` — but thin on **authoring** and **live operation**. The listed
command groups (`devices`, `sessions`, `activations`, `claims`, `ui`,
`flow`, `observe`, `presence`, `world`, `scheduler`, `app`, `logs`,
`telephony`, `ai`, `config`, `term`) expose the running system for reading
and limited lifecycle control; they do not let a user *create new
behavior* from the REPL.

Building the chat use case made this concrete. Chat is a deliberately
simple scenario — a room, an identity per device, a message log, a
fan-out on post — and it still required Go edits across seven files in
five packages: `chat/room.go`, `scenario/chat.go`, `scenario/register.go`,
`ui/descriptor.go`, `transport/chat_wire.go`, `transport/control_stream.go`,
`admin/server.go`. None of these edits carry intrinsically chat-specific
architecture. They are generic capabilities (keyed TTL store, UI
composition, input routing, broadcast, out-of-band trigger) that every
use case needs and that every use case currently re-implements in Go.

This plan identifies the **missing authoring surfaces** that would let the
bulk of `usecases.md` be implemented, inspected, and iterated without Go
recompilation. It is not chat-specific; chat is used throughout only as a
worked example because the gaps surfaced there are representative.

## Scope

This plan extends the REPL **command surface** and the **typed services**
underneath it. It does not redesign:

- the closed UI primitive set (see [server-driven-ui.md](server-driven-ui.md))
- the scenario engine lifecycle (see [scenario-engine.md](scenario-engine.md))
- the TAR/TAL package model (see [application-runtime.md](application-runtime.md))
- the AI assistance and approval pipeline (already covered in the REPL plan)

Those models are sound; the missing pieces are the *typed APIs and REPL
command groups* that let a user exercise them at runtime.

## Capability Gap Analysis

Each row below names a capability the chat implementation needed, the
broad set of use cases that require the same thing, and the nearest
existing surface. "Existing surface" names the module the chat code had
to edit directly; "gap" describes what is missing from the REPL.

| # | Capability | Use cases needing it | Existing surface | Gap |
|---|------------|----------------------|------------------|-----|
| G1 | Keyed in-memory state with TTL | M1, M2, AH13, AH14, AH15, T1, T2, chat | ad-hoc Go maps (e.g. `chat.Room`) | No REPL surface to create, read, write, or expire structured keyed state without a Go type. |
| G2 | Per-device, per-scenario mutable state (name, preferences, cursor) | P2, AH12, AH5, chat | scenario-specific Go structs | No REPL surface for "bind a value to (device, scenario) and retrieve it later." |
| G3 | Compose and publish a UI view | every UI-bearing scenario; esp. S1, D1, C5, chat | hand-written `ui.Descriptor` builders | No REPL surface to declare a view, bind it to a root id, and push it to a device. |
| G4 | Apply a targeted UI patch | M3, M4, D2, C1, chat | `UpdateUI{ComponentID, Node}` wrapped in transport helpers | No REPL command to emit an `UpdateUI` to one or many devices from a typed tree expression. |
| G5 | Register an input/event handler | every interactive scenario; chat | `handleInput` switch in `control_stream.go` | No REPL surface to register "when device X's component Y fires action Z, run command W." |
| G6 | Broadcast a UI message to a device set | C2, C3, M3, S1, AO3, chat | `RelayToDeviceID` + `globalSessionRelayRegistry` | No REPL command to fan out an arbitrary server message to a named device cohort. |
| G7 | Out-of-band trigger emission | AA1, AA2, AA4, B3, chat (admin post) | ad-hoc `/admin/...` HTTP handlers | No REPL command that emits a typed `Intent`/`Event` onto the bus as if it came from voice/UI/webhook. |
| G8 | Virtual/simulated devices and inputs | AA6, B3, I8, test harness | integration-test helpers in Go | No REPL surface to register a fake device, deliver synthetic input, or observe outbound messages to it. |
| G9 | Define a scenario without a TAR package | I10, chat, all prototyping | full TAR app on disk; `engine.Register` in Go | No REPL surface for an inline scenario: `match` intent list, `on start/stop/input/event` actions, without writing a package. |
| G10 | Inspect and unregister runtime-defined artifacts | ops discipline | nothing — Go-defined things are immortal in process | No REPL surface that lists REPL-authored scenarios/views/handlers/stores and safely removes them. |
| G11 | Assertions and scripted verification | I8, AA6, every end-to-end test | Go test files | No REPL command surface for "expect device X to receive Y within N ms" usable both interactively and from a script. |
| G12 | Cross-session attachment to scenario UI | P2, C6, chat history to late joiner | ad-hoc broadcast wiring | No general "subscribe this session/device to that scenario's UI stream" command. |

G1–G8 are the capabilities that get rebuilt *every* time a new scenario
is added. G9–G12 are the capabilities that make the first set reachable
and testable from inside the REPL rather than through Go.

## Proposed Command Groups

The additions slot into the existing REPL command-group vocabulary. Each
group below is new or substantially extended. Every command classifies as
`read_only | operational | mutating` for the approval pipeline described
in the existing REPL plan.

### `store` — keyed state with TTL (addresses G1, G2)

`store` already exists as a TAL host module; this exposes the same typed
service to the REPL.

```text
store ns ls                                      read_only
store put <ns> <key> <value> [--ttl 24h]         mutating
store get <ns> <key>                             read_only
store del <ns> <key>                             mutating
store ls <ns> [--prefix <p>]                     read_only
store watch <ns> [--prefix <p>]                  operational
store bind <ns> <key> --to <device>:<scenario>   mutating  # G2 — scope to (device, scenario)
```

Values are structured (JSON-compatible) and versioned. TTL is per-record
and enforced by the store, not by callers.

### `ui` — view and patch authoring (addresses G3, G4, G6, G12)

Today `ui` in the existing plan is an introspection group. It needs
authoring verbs.

```text
ui show <device> [--root <id>]                              read_only
ui push <device> <descriptor-expr> [--root <id>]            mutating
ui patch <device> <component-id> <descriptor-expr>          mutating
ui broadcast <cohort> <descriptor-expr> [--patch <id>]      mutating   # G6
ui subscribe <device> --to <activation|cohort>              mutating   # G12
ui snapshot <device> [--format json]                        read_only
```

Cohorts are named device sets (see `devices cohort` below). A
`descriptor-expr` is a small literal form over the closed UI primitive
set — no new primitives, only a typed way to compose them from text.
The REPL holds a single reusable descriptor builder that the TAL `ui`
module, `ui push`, and internal scenarios all share. This matters: it
stops the ad-hoc "one UI builder per scenario in Go" pattern that the
chat work created.

### `devices cohort` — named device sets (addresses G6)

```text
devices cohort ls                                           read_only
devices cohort show <name>                                  read_only
devices cohort put <name> --where <selector>                mutating
devices cohort put <name> --ids <id>,<id>,...               mutating
devices cohort del <name>                                   mutating
```

`<selector>` reuses the existing placement selector grammar
(`zone=kitchen capability=display`) plus a `scenario=<name>` predicate so
cohorts like "everyone currently in chat" are expressible without code.

### `bus` — typed intent/event emission (addresses G7)

```text
bus emit intent <action> [slots...] [--scope device=<id>]   mutating
bus emit event <kind> [attrs...] [--subject <s>]            mutating
bus tail [--kind <k>] [--source <s>]                        operational
bus replay <from> <to> [--filter ...]                       operational
```

`bus emit` replaces the one-off `/admin/api/chat/send` style endpoints
with a single typed path. Admin HTTP stays for external integrations;
the REPL never needs a bespoke endpoint again.

### `handlers` — input/event routing (addresses G5)

```text
handlers ls                                                 read_only
handlers on <selector> <action> --run <command>             mutating
handlers on <selector> <action> --emit intent <action> ...  mutating
handlers off <handler-id>                                   mutating
```

`<selector>` matches on `scenario=<name>`, `component=<id>`, and/or
`device=<id>`. `--run` invokes another REPL command (any classification;
mutating commands still require the pipeline's approval before they
execute). `--emit` enqueues a typed bus trigger. Handlers are first-class
records with IDs, so they show up in `handlers ls`, can be disabled, and
are logged when they fire.

### `scenarios` — inline definition (addresses G9, G10)

```text
scenarios ls                                                read_only
scenarios show <name>                                       read_only
scenarios define <name> --match <intent-list> [--priority normal]
                         [--on-start <command>]
                         [--on-input <handler-spec>]
                         [--on-stop <command>]              mutating
scenarios undefine <name>                                   mutating
```

This is a minimal inline alternative to a full TAR package. It is *not*
TAL: it composes existing REPL commands into lifecycle hooks. A real app
still graduates to a TAR package on disk; this exists for prototyping,
demos, and ops-level one-offs — exactly what the chat work would have
needed.

### `sim` — virtual devices and scripted verification (addresses G8, G11)

```text
sim device new <id> [--caps display,keyboard]               mutating
sim device rm <id>                                          mutating
sim input <id> <component-id> <action> [<value>]            mutating
sim ui <id>                                                 read_only    # last descriptor pushed to this device
sim expect <id> ui contains <expr> [--within 2s]            operational
sim expect <id> message <selector> [--within 2s]            operational
sim record <id> [--duration 30s]                            operational
sim script <path>                                           operational
```

`sim device` devices participate in the full registry, placement,
claims, and transport path — they differ from real devices only in that
their IO is captured instead of forwarded. `sim expect` returns a
success/failure exit code and an optional trace, making it usable both
interactively and from a shell script invoking the REPL non-interactively.

### `scripts` — non-interactive REPL execution

```text
scripts run <path> [--json]                                 operational
scripts dry-run <path>                                      read_only
```

`scripts` runs a newline-delimited sequence of REPL commands with
ordinary classification semantics: `mutating` steps still prompt unless
`--yes` is passed for a whole-script confirmation. This is how CI
invokes the REPL for use-case validation (I8, AA6).

## Worked Example — Chat Without Go Changes

The chat implementation reduces to a single REPL script under this plan.
It is shown compactly to make the dependency on each capability concrete,
not as a final spec.

```text
store put chat.config retention_s 86400
handlers on scenario=chat component=chat_name_input submit \
    --run 'store put chat.identity "$device" $value --ttl 24h; \
           ui push $device $(chat_view $device)'
handlers on scenario=chat component=chat_message_input submit \
    --run 'store put chat.messages $(uuid) {device:$device,name:$(store get chat.identity $device),text:$value} --ttl 24h; \
           ui broadcast chat_members --patch chat_messages $(chat_message_list)'
devices cohort put chat_members --where scenario=chat
scenarios define chat --match intent=chat,intent="open chat","join chat" \
    --on-start 'ui push $device $(chat_entry_view $device)' \
    --on-stop  'store del chat.identity $device'
```

The two `chat_view` and `chat_message_list` expressions are ordinary
`ui.descriptor` compositions over the existing primitive set; they live
in `store put ui.views.*` or in a small TAR package that only ships
descriptor builders. Either way, the scenario wiring, state, input
routing, and fan-out are all REPL-authored.

The same pattern is what AH5 (bedtime story), AH8 (smoke alarm), M3
(red alert), and AO3 (PA announcement) need. None of them should have to
add a file under `internal/transport/` or `internal/scenario/`.

## Typed Services to Add

Each new command group maps to one typed service, co-located with the
kernel services already named in
[application-runtime.md](application-runtime.md) (`placement`, `claims`,
`ui`, `flow`, `observe`, `recent`, `presence`, `world`, `scheduler`,
`store`, `telephony`, `pty`, `ai`, `bus`, `log`):

- `StoreService` — gains TTL, namespaces, watch, scoped binding.
- `UiService` — gains `Push`, `Patch`, `Broadcast`, `Subscribe`,
  `Snapshot`. Today the UI surface is internal to transport.
- `BusService` — `Emit`, `Tail`, `Replay`. Today the bus is implicit in
  scenario matching.
- `HandlerService` — `Register`, `Unregister`, `List`, `Trigger`.
- `CohortService` — device set CRUD plus live membership evaluation.
- `ScenarioAuthoringService` — inline scenario CRUD; distinct from and
  layered above the existing scenario engine `Register`.
- `SimService` — virtual device registration, input injection, output
  capture, assertion evaluation.

All of these sit under the existing REPL command-registry pipeline
(classification, approval, streaming dispatch) and are equally available
to MCP origins per [agent-delegation.md](agent-delegation.md).

## Interactions With Existing Plans

- **[scenario-engine.md](scenario-engine.md)** — `scenarios define` is an
  additional factory path into the existing engine. Runtime-defined
  scenarios use the same `ScenarioDefinition`/`ScenarioActivation`
  interfaces; they are not a parallel lifecycle. `Start/Stop/Suspend/Resume`
  still go through the engine's supervisor.
- **[application-runtime.md](application-runtime.md)** — TAR/TAL remains
  the authoring path for durable applications. Inline `scenarios define`
  is the cheap prototyping path; graduation to a TAR package is a
  copy-out, not a rewrite, because both targets use the same typed
  service surface.
- **[server-driven-ui.md](server-driven-ui.md)** — `ui push/patch/broadcast`
  adds no new primitives. The closed UI contract is unchanged; what
  changes is who can compose primitives (now: the REPL, not only
  hand-written Go).
- **[repl-and-shell.md](repl-and-shell.md)** — this document adds command
  groups alongside the existing `devices`/`activations`/`claims`/`ai`/…
  set. Classification metadata (`read_only | operational | mutating`),
  approval pipeline, streaming dispatch, and AI tool-use mediation all
  apply to the new groups exactly as to the existing ones.
- **[agent-delegation.md](agent-delegation.md)** — every new command is
  usable from MCP origins with the same approval model. This matters:
  AA1, AA2, AA4, AA5 want these surfaces programmatically.

## Use-Case Coverage

New or strengthened coverage from this plan:

- **I10** — "add a new server-side scenario without touching client
  code" becomes "without touching client *or* server code" for the
  subset of scenarios expressible via `scenarios define`.
- **I8, AA6** — `sim` + `scripts` provide the integration-test surface
  the CI pipeline and development agents currently need Go for.
- **B3** — `bus emit event` gives the diagnostics engine a typed path
  to open a pending bug report without a bespoke HTTP endpoint.
- **AA1, AA2, AA4, AA5** — automation agents can trigger scenarios,
  subscribe to observations, and react to events through the REPL
  command surface rather than through scenario-specific admin
  endpoints.
- **M3, M4, C2, C3, AO3** — broadcast scenarios become compositional:
  a red-alert-like scenario is a short script rather than a Go file.
- **P2, C6, chat** — `ui subscribe` gives late joiners and secondary
  attachments a named way to pick up an active UI stream.

The use cases this plan does **not** move the needle on are those whose
heavy lifting is in the IO router, placement engine, world model, or
telephony bridge (C4, C5, S1, S2, S3, AH7, AH11, I5). Those remain
Go-level work; this plan only removes the *glue* that currently gets
rewritten per scenario.

## Acceptance Criteria

- A user can implement a basic multi-device UI scenario — identity,
  input, log, fan-out, retention — using only REPL commands, without
  Go recompilation, and without new UI primitives.
- Every new command group is classified (`read_only | operational |
  mutating`) and honored by the existing approval pipeline.
- All REPL-authored artifacts (stores, cohorts, handlers, views,
  scenarios, sim devices) are listable, inspectable, and removable via
  their group's `ls`/`show`/`rm`/`undefine` commands.
- `sim` produces reproducible runs: the same script yields the same
  captured output on a clean server, and `sim expect` exits non-zero on
  violation.
- Documentation (`docs open api/UiService`, `docs open api/BusService`,
  …) is generated from service metadata for each new service, per the
  existing REPL docs system.
- An inline scenario authored with `scenarios define` and a scenario
  authored as a TAR package are indistinguishable to the scenario
  engine's supervisor (same lifecycle, same claim behavior, same
  suspend/resume semantics).
- Every new command is reachable from MCP origins with the same
  approval model as human REPL input.

## Implementation Phases

### Phase 1 — `store` and `bus` services

Typed TTL store service exposed to both TAL and REPL. Typed bus service
with `Emit`/`Tail`. These are the lowest-level gaps (G1, G7) and unblock
the rest.

### Phase 2 — `ui` authoring and `devices cohort`

`UiService` with `Push/Patch/Broadcast/Subscribe/Snapshot`. Cohort CRUD
backed by live selector evaluation. At the end of this phase, a REPL
session can drive a screen without Go changes as long as inputs are
ignored.

### Phase 3 — `handlers`

Input and event routing with classification-aware `--run` execution.
First point at which interactive scenarios are buildable from the REPL.

### Phase 4 — `scenarios define` and `scenarios undefine`

Inline scenario authoring. Reuses Phase 1–3 primitives for lifecycle
hooks; no parallel engine. Includes an explicit rollback path for
failed definitions.

### Phase 5 — `sim` and `scripts`

Virtual devices, input injection, output capture, assertions, scripted
execution. This is what ties the prior phases to CI (I8) and to the
agent-driven test surface (AA6).

### Phase 6 — documentation and examples

Generated docs for each new service. Worked examples for chat,
red-alert-like broadcast, timer + reminder, presence-query, and a
sim-only assertion script. Added to the REPL `docs examples` index.

## Related Plans

- [repl-and-shell.md](repl-and-shell.md) — base REPL plan: sessions,
  classification, approval pipeline, AI assistance.
- [application-runtime.md](application-runtime.md) — TAR/TAL durable
  authoring; uses the same typed services this plan extends.
- [scenario-engine.md](scenario-engine.md) — supervisor for both
  Go-defined and REPL-defined scenarios.
- [server-driven-ui.md](server-driven-ui.md) — closed UI primitive
  contract; unchanged.
- [agent-delegation.md](agent-delegation.md) — MCP exposure of the
  REPL command surface.
- [usecases.md](../usecases.md) — the user stories this plan helps move
  out of Go and into the REPL.
