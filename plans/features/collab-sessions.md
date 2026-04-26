---
title: "Collaborative Sessions Plan"
kind: plan
status: shipped-untested
owner: copilot
validation: none
last-reviewed: 2026-04-26
---

# Collaborative Sessions Plan

See `application-runtime.md`, `repl-and-shell.md`, and `repl-capability-closure.md` for the broader runtime and REPL context.

## Design Principle

The system needs a general-purpose interactive session substrate that is broader than PTY-backed REPL sessions. REPL remains one session kind, but the same underlying model should support co-view, co-control, lessons, games, and shared app experiences.

## Goals

- generalize the session model beyond REPL/PTX,
- support join/leave/detach/reattach for multi-party app sessions,
- support consent-based co-view and co-control,
- support durable participant-aware session state,
- preserve the thin-client architecture.

## Session Kinds

Minimum built-in kinds:

- `repl`
- `shared_view`
- `co_control`
- `lesson`
- `game`
- `chat_room`
- `artifact_view`

Apps may define additional logical specializations while still using the common session substrate.

## Data Model

### InteractiveSession

Fields:

- stable `session_id`
- `kind`
- owner activation or owner app ref
- participants
- attached devices
- presentation or control owner
- share policy
- serialized session state
- created / updated / idle timestamps

### SessionParticipant

Fields:

- participant identity ref
- joined time
- attached device refs
- role within session
- capabilities granted (`view`, `annotate`, `control`, `moderate`)

## TAL Host Module

Add `session`.

Suggested functions:

- `session.create(kind, spec)`
- `session.join(session_id, participant)`
- `session.leave(session_id, participant)`
- `session.members(session_id)`
- `session.state(session_id)`
- `session.share_view(session_id, targets)`
- `session.request_control(session_id, participant)`
- `session.grant_control(session_id, participant)`
- `session.revoke_control(session_id, participant)`
- `session.attach_device(session_id, device_ref)`
- `session.detach_device(session_id, device_ref)`

## Services

### InteractiveSessionService

- `CreateSession`
- `GetSession`
- `ListSessions`
- `JoinSession`
- `LeaveSession`
- `AttachDevice`
- `DetachDevice`
- `RequestControl`
- `GrantControl`
- `RevokeControl`
- `TerminateSession`

## REPL Surface

Add command group `session`.

Examples:

```text
session ls
session show sess_42
session new --kind shared_view
session join sess_42
session leave sess_42
session members sess_42
session share sess_42 --to zone:kitchen
session control request sess_42 mom
session control grant sess_42 mom
session control revoke sess_42 mom
```

`ReplSession` remains, but is treated as the `repl` specialization of this broader model.

## Control and Consent Model

Remote control must be approval-gated.

Rules:

- co-view may be allowed by session policy,
- control elevation always requires an explicit grant,
- control grants are revocable,
- grants and revocations are logged,
- apps do not get silent escalation.

## Use Cases Enabled

This plan directly supports:

- remote help and shared navigation,
- shared lesson sessions,
- join/resume game sessions,
- shared artifact viewing and annotation,
- escalation from text collaboration to richer coordinated sessions.

## Acceptance Criteria

- a TAL app can create a non-REPL session and manage participants deterministically,
- participants can detach and reattach without losing session identity,
- remote control is auditable, revocable, and typed,
- REPL can inspect and operate all session kinds, not just REPL sessions.
