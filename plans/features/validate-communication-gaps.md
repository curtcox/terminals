---
title: "Communication Use Case Validation — Gaps (C4, C6)"
kind: plan
status: planned
owner: curtcox
validation: none
last-reviewed: 2026-05-17
---

# Communication Use Case Validation — Gaps

The core communication use cases C1–C3 and C5 are automated. This plan covers
the two remaining gaps: external SIP calling (C4) and a dedicated video call
terminal role (C6).

## Use Cases in Scope

| ID | Description | Work type |
|----|-------------|-----------|
| C4 | Say "call Mom" → SIP call placed via server-side SIP stack | Feature + validation |
| C6 | Tablet configured as a dedicated video call terminal | Validation (C5 behavior, new role config) |

## C4 — SIP Phone Call

C4 requires a server-side SIP client that can initiate outgoing calls.
The existing video-call scenario (C5) establishes the in-house device-to-device
path; C4 extends this to an external SIP peer.

### Implementation steps

1. **Define protobuf messages** for SIP dial (`SipDial`), ringing
   (`SipRinging`), answered (`SipAnswered`), and hangup (`SipHangup`).
2. **Add `SipClient` interface** in `terminal_server/internal/sip/` behind a
   fake for testing, matching the pattern used for `TTS` and `LLM`.
3. **Write `SipScenario`** (or extend `VoiceCallScenario`) to handle a
   `sip_dial` intent: parse the contact name, look up a SIP address from a
   contacts store stub, initiate the call via `SipClient`, and route two-way
   audio.
4. **Write harness test** `TestUseCaseC4WithEvidence` using `FakeSipClient`.
   Verify: intent triggers dial, ringing state sent to requesting device, audio
   route opened, hangup clears route.
5. Wire into `scripts/usecase-validate.sh` (`C4` case + `all_ids`).

## C6 — Dedicated Video Call Terminal

C6 is a configuration / role use case: a tablet that registers as
`role:video_call_terminal` and automatically accepts incoming video call
requests. The underlying video call scenario is already exercised by C5.

### Implementation steps

1. **Add `role` field to capability manifest** proto (or reuse existing tags).
2. **Extend `VideoCallScenario`** to auto-accept when the target device's role
   is `video_call_terminal`.
3. **Write harness test** `TestUseCaseC6WithEvidence`: register a terminal with
   `role:video_call_terminal`, initiate a call from a second terminal, verify
   auto-accept and two-way route without explicit user accept action.
4. Wire into validate script (`C6` case + `all_ids`).

## Milestones

1. **M1** — C6 test passing (no new features, role-config extension only).
2. **M2** — SIP interface + fake defined; C4 test written and failing.
3. **M3** — `SipScenario` implemented; C4 test passing.
4. **M4** — Both IDs registered in `usecase-validate.sh`; CI green.
