Harden scenario priority and preemption (Phase 7). Today the engine
suspends lower-priority scenarios when a higher-priority one activates and
pops the suspend stack on `Stop`, but it never notifies the suspended
scenario that it has lost its IO, and never asks it to resume. That means a
long-running scenario (for example `AudioMonitorScenario` holding a live
audio-hub subscription) keeps its side effects alive while `red_alert`
preempts the device, which will eventually produce duplicate audio routes
and zombie goroutines once real IO lands.

Close the gap so preemption is observable and reversible.

**Scope**

1. Extend the `scenario.Scenario` interface (in
   `internal/scenario/scenario.go`) with optional `Suspend()` / `Resume(ctx,
   env)` hooks. Default behavior (for scenarios that don't implement them)
   must remain a no-op. Prefer a small optional-interface pattern
   (`type Suspendable interface { Suspend() error }`) over forcing every
   existing scenario to grow two methods.

2. Have `Engine.Activate` call `Suspend()` on the scenario being
   suspended, and `Engine.Stop` call `Resume(ctx, env)` on the scenario
   being restored from the suspend stack. The engine already has the
   device-keyed suspend stack in `suspendedDev`; extend it to remember the
   `Scenario` instance (not just name/priority) so we can dispatch the
   hooks.

3. Implement `Suspend`/`Resume` on `AudioMonitorScenario`:
   - `Suspend` cancels the classifier goroutine and releases the live
     `DeviceAudio` subscription (same code path as today's `Stop`).
   - `Resume` re-subscribes to device audio and restarts the classifier,
     using the original target and source-device from the trigger stored
     on the scenario.

4. Tests:
   - Engine-level: activate `PriorityNormal` scenario with a fake
     suspendable, activate `PriorityCritical` on the same device, assert
     `Suspend` was called exactly once. `Stop` the critical scenario,
     assert `Resume` was called exactly once and the registry's active
     entry is back to the original scenario.
   - Runtime-level: start `audio_monitor` for `d1`, subscribe count == 1.
     Trigger `red_alert` targeting `d1`, assert subscriber count drops to
     0. Stop `red_alert`, assert subscriber count returns to 1 and a
     published PCM event is still routed to the classifier.

**Wiring**

No changes needed in `cmd/server` — `RegisterBuiltins` already registers
`AlertScenario` at `PriorityCritical`. The builtin just needs the new
hooks.

**Out of scope**

Scenario.Resume that has to re-open IO routes (IntercomScenario /
PASystemScenario) — they own their IO once and don't tear it down in
Stop. Leave a TODO there; they can be migrated after the engine contract
is settled.
