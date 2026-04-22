# Terminal UI

See [masterplan.md](../masterplan.md) for overall system context, and
[server-driven-ui.md](server-driven-ui.md) for the descriptor format that
renders inside the surfaces defined here.

This plan specifies the user-facing surface of a terminal with a screen:
tablets and browsers running the Flutter client. The client is a generic
thin client — all scenario-specific behavior stays server-side (see
[CLAUDE.md](../CLAUDE.md) core rule 1).

## Scope

In scope:

- Screen layout, modes, and transitions on a single terminal
- How user-selected apps occupy the display
- The always-present menu affordance
- Idle behavior, privacy mode, wake-word expectations
- Orientation and window-size reporting

Out of scope for v1:

- Split-screen and multi-user handoff on a single display
- Hardware-button or gesture escape hatches that bypass the server-app
  contract (may be revisited if needed)

## I/O Model

Inputs and outputs on a single terminal are independently routable and
server-controlled. A single terminal can simultaneously:

- Stream its camera, microphone, and keypress/touch events into multiple
  independent server-side activities.
- Render a single app full-screen for the local user.

| Stream                       | Direction | Typical binding                                                   |
| ---------------------------- | --------- | ----------------------------------------------------------------- |
| Camera video                 | in        | Often fanned out to several activities                            |
| Microphone audio             | in        | Often fanned out to several activities                            |
| Keypress / touch events      | in        | Routed by server; active even in privacy mode                     |
| Display video                | out       | Usually bound to one app full-screen                              |
| Display audio                | out       | Usually bound to the same app as display video                    |
| Display pointer / gesture    | in        | Usually bound to the same app as display video                    |

"Usually" means the default. The server can decompose the bundle at any
time; the client must not assume display video, audio, and pointer are
always one unit.

## Screen Modes

The terminal screen is always in exactly one of these modes. The server
decides which one; the client renders it.

### 1. Idle mode

The terminal has no user-selected app in focus. The idle screen is fully
server-driven. Typical server choices include:

- Still photos (slideshow, ambient art)
- Live audio/video feeds from other terminals
- Any other server-driven UI descriptor

The client does not pick idle content. It renders what the server sends.

The corner menu affordance is still visible in idle mode so the user can
launch an app or file a bug.

### 2. Full-screen app mode

Default for any user-selected app. The entire screen is dedicated to the
app's rendered UI, **except** for a single small corner icon (see below).

### 3. Menu-overlay mode

A menu is overlaid on top of the current mode (idle or full-screen).
Opened by selecting the corner icon.

## The Corner Icon

A single, always-present affordance by which the user can leave the
current activity, switch apps, toggle privacy mode, or file a bug.

- **Default position**: bottom-right corner of the screen.
- **Default visibility**: always visible.
- **Configurable by**: both the user (persistent preference) and the
  active activity (situational override from the server). Either can
  change position (any of the four corners) and visibility rules (e.g.
  fade on idle, reveal on proximity). The server is trusted to keep the
  icon reachable. No client-enforced escape hatch in v1.
- **Selecting it**: overlays the menu on top of the current mode. The
  underlying app is not dismissed.

## Menu Overlay

The menu is a server-driven list presented over the current mode.
Contents include at minimum:

- A list of other apps the user may launch
- **File a bug** entry
- **Privacy mode toggle** (enter / exit privacy mode for this terminal)

Diagnostics and terminal settings are not first-class menu entries —
they are ordinary apps that appear in the app list when appropriate.

### Behavior of the underlying app while the menu is open

The server chooses, per activity, how the underlying app behaves. Three
modes must all be supported, plus mixes of them:

1. **Live**: app continues to receive all inputs and drive outputs.
2. **Paused**: app receives no inputs and its outputs are frozen/muted.
3. **Mixed**: some streams continue, others are suspended.

The default mix: **audio continues, pointer input is suspended**. This
is the common case — a user opens the menu mid-call and wants to keep
hearing the call while the pointer addresses the menu.

The client must expose enough state to the server for it to make this
choice per activity; the client must not hardcode one behavior.

## Idle-Screen Details

The idle screen is a server-rendered surface with the corner icon on
top. If the server has not yet pushed a descriptor, the client shows a
neutral placeholder. The client does not cache or choose idle content.

## Privacy Mode

A terminal state, togglable from the menu. When active:

- The terminal **disables** microphone and video-camera capture.
- The terminal **continues** to report keypress and touch events.
- Wake-word detection is suspended (see below).
- There is **no persistent on-screen indicator** that the terminal is in
  privacy mode, and no persistent indicator when the mic or camera is
  live. This is a deliberate choice. If an indicator becomes desirable,
  it will be added as a server-driven UI element, not a client chrome
  element.

Toggling privacy mode is reflected to the server immediately so
server-side activities relying on that terminal's mic/camera can
degrade.

## Wake Words

Terminals that support on-device wake-word detection must always be
listening, **except** when in privacy mode.

- The set of wake words being listened for, and how each is handled, is
  situational and controlled by the server. The client exposes a
  capability (on-device detection of a configurable wake-word set) and
  reports detections upstream.
- The client does **not** decide what a wake word means. It reports the
  event (wake-word id, optional trailing audio) to the server.
- The server decides the response. Possible responses include silent
  service of a request, launching an app in the corner-menu sense,
  or directly indicating to the user (via the display-audio or
  display-video bundle) that the request is complete or rejected.

Because the response is server-driven, direct feedback to the user may
appear as a transient overlay, an audio cue, or any other server-driven
UI element — not a client-chrome toast.

## Identity

The client does not render a login screen or identity picker. The
server tracks user locations (which user is at which terminal) and
determines who initiated any given action from that context. Most
actions can be initiated by anyone present; some are restricted, and
restriction is enforced server-side.

Implication for this UI plan: menu entries and app lists shown on a
terminal may legitimately vary by who the server believes is present,
but the client itself does not gate on identity.

## Orientation and Window Size

Both portrait and landscape must be supported on tablets, and arbitrary
window sizes in the browser client.

- The client reports the current window size and orientation to the
  server on connect and **on every change** (rotation, window resize,
  browser zoom that changes effective dimensions).
- The server may respond with a new UI descriptor; the client re-renders
  accordingly. The client does not make layout decisions beyond what the
  descriptor specifies.
- Full-screen app mode fills whatever the current viewport is — there
  is no assumed aspect ratio.

## Summary of Client Responsibilities

The Flutter client on a tablet or browser is responsible for:

1. Rendering the server's UI descriptor for the current mode (idle,
   full-screen app, menu overlay).
2. Maintaining the corner icon per server + user configuration and
   opening the menu overlay on selection.
3. Capturing camera, microphone, keypress, touch, and pointer events
   and routing them to the server per current server instructions,
   respecting privacy mode.
4. Playing server-provided display video and display audio.
5. Detecting on-device wake words (when supported and not in privacy
   mode) and reporting them.
6. Reporting window size and orientation on connect and on change.
7. Applying the server-specified behavior (live / paused / mixed) for
   the underlying app when the menu overlay is open.

The client does none of the following:

- Pick idle-screen content
- Interpret wake words
- Render persistent privacy or capture indicators
- Render a login or identity screen
- Decide app layout beyond the server's descriptor
- Enforce a client-side escape hatch out of a full-screen app

## Implementation Plan

The plan is organized as phases. Each phase lists what ships and, as the
primary deliverable, the **automated tests** that gate it. No phase is
"done" until its tests are green under `make all-check`. Tests are
written alongside (or before) the code, not after.

Test layers in use:

- **Protobuf contract tests** (`make proto-lint`, plus Go-side
  round-trip tests): the canonical IO shape is defined before any
  server or client code in a phase.
- **Go server unit tests** (`make server-test`): state machines,
  descriptor generation, config merging, privacy gating, wake-word
  dispatch — all exercised without a client.
- **Go server integration tests**: in-process server + scripted client
  stub driving protobuf frames over the real transport, asserting
  emitted descriptors and side effects.
- **Flutter widget tests** (`make client-test`): each screen mode, the
  corner icon, menu overlay, orientation changes, privacy-mode input
  gating — driven by synthetic descriptors.
- **Flutter integration tests**: client connected to a stub server,
  exercising full descriptor round-trips and input routing.
- **Use-case validation gate** (`usecase-validate` / `make
  usecase-validate`): end-to-end scenarios registered in
  [usecases.md](../usecases.md) that assert observable behavior across
  client and server.

### Phase A — Protocol additions

Extend `api/proto/` with the messages needed by this plan. Nothing else
in this plan proceeds until Phase A is green.

Messages / fields (names illustrative):

- `ScreenMode` enum: `IDLE`, `APP_FULLSCREEN`, `MENU_OVERLAY`
- `SetScreenMode` server→client
- `CornerIconConfig` (corner, visibility rule) — server→client, merged
  from user pref + activity override server-side
- `OpenMenuRequest` client→server (user tapped icon)
- `MenuDescriptor` server→client (apps list, file-bug entry, privacy
  toggle)
- `MenuUnderlyingBehavior` per-activity: `LIVE | PAUSED | MIXED` with
  per-stream overrides (audio/video/pointer)
- `PrivacyModeState` + toggle request
- `WakeWordEvent` client→server (wake-word id, optional audio blob
  reference)
- `ViewportReport` client→server (width, height, orientation, density)

Tests:

- `make proto-lint` passes.
- Go round-trip test per new message: marshal → unmarshal → equal.
- Schema-freeze test: golden files of the descriptor shapes so future
  refactors don't silently change the wire format.

### Phase B — Server screen-mode state machine

Implement per-terminal mode state in `terminal_server/internal/terminal`
(or `internal/ui`), driven by server decisions and client events.

Tests (Go unit, `make server-test`):

- Transitions: `IDLE → APP_FULLSCREEN → MENU_OVERLAY → APP_FULLSCREEN`,
  and closing menu returns to the prior mode (not always `IDLE`).
- Opening the menu from `IDLE` returns to `IDLE` on close.
- Launching an app from the menu transitions to `APP_FULLSCREEN` with
  the new app, closing the menu.
- Re-entry safety: duplicate `OpenMenuRequest` is idempotent.
- Reconnect: a reconnecting terminal is told its current mode.

### Phase C — Corner-icon configuration

Implement the merge of user preference and active-activity override,
server-side, producing a single `CornerIconConfig` sent to the client.

Tests (Go unit):

- User pref alone: rendered as configured.
- Activity override alone: rendered as configured.
- Both set: activity wins for the duration of the activity; on exit,
  the user pref is restored.
- Absent config: bottom-right, always visible (the documented
  defaults).

### Phase D — Menu overlay and underlying-app behavior

Server-side: menu construction (apps, file-bug, privacy toggle) and
`MenuUnderlyingBehavior` dispatch per activity.

Tests (Go unit + integration):

- Menu contents reflect the server's app registry and include the
  file-bug and privacy-toggle entries.
- Diagnostics and terminal-settings apps appear as ordinary entries,
  not as fixed chrome.
- Underlying behavior: `LIVE`, `PAUSED`, and `MIXED` all round-trip
  through the protocol and reach the server-side router.
- Default when an activity does not specify: `MIXED` with
  audio=continue, pointer=paused. One unit test pins this default so
  it can't drift accidentally.
- Input routing: while menu is open in the default mix, pointer events
  from the terminal are not delivered to the underlying activity;
  audio capture and playback still are.

### Phase E — Privacy mode

Implement the server-side gate: when a terminal is in privacy mode,
mic/camera frames from it must not be delivered to any activity;
keypress and touch must still route normally.

Tests (Go unit + integration):

- Toggling privacy mode updates the terminal's device capability set
  immediately.
- Mic/camera frames submitted while in privacy mode are dropped at the
  server boundary (router never forwards them).
- Keypress and touch events are still routed.
- Wake-word detection is suspended on the client while privacy mode
  is active — asserted by a client widget test that stubs the
  detector.
- Exiting privacy mode restores routing without requiring a new
  activity.
- No client-chrome privacy indicator is rendered — widget test asserts
  absence.

### Phase F — Wake-word reporting and response

Server-side dispatch of `WakeWordEvent` based on identity-at-location
plus server policy.

Tests (Go unit + integration):

- A wake-word event from a terminal with a known occupant is dispatched
  according to the configured policy for that user.
- An event from a terminal with no resolved occupant is handled by the
  default policy (documented separately; tested here for the presence
  of *some* policy, not a specific outcome).
- The server can respond with: silent service, activity launch, or a
  direct user-feedback UI descriptor. One integration test per
  response kind confirms the client surface.
- Privacy mode suppresses events at the client boundary — no
  `WakeWordEvent` reaches the server.

### Phase G — Idle screen

Server emits idle descriptors; the client renders them; the corner
icon remains reachable.

Tests:

- Go unit test: the idle producer emits a valid descriptor (still
  photo, live A/V, or arbitrary UI) for a terminal with no active app.
- Flutter widget test: given an idle descriptor, the client renders it
  and overlays the corner icon.
- Integration: entering idle (no app focused) causes the server to
  push an idle descriptor; launching an app replaces it.

### Phase H — Orientation and viewport reporting

Client reports `ViewportReport` on connect and on every change
(rotation, resize, zoom that changes effective dimensions); server
reacts by sending a new descriptor when warranted.

Tests:

- Flutter widget test: rotating the test harness from portrait to
  landscape emits a `ViewportReport`; resizing the window likewise.
- Debouncing test: rapid resizes are coalesced so the server is not
  flooded (thresholds documented in the test).
- Go unit test: an activity that specifies layout variants returns the
  appropriate descriptor for a reported viewport.

### Phase I — End-to-end use cases

Register new entries in [usecases.md](../usecases.md) and wire them
into the validation gate via the `usecase-implement` /
`usecase-validate` skills. Candidate IDs (exact IDs TBD at
registration):

- **UI-IDLE-1**: terminal with no app shows a server-driven idle
  descriptor; corner icon is present.
- **UI-MENU-1**: tapping the corner icon during a full-screen app
  opens the menu overlay; closing it returns to the same app.
- **UI-MENU-2**: while the menu is open in default mix, the
  underlying app still receives audio but no pointer events.
- **UI-PRIV-1**: enabling privacy mode stops mic/camera delivery and
  suspends wake-word detection; keypress/touch still route.
- **UI-WAKE-1**: a wake word detected on a terminal with a resolved
  occupant is dispatched according to server policy and the terminal
  shows the server-chosen feedback (if any).
- **UI-ROT-1**: rotating the tablet client emits a viewport report
  and, for an activity with layout variants, triggers a descriptor
  swap.

Each use case is implemented per the `usecase-implement` workflow and
must pass `make usecase-validate` before the phase ships.

### Phase J — Regression and CI wiring

- All new Go tests included in `make server-test`.
- All new Flutter tests included in `make client-test`.
- Proto round-trip and schema-freeze tests included in
  `make proto-lint` or a sibling target invoked from `make all-check`.
- Use-case gate included in `make all-check` (or whatever umbrella
  target CI runs — see [ci.md](ci.md)).
- No phase merges without its tests attached to the same commit
  series; tests do not lag implementation.

### Deliberate non-tests

These are intentionally **not** covered by automated tests in v1:

- Visual appearance (colors, fonts, exact icon glyph) — the server-
  driven UI descriptors decide this, and visual regression testing
  is out of scope here.
- Wake-word detection accuracy — that is a model/device concern,
  validated separately, not in this plan's gate.
- Network-partition recovery beyond reconnect of mode state — covered
  by the connection-reliability plan.

## Open Questions / Future Work

- Whether a client-enforced escape hatch (edge-swipe, long-press) is
  needed if an activity-configured corner icon ever traps a user. Not
  in v1 — trust the server-app contract.
- Split-screen and handoff for multi-user use of a single display.
- Accessibility affordances (text scaling, reduced motion, high
  contrast) — these should flow through the server-driven UI descriptor
  rather than as client-side toggles.
