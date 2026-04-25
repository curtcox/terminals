---
title: "Identity and Audience Plan"
kind: plan
status: planned
owner: unowned
validation: none
last-reviewed: 2026-04-25
---

# Identity and Audience Plan

See `application-runtime.md`, `repl-and-shell.md`, and `repl-capability-closure.md` for the surrounding runtime and REPL context.

## Design Principle

People and audiences are first-class server concepts. Apps should not infer personhood indirectly from device IDs alone. The system needs a typed identity layer so that person-directed, role-directed, and household-directed workflows can be authored once and reused across apps.

## Goals

- represent people, groups, roles, aliases, and preferences,
- support audience resolution for commands like `kids`, `parents`, `mom`, or `everyone in kitchen`,
- support acknowledgement and read-state for alerts, bulletins, and messages,
- support per-person targeting in TAL and REPL without introducing user auth or client-specific logic.

## Non-Goals

- no full enterprise IAM system,
- no mandatory auth model beyond the current trusted-LAN assumption,
- no coupling of identity records to a specific device implementation.

## Data Model

### Person

Fields:

- stable `person_id`
- display name
- aliases / voice synonyms
- household roles
- default zones / associated devices
- notification preferences
- optional caregiver relationships

### Group

Fields:

- stable `group_id`
- display name
- members
- computed or static membership rules

### AudienceSpec

An `AudienceSpec` resolves semantic targets into people and devices.

Examples:

- `person:mom`
- `group:kids`
- `role:caregiver`
- `zone:kitchen`
- `household:all`
- compound forms such as `group:kids in zone:upstairs`

### Acknowledgement

A durable record of:

- subject reference,
- actor reference,
- time,
- mode (`seen`, `read`, `heard`, `dismissed`, `confirmed`).

An actor reference is a discriminated union over the
acknowledging party, not a `person_ref` alone. Supported actor
kinds:

- `person:<person_id>` — a known person,
- `device:<device_id>` — a device acting on its own behalf
  (kiosk tap, M4-style alert acknowledgement, idle-screen
  dismiss),
- `agent:<agent_id>` — an automated agent or MCP origin acting
  under delegated authority (see
  [agent-delegation.md](agent-delegation.md)),
- `anonymous:<origin>` — an unattributed ack with an origin tag
  (e.g., `anonymous:sip`, `anonymous:webhook`) for off-device
  intake channels with no person identity attached.

This keeps `IdentityService` as the single canonical ack owner
(per the layering in
[repl-capability-plan.md](repl-capability/plan.md)) while
admitting device-level, agent, and anonymous acks without
inventing a parallel ack substrate on `MessagingService`,
`ArtifactService`, or the monitoring flows.

## TAL Host Module

Add `identity`.

Suggested functions:

- `identity.people()`
- `identity.show(person_ref)`
- `identity.groups()`
- `identity.resolve(target_spec)`
- `identity.audience(target_spec)`
- `identity.preferences(person_ref)`
- `identity.ack(subject_ref, actor, mode)`
- `identity.ack_status(subject_ref)`

## Services

### IdentityService

- `ListPeople`
- `GetPerson`
- `ListGroups`
- `GetGroup`
- `ResolveAudience`
- `GetPreferences`
- `RecordAcknowledgement`
- `GetAcknowledgements`

## REPL Surface

Add command group `identity`.

Examples:

```text
identity ls
identity show mom
identity groups
identity resolve kids
identity resolve 'group:kids in zone:upstairs'
identity prefs mom
identity ack ls
identity ack show bulletin_42
identity ack record bulletin_42 --actor person:mom --mode read
identity ack record alert_17  --actor device:kitchen-screen --mode dismissed
identity ack record alert_17  --actor agent:voice-bot --mode confirmed
identity ack record report_93 --actor anonymous:sip --mode heard
```

## Use Cases Enabled

This plan directly supports:

- person-targeted or caregiver-targeted notifications,
- learner assignment and progress per person,
- direct messaging to a person,
- household summaries and acknowledgement workflows,
- audience-aware bulletin and reminder delivery.

## Acceptance Criteria

- TAL apps can target people and groups without using raw device IDs,
- REPL can inspect and resolve audiences and acknowledgement state,
- identity references can be joined with presence and placement cleanly,
- audience resolution behaves deterministically under test fixtures.
