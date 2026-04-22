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

## Open Questions / Future Work

- Whether a client-enforced escape hatch (edge-swipe, long-press) is
  needed if an activity-configured corner icon ever traps a user. Not
  in v1 — trust the server-app contract.
- Split-screen and handoff for multi-user use of a single display.
- Accessibility affordances (text scaling, reduced motion, high
  contrast) — these should flow through the server-driven UI descriptor
  rather than as client-side toggles.
