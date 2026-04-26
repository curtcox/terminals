---
title: "Messaging and Boards Plan"
kind: plan
status: shipped-validated
owner: copilot
validation: automated:PL1
last-reviewed: 2026-04-26
---

# Messaging and Boards Plan

See `repl-capability-plan.md` for the overall closure rationale
and the layering this plan sits inside. Acknowledgement and
read-state ownership is defined in `identity-and-audience.md`;
the ack-related functions named below are thin helpers over
`IdentityService`, not a parallel ack substrate.

## Design Principle

Durable conversation and bulletin objects are first-class control-plane resources. They should not be approximated with ad hoc `store` records and custom UI code in each app.

## Goals

- room chat,
- direct person/device messaging,
- persistent boards and bulletins,
- threaded replies,
- unread and acknowledgement state,
- consistent searchability and timeline visibility.

## Non-Goals

- no requirement to integrate external messaging systems in phase one,
- no peer-to-peer client messaging path,
- no app-specific message schema drift.

## Data Model

### MessageRoom

- `room_id`
- display name
- audience or membership policy
- retention policy

### Message

- `message_id`
- room or direct target ref
- sender ref
- body
- created time
- thread parent ref
- delivery state
- tags

Ack/read state is not stored on `Message`; it lives on
`IdentityService` as an `Acknowledgement` record keyed by the
message's `subject_ref`. `MessagingService` reads that state
through `IdentityService.GetAcknowledgements` when surfacing
per-actor read/ack status.

### BoardPost

- `post_id`
- board ref
- author ref
- content
- pin state
- acknowledgement policy
- thread root ref

## TAL Host Module

Add `message`.

Suggested functions:

- `message.room_create(name, spec)`
- `message.room_post(room_ref, content, opts)`
- `message.dm_send(target_ref, content, opts)`
- `message.thread_reply(root_ref, content, opts)`
- `message.board_post(board_ref, content, opts)`
- `message.pin(subject_ref, audience)`
- `message.list(scope, filters)`
- `message.get(message_id)`
- `message.unread(target_ref)` — convenience filter; delegates to
  `identity.ack_status` on the message/room subject refs.
- `message.ack(subject_ref, actor)` — convenience wrapper;
  delegates to `identity.ack(subject_ref, actor, mode="read")`.

## Services

### MessagingService

- `CreateRoom`
- `GetRoom`
- `ListRooms`
- `PostRoomMessage`
- `SendDirectMessage`
- `ReplyThread`
- `CreateBoardPost`
- `PinSubject`
- `ListMessages`
- `GetMessage`
- `ListUnread` — delegates to `IdentityService.GetAcknowledgements`
- `AcknowledgeSubject` — delegates to
  `IdentityService.RecordAcknowledgement`

## REPL Surface

Add `message` and `board` command groups.

Examples:

```text
message rooms
message room new kitchen
message post --room kitchen 'Dinner in 10'
message dm mom 'Come downstairs'
message unread
message ack msg_42

board ls
board show family
board post family 'Need milk'
board pin post_42 --audience household:all
board thread post_42
```

## Search and Timeline Requirements

All messages and board posts must be indexable by the search subsystem and must appear in timeline views when relevant.

## Use Cases Enabled

This plan directly supports:

- room-based chat,
- direct typed messages,
- shared notes and threaded discussion,
- family bulletins pinned to idle screens,
- searchable message and board history.

## Acceptance Criteria

- TAL can create and operate room and direct message flows without app-specific storage conventions,
- REPL can inspect rooms, threads, unread state, and pinned bulletins,
- messages and boards are searchable and timeline-visible,
- acknowledgements are durable and typed, owned by
  `IdentityService` and reachable from messaging via thin
  delegating helpers.
