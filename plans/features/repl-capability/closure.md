# REPL Capability Closure Plan

See `../masterplan.md` for overall system context, `../usecases.md` for the original use cases, `plato_inspired_usecases.md` for the PLATO-inspired use cases, `application-runtime.md` for the TAL/TAR runtime, and `repl-and-shell.md` for the REPL plan this document extends.

## Design Principle

Every application capability required by a supported use case must exist in five aligned forms:

1. a TAL host module,
2. a typed control-plane service,
3. a REPL command surface,
4. in-REPL documentation and examples,
5. simulation and test support.

The REPL remains a typed control-plane shell. This plan does **not** turn it into a Unix shell or expose arbitrary host capabilities.

## Problem Statement

The existing runtime and REPL plans provide strong coverage for orchestration-heavy scenarios: media routing, observation, scheduling, device placement, telephony, PTY-backed terminal access, and AI-assisted operator workflows. They do not yet fully cover the collaborative, durable-content, identity-aware, and searchable-memory use cases described in `plato_inspired_usecases.md`.

The missing substrate is not client logic. The missing substrate is a small set of additional server-side, typed capabilities that TAL apps and the REPL can both use.

## Closure Rule

No use case is considered supported unless a human or agent can exercise the required capability through the REPL using typed requests and responses.

This means:

- no hidden-only capability available to apps but not visible in REPL,
- no app-only service that bypasses the typed command surface,
- no REPL command that shells out to host state instead of authoritative kernel services.

## Capability Inventory

The following new capability families are required in addition to the current runtime surface:

- `identity` — people, groups, aliases, audiences, acknowledgement state, preferences.
- `session` — generalized interactive sessions beyond PTY-backed REPL sessions.
- `message` — room chat, direct messages, boards, bulletins, threads, unread state.
- `artifact` — durable shared documents, canvases, templates, annotations, lesson content.
- `search` — unified search across messages, artifacts, activity streams, logs, and structured records.
- `memory` — optional higher-level household/activity-memory service built on top of search and content indexing.

The following currently implicit backends must become explicit REPL command groups:

- `placement`
- `recent`
- `store`
- `bus`

## Command Surface Expansion

Add the following first-class REPL groups:

- `identity`
- `session`
- `message`
- `board`
- `artifact`
- `canvas`
- `search`
- `memory`
- `placement`
- `recent`
- `store`
- `bus`

These sit alongside existing groups such as `devices`, `activations`, `claims`, `ui`, `flow`, `observe`, `presence`, `world`, `scheduler`, `app`, `logs`, `telephony`, `ai`, and `docs`.

## Service Alignment

For each new capability, add a typed service:

- `IdentityService`
- `InteractiveSessionService`
- `MessagingService`
- `ArtifactService`
- `SearchService`
- `MemoryService` (optional as a separate service; may be folded into `SearchService` if desired)

The authoritative backend for every REPL command in these groups must be one of the above services or an existing typed service.

## TAL Alignment

The runtime must expose corresponding host modules:

- `identity`
- `session`
- `message`
- `artifact`
- `search`
- `memory` (optional if kept separate)

Existing modules remain available and continue to handle orchestration-heavy use cases:

- `placement`, `claims`, `ui`, `flow`, `observe`, `recent`, `presence`, `world`, `scheduler`, `store`, `telephony`, `pty`, `ai`, `http`, `bus`, `log`

## Use-Case Coverage Matrix

| Use-case family | TAL modules | REPL groups | Status target |
|---|---|---|---|
| Intercom / PA / calling | placement, claims, flow, telephony, ui, ai | devices, flow, telephony, activations | already covered |
| Voice assistant | ai, ui, placement, bus | ai, activations, logs | already covered |
| Monitoring / alerts | flow, observe, recent, scheduler, ui, bus | observe, recent, scheduler, logs | already covered |
| Messaging / boards | identity, message, search | identity, message, board, search | new |
| Shared help / co-control | session, identity, ui | session, identity, activations | new |
| Lessons / guided practice | session, artifact, scheduler, identity, search, ai | session, artifact, scheduler, identity, search, ai | new |
| Shared canvas / symbols | artifact, session, ui | artifact, canvas, session | new |
| Multiplayer games | session, identity, artifact or store | session, identity, artifact | new |
| Household knowledge / memory | search, memory, message, artifact | search, memory, board, artifact | new |

## Documentation Requirements

For every new command group, provide:

- concise `help <group>` output,
- generated API documentation under `docs/repl/api/`,
- hand-authored usage guides under `docs/repl/commands/`,
- at least one example under `docs/repl/examples/`.

Required new example topics:

- `start-room-chat`
- `send-direct-message`
- `pin-family-bulletin`
- `remote-help-session`
- `shared-lesson-session`
- `annotate-shared-canvas`
- `search-household-memory`
- `review-learner-progress`
- `resume-multiplayer-session`

## Simulation Requirements

Each new capability must be usable in simulation and tests.

Minimum requirements:

- create and join sessions in simulation,
- post/read/ack messages in simulation,
- create and patch artifacts in simulation,
- run search queries against seeded data,
- validate audience resolution against fixture identities.

## Acceptance Criteria

- every use case in `../usecases.md` and `plato_inspired_usecases.md` maps cleanly to REPL-visible typed capabilities,
- no required use case depends on an app-only service that lacks REPL visibility,
- REPL and TAL expose the same capability families at the same conceptual granularity,
- agents using the REPL command surface can operate the same features humans use,
- new docs and examples are available entirely from inside the REPL,
- new capabilities are covered by simulation and integration tests.

## Implementation Order

1. identity and audience
2. generalized sessions
3. messaging and boards
4. shared artifacts and canvas
5. search and memory
6. REPL/runtime revisions and documentation
7. cross-use-case validation and simulation
