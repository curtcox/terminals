# TAL Example: Kitchen Timer (Use Case T1)

This document walks through a single Terminals Application Language (TAL)
app end-to-end so the language and its runtime contract are easy to read
off a worked example. The app implements **T1** from [usecases.md](../usecases.md):

> **Cook** — "Set a timer for 10 minutes" — be alerted when my food is
> ready without watching the clock.

TAL and the Terminals Application Runtime (TAR) are specified in
[plans/application-runtime.md](../plans/application-runtime.md). TAL is
not yet implemented; this example is written against that spec so the
contract stays concrete. Every host call below corresponds to a module
named there.

---

## Why this use case

T1 is small enough to fit on one page and still touches every part of
the TAL contract that other apps reuse:

| Concern                               | TAL surface used       |
|---------------------------------------|------------------------|
| Typed trigger matching                | `match` / `ActivationRequest` |
| Choosing the right device semantically | `placement`            |
| Composing server-driven UI            | `ui`                   |
| Driving time                          | `scheduler`            |
| Speaking on a speaker                 | `ai.tts`               |
| Serializable activation state        | the state dict         |
| Suspend / resume                      | `suspend` / `resume`   |
| Structured logs                       | `log`                  |

The app never ships any client-side code, matching rule #1 of
[CLAUDE.md](../CLAUDE.md): "never add scenario-specific behavior to the
client."

---

## Package layout

```text
terminal_server/apps/
└── kitchen_timer/
    ├── manifest.toml
    ├── main.tal
    └── tests/
        └── kitchen_timer_test.tal
```

---

## `manifest.toml`

The manifest is the contract between TAR and the package. An undeclared
permission means the corresponding host module is not loadable from
`main.tal`.

```toml
name                = "kitchen_timer"
version             = "0.1.0"
language            = "tal/1"
requires_kernel_api = "1.x"
description         = "Voice-activated countdown timer with spoken and visual alerts."

permissions = [
    "placement.read",   # resolve the nearest display to the cook
    "ui.set",           # render the initial countdown view
    "ui.patch",         # tick the remaining-time field and banner
    "scheduler",        # one-shot expiry + 1 Hz tick
    "ai.tts",           # speak "Your pasta is ready."
    "bus.emit",         # emit a `timer.expired` event for other apps
]

exports = ["kitchen_timer"]

# Closed finite domains for every req.* field used by match().
# See application-distribution.md §5.4.1.
[match.domain]
"req.kind"   = ["intent"]
"req.action" = ["timer.set", "timer.start"]
```

Notes:

- `language = "tal/1"` pins the TAL dialect. Future breaking dialect
  changes bump the major.
- `requires_kernel_api = "1.x"` constrains the host module set. TAR
  refuses to load if a required module's contract has moved.
- `exports` names the application definitions registered with the
  scenario engine. This package registers exactly one.

---

## `main.tal`

TAL is a Starlark-like, deterministic language hosted in the Go server.
No threads, no sockets, no raw filesystem, no clock other than what
host APIs hand you. A module is read once at load time; everything
mutable lives in activation state.

```python
# kitchen_timer/main.tal
#
# One activation = one running timer. The scenario engine is our
# supervisor; TAL only describes reactions to typed triggers.

load("placement",  nearest = "nearest")
load("scheduler",  after   = "after", every = "every", cancel = "cancel")
load("ui",         view    = "view",  patch = "patch", clear = "clear",
                   column  = "column", heading = "heading", text  = "text")
load("ai",         tts     = "tts")
load("bus",        emit    = "emit")
load("log",        info    = "info")


# ---------------------------------------------------------------------
# Definition — what activation requests this app accepts.
# ---------------------------------------------------------------------

def name():
    return "kitchen_timer"


def match(req):
    # TAL `match` is pure Boolean DNF over the closed atom set
    # declared in manifest.toml's [match.domain] (see
    # application-distribution.md §5.4.1). No negation, no helpers,
    # no state — Gate 4 rejects anything else.
    return req.kind == "intent" && req.action in ["timer.set", "timer.start"]


def new_activation(req):
    # Resolve the display the cook is standing in front of.
    # `placement.nearest` is a pure query; no observable side effect.
    target = nearest(
        to     = req.source_device,
        role   = "display",
        zone   = "kitchen",
        fallback = req.source_device,
    )

    duration = int(req.slots.get("duration_seconds", 0))
    label    = req.slots.get("label", "timer")

    if duration <= 0:
        # Refuse the activation by returning None. TAR reports this
        # back to the trigger source so voice can speak an error.
        return None

    return {
        "id": req.activation_id,
        "state": {
            "label":     label,
            "target":    target,
            "total":     duration,
            "remaining": duration,
            "status":    "running",
        },
    }


# ---------------------------------------------------------------------
# Lifecycle — Start / Handle / Suspend / Resume / Stop.
# Each returns a Result: {state, ops, emit, done}. Fields default to
# "no change" (state), [] (ops/emit), False (done).
# ---------------------------------------------------------------------

def start(env, state):
    info("timer.start", label = state["label"], seconds = state["total"])
    return {
        "state": state,
        "ops": [
            ui.view(state["target"], root = "timer", body = _countdown(state)),
            scheduler.every(id = "tick",    seconds = 1,                event = "tick"),
            scheduler.after(id = "expiry",  seconds = state["remaining"], event = "expired"),
        ],
    }


def handle(env, state, trigger):
    # One mailbox per activation; triggers arrive in order.
    if trigger.kind == "event" and trigger.name == "tick":
        return _on_tick(state)

    if trigger.kind == "event" and trigger.name == "expired":
        return _on_expired(state)

    if trigger.kind == "intent" and trigger.action == "timer.cancel":
        return _on_cancel(state)

    # Unknown trigger: commit nothing, let the engine log it.
    return {}


def suspend(env, state):
    # Only serializable state survives suspend. `scheduler` handles are
    # refs, not live timers, so we capture wall-clock delta via env.
    state["remaining"] = max(state["remaining"] - env.seconds_since_start, 0)
    state["status"]    = "suspended"
    return {"state": state}


def resume(env, state):
    state["status"] = "running"
    # Re-render and re-arm. `start` is idempotent by construction
    # because the scheduler IDs above are fixed strings.
    return start(env, state)


def stop(env, state):
    return {
        "ops": [
            scheduler.cancel(id = "tick"),
            scheduler.cancel(id = "expiry"),
            ui.clear(state["target"], root = "timer"),
        ],
    }


# ---------------------------------------------------------------------
# Event handlers — small, pure, test-friendly.
# ---------------------------------------------------------------------

def _on_tick(state):
    state["remaining"] = max(state["remaining"] - 1, 0)
    return {
        "state": state,
        "ops": [
            ui.patch(
                state["target"],
                component = "remaining",
                node      = ui.text(body = _mmss(state["remaining"])),
            ),
        ],
    }


def _on_expired(state):
    state["status"]    = "done"
    state["remaining"] = 0
    return {
        "state": state,
        "done":  True,
        "ops": [
            ui.patch(
                state["target"],
                component = "banner",
                node      = ui.text(body = "Timer done!", style = "alert"),
            ),
            ai.tts(state["target"], "Your %s is ready." % state["label"]),
            scheduler.cancel(id = "tick"),
        ],
        "emit": [
            bus.emit(kind = "timer.expired",
                     subject = state["label"],
                     duration = state["total"]),
        ],
    }


def _on_cancel(state):
    state["status"] = "cancelled"
    return {
        "state": state,
        "done":  True,
        "ops": [
            scheduler.cancel(id = "tick"),
            scheduler.cancel(id = "expiry"),
            ui.clear(state["target"], root = "timer"),
        ],
    }


# ---------------------------------------------------------------------
# View helpers — these return descriptor values, not side effects.
# ---------------------------------------------------------------------

def _countdown(state):
    return ui.column(
        ui.heading(body = state["label"]),
        ui.text(id = "remaining", body = _mmss(state["remaining"])),
        ui.text(id = "banner",    body = ""),
    )


def _mmss(seconds):
    m = seconds // 60
    s = seconds %  60
    return "%02d:%02d" % (m, s)
```

---

## `tests/kitchen_timer_test.tal`

TAL ships with its own simulation harness (`term sim run`). Tests are
plain TAL that drive the activation with synthetic triggers and assert
on emitted ops. No real clock, no real speaker.

```python
load("sim", "run", "trigger", "advance")
load("assert", "contains", "equals")

def test_ten_minute_timer_expires_and_speaks():
    act = run("kitchen_timer", intent = {
        "action": "timer.set",
        "slots":  {"duration_seconds": "600", "label": "pasta"},
        "source_device": "tab.kitchen.counter",
    })

    # 1 Hz ticks patch the remaining-time field for ten minutes.
    advance(act, seconds = 600)

    ops = act.committed_ops
    assert.contains(ops, kind = "ai.tts",
                    body  = "Your pasta is ready.")
    assert.contains(ops, kind = "ui.patch",
                    component = "banner",
                    style     = "alert")
    assert.equals(act.state["status"], "done")


def test_cancel_stops_the_timer_cleanly():
    act = run("kitchen_timer", intent = {
        "action": "timer.set",
        "slots":  {"duration_seconds": "60", "label": "quickbread"},
        "source_device": "tab.kitchen.counter",
    })
    advance(act, seconds = 10)
    trigger(act, intent = {"action": "timer.cancel"})

    assert.equals(act.state["status"], "cancelled")
    assert.contains(act.committed_ops, kind = "scheduler.cancel", id = "tick")
    assert.contains(act.committed_ops, kind = "ui.clear")
```

---

## How a request flows through the app

1. Voice capture on any terminal produces a typed `Intent` on the bus:
   `{action: "timer.set", slots: {duration_seconds: "600", label: "pasta"}}`.
2. The scenario engine walks registered apps and calls `match(req)`;
   `kitchen_timer` returns True.
3. TAR calls `new_activation(req)`. Placement resolves the nearest
   kitchen display. Duration 0 would cause the return value to be
   `None`, and the engine would reject the activation.
4. The engine calls `start(env, state)`. TAR commits three ops
   atomically — publish view, arm 1 Hz tick, arm 10-minute expiry — or
   commits none.
5. Each tick arrives as `handle(env, state, trigger)` with
   `trigger.kind == "event"` and `trigger.name == "tick"`. The handler
   decrements `remaining`, patches the UI, and the engine snapshots
   state for crash recovery.
6. When `expiry` fires, the handler emits a TTS op, patches the
   banner, emits a `timer.expired` event on the bus for downstream
   apps, and returns `done = True`. The engine then calls `stop`.
7. If the device disconnects before expiry, `suspend` runs and the
   engine persists state. On reconnect `resume` re-arms the scheduler
   with the remaining time and re-renders the view.

---

## TAL contract cheatsheet

Everything a TAL module returns from a lifecycle hook is a **Result**.
Any field not set keeps the default:

```text
Result {
    state:  any   # new activation state; default: unchanged
    ops:    list  # host ops to commit atomically; default: []
    emit:   list  # triggers to inject back onto the bus; default: []
    done:   bool  # end this activation after commit; default: False
}
```

Rules TAL relies on (from
[application-runtime.md](../plans/application-runtime.md) §TAL
Execution Model):

- **Serial mailbox.** `handle` never runs concurrently for the same
  activation. Helpers can safely read and write `state` in place.
- **State must be JSON-serializable.** Live handles (scheduler IDs,
  placement targets) are opaque string refs the host resolves — not
  pointers.
- **All-or-nothing commits.** If any op in `ops` fails validation, the
  whole hook's effects are rolled back. `state` is only persisted
  alongside a successful commit.
- **Stable scheduler IDs.** Passing the same `id` on `scheduler.after`
  is idempotent. This is why `start` is safe to call from `resume`.
- **No direct clock.** `env.now`, `env.seconds_since_start`, etc. are
  the only source of time; this is what makes the sim harness
  reproducible.
- **No undeclared hosts.** `load("telephony", …)` fails at package load
  if `permissions` does not include a matching `telephony.*` entry.

---

## What this example intentionally leaves out

- **Multiple concurrent timers.** This app keeps one timer per
  activation. The scenario engine already supports many activations of
  one definition; `"set a second timer"` creates a sibling activation.
  No in-app list needed.
- **Cross-device follow.** If the cook leaves the kitchen, a future
  revision can subscribe to presence updates and re-place the view.
  That is a `presence` concern, not core timer logic.
- **Snoozing.** A `timer.snooze` intent would be a one-line addition to
  `handle`: cancel both schedules, add `remaining`, re-run `start`.

Keeping these out matches the "don't design for hypothetical future
requirements" rule in [CLAUDE.md](../CLAUDE.md). When those use cases
land, they extend the same pattern.

---

## Related

- [plans/application-runtime.md](../plans/application-runtime.md) —
  full TAL/TAR specification.
- [plans/scenario-engine.md](../plans/scenario-engine.md) — the
  supervisor that calls `start` / `handle` / `stop` / `suspend` /
  `resume`.
- [usecases.md](../usecases.md) — T1 and adjacent timer/reminder
  stories.
- [plans/repl-capability-plan.md](../plans/repl-capability-plan.md) —
  how the same capabilities are reachable from the REPL, so this app
  is operable without a TAR package during prototyping.
