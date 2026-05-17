---
title: "PLATO Extension Use Case Validation (PL2–PL27, excl. PL8/PL20)"
kind: plan
status: planned
owner: curtcox
validation: none
last-reviewed: 2026-05-17
---

# PLATO Extension Use Case Validation

PL1, PL8, and PL20 are already automated. This plan covers the remaining 24
PLATO-inspired use cases. They are grouped into four feature clusters;
each cluster can ship as an independent milestone.

## Use Cases in Scope

| Cluster | IDs | Description |
|---------|-----|-------------|
| Messaging | PL2–PL6 | Direct messages, shared board, pinned bulletins, replies, search |
| Collaboration | PL7, PL9–PL11 | Screen share, remote control, presence monitoring, escalation |
| Learning | PL12–PL17 | Lessons, adaptive quizzes, assignment, resume, progress, audio coaching |
| Creative / Social | PL18–PL19, PL21–PL27 | Drawing canvas, visual cues, multiplayer games, trivia, restrictions, community board, household memory |

## Cluster 1 — Messaging (PL2–PL6)

The shared household board is partially implemented via `messaging-and-boards.md`.
PL1 (text chat room) is automated. These five extend it.

**PL2** — Direct message to a specific device/person. Extend the board scenario
to support a `recipient` filter. `TestUseCasePL2WithEvidence`: post a DM to
`device_b`; verify only `device_b` receives it, not `device_c`.

**PL3** — Persistent note on shared household board. Write a scenario that
posts a note with `persist: true` and verifies it survives a simulated server
restart (reload from board store). `TestUseCasePL3WithEvidence`.

**PL4** — High-priority bulletin pinned to every idle screen. Extend the
display scenario with a `pin_all_idle` flag. All registered idle terminals
receive a `SetUI` with the bulletin. `TestUseCasePL4WithEvidence`.

**PL5** — Reply to a shared note. Post a note, then post a reply referencing
the note ID. Verify the reply is threaded (parent_id set). `TestUseCasePL5WithEvidence`.

**PL6** — Search notes by topic/date. Implement a `SearchNotes` query handler.
Post several notes with different topics. Query by topic; verify correct subset
returned. `TestUseCasePL6WithEvidence`.

## Cluster 2 — Collaboration (PL7, PL9–PL11)

PL8 (shared live session) is already automated.

**PL7** — Screen-share: observer requests to see another device's current UI.
Server forwards the observed device's latest `SetUI` frame to the observer.
`TestUseCasePL7WithEvidence`: trigger SetUI on device A; request screen-share;
verify device B receives the same UI frame.

**PL9** — Remote control: temporary navigation takeover after approval.
Observer sends a `RequestControl` message; observed device receives an approval
prompt (SetUI); on simulated approval, observer's input events are forwarded.
`TestUseCasePL9WithEvidence`.

**PL10** — Child engagement monitoring: track whether a device is actively
used. Idle timer + last-input timestamp. `TestUseCasePL10WithEvidence`:
advance fake clock past idle threshold; verify parent device receives
"device idle" notification.

**PL11** — Escalate from text to voice/intercom. During a board session, a
"help" voice command upgrades to intercom. `TestUseCasePL11WithEvidence`:
start board session, send voice "help", verify intercom route opened.

## Cluster 3 — Learning (PL12–PL17)

This cluster requires a new `LessonScenario` subsystem.

**Core abstraction**: A `Lesson` struct with steps (question, accepted answers,
feedback text). The `LessonScenario` renders one step at a time via `SetUI`
and accepts `InputEvent` responses.

**PL12** — Publish a guided lesson to any terminal. `TestUseCasePL12WithEvidence`:
register a lesson YAML; connect a learner terminal; verify first step rendered.

**PL13** — Adaptive quiz: wrong answer → corrective feedback → retry.
`TestUseCasePL13WithEvidence`: inject wrong answer; verify feedback shown; inject
correct answer; verify advancement.

**PL14** — Assign lesson at scheduled time to a specific device.
Use fake clock. `TestUseCasePL14WithEvidence`: schedule lesson for T+1m;
advance clock; verify assigned terminal receives lesson start.

**PL15** — Resume lesson from a different terminal.
`TestUseCasePL15WithEvidence`: complete step 1 on terminal A; connect terminal B
with same user identity; verify lesson resumes at step 2.

**PL16** — Progress review: parent queries completion history.
`TestUseCasePL16WithEvidence`: complete a lesson; query progress; verify
step count and wrong-answer log.

**PL17** — Audio coaching (timing/pitch feedback via microphone).
Register a microphone terminal. Inject audio frames. `FakeAudioCoach` returns
timing feedback. Verify feedback broadcast to learner. `TestUseCasePL17WithEvidence`.

## Cluster 4 — Creative / Social (PL18–PL19, PL21–PL27)

**PL18** — Shared drawing canvas: tablet sends draw events; canvas state
broadcast to a display terminal. `TestUseCasePL18WithEvidence`: inject stroke
events; verify canvas frame received by display.

**PL19** — Quick visual cue / sketch to a room. One-shot: a pre-drawn symbol
is displayed on the target room device. `TestUseCasePL19WithEvidence`.

**PL21** — Multiplayer text/graphical game across terminals.
Implement `GameScenario` (generic turn-based). `TestUseCasePL21WithEvidence`:
two players registered; player 1 submits move; verify player 2 receives updated
state.

**PL22** — Trivia/quiz game on nearest screens.
`GameScenario` with trivia mode. `TestUseCasePL22WithEvidence`: start trivia;
broadcast question; verify all registered room devices receive it.

**PL23** — Join ongoing game from a different device.
`TestUseCasePL23WithEvidence`: game in progress; new terminal connects with
same player identity; verify score and state restored.

**PL24** — Restrict game availability by time/room/role.
`TestUseCasePL24WithEvidence`: game start blocked outside allowed hours (fake
clock); game start allowed within hours.

**PL25** — Community board (neighborhood-level).
Extend the messaging board with a `scope: community` field. Requires a
community broadcast route. `TestUseCasePL25WithEvidence`.

**PL26** — Household memory: accumulated context stored across sessions.
`MemoryScenario` that persists key facts. `TestUseCasePL26WithEvidence`: store
a fact; restart session; query fact; verify returned.

**PL27** — Community knowledge search.
Extend PL6 search to include community-scope boards. `TestUseCasePL27WithEvidence`.

## Milestones

1. **M1** — Cluster 1 (PL2–PL6): messaging extensions. Board infrastructure
   already exists; mostly incremental.
2. **M2** — Cluster 2 (PL7, PL9–PL11): collaboration. Screen-share and idle
   monitoring are the key new paths.
3. **M3** — Cluster 3 (PL12–PL17): `LessonScenario` built and all six tests
   passing.
4. **M4** — Cluster 4 (PL18–PL19, PL21–PL27): `GameScenario` + canvas +
   community board. All 24 IDs registered in validate script; CI green.
