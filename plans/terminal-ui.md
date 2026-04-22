# Terminal UI

See [masterplan.md](../masterplan.md) for overall system context. This
plan specifies the user-facing surface of a terminal with a screen —
tablets and browsers running the Flutter client — by composing existing
primitives, not by inventing new wire contracts.

Load-bearing precedents this plan leans on:

- [server-driven-ui.md](server-driven-ui.md) — closed set of
  `ui.v1` primitives and `SetUI` / `UpdateUI` / `TransitionUI`. Adding
  a new primitive to satisfy a scenario is forbidden.
- [io-abstraction.md](io-abstraction.md) — display resources
  (`display.<id>.main` exclusive, `display.<id>.overlay` shared) and
  the claim manager that arbitrates them.
- [bug-reporting.md](bug-reporting.md) — server-composed
  `withBugReportAffordance` scenario wrapper injects a report button
  into every root UI tree. The corner affordance in this plan follows
  the same pattern.
- [identity-and-audience.md](identity-and-audience.md) — actor forms
  `person` / `device` / `agent` / `anonymous`, possibly uncertain.
- [capabilities.proto](../api/terminals/capabilities/v1/capabilities.proto)
  + `CapabilityDelta` — the existing channel for viewport, orientation,
  and per-resource availability changes.

The client (see [CLAUDE.md](../CLAUDE.md) core rule 1) remains generic:
it renders descriptors, captures inputs, plays outputs, reports
events, detects wake words, and reports capability changes. Nothing
else.

## Scope

In scope:

- What the screen shows in idle, app, and menu situations
- How the always-available affordance is composed server-side
- Privacy mode expressed as a capability change
- Wake-word expectations and their routing
- Viewport / orientation reporting via existing capability flow

Out of scope for v1:

- Split-screen main-layer sharing on one display (distinct main-layer
  claims must still stay exclusive per the claim manager; overlay
  sharing is in scope because the resource model already supports it).
- Multi-user handoff on a single display.
- A client-side escape hatch that bypasses the server-composed
  affordance. Replaced with a server-enforced reachability invariant
  (see below).

## I/O Model

Inputs and outputs on a single terminal are independently claimable
resources. The claim manager arbitrates; the client has no opinion.

| Stream / surface              | Resource kind                      | Typical claim binding                             |
| ----------------------------- | ---------------------------------- | ------------------------------------------------- |
| Display video + pointer focus | `display.<id>.main` (exclusive)    | One activation at a time (the "current app")      |
| Overlay video                 | `display.<id>.overlay` (shared)    | Zero or more activations concurrent with main     |
| Display audio output          | `audio_out.<id>` (exclusive)       | Usually the same activation as `.main`            |
| Camera (dedicated)            | `camera.<id>.capture` (exclusive)  | One activation                                    |
| Camera (tap)                  | `camera.<id>.analyze` (shared)     | Many activations in parallel                      |
| Microphone (dedicated)        | `audio_in.<id>.capture` (exclusive)| One activation                                    |
| Microphone (tap)              | `audio_in.<id>.analyze` (shared)   | Many activations in parallel                      |
| Keyboard / touch / pointer    | `keyboard.<id>` / `touch.<id>` / `pointer.<id>` (shared) | Active regardless of privacy mode |

There is no global "screen mode". The screen state is whatever the
`.main` and `.overlay` claims currently render — possibly both at
once.

## Screen Situations

Three common situations, expressed as combinations of claims on the
primary display's resources. The server-driven UI tree for each is
composed from the closed primitive set — no new widget, no new wire
message, no client chrome.

### 1. Idle

No user-selected app; the server has chosen some activation to hold
`display.<id>.main`. Typical choices (server-side, not client-side):

- Still-photo slideshow: `fullscreen` wrapping an `image` rotated by
  `UpdateUI` on a timer.
- Live A/V from another terminal: `fullscreen` wrapping a
  `video_surface` and an `audio_visualizer`.
- Any other composition of primitives.

The corner affordance (below) is injected into this tree by the same
scenario wrapper that injects it into app trees.

### 2. App full-screen

User launched an app. The activation holds `display.<id>.main`
exclusive plus `audio_out.<id>` exclusive plus typically
`pointer.<id>` / `touch.<id>`. Its root descriptor is a
`FullscreenWidget` wrapping the app UI.

### 3. Menu overlaid on current situation

The menu is **not** a mode. It is a second activation holding
`display.<id>.overlay` shared, launched in response to a `UIAction`
from the corner affordance. The underlying main activation continues
to hold its claims; what happens to its inputs while the overlay is
up is governed by the input-routing policy for the overlay claim
(below).

Closing the menu means the overlay activation releases its claim and
terminates. The main activation never changed state.

## The Corner Affordance

A server-composed element injected into every main-layer UI tree by a
scenario wrapper (analogous to the existing `withBugReportAffordance`
in [bug-reporting.md](bug-reporting.md)). Working name:
`withCornerAffordance`.

Its descriptor composition, using only the closed primitive set:

- A `GestureAreaWidget` or a positioned `ButtonWidget` in the
  user-preferred corner of the layout tree, with
  `action = "corner.open"`.
- The positioning is expressed with the existing layout primitives
  (`padding`, `expand`, `row`, `stack`); no new widget.
- Visibility, position (one of four corners), and hit-target size are
  parameters to the wrapper resolved server-side from the merge of
  user preference and activity override. The client never sees the
  inputs to that merge — it renders the result.

When the user activates it, the client emits `io.v1.InputEvent.ui_action
= UIAction{component_id: "corner", action: "open"}`. The server
responds by starting an overlay-layer activation that holds
`display.<id>.overlay` and renders the menu descriptor.

### Server-enforced reachability invariant

Because the affordance is part of a scenario wrapper and not client
chrome, we can enforce at the server what the client cannot. The
invariant is defined on the **typed wrapper contract**, not on
arbitrary descriptor trees — tree-walking "any tree containing a
button" is brittle, as Codex noted. Concretely:

- `withCornerAffordance` emits a node with a **reserved `id` prefix**
  `__affordance.corner__` and a `UIAction` handler for action
  `corner.open`.
- The wrapper enforces, in code, minimum hit-target size (for
  example, >= 44×44 logical px, exact value documented alongside the
  wrapper), top-of-stack Z-order relative to the scenario's root
  content, and non-occluded geometry under the current
  `ScreenCapability.safe_area`.
- A CI gate validates the **output of the wrapper specifically** for
  every main-layer scenario in the registry: the emitted tree
  contains exactly one node with the reserved id prefix, its props
  satisfy the minimums, and its action handler is wired to
  `corner.open`.
- Anything outside the wrapper — third-party descriptor content,
  hand-rolled scenarios — is not audited by the invariant. Main-layer
  scenarios are required to pass through the wrapper; skipping it is
  itself a CI failure (see opt-out governance below).

**Opt-out governance.** Some scenarios (kiosk demos, safety-critical
foreground lock) legitimately need to run without the affordance.
"Reviewed opt-out" is not a prose escape hatch; it is a concrete
mechanism:

- A checked-in allowlist at
  `terminal_server/internal/scenario/affordance_optouts.yaml` with
  entries keyed by `scenario_id`, each carrying `reason`,
  `approver`, `expires_at`, and optionally a `replacement_affordance`
  pointer (if the scenario substitutes its own audited equivalent).
- The scenario registry refuses to load a scenario that skips
  `withCornerAffordance` unless it appears in the allowlist with a
  non-expired entry.
- A CI test fails the build if the allowlist contains an expired
  entry, an entry whose referenced scenario no longer exists, or an
  entry whose `replacement_affordance` (if any) does not itself pass
  the typed-wrapper check.
- PR review process for additions to the allowlist is documented in
  [ci.md](ci.md); this plan requires the file and the CI check, not
  a specific review workflow.

This replaces the client-side escape hatch that was contemplated and
deferred.

## The Menu Overlay

A scenario that holds `display.<id>.overlay` shared. Its descriptor is
an ordinary `ui.v1.Node` tree — likely a `Stack` wrapping a translucent
background plus a `Stack`/`Row` of `Button` widgets:

- One `Button` per launchable app (server-chosen list; diagnostics,
  terminal settings, bug reporting are ordinary entries).
- One `Button` whose action is `privacy.toggle`.
- One `Button` whose action is `bug.open` (the existing bug-reporting
  intent; see [bug-reporting.md](bug-reporting.md)).

Every interaction is a `UIAction` dispatched via the existing input
channel. No new control-plane messages.

### Action routing and component-id scoping

`io.v1.UIAction{component_id, action, value}` carries no layer or
activation context. When the main layer and the overlay layer are
both active, a bare `component_id` is ambiguous — two trees could
legally contain the same logical name. Rule:

- **Scenario authors write logical component IDs**
  (e.g. `submit`, `privacy_toggle`) in their descriptor trees.
- **The descriptor generator rewrites** every `id` field on every
  `ui.v1.Node` it emits to a **server-assigned, globally unique ID**
  scoped by owning activation (convention:
  `<activation_id>:<logical_id>`). The client only ever sees the
  rewritten form.
- **The server maintains** an action-ownership map from rewritten
  `component_id` → owning activation, populated at `SetUI` /
  `UpdateUI` time and torn down at activation exit.
- **Incoming `UIAction`s** are routed to the owning activation by
  lookup. Unknown or stale component IDs are dropped with an
  observable server-side counter, not delivered.
- **A server-side validator** rejects a `SetUI` whose tree contains a
  duplicate `id` or an unrewritten logical id leaking through. This
  is a scenario-engine bug, not a protocol bug, and should fail
  loudly in tests.

No wire change is required: the scoping lives in the bytes of the
`id` field, which already exists on every `Node`.

### Input routing while the overlay is up

The claim manager and router, not the client, decide what the main
activation receives while the overlay is active. Three observable
dispositions must be supported:

1. **Main stays live**: main activation continues to receive pointer,
   touch, audio, camera — nothing changes.
2. **Main paused**: router temporarily stops delivering input events
   and suspends the main-side media plan on relevant edges.
3. **Mixed**: scripted per-stream (typical default — audio continues,
   pointer routes to the overlay).

All three are the existing router behavior of routing inputs to one
claim or another, or pausing the relevant edges in the media plan.
Nothing about it is UI-specific. The default mix (audio continues,
pointer routes to overlay) is the server policy, tested as such.

**Where the policy lives.** The policy is a **server-internal
attribute of the overlay activation**, not a wire field. Concretely:
the activation record in `internal/scenario` (or the claim manager's
activation metadata — exact package chosen at implementation time)
gains an `overlay_input_policy` field with value `LIVE` / `PAUSED` /
`MIXED` plus a per-stream override map. The router consults this
attribute when resolving each incoming event's destination while an
overlay claim is active on the same display. Scenarios set the
attribute at activation-start; there is no `ConnectRequest` /
`ConnectResponse` carrier for it, and it is not part of
[io-abstraction.md](io-abstraction.md)'s public `Claim` struct.

## Idle Content Rendering

The client renders whatever main-layer descriptor the server has
currently set via `SetUI`. If no descriptor has yet been received for
a terminal with a screen, the client renders a minimal server-defined
placeholder descriptor that ships with the scenario runtime (not an
ad-hoc client widget). The client does not cache idle content across
sessions.

## Privacy Mode

Expressed as a capability withdrawal. The client is the source of
truth for its own hardware access.

- The user toggles privacy mode via the menu's `privacy.toggle`
  `UIAction` (or any other server-authored entry point).
- On entering privacy mode, the client:
  1. Stops local capture on mic and camera *before* emitting the
     delta.
  2. Emits a `CapabilityDelta` with `generation = N+1` whose embedded
     `DeviceCapabilities` omits the mic and camera **capability
     messages entirely**. This plan pins that as the canonical
     withdrawal encoding: the `microphone` field and the `camera`
     field are unset on the `DeviceCapabilities` message. Neither
     `AudioInputCapability` nor `CameraCapability` has a `supported`
     boolean in [capabilities.proto](../api/terminals/capabilities/v1/capabilities.proto),
     so the pre-rewrite phrasing is invalid; the canonical form is
     "field absent," and a single round-trip test pins it.
  3. Disables local wake-word detection.
- The server's claim manager observes the delta and, per
  [io-abstraction.md](io-abstraction.md), **invalidates** any claim on
  the disappeared resources. Activations relying on mic/camera
  suspend per the existing invalidation path.
- On exiting privacy mode, the reverse: client re-emits the
  capabilities with a fresh `generation`, claims re-grant.

Privacy-mode state is **not** a new field; it is fully expressed by
the absence of mic/camera in the terminal's current capability
snapshot. There is no persistent on-screen indicator, and no
client-chrome indicator. If the server wants one, it composes it into
the UI descriptor.

### Cutover semantics

The race is between "client decides to enter privacy mode" and
"in-flight mic/camera frames". The protocol already carries
`generation` on `CapabilitySnapshot`/`CapabilityDelta`. The rule:

- The client must **not** emit any mic or camera frame after the local
  cutover — defined as the instant the client's capture stack returns
  its final frame on the path to the transport.
- The server must drop any mic/camera frame whose producing
  activation's claim has been invalidated by a capability change.
- Test fixture asserts that from the moment the delta is sent, zero
  mic/camera frames reach any server-side activation.

**Test-observable sequencing signal.** `VoiceAudio` and camera frame
messages in the current contracts do not carry per-frame timestamps,
so the cutover test relies on harness-side tagging rather than
in-protocol timestamps:

- The Flutter test harness wraps the client's capture producers to
  stamp each produced frame with a **monotonic harness counter** and
  records the counter value at the moment `stopCapture()` returns.
- The server-side test harness records the counter value on every
  received frame.
- The assertion is: no received frame carries a counter value
  strictly greater than the recorded cutover value. This avoids any
  dependence on wall-clock timestamps that the wire format does not
  provide.

This generalizes the existing capability-invalidation path; nothing
new is invented on the wire.

## Wake Words

Terminals that support on-device wake-word detection must always be
listening **unless privacy mode has withdrawn the microphone
capability**. Detection emits audio into the existing voice pipeline.

- The client does not interpret wake words. It detects presence,
  starts or continues streaming audio (as the voice pipeline already
  does for `VoiceAudio` frames), and emits events via the existing
  channel.
- The server decides what to do with the audio — STT, intent routing,
  activation launch, audible acknowledgement — using the existing
  voice and intent pipelines.
- For v1, the set of wake words a given device recognizes is whatever
  the client's on-device detector is built with. Situational
  variation in what wake words *mean* is server-side intent routing,
  not client reconfiguration. If per-activation wake-word vocabularies
  are needed later, that is a distinct protocol addition with its own
  plan; this plan does not presume one.

### Multi-terminal deduplication

When one utterance is heard by two terminals in the same room, both
will emit voice events. Dedupe is a server-side policy on the voice
pipeline, not a client concern. This plan requires that a test exist
asserting at most one intent is dispatched per utterance within a
configurable window; it does not prescribe the winner policy.

**This is new infrastructure.** [phase-5-voice.md](phase-5-voice.md)
documents STT, LLM, TTS, and the intent bus — it does **not**
document a dedupe stage. Phase E below owns adding that stage to the
voice pipeline, in addition to the client-side work. The plan should
not read as though dedupe already exists.

## Identity and Actors

The client does not render a login or identity picker. Any action
taken at a terminal is attributed by the server to an `actor` of one
of the kinds in [identity-and-audience.md](identity-and-audience.md):
`person:<id>`, `device:<id>`, `agent:<id>`, or `anonymous:<origin>`.
Attribution may be uncertain; the server's identity/presence policy
resolves it and may fall back to `device:` when no person can be
identified with confidence.

Menu contents and permitted actions may legitimately vary with
resolved actor. The client does not gate on identity.

## Viewport and Orientation

Handled via the existing capability flow, not new messages.

- On connect, the client sends a `CapabilitySnapshot` including
  `ScreenCapability{width, height, density, orientation,
  fullscreen_supported, multi_window_supported, safe_area}`.
- On any change — rotation, window resize, browser zoom that moves
  effective pixel dimensions, tab foreground/background that changes
  the `safe_area` — the client sends a `CapabilityDelta` with a fresh
  `generation`.
- Rapid changes are coalesced on the client with a short debounce so
  the server is not flooded; debounce interval is a client constant
  documented in the client architecture plan.
- The server decides whether to re-emit `SetUI` or apply `UpdateUI`
  for any layout variants. The client does not make layout decisions
  beyond the descriptor.

## Summary of Client Responsibilities

The Flutter client on tablet or browser:

1. Renders `ui.v1` descriptors via `SetUI` / `UpdateUI` /
   `TransitionUI`.
2. Emits `InputEvent` (key, pointer, touch, ui_action),
   `SensorData`, voice audio frames, capability snapshots and deltas
   per existing contracts.
3. Captures mic/camera/keyboard/touch/pointer and obeys the server's
   stream start/stop and route commands.
4. Plays audio and displays video per existing IO commands.
5. Detects on-device wake words when mic capability is present.
6. Emits `CapabilityDelta` on viewport, orientation, or hardware
   change.
7. Withdraws mic/camera capability on local privacy-mode toggle; adds
   them back on exit.

The client does **not**:

- Own the corner affordance as chrome; it is a server-composed
  descriptor element.
- Maintain a persistent UI-mode enum.
- Pick idle content.
- Interpret wake words or decide their meanings.
- Render a persistent privacy, mic, or camera indicator.
- Store durable user UI preferences; those live server-side.

## Open Questions / Future Work

- Per-activation wake-word vocabularies. Keep out of v1 unless a
  concrete use case requires it.
- A protocol-level convention for "soft corner affordance" vs "hard
  corner affordance" — the reachability invariant as specified is
  binary. May need nuance for e.g. kiosk demos that truly want no
  escape.
- Split-screen on the main layer: would require lifting `display.main`
  to a tiled, non-exclusive resource. Deferred until a driving use
  case exists.

---

# Implementation Plan

Each phase's deliverable is the **automated tests** that gate it.
Tests run under `make all-check` from Phase A onward — CI wiring is
not deferred. Tests are written alongside or before the code, never
after.

**CI-gating clause, applied to every phase below.** Each phase's new
tests land in the same commit series as the code and are included
in the relevant `make` target at that phase: Go tests in
`make server-test`, Flutter tests in `make client-test`, proto
round-trip tests in `make proto-lint`, use-case gates in
`make usecase-validate`. `make all-check` invokes all of the above.
No phase is "done" until its additions are reachable from
`make all-check` on that phase's PR.

Test layers in use:

- **Protobuf contract tests** (`make proto-lint` plus Go round-trip
  tests). This plan adds **no new proto messages** unless a phase
  below explicitly justifies one.
- **Go server unit tests** (`make server-test`): descriptor
  generation, claim-manager interactions, actor resolution policy,
  privacy-mode cutover, capability-delta handling.
- **Go server integration tests**: in-process server + scripted client
  stub driving `ConnectRequest` frames over the real transport,
  asserting emitted `ConnectResponse` frames and server-side state.
- **Flutter widget tests** (`make client-test`): renderer behavior
  for descriptors composed from the closed primitive set, input event
  emission, capability delta emission.
- **Flutter integration tests**: client connected to a stub server,
  driving end-to-end descriptor round-trips.
- **Use-case validation gate** (`usecase-validate` skill /
  `make usecase-validate`): end-to-end scenarios registered in
  [usecases.md](../usecases.md).

## Phase A — Scenario wrapper: `withCornerAffordance`

Implement a Go scenario wrapper that injects a corner affordance into
the root descriptor of any main-layer activation on a terminal with a
screen. Parameterized by user pref + activity override, merged
server-side, emitting the resulting position/visibility. No new proto
messages.

Tests (`make server-test`, wired into `make all-check` from day one):

- Unit: given a base descriptor and a corner-config input, the
  wrapper produces a descriptor whose tree contains a single
  corner-affordance subtree at the specified corner with the
  specified visibility, and whose other content is unchanged.
- Unit: user pref alone → user's corner. Activity override alone →
  activity's corner. Both → activity wins while active, user pref
  restored on activity exit.
- Unit: absent config → bottom-right, always visible.
- **Reachability invariant test**: for every main-layer scenario in
  the registry (fixture-generated descriptors), the wrapped
  descriptor is checked for the invariant "a corner-affordance
  subtree with a `UIAction` handler for `corner.open` exists, or a
  documented wrapper-level opt-out is present". Failure is a build
  failure.
- Widget: given a wrapped descriptor, the Flutter renderer shows the
  affordance at the configured corner and emits a `UIAction` on
  activation. The emitted `component_id` matches the
  server-assigned scoped id (prefix `__affordance.corner__`) that
  the wrapper placed on the node; the test does not hardcode a bare
  string.
- Unit (opt-out governance): the scenario registry refuses to load a
  main-layer scenario that skips `withCornerAffordance` unless an
  entry exists in `affordance_optouts.yaml` with an unexpired
  `expires_at`.
- Unit (opt-out governance): an expired allowlist entry, a reference
  to a missing scenario, or a `replacement_affordance` that fails
  the typed-wrapper check, each cause the build to fail.

## Phase B — Menu overlay activation

Implement a scenario whose root descriptor is the menu, and which
requests `display.<id>.overlay` shared when started. It responds to
`UIAction{component_id: "corner", action: "open"}` by starting; to
any `close` action or a second `corner.open` by terminating.

Tests:

- Integration: `corner.open` UIAction from the client stub causes the
  server to grant an overlay claim and emit a `SetUI` on the
  overlay device layer.
- Integration: the main activation's `display.main` claim is
  unaffected; re-issuing main-layer `SetUI` is not required.
- Unit: menu descriptor contents reflect the server's registered
  apps and always include bug-report and privacy-toggle buttons.
- Unit: diagnostics and terminal-settings apps appear as ordinary
  entries when registered; nothing special-cased.
- Integration: second `corner.open` while menu is up is idempotent
  (no duplicate overlay activation).
- Integration: an explicit `close` action on the menu (e.g. the
  menu's close button) terminates the overlay activation and
  releases its `display.<id>.overlay` claim.
- Unit (action routing): the descriptor generator rewrites logical
  `id`s on emitted `ui.v1.Node` trees to scoped ids
  (`<activation_id>:<logical>`); a `SetUI` that contains a duplicate
  id or leaks an unrewritten logical id is rejected by the
  server-side validator.
- Unit (action routing): the action-ownership map is populated at
  `SetUI` / `UpdateUI` time and torn down at activation exit; an
  incoming `UIAction` with an unknown scoped `component_id` is
  dropped and increments a server-side counter rather than being
  delivered.

## Phase C — Overlay input-routing policies

Implement the per-activity routing policy for what the main activation
receives while the overlay is active: `LIVE`, `PAUSED`, `MIXED`. This
rides on the existing router; no new wire message.

Tests:

- Unit (router): each policy produces the expected delivery pattern
  for a synthetic input stream.
- Integration: default policy (`MIXED` with audio=live,
  pointer=routed-to-overlay) is what gets applied when an activity
  does not specify. Pinned by an explicit test so the default can't
  drift.
- Integration: `LIVE` policy — underlying app receives pointer events
  while the menu is open.
- Integration: `PAUSED` policy — underlying app's media-plan edges on
  the affected streams are torn down by the router, and restored on
  menu close.

## Phase D — Privacy mode via capability withdrawal

Wire a server-authored `privacy.toggle` UIAction to a client-side
handler that withdraws/re-adds mic and camera in the terminal's
capability set.

Tests:

- Widget: `privacy.toggle` UIAction triggers a `CapabilityDelta`
  whose embedded `DeviceCapabilities` has the `microphone` and
  `camera` fields **absent** (not present with empty sub-messages),
  and a monotonically-incremented `generation`. One proto round-trip
  test pins the exact canonical encoding.
- Widget: local capture stops **before** the delta is emitted; a
  fixture stub for the capture APIs asserts the stop call precedes
  the delta send.
- Integration (race cutover — **blocker** fix): inject mic frames at
  a fixed rate into the client stub; trigger privacy mode; using the
  harness-side monotonic frame counter (see **Cutover semantics**
  above), assert **zero** frames with a counter value strictly
  greater than the recorded cutover value reach any server-side
  activation. Repeat for camera.
- Integration: any active claim on the terminal's mic/camera is
  invalidated server-side on the delta.
- Integration: exiting privacy mode re-emits capabilities with a
  fresh generation; claim grants resume without requiring a fresh
  activation start.
- Widget: **no persistent privacy or capture indicator** in the
  rendered tree that originates from the client. The test is keyed
  to a client-chrome namespace, distinct from any indicator the
  server-composed descriptor may contain, so it can tell the two
  apart.

## Phase E — Wake words

Client's on-device detector is always live while mic capability is
present. Detected wake words feed the existing voice pipeline.

**This phase introduces two pieces of new infrastructure** that were
not acknowledged as new in the previous revision:

1. **A multi-client voice integration fixture.** Existing tests exercise
   one client at a time; Phase E's dedupe test needs two clients
   attached to a single server instance with coordinated wake-word
   injection. The fixture is a prerequisite for the dedupe test in
   this phase and must land first.
2. **A voice-pipeline dedupe stage.** Per the note in "Multi-terminal
   deduplication" above, [phase-5-voice.md](phase-5-voice.md) does
   not document dedupe. Phase E adds a dedupe stage upstream of
   intent dispatch, configurable by a window length and a
   pluggable winner policy (confidence, first-timestamp, closest
   terminal via the placement engine).

Tests:

- Widget: detector is enabled when mic is in capability set; disabled
  on privacy mode (no mic capability). Fixture harness replaces the
  detector with a stub.
- Widget: with mic capability present and privacy mode off, a
  simulated utterance causes the voice pipeline to begin streaming
  `VoiceAudio` frames.
- Go unit: the new dedupe stage, given two wake-word events within
  the configured window, emits one downstream intent per the
  configured winner policy. Each winner policy has its own unit
  test; the default policy is pinned.
- Integration (dedupe end-to-end): using the multi-client fixture,
  two terminals each emit a wake-word-triggered voice stream for the
  same utterance within the window; server dispatches **at most
  one** intent. The winner policy is a fixture parameter, not
  hardcoded.
- Integration: server response to a detected wake word may be (a)
  silent service, (b) activation launch, (c) an audible/visible
  server-composed descriptor update. One test per disposition, with
  the client assertions performed by the Flutter integration
  harness — **not** by the Go server test.

## Phase F — Viewport and orientation via capability delta

Use existing `CapabilitySnapshot` / `CapabilityDelta` on
`ScreenCapability`. No new proto.

Tests:

- Widget: on connect, a `CapabilitySnapshot` is emitted including
  `ScreenCapability` with current width, height, orientation,
  density, `safe_area`.
- Widget: rotating the test harness emits a `CapabilityDelta` with a
  fresh generation and the new `orientation`.
- Widget: resizing the browser window emits a `CapabilityDelta`.
- Widget: browser zoom change that alters effective pixel dimensions
  emits a `CapabilityDelta`.
- Widget: rapid resizes are coalesced; emission count per unit time
  is bounded, asserted against the documented debounce constant.
- Integration: connect-time snapshot reaches the server and is
  usable by scenarios to pick layout variants.
- Integration: orientation change **while the menu overlay is open**
  preserves the overlay activation and preserves the main
  activation's state; both re-render per server descriptors.
- Integration: browser tab backgrounding / foregrounding is reported
  as a capability change (via `safe_area` or a documented marker on
  `ScreenCapability`); this covers Codex #6.6.

## Phase G — Reconnect

Reconnect must restore the user-visible state, including an open
menu overlay.

Tests:

- Integration: client disconnects while a main-layer app is active;
  on resume, server replays the current main-layer `SetUI` and any
  active media plan.
- Integration: client disconnects **with the menu overlay open**; on
  resume, both main-layer and overlay-layer state are replayed.
- Unit: a mid-flight `UIAction` for `corner.open` whose response
  arrived post-disconnect is idempotent on resume (no ghost overlay
  activation).

## Phase G2 — Idle rendering and identity invariants

Covers spec-level claims that the first revision of the plan asserted
in prose without a corresponding test.

Tests:

- Widget (cold start): on first connect, before any `SetUI` has been
  received, the client renders the server-defined placeholder
  descriptor that ships with the scenario runtime. No ad-hoc client
  widget is rendered in its place. Asserted by rendering-tree
  inspection against a chrome-namespace vs descriptor-namespace
  split.
- Widget (no idle cache across sessions): disconnect the client from
  a session whose last `SetUI` was an idle descriptor; reconnect as
  a new session; assert the prior idle tree is not re-rendered
  before a fresh server push. The test inspects the renderer's
  active descriptor between the disconnect and the server's first
  post-reconnect `SetUI`.
- Widget (no identity surface): inspect the rendered tree across a
  representative set of descriptors for any login, user-picker, or
  actor-display element that originates from the **client-chrome
  namespace**. Any such element is a test failure. Server-composed
  descriptor elements that display identity are not subject to this
  assertion — the test distinguishes by namespace.
- Integration (no client-side identity gating): a `UIAction` that the
  server decides is restricted for the current resolved actor is
  blocked server-side (policy returns an error / no-op / redirect);
  the client never gates on identity and always forwards the
  action. Fixture asserts both sides of this behavior.

## Phase H — End-to-end use cases

Register in [usecases.md](../usecases.md) and wire into
`make usecase-validate` per the `usecase-implement` skill. Candidate
IDs (exact IDs at registration time):

- **UI-IDLE-1**: terminal with no user-launched app shows a
  server-driven main-layer descriptor; the corner affordance is
  present.
- **UI-CORNER-1**: activating the corner affordance opens the menu
  overlay without disturbing main-layer state.
- **UI-CORNER-2**: menu-overlay default routing (audio stays live,
  pointer routes to overlay) is observed in a real round-trip.
- **UI-PRIV-1**: toggling privacy mode stops mic/camera frame
  delivery atomically across the capability cutover (no post-cutover
  frames reach the server), and restores both on exit.
- **UI-PRIV-2**: wake-word detection is suspended in privacy mode;
  keypress/touch still route.
- **UI-WAKE-1**: a wake word heard by a single terminal triggers the
  server-configured intent and the server-chosen feedback.
- **UI-WAKE-2**: a wake word heard simultaneously by two terminals
  dispatches at most one intent.
- **UI-ROT-1**: rotating the tablet client emits a capability delta
  and, for a scenario with layout variants, triggers a descriptor
  swap.
- **UI-RECON-1**: reconnect-mid-menu restores both layers.
- **UI-INVARIANT-1**: the reachability invariant check (Phase A)
  runs against every registered scenario as part of the gate.

Each use case is implemented per `usecase-implement` and must pass
`make usecase-validate` before the phase ships.

## Phase I — Hardening

Not a feature phase; a cleanup pass.

- Confirm every spec claim has a corresponding test citation from
  one of the phases above; fill any remaining gap.
- Confirm no test asserts client-side rendering from within a Go-only
  integration test — those assertions belong in Flutter integration
  tests. A lint script reviews the phase's test matrix for
  layer-correctness.
- Confirm CI gating has been real the whole way — no test added to
  the plan is excluded from `make all-check`.

## Deliberate non-tests

Out of scope for this plan's automated gate:

- Visual appearance (colors, fonts, iconography). Server-driven
  descriptors own this; visual regression testing is a separate
  concern.
- Wake-word detection accuracy. A model/device concern, gated
  elsewhere.
- Deep network-partition behavior beyond the specific reconnect
  scenarios in Phase G. The connection-reliability plan covers the
  rest.

## What this plan is explicitly not adding to the wire

To make Codex's core finding unmistakable, here is the list of
messages that might tempt a future editor to add and the existing
contract that subsumes each:

| Tempting new message     | Use existing instead                                   |
| ------------------------ | ------------------------------------------------------ |
| `SetScreenMode`          | `SetUI` — modes are descriptor state, not wire state   |
| `MenuDescriptor`         | A normal `ui.v1.Node` tree in a `SetUI`                |
| `OpenMenuRequest`        | `io.v1.UIAction{component_id, action}`                 |
| `CornerIconConfig` wire  | Server-side merge; emitted into the descriptor tree    |
| `ViewportReport`         | `CapabilityDelta` on `ScreenCapability`                |
| `SetPrivacyMode` command | `CapabilityDelta` withdrawing mic/camera endpoints     |
| `WakeWordEvent`          | Existing `VoiceAudio` + intent pipeline                |

If a future need genuinely cannot be expressed within the existing
contracts, that is a separate plan with its own justification.
