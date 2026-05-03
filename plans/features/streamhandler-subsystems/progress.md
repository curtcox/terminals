# StreamHandler Subsystems Progress

- 2026-05-02: Created plan. Status: proposed. No implementation started.

## 2026-05-02 — Phase 1

Status: complete

Changes:
- Added private `newStreamHandler(control, runtime)` helper in `terminal_server/internal/transport/control_stream.go` that centralizes all field defaults.
- Rewrote `NewStreamHandler` and `NewStreamHandlerWithRuntime` to delegate to the helper.
- Added field-grouping comments inside the helper (transport dispatch / metrics, terminal+REPL, UI session state, photo-frame defaults, media+route-replay+voice, diagnostics+collaborator defaults) to mark the boundaries the later phases will extract along.
- Added `control_stream_constructor_test.go` with a constructor-invariant test that runs both public constructors and asserts every map, default limit, default duration, and default collaborator (terminal manager, REPL service, no-op recording manager, UI ownership tracker, wake-word dedupe, menu-app policy) is initialized.

Validation:
- `cd terminal_server && go test ./internal/transport -run TestStreamHandlerConstructors -count=1 -v` — pass.
- `cd terminal_server && go test ./internal/transport -count=1 -race` — pass (10.2s).
- `cd terminal_server && go test ./...` — pass (all transport + adjacent suites).
- `cd terminal_server && go build ./...` — pass.

Notes:
- No behavior changes intended. The two constructors were verbatim duplicates apart from `runtime`; the helper preserves order and values.
- Field-grouping comments are intentionally lightweight markers — they will become file boundaries once Phases 2–8 land.
- Phase 0 characterization gaps (reconnect with prior UI, route replay, stale capability generation, bug-report unavailable, command validation errors) are deferred to PR 2 per the plan's PR table.

## 2026-05-02 — Phase 0 (PR 2: characterization tests)

Status: complete

Changes:
- Audited existing tests in `control_stream_test.go`, `control_stream_reconnect_test.go`, `control_stream_command_test.go`, and `errors_test.go` against the plan's high-risk paths. Confirmed pre-existing coverage for: initial capability snapshot (`TestHandleMessageCapabilitySnapshotBootstrapsUnknownDevice`), capability ack reporting (`TestHandleMessageCapabilityLifecycleAckReportsSnapshotAppliedAndGeneration`), stale capability delta and snapshot (`TestHandleMessageCapabilityDeltaRejectsStaleGeneration`, `TestHandleMessageCapabilitySnapshotRejectsStaleGeneration`), rebaseline RegisterAck (`TestHandleMessageCapabilitySnapshotReturnsRegisterAckOnRebaseline`), reconnect with prior UI + route replay (`TestGeneratedSessionUI_RECON_1`), bug-report-unavailable (`TestHandleMessageBugReportRequiresIntake`), bug-report ack happy path (`TestHandleMessageBugReportReturnsAck`), command validation error sentinels (`TestHandleMessageCommandRejectsInvalidAction`, `TestHandleMessageRejectsInvalidCommandKind`, `TestHandleMessageRejectsMissingManualIntent`, `TestHandleMessageRejectsMissingVoiceText`, `TestHandleMessageRejectsMissingCommandDeviceID`), and the error-code mapping table (`TestErrorCodeFor`).
- Added `control_stream_characterization_test.go` to fill the remaining gaps:
  - `TestHandleMessageBugReportIntakeErrorPropagates` — pins the path where `BugReportIntake.File` returns an error: response is exactly one `ServerMessage` with `BugReportAck == nil`, `Error` equal to the intake error's text, and `ErrorCode == ErrorCodeUnknown` (no dedicated error-code mapping today). Locks the contract that `DiagnosticsIntake` must preserve in Phase 6.
  - `TestHandleMessageCommandValidationErrorsReturnSingleErrorResponse` — table-driven across the five command validation sentinels (invalid action, invalid kind, missing manual intent, missing voice text, missing device id). Pins that each emits exactly one `ServerMessage` populated with the stable error code, the sentinel's `Error()` text, an empty `CommandAck`, and forwards the sentinel via `errors.Is`. Locks the response shape `CommandDispatcher` must replicate in Phase 5.
  - `TestHandleMessageCommandRecordsValidationErrorInRecentEvents` — pins that command validation failures still append a `CommandEvent` to the recent-command audit buffer with `Outcome` = `"error:" + err.Error()`. This is the recent-command audit behavior `CommandDispatcher` will own in Phase 5.

Validation:
- `cd terminal_server && go test ./internal/transport -run 'BugReportIntakeError|CommandValidationErrorsReturnSingle|CommandRecordsValidationErrorIn' -count=1 -v` — 3 tests / 7 subtests, all pass.
- `cd terminal_server && go test ./internal/transport -count=1` — pass (9.0s).
- `cd terminal_server && go test ./...` — pass.
- `cd terminal_server && go test -race ./internal/transport -count=1` — pass (10.2s).
- `cd terminal_server && go test -race ./...` — pass.

Notes:
- No production changes; tests only.
- No pre-existing surprises surfaced. Worth flagging for Phase 6: bug-report intake errors currently fall through to `ErrorCodeUnknown` because `errorCodeFor` has no specific mapping for arbitrary intake errors — intentional today, but `DiagnosticsIntake` should keep that mapping unless a follow-up explicitly adds a code.
- Skipped writing a separate "reconnect with previous UI but no route replay" test — `TestGeneratedSessionUI_RECON_1` already exercises both replay paths, and the SetUI-only sub-path is not high-risk for the upcoming Phase 2 (CapabilityLifecycle) extraction (UI replay stays in `StreamHandler` per the plan).

## 2026-05-02 — Phase 2 (PR 3: extract CapabilityLifecycle)

Status: complete

Changes:
- Added `terminal_server/internal/transport/capability_lifecycle.go` with `CapabilityLifecycle` struct, `NewCapabilityLifecycle(*ControlService)`, and a single `CapabilityResult` shape exposing `Messages`, `BeforeCaps`, `AfterCaps`, `AfterDeviceName`, `IsInitialBaseline`, `HadPriorDevice`, and a `RegisterAck` pointer (same pointer as the one in `Messages`) so callers can attach an initial UI descriptor without rebuilding the ack.
- Methods: `HandleHello` returns `[]ServerMessage`; `HandleRegister` returns the raw `RegisterResponse` (UI replay stays in `StreamHandler`); `HandleSnapshot`/`HandleDelta` return `CapabilityResult`; `HandleUpdateCapabilities` covers the deprecated `Capability` message and returns no ack on success.
- The implicit-Hello-on-snapshot-for-unknown-device fallback (previously inlined in `HandleMessage`) moved into `HandleSnapshot` to keep the lifecycle responsible for its own compatibility behavior.
- `capabilityInvalidations` (package-level helper) is reused unchanged. `handleCapabilityChangeEffects` and the suspended-claim helpers stay on `StreamHandler` per the plan's "scenario capability-change effects" boundary.
- Wired `capabilityLifecycle` into `newStreamHandler` so both public constructors initialize it; added a field-grouping comment on `StreamHandler`.
- Rewrote the Hello / CapabilitySnap / CapabilityDelta / Register / Capability branches in `HandleMessage` (control_stream.go) to delegate to the new collaborator. UI replay, overlay replay, route replay, and the `handleCapabilityChangeEffects` call sequence remain in `StreamHandler` and now drive off `result.HadPriorDevice`, `result.IsInitialBaseline`, `result.AfterDeviceName`, `result.RegisterAck`, and `result.BeforeCaps`/`AfterCaps`. The previous `before.Generation == 0 || hasUI` predicate translates to `!result.HadPriorDevice || hasUI` — same boolean.
- Removed the now-unused `internal/device` import from `control_stream.go`.
- Added `capability_lifecycle_test.go` with 9 focused tests: HelloAck shape, RegisterAck shape, initial-baseline detection, second-snapshot is-not-baseline, delta updates capabilities, stale-generation error preserves `errors.Is(err, device.ErrStaleGeneration)`, malformed-device-id error, deprecated-Capability path, and capability invalidations on resource loss.

Validation:
- `cd terminal_server && go build ./...` — pass.
- `cd terminal_server && go test ./internal/transport -run 'HandleHello|HandleRegister|HandleSnapshot|HandleDelta|HandleUpdate' -count=1 -v` — all new tests pass.
- `cd terminal_server && go test ./internal/transport -count=1` — pass (9.1s). The Phase 0 characterization tests (`TestHandleMessageBugReportIntakeErrorPropagates`, `TestHandleMessageCommandValidationErrorsReturnSingleErrorResponse`, `TestHandleMessageCommandRecordsValidationErrorInRecentEvents`) and reconnect tests pass without modification — load-bearing signal that snapshot/delta/register/UI-replay/route-replay behavior did not drift.
- `cd terminal_server && go test ./...` — pass.
- `cd terminal_server && go test -race ./...` — pass (transport 10.2s).

Judgment calls:
- **`CapabilityResult` shape**: kept the higher-level result (per the plan's lean) so `StreamHandler` still owns UI replay and the `handleCapabilityChangeEffects` call. Added `AfterDeviceName` and `HadPriorDevice` to the documented shape because both are needed by the existing UI-replay logic and adding them keeps the predicate translation a one-liner. `RegisterAck` is exposed as a direct pointer (aliased into `Messages`) so the snapshot path can mutate `Initial` without re-walking the message slice.
- **No interface on the seam.** Concrete `*CapabilityLifecycle` is wired in. The single test seam needed (a `*ControlService` constructed against `device.NewManager()`) is already easy to set up.
- **Register stays partially in `StreamHandler`.** The Register path's UI replay reads `resp.Initial` (built by `control.Register`) rather than rebuilding it, and the lifecycle returns `RegisterResponse` directly. Wrapping it in `CapabilityResult` would have forced fake before/after caps, so the asymmetry with Snapshot is intentional.

Notes:
- No protobuf changes. No client changes. `control_stream.go` is now ~5000 lines (capability lifecycle code moved out, but characterization tests and the new file are in separate files).
- `handleCapabilityChangeEffects`, `rememberSuspendedClaims`, `restoreSuspendedClaims`, and the package-level `capabilityInvalidations`/`capabilityResources`/`emitCapabilityEvents`/`shouldDisconnectRouteForLostResources` helpers stay in `control_stream.go`. They straddle UI/route replay and scenario IO; per the plan, scenario capability-change effects are not owned by `CapabilityLifecycle`.
- Stopping after this PR. Awaiting confirmation before starting PR 4 (Phase 3: extract `RouteReplayStore`).

## 2026-05-02 — Phase 3 (PR 4: extract RouteReplayStore)

Status: complete

Changes:
- Added `terminal_server/internal/transport/route_replay.go` with concrete `RouteReplayStore` (own `sync.Mutex`, own `map[string][]iorouter.Route`). API: `NewRouteReplayStore`, `Capture(deviceID, routes)`, `Snapshot(deviceID)`, `Clear(deviceID)`, `MessagesForDevice(deviceID, liveRoutes, useCapturedFallback)`. Both Capture and Snapshot copy the slice so internal state cannot be mutated through the boundary.
- Removed `routeReplayByDevice map[string][]iorouter.Route` field and the `routeReplaySnapshotForDevice` method from `StreamHandler`. Replaced with `routeReplay *RouteReplayStore` initialized in `newStreamHandler`.
- `disconnectRoutesForDevice` now calls `h.routeReplay.Capture(deviceID, routes)` instead of writing directly under `h.mu`.
- The CapabilitySnap branch (was ~lines 759–788) and the Register branch (was ~lines 840–865) of `HandleMessage` previously each had a duplicated 16-line StartStream/RouteStream construction loop. Both are now single-line calls into `h.routeReplay.MessagesForDevice(deviceID, h.routeSnapshotForDevice(deviceID), useFallback)` — `useFallback=true` for Snap (preserves the captured-replay-on-empty-live behavior), `useFallback=false` for Register (preserves the Register branch's live-only behavior).
- Updated `control_stream_constructor_test.go` to assert `h.routeReplay != nil` instead of the old map field.
- Added `route_replay_test.go` with 9 focused tests: live-preferred-over-captured, fallback-when-live-empty, no-fallback-returns-nil, metadata-matches-route-delta (origin=route_delta, webrtc_mode=server_managed, matching StartStream/RouteStream IDs), per-device isolation, scoped Clear, Snapshot returns copy, Capture copies input, empty-deviceID is no-op, plus a concurrent Capture+read race test across multiple devices.

Validation:
- `cd terminal_server && go test ./internal/transport -count=1` — pass (9.2s).
- `cd terminal_server && go test ./...` — pass.
- `cd terminal_server && go test -race ./internal/transport -count=1` — pass (10.2s).
- `cd terminal_server && go test -race ./...` — pass.
- `cd terminal_server && go build ./...` — pass.
- `TestGeneratedSessionUI_RECON_1` (the load-bearing reconnect characterization) and Phase 0 characterization tests pass unmodified — confirms behavior preservation across both Snap and Register replay paths.

Judgment calls:
- **`MessagesForDevice` returns `[]ServerMessage`, not `[]iorouter.Route`.** The plan-suggested target. Eliminates a 16-line duplicated StartStream/RouteStream loop in both `HandleMessage` branches and lets the store own the routeID + metadata construction. The Snap-vs-Register asymmetry (Snap uses captured-replay fallback, Register only uses live routes) is preserved via the `useCapturedFallback` boolean parameter rather than two separate methods — the predicate is small enough that a flag stays clearer than e.g. `MessagesForDeviceWithFallback`. Ties the store to the `ServerMessage` type, which is fine for now (Phase 10 is not on the table); if a future move to a non-transport package is needed, swapping back to a route-returning shape is a localized change.
- **Capture takes the live route slice and copies internally.** Caller (`disconnectRoutesForDevice`) no longer needs to allocate a snapshot. Reduces a couple lines at the call site and keeps the copy responsibility with the type that owns the storage.
- **Did not wire `Clear` into a call site.** Per the plan, the method is exposed for a future caller (e.g. on session reset). Today nothing clears the replay; preserving that behavior. A focused test pins `Clear`'s scoping so a future caller can rely on it.

Notes:
- No protobuf changes. No client changes. No behavior changes intended; characterization tests confirm.
- `routeSnapshotForDevice` (the live-router read) stays in `StreamHandler` — it depends on `h.runtime.Env.IO` and is shared with command-handling paths well beyond the replay seam.
- `control_stream.go` dropped to 4898 lines (down ~30 from the reconcile loop dedupe, plus the `routeReplaySnapshotForDevice` method). The capture-side write path moved entirely; the read-side now goes through the store.
- `internal/transport` test count went up by 9 (one new test file). Suite still passes under `-race`.
- Stopping after this PR. Awaiting confirmation before starting PR 5 (Phase 4: extract `UISessionState`).


## 2026-05-02 — Phase 4 (PR 5: extract UISessionState)

Status: complete

Changes:
- Added `terminal_server/internal/transport/ui_session_state.go` with `UISessionState` (own `sync.Mutex`, own per-device maps). API: `NewUISessionState`, `RememberSetUI(deviceID, []ServerMessage)`, `LastSetUI(deviceID) (ui.Descriptor, bool)`, `SwapMainUIActivation(deviceID, activationID) string`, `ForgetMainUIActivation(deviceID)`, `CaptureMultiWindowResume(deviceID, priorScenario)`, `TakeMultiWindowResume(deviceID) (priorScenario string, priorUI ui.Descriptor, hasPriorUI, taken bool)`, `UIHostBeforeCountAndAdvance(deviceID, totalCount) int`, `MarkUIHostDelivered(map[string]struct{}, totalCount)`.
- Moved fields off `StreamHandler` and into the store: `lastSetUIByDevice`, `lastUIHostEventByDev`, `mainUIActivationByDev`, `multiWindowResume`. Replaced with a single `uiSession *UISessionState` field; constructor initializer collapses to one line.
- Moved helper bodies into the store: `rememberSetUI` (now `RememberSetUI`, scans the outgoing slice for non-relayed `SetUI` and stores the most recent — same semantics, must be called with what is about to be sent), `swapMainUIActivation` (now a one-line passthrough to `SwapMainUIActivation`), `captureMultiWindowResume` (now a one-line passthrough; the atomic check-existing+read-lastUI+write-resume happens inside the store), the lookup half of `restoreMultiWindowResume` (now `TakeMultiWindowResume`; `restoreMultiWindowResume` retains the transition-shaping wrapper because it returns transport types and consults `enterTransitionForScenario`), the `lastUIHostEventByDev` read+advance pair in `uiHostMessagesForDevice`, the bulk `delivered` write at the end of `uiHostMessagesSince`, and the `delete(h.mainUIActivationByDev, ...)` line in `HandleDisconnect` (now `ForgetMainUIActivation`).
- `HandleMessage`: the CapabilitySnap and Register branches both did `h.mu.Lock()` to read `lastSetUIByDevice` and `menuOverlayByDevice` together; that block is now split — the UI lookup goes through `h.uiSession.LastSetUI` (own lock) and only the overlay lookup remains under `h.mu`. The two-lock sequence is allowed because the overlay-and-UI reads were already independent observations, not transactional.
- Updated `control_stream_constructor_test.go` to replace four field-presence assertions with one `h.uiSession != nil` assertion (matching the `routeReplay` pattern from PR 4); kept the `menuOverlayByDevice` assertion since that field stayed on `StreamHandler`.
- Added `ui_session_state_test.go` with 13 focused tests covering: remember-then-recall, relayed-SetUI is not stored for sender, blank-deviceID is a no-op, last-SetUI-in-slice wins, swap returns prior, forget-then-swap returns empty, capture+take round-trip, capture skips `multi_window` scenario, first-capture wins on duplicate, before-count read+advance, mark-delivered bulk write, per-device isolation across all four maps, and a concurrent read/write race across all 7 mutating ops × 8 goroutines × 200 iterations.

Validation:
- `cd terminal_server && go test ./internal/transport -count=1` — pass (9.0s).
- `cd terminal_server && go test ./...` — pass.
- `cd terminal_server && go test -race ./internal/transport -count=1` — pass (10.2s).
- `cd terminal_server && go test -race ./...` — pass.
- `cd terminal_server && go build ./...` — pass.
- `make server-lint` — only pre-existing warnings remain (`route_replay_test.go:153` unused-`t`, `control_stream_characterization_test.go:56` gofumpt). Confirmed pre-existing via `git stash`.
- Behavior-preservation: `TestGeneratedSessionUI_RECON_1` (load-bearing reconnect), `control_stream_command_test.go`, `control_stream_characterization_test.go`, `control_stream_keytext_test.go`, `errors_test.go`, `capability_lifecycle_test.go`, `route_replay_test.go` all pass unmodified.

Judgment calls:
- **Left `menuOverlayByDevice` and `menuOverlayDescriptor` on `StreamHandler`.** Per the plan's escape hatch: overlay state is read by `isMenuOverlayOpen`, `overlayPolicyForDevice`, `shouldDropMainStreamWhileOverlayOpen`, `shouldDropMainInputWhileOverlayOpen`, `openMenuOverlay`, `closeMenuOverlay` — overlay-input policy logic dominates the call sites, and `menuOverlayDescriptor` walks `h.runtime.Engine.RegistrySnapshot()` plus a half-dozen scoped-component-id helpers that are tightly bound to `StreamHandler`'s renderer surface. Moving overlay state in this PR would either drag the policy/render logic with it (scope creep beyond Phase 4) or leave the policy half on `StreamHandler` reaching across the new boundary. The four maps that *did* move all share a common consumer (the reconnect/heartbeat replay path) and are not consulted by overlay-input policy. Defer overlay extraction to a later phase if the plan calls for it; otherwise it stays put.
- **High-level API where transactions matter, accessor pairs where they don't.** `RememberSetUI` owns the scan-for-SetUI logic so call sites stay one-line. `CaptureMultiWindowResume` owns the read-existing+read-lastUI+write atomic so the helper isn't fragmented across the lock. `SwapMainUIActivation` returns the prior in one call (matches the existing helper's contract). On the read-and-advance side, `UIHostBeforeCountAndAdvance` is one atomic read+write per the existing `h.mu` block; `MarkUIHostDelivered` is the bulk-update half. `LastSetUI` and `ForgetMainUIActivation` are simple per-map accessors because their callers don't compose with anything else under the lock today.
- **`TakeMultiWindowResume` returns four scalar values, not the unexported `multiWindowResumeState` struct.** Avoids `revive` complaining about an exported method returning an unexported type. The struct stays unexported because nothing outside `StreamHandler` and the store touches it.
- **Order-of-operations in `RememberSetUI` preserves "remember what was sent."** Existing call sites pass the `out` slice that is about to be returned from `HandleMessage`; the loop walks that slice and stashes the last non-relayed SetUI. Identical to the prior implementation — no reordering relative to the wire.
- **Forget-on-disconnect went through the store.** `HandleDisconnect` now calls `h.uiSession.ForgetMainUIActivation`; the old inline `delete` is gone. Other state (lastSetUI, multiWindowResume, lastUIHostEvent) is intentionally *not* cleared on disconnect to preserve the reconnect-replay behavior pinned by `TestGeneratedSessionUI_RECON_1`.

Moved vs. stayed:
- Moved (state): `lastSetUIByDevice`, `lastUIHostEventByDev`, `mainUIActivationByDev`, `multiWindowResume`.
- Moved (behavior): `rememberSetUI` body; `swapMainUIActivation` body; `captureMultiWindowResume` body; the take-and-delete half of `restoreMultiWindowResume`; the read+advance block inside `uiHostMessagesForDevice`; the bulk-update tail of `uiHostMessagesSince`; the `mainUIActivationByDev` delete inside `HandleDisconnect`.
- Stayed (state): `menuOverlayByDevice`, `menuOverlayPolicy`.
- Stayed (behavior): `restoreMultiWindowResume` wrapper (transition shaping); `menuOverlayDescriptor` and the menu-overlay open/close/policy helpers; `uiHostMessagesSince` body (consults `h.runtime.Env.UI` — not state); `uiHostEventCount`; `handleCapabilityChangeEffects` (cross-subsystem; consults UI state via the new accessors but stays as orchestrator).

Notes:
- `control_stream.go` dropped from 4898 → 4826 lines.
- `StreamHandler.mu` is no longer used to guard any of the moved maps. The fields it still guards are unrelated (overlay state, media streams, sensors, etc.).
- Left the plan status `building`. Stopping after this PR. Awaiting confirmation before starting PR 6 (Phase 5: extract `CommandDispatcher` — characterization-pinned in PR 2).


## 2026-05-02 — Phase 5 (PR 6: extract CommandDispatcher)

Status: complete

Changes:
- Added `terminal_server/internal/transport/command_dispatcher.go` with `CommandDispatcher`. API: `NewCommandDispatcher(h *StreamHandler, runCommand func(context.Context, *CommandRequest) (ServerMessage, error)) *CommandDispatcher`, `Dispatch(ctx, *CommandRequest) ([]ServerMessage, error)`, `BroadcastNotificationsForCommand(cmd, result, beforeCount) []ServerMessage`, plus the unexported `appendEventLocked(CommandEvent)` helper.
- `Dispatch` is the single high-level orchestrator. It owns: dedupe lookup against `h.seen` with the `"deduped"` audit append, pre-command snapshots (`routeSnapshotForDevice`, `broadcastEventCount`, `uiHostEventCount`), invocation of the run-command callback, audit append on error with the `"error:" + err.Error()` outcome, audit append on success with `commandOutcome(...)`, multi-window resume capture, dedupe seen-map record, post-command response assembly (via existing `commandResponses` / `routeUpdatesForCommand` / `paTransitionsForCommand` / `paOverlayClearsForCommand` / `BroadcastNotificationsForCommand` / `uiHostMessagesSince`), and the final `h.uiSession.RememberSetUI`. The order matches the prior inline body byte-for-byte.
- `HandleMessage`'s `Command` branch collapsed from ~92 lines to a single delegation: `return h.commandDispatcher.Dispatch(ctx, msg.Command)`.
- `broadcastNotificationsForCommand` moved from `StreamHandler` (now `CommandDispatcher.BroadcastNotificationsForCommand`). The two non-dispatch call sites in the manual-passthrough handler (around former lines 3688, 3730) now go through `h.commandDispatcher.BroadcastNotificationsForCommand`.
- `appendCommandEventLocked` moved off `StreamHandler` into the dispatcher's unexported `appendEventLocked`.
- `StreamHandler` gains a `commandDispatcher *CommandDispatcher` field; constructor wires `handler.commandDispatcher = NewCommandDispatcher(handler, handler.handleCommand)` after the other collaborators.
- Updated `control_stream_constructor_test.go` to add a `commandDispatcher != nil` assertion (kept the existing `recent` / `recentLimit` checks because those fields stayed on `StreamHandler` — see judgment call).
- Added `command_dispatcher_test.go` with 10 focused tests covering: each of the five validation sentinels (`ErrInvalidCommandKind`, `ErrMissingCommandDeviceID`, `ErrMissingCommandIntent`, `ErrMissingCommandText`, `ErrInvalidCommandAction`) round-tripping through `errors.Is` and the single-`ServerMessage` error response shape, audit append on success, audit append on error with the `"error:" + sentinel.Error()` prefix, audit-buffer trim at `recentLimit` with FIFO eviction, broadcast fan-out only on scenario start/stop (and nil-cmd guard), post-command UI host before-count advancement, and a concurrent-dispatch race test under `-race`.

Validation:
- `cd terminal_server && go test ./internal/transport -count=1` — pass (9.1s).
- `cd terminal_server && go test ./...` — pass.
- `cd terminal_server && go test -race ./internal/transport -count=1` — pass (10.2s).
- `cd terminal_server && go test -race ./...` — pass.
- `cd terminal_server && go build ./...` — pass.
- Pre-existing lint warnings in `route_replay_test.go:153` and `control_stream_characterization_test.go:56` left alone — confirmed still pre-existing.
- Behavior-preservation: `TestHandleMessageCommandValidationErrorsReturnSingleErrorResponse`, `TestHandleMessageCommandRecordsValidationErrorInRecentEvents`, `TestHandleMessageRecentCommandsEviction`, `TestHandleMessageRejectsInvalidCommandKind`, `TestGeneratedSessionUI_RECON_1`, `control_stream_command_test.go`, `control_stream_keytext_test.go`, `capability_lifecycle_test.go`, `route_replay_test.go`, `ui_session_state_test.go`, `errors_test.go` all pass unmodified.

Judgment calls:
- **Single high-level `Dispatch` entry point with a run-command callback.** Per the plan's nudge. The dispatcher owns the orchestration shape (validation → audit → fan-out → RememberSetUI). The actual command body — scenario-engine routing, manual passthroughs, voice intents, system intents, bug-report intake — stays on `StreamHandler` because it crosses subsystem boundaries the dispatcher should not own. The callback is wired as a function-typed field (`runCommand func(context.Context, *CommandRequest) (ServerMessage, error)`) bound to `handler.handleCommand` in the constructor. A function field is cleaner than a single-method interface here: there is exactly one implementation, the signature is small, and Go function values give us the same indirection without the interface ceremony.
- **Audit-buffer fields stay on `StreamHandler`.** This is the explicit judgment call from the plan's "Where the recent-command audit lives" prompt, resolved against the constraint that `control_stream_command_test.go` (`TestHandleMessageRecentCommandsEviction` writes `handler.recentLimit = 2` and reads `handler.recent`) and `control_stream_characterization_test.go` (`TestHandleMessageCommandRecordsValidationErrorInRecentEvents` reads `handler.recent` under `handler.mu`) must stay green unmodified. Moving the fields into the dispatcher would require either touching those characterization tests (forbidden) or maintaining duplicate state in two places (worse). The dispatcher owns the *logic* (`appendEventLocked` lives in `command_dispatcher.go`), the buffer is only mutated through that helper, and `h.recent` / `h.recentLimit` are now read in only two places: `appendEventLocked` and the `SystemIntentRecentCommands` handler (a system-intent reader, not a writer). The audit buffer continues to be guarded by `h.mu`, the same lock that already serializes the dedupe `seen` / `seenOrder` updates the dispatcher coordinates around — so commingling under one lock matches the existing transaction boundary rather than fighting it. This deviates from the "collaborator owns its own mutex" principle for this one PR; the field migration can happen in a later cleanup once the characterization tests are willing to move.
- **Validation sentinels left as package-level vars in `control_stream.go`.** They're used both inside `handleCommand` (which stays on `StreamHandler`) and externally via `errors.Is`; package-scope means file location is cosmetic. Moving them is shuffle work that doesn't strengthen the boundary. Preserving file layout keeps `git blame` clean for callers.
- **Pre-command UI-host snapshot timing preserved exactly.** `Dispatch` snapshots `beforeUIEvents := h.uiHostEventCount()` *before* invoking `runCommand`, matching the prior inline order. The post-command `uiHostMessagesSince(beforeUIEvents, ...)` call therefore sees only events emitted during the command body — same as before.
- **Broadcast-notification ordering preserved.** Within `Dispatch`, the post-command response slice is assembled in the same order as the prior inline body: `commandResponses` → `routeUpdatesForCommand` → `paTransitionsForCommand` → `paOverlayClearsForCommand` → `BroadcastNotificationsForCommand` → `uiHostMessagesSince` → `RememberSetUI`. Verified by reading the diff of the deleted inline block against the new dispatcher body — same calls, same arguments, same order, same conditional `len(...) > 0` guards.
- **Validation error response shape preserved.** Single `ServerMessage{ErrorCode: errorCodeFor(err), Error: err.Error()}` returned alongside the sentinel as the second return value. Confirmed by `TestHandleMessageCommandValidationErrorsReturnSingleErrorResponse` passing unmodified, plus the new `TestDispatcherValidationSentinel*` tests asserting the same shape directly.
- **Did not move `handleCommand`, `handleSystemCommand`, `handleVoiceCommand`, `manualPassthroughTrigger`, `commandResponses`, route-snapshot/route-update/PA-transition/overlay-clear helpers, or `commandOutcome` / `defaultAction`.** All cross subsystem boundaries (scenario runtime, route engine, PA UI affordances) the dispatcher should not own. The dispatcher consults the route/PA helpers but doesn't implement them.

Moved vs. stayed:
- Moved (state): none — see audit-buffer judgment call.
- Moved (behavior): the `Command` branch body of `HandleMessage` (~92 lines, dedupe lookup + pre-command snapshots + run-command + audit append on both paths + multi-window resume + dedupe seen-record + post-command response assembly + RememberSetUI); `broadcastNotificationsForCommand` (now `BroadcastNotificationsForCommand`); `appendCommandEventLocked` (now `appendEventLocked`).
- Stayed (state): `recent`, `recentLimit`, `seen`, `seenOrder`, `seenLimit`. The audit-buffer fields stay on `StreamHandler` for test compatibility (the dedupe state never moved — it predates this plan).
- Stayed (behavior): `handleCommand` and the entire command-body dispatch table (system / voice / manual passthrough / playback metadata / terminal refresh / generic manual trigger); `handleSystemCommand`; `commandResponses`; `routeUpdatesForCommand`, `paTransitionsForCommand`, `paOverlayClearsForCommand`, `routeSnapshotForDevice`, `broadcastEventCount`, `uiHostEventCount`, `uiHostMessagesSince`, `broadcastNotificationsSince`; `captureMultiWindowResume` wrapper, `commandOutcome`, `defaultAction`. All consulted by the dispatcher via `h.<helper>` but their implementations cross subsystem boundaries.

Notes:
- `control_stream.go` dropped from 4826 → 4720 lines (net 106-line shrink: the 92-line inline branch became one delegation, plus the moved `broadcastNotificationsForCommand` and `appendCommandEventLocked` bodies). `command_dispatcher.go` is 169 lines.
- No `.proto` changes, no `terminal_client/` changes, no package moves — Phase 10 still on the table later.
- Plan status stays `building`. Stopping after this PR. Awaiting confirmation before starting PR 7.

## 2026-05-02 — Phase 6 (PR 7: extract DiagnosticsIntake)

Status: complete

Changes:
- Added `terminal_server/internal/transport/diagnostics_intake.go` (49 lines) with `DiagnosticsIntake` collaborator: `NewDiagnosticsIntake(BugReportIntake) *DiagnosticsIntake`, `SetIntake(BugReportIntake)`, and `HandleBugReport(ctx, *diagnosticsv1.BugReport) (ServerMessage, error)`. Owns its own `sync.Mutex`. Returns `(ServerMessage{}, ErrBugReportIntakeUnavailable)` when intake is nil; propagates `intake.File` errors unchanged; returns `ServerMessage{BugReportAck: ack}` on success.
- Replaced `bugReports BugReportIntake` field on `StreamHandler` with `diagnostics *DiagnosticsIntake`. Constructor wires `handler.diagnostics = NewDiagnosticsIntake(nil)` so the field is never nil and the mutex is safe to use before any `SetIntake` call.
- `SetBugReportIntake` is now a one-line passthrough to `h.diagnostics.SetIntake(intake)` — kept as a back-compat shim per the plan so external wiring doesn't break.
- `HandleMessage`'s `case msg.BugReport != nil` branch now delegates to `h.diagnostics.HandleBugReport`; the metric increment (`h.metrics.protocolErrors.Add(1)`) and the `[]ServerMessage{{ErrorCode: errorCodeFor(err), Error: err.Error()}}` wire-format wrapping for the error case stay on `StreamHandler` at the dispatch boundary, mirroring prior extractions.
- `handleBugReportUIAction` (the menu-driven path at control_stream.go:2970) was refactored to also route through `h.diagnostics.HandleBugReport` so both the wire-level BugReport branch and the UI-action path share one nil-check and one error-mapping policy. The function still owns its own `[]ServerMessage` shape (ack + `Notification: "Bug report filed: " + ack.GetReportId()`), but the ack is unwrapped from the collaborator's `ServerMessage{BugReportAck: ack}` return.
- Added `diagnostics_intake_test.go` with 5 focused tests: nil intake returns `ErrBugReportIntakeUnavailable` (and ack is nil); successful intake returns the ack inside `ServerMessage{BugReportAck: ...}`; intake error propagates unchanged via `errors.Is` against a sentinel; context cancellation propagates (stub blocks on `ctx.Done`); concurrent `SetIntake` + `HandleBugReport` race test passes under `-race` (uses a stateless stub so the race detector isolates the collaborator's lock, not the test's bookkeeping).
- Updated `control_stream_constructor_test.go` to assert `h.diagnostics != nil`. Removed the pre-existing `bugReports` field-presence check is N/A — there was no such check before this PR.

Validation:
- `cd terminal_server && go test ./internal/transport -count=1` — pass (9.1s).
- `cd terminal_server && go test ./...` — pass.
- `cd terminal_server && go test -race ./internal/transport -count=1` — pass (10.2s).
- `cd terminal_server && go build ./...` — pass.
- Behavior-preservation: `TestHandleMessageBugReportRequiresIntake`, `TestHandleMessageBugReportReturnsAck`, `TestHandleMessageInputBugReportActionFilesReport`, `TestHandleMessageInputBugReportActionRespectsModalitySources`, `TestHandleMessageBugReportIntakeErrorPropagates` (the Phase 0 characterization test) all pass unmodified.

Judgment calls:
- **Mutex on the collaborator: yes.** Per the plan's leaning. `DiagnosticsIntake` owns a `sync.Mutex` and serializes the `intake` field across `SetIntake` and `HandleBugReport`. The pre-extraction code did `if h.bugReports == nil { ... } ack, err := h.bugReports.File(...)` outside `h.mu`, which was a mild data race against `SetBugReportIntake`. No test exercised that race, but the cost of fixing it (one mutex, one snapshot read) is trivial and matches the precedent set by `UISessionState` and `RouteReplayStore`. The mutex is held only across the field read, then released before calling `intake.File` — so the underlying intake (which can block on I/O) does not serialize against `SetIntake`.
- **`handleBugReportUIAction` routed through the collaborator: yes.** The menu-driven path's response shape (ack + Notification) composed cleanly with `HandleBugReport`'s single-`ServerMessage` return — unwrap the ack from `ServerMessage{BugReportAck: ack}`, build the `Notification` from `ack.GetReportId()`. Both code paths now share one nil-check and one error-mapping policy. The `nil, err` return on intake error in `handleBugReportUIAction` matches prior behavior exactly (caller decides how to surface it; this path returns the error to the input handler which then maps it).
- **`ErrBugReportIntakeUnavailable` stays in `control_stream.go`.** Same reasoning as PR 6's command sentinels: package-level vars are file-cosmetic; moving is shuffle work that doesn't strengthen the boundary.
- **Constructor wiring with `NewDiagnosticsIntake(nil)`.** Mirrors `NewRouteReplayStore` / `NewUISessionState`. The field is never nil; `HandleBugReport` can safely lock its own mutex even before `SetIntake` is called.

Moved vs. stayed:
- Moved (state): the `bugReports BugReportIntake` field — now lives as `intake BugReportIntake` inside `DiagnosticsIntake`, guarded by a new `sync.Mutex` on the collaborator (not `h.mu`).
- Moved (behavior): the nil-intake check, the `intake.File(ctx, report)` call, and the `ServerMessage{BugReportAck: ack}` assembly. The wire-level branch in `HandleMessage` and the menu-driven `handleBugReportUIAction` both delegate through it.
- Stayed (state): nothing else — `DiagnosticsIntake` is pure leaf state (no back-reference to `StreamHandler`, unlike `CommandDispatcher`).
- Stayed (behavior): metrics increment (`h.metrics.protocolErrors.Add(1)`) and the wire-format error wrapping in `HandleMessage`'s BugReport branch — same dispatch-boundary pattern PRs 4–6 used. `handleBugReportUIAction`'s response shape (ack + Notification) and `parseBugReportUIAction` (a pure parser) stay on `StreamHandler`. The UI-affordance helpers `decorateBugReportAffordance` / `withBugReportAffordance` / `hasBugReportAffordance` (control_stream.go:1476–1520) stayed — they render the report-button, they don't file reports. `SetBugReportIntake` stayed as a delegating compatibility method.

Notes:
- `control_stream.go` dropped from 4720 → 4712 lines (net 8-line shrink: the 11-line BugReport branch became 6 lines; the 22-line `handleBugReportUIAction` became 17; partly offset by the new collaborator file). `diagnostics_intake.go` is 49 lines.
- Pre-existing lint warnings in `route_replay_test.go:153` and `control_stream_characterization_test.go:56` left alone — confirmed still pre-existing.
- Plan status stays `building`. Stopping after this PR. Awaiting confirmation before starting PR 8 (Phase 7).

## 2026-05-03 — Phase 7 (PR 8: extract VoicePipeline)

Status: complete

Changes:
- Added `terminal_server/internal/transport/voice_pipeline.go` with `VoicePipeline`: `NewVoicePipeline(handler *StreamHandler) *VoicePipeline`, `SetDeviceAudioPublisher(DeviceAudioPublisher)`, and `HandleAudio(ctx, *VoiceAudioRequest) ([]ServerMessage, error)`.
- Moved per-device voice audio buffers from `StreamHandler.voiceAudioBuffers` into `VoicePipeline.buffers`, guarded by the collaborator's own `sync.Mutex`.
- Moved the live mic-audio publisher from `StreamHandler.deviceAudio` into `VoicePipeline.deviceAudio`; `StreamHandler.SetDeviceAudioPublisher` remains as the public compatibility method and now delegates to the pipeline.
- Moved the raw voice-audio path body out of `control_stream.go`: buffering, publisher fan-out, recording chunk tap, STT transcript selection, wake-word detection/dedupe, runtime voice dispatch, response UI patching, TTS synthesis, `voiceAudioReader`, `readAudioPlayback`, and `recordVoiceAudioChunk` now live in `voice_pipeline.go`.
- `StreamHandler.handleVoiceAudio` collapsed to `return h.voicePipeline.HandleAudio(ctx, va)`. The `HandleMessage` voice branch still owns transport metrics, overlay drop policy, and wire-format error mapping.
- Updated `control_stream_constructor_test.go` to assert `h.voicePipeline != nil` instead of the removed `h.voiceAudioBuffers != nil`.
- Added `voice_pipeline_test.go` with focused tests covering partial buffer accumulation, final-buffer clearing, defensive copying of buffered chunks, publisher fan-out for non-empty chunks only, and final-buffer clearing even when the downstream runtime is not configured.

Validation:
- `cd terminal_server && GOCACHE=/tmp/terminals-go-build go test ./internal/transport -run 'TestVoicePipeline|TestStreamHandlerConstructors|TestControlStreamVoiceAudio' -count=1` — pass.
- `cd terminal_server && GOCACHE=/tmp/terminals-go-build go test ./internal/transport -count=1` — pass.
- `cd terminal_server && GOCACHE=/tmp/terminals-go-build go test ./... -count=1` — pass.
- `cd terminal_server && GOCACHE=/tmp/terminals-go-build go test -race ./internal/transport -count=1` — pass.
- Initial sandboxed `go test ./internal/transport -count=1` failed before compiling because Go could not open the default build cache under `~/Library/Caches/go-build`; rerunning with `GOCACHE=/tmp/terminals-go-build` fixed that.
- Sandboxed full transport test also hit existing websocket tests that bind `127.0.0.1:0`; reran the package and full server suite outside the sandbox with approval.

Judgment calls:
- **VoicePipeline keeps a `*StreamHandler` back-reference for now.** The voice path still depends on runtime dispatch, broadcast lookup, device capability checks, wake-word dedupe, recording manager access, and UI response assembly. Passing all of those as callbacks would make this phase look cleaner on paper but add a noisy interface before the media extraction settles. The state ownership still moves: the mutable voice buffer and publisher are no longer on `StreamHandler`.
- **Wake-word dedupe stayed on `StreamHandler`.** It is part of voice command admission, but existing tests directly override `handler.wakeWordDedupe` to exercise winner policies. Moving it now would force test rewrites or an extra setter that does not otherwise exist. The pipeline uses the existing field through the handler and the plan allowed moving it only if it was clearly part of the audio path.
- **Recording manager stayed on `StreamHandler`.** Phase 8 owns media/recording extraction. The pipeline snapshots `h.recording` and calls the moved `recordVoiceAudioChunk` helper, preserving the prior behavior without pulling recording ownership forward.

Moved vs. stayed:
- Moved (state): `voiceAudioBuffers`, `deviceAudio`.
- Moved (behavior): voice-audio buffer assembly/final clearing, live publisher fan-out, recording chunk write helper, STT transcript selection, wake-word gate/dedupe invocation, voice runtime invocation, response UI/TTS assembly, `voiceAudioReader`, and `readAudioPlayback`.
- Stayed (state): `wakeWordDedupe`, `recording`.
- Stayed (behavior): `HandleMessage` metrics/error mapping/overlay policy, `deviceAllowsVoiceAudio` capability policy, `latestBroadcastForDevice` broadcast selection, and broader runtime/scenario helpers.

Notes:
- `control_stream.go` dropped from 4712 → 4524 lines. `voice_pipeline.go` is 251 lines.
- No `.proto` changes, no `terminal_client/` changes, no package moves.
- Follow-up remains for bounded per-device voice buffers; this PR preserves the existing unbounded behavior intentionally.
- Plan status stays `building`. Next planned phase is Phase 8 (`MediaControlState`).

## 2026-05-03 — Phase 8 (PR 9: extract MediaControlState)

Status: complete

Changes:
- Added `terminal_server/internal/transport/media_control_state.go` with `MediaControlState`: `NewMediaControlState()`, `SetRecordingManager`, `SetWebRTCSignalEngine`, `RegisterStream`, `UnregisterStream`, `MarkStreamReady`, WebRTC peer/engine lookup, media/recording status helpers, recording event reads, playback artifact listing, and playback metadata lookup.
- Moved media stream state from `StreamHandler.mediaStreams` into `MediaControlState.streams`, guarded by the collaborator's own `sync.Mutex`.
- Moved recording manager state from `StreamHandler.recording` into `MediaControlState.recording`; `StreamHandler.SetRecordingManager` remains as the public compatibility method and now delegates to the collaborator.
- Moved WebRTC signaling engine state from `StreamHandler.webrtc` into `MediaControlState.webrtc`; `StreamHandler.SetWebRTCSignalEngine` remains as the public compatibility method and now delegates to the collaborator.
- Kept existing `StreamHandler` helper names (`registerMediaStream`, `unregisterMediaStream`, `markStreamReady`, `serverManagedSignalEngine`, `peerDeviceForStream`, `mediaStreamStatusData`, `recordingStatusData`, `listPlaybackArtifacts`, `playbackMetadataForTarget`) as thin delegates so command/reconnect/WebRTC call sites and tests keep their existing shape.
- Updated `VoicePipeline`'s recording-manager access to use `h.mediaControl.CurrentRecordingManager()`.
- Updated `control_stream_constructor_test.go` to assert `h.mediaControl != nil` instead of direct `mediaStreams` / `recording` fields.
- Added `media_control_state_test.go` with focused tests for recorder start/stop hooks, ready-state accounting for unknown streams, defensive metadata copying for server-managed WebRTC, peer lookup, and engine stream removal.

Validation:
- `cd terminal_server && GOCACHE=/tmp/terminals-go-build go test ./internal/transport -run 'TestMediaControlState|TestStreamHandlerConstructors|TestHandleMessageWebRTCSignal|TestUnregisterMediaStream|TestControlStreamVoiceAudio' -count=1` — pass.
- `cd terminal_server && GOCACHE=/tmp/terminals-go-build go test ./internal/transport -count=1` — pass.
- `cd terminal_server && GOCACHE=/tmp/terminals-go-build go test ./... -count=1` — pass outside sandbox.
- `cd terminal_server && GOCACHE=/tmp/terminals-go-build go test -race ./internal/transport -count=1` — pass outside sandbox.
- Initial sandboxed `go test ./... -count=1` failed because several existing packages/tests need loopback listeners or host discovery (`cmd/server`, `internal/discovery`, `internal/mcpadapter`, `internal/repl`, and transport websocket tests); rerunning outside the sandbox passed.

Judgment calls:
- **Kept `mediaStreamState` as a package-local type.** It is still a transport-only data shape used by the new collaborator; moving it into the file would be cosmetic, and keeping the package-local type avoids churn for nearby helper tests.
- **Left `StreamHandler` wrappers in place.** The plan's public-stability goal matters here: tests and command routing already call the helper names directly. The behavior moved, while the orchestration surface stayed stable.
- **Recorder operations stay outside the collaborator mutex.** `RegisterStream` / `UnregisterStream` snapshot the recorder under lock, release the lock, then call `Start` / `Stop`. This preserves the previous non-blocking lock posture and avoids holding media state while recording backends do I/O.
- **Route-message construction stayed in `StreamHandler`.** `routeUpdatesForCommand` still owns route-delta message fan-out and calls `registerMediaStream` / `unregisterMediaStream`; the new collaborator owns lifecycle state and hooks, not command policy.

Moved vs. stayed:
- Moved (state): `mediaStreams`, `recording`, `webrtc`.
- Moved (behavior): media registration/unregistration storage, ready marking, media status assembly, recording status assembly, WebRTC server-managed lookup, stream peer lookup, recorder start/stop calls, WebRTC stream removal, recording event reads, playback artifact listing, playback metadata lookup.
- Stayed (state): sensor snapshots, suspended claims, route replay, and wake-word dedupe.
- Stayed (behavior): `HandleMessage` metrics/error handling, command route-delta construction and peer fan-out, route replay, disconnect orchestration, and system-intent response shaping.

Notes:
- `control_stream.go` dropped from 4524 → 4374 lines. `media_control_state.go` is 277 lines and `media_control_state_test.go` is 114 lines.
- No `.proto` changes, no `terminal_client/` changes, no package moves.
- Plan status stays `building`. Next planned phase is Phase 9 (`HandleMessage` cleanup).

## 2026-05-03 — Phase 9 (PR 10: simplify StreamHandler dispatch)

Status: complete

Changes:
- Slimmed `StreamHandler.HandleMessage` down to a high-level transport dispatch switch: each branch now delegates to a focused helper for capability lifecycle, heartbeat, telemetry/observations, media/voice/input, or diagnostics.
- Added `protocolError(err)` to centralize the repeated protocol-error metric increment and `ServerMessage{ErrorCode, Error}` assembly used by non-command branches.
- Extracted the capability/reconnect replay orchestration into `handleCapabilitySnapshotMessage`, `handleCapabilityDeltaMessage`, `handleRegisterMessage`, and related helpers. Capability persistence still lives in `CapabilityLifecycle`; `StreamHandler` keeps the cross-subsystem replay/effect coordination.
- Extracted telemetry and observation helpers, including a small package-local `observationSink` interface so observation/artifact-ready routing does not repeat runtime shape checks.
- Extracted heartbeat, stream-ready, WebRTC signal, voice-audio, input, and bug-report wire handlers. The already-extracted collaborators still own their state; `StreamHandler` keeps metrics, overlay admission policy, and wire error mapping.
- Cleaned up lint drift from earlier phases: added revive-required comments to exported `MediaControlState` / `VoicePipeline` methods, changed two intentionally-unused test `*testing.T` parameters to `_`, and let `golangci-lint --fix` apply the gofumpt alignment in `control_stream_characterization_test.go`.

Validation:
- `cd terminal_server && go test ./internal/transport -run 'TestStreamHandlerConstructors|TestHandleMessage|TestControlStream|TestMediaControlState|TestVoicePipeline' -count=1` — pass.
- `cd terminal_server && go test ./internal/transport -count=1` — pass.
- `cd terminal_server && GOCACHE=/tmp/terminals-go-build go test ./... -count=1` — pass.
- `cd terminal_server && GOCACHE=/tmp/terminals-go-build go test -race ./... -count=1` — pass outside sandbox.
- `GOCACHE=/tmp/terminals-go-build GOLANGCI_LINT_CACHE=/tmp/terminals-golangci-lint make server-lint` — pass.
- Initial sandboxed full race run failed before exercising the new code because existing server tests need loopback listeners (`listen tcp 127.0.0.1:0` / `httptest`); rerunning outside the sandbox passed.

Judgment calls:
- **Kept helpers in `control_stream.go`.** This phase was about making dispatch readable, not adding new ownership seams. Moving helpers to new files would make navigation easier in isolation but would blur the boundary with the existing subsystem files.
- **Kept command error handling inside `CommandDispatcher`.** The dispatcher already owns command metrics, dedupe, audit records, and command error response shape. Pulling that back into `HandleMessage` would undo the Phase 5 seam.
- **Kept WebRTC signal errors unchanged.** `handleWebRTCSignal` currently returns only messages and swallows/logs engine errors through the existing response behavior. The new wrapper preserves that exact shape instead of inventing a new error path.
- **Used a tiny `observationSink` interface.** This removes duplicated anonymous type assertions while keeping observation intake transport-local and runtime-agnostic.

Moved vs. stayed:
- Moved (within `control_stream.go` helpers): repeated branch bodies for hello/register/capability snapshot/delta/update, heartbeat, sensor, observation/artifact-ready, flow stats, clock samples, stream-ready, WebRTC signal, voice audio, input, and bug report.
- Stayed (state): no new state moved in this phase.
- Stayed (behavior): subsystem collaborators and all cross-subsystem orchestration remain under `StreamHandler`; no `.proto` changes, no `terminal_client/` changes, no package moves.

Notes:
- `control_stream.go` is now 4447 lines after dispatch cleanup and lint-formatting.
- `make server-lint` is now clean; earlier Phase 8/7 revive comments and two test unused-parameter warnings were resolved as part of this finish pass.
- Plan status remains `building`. The only remaining planned work is optional Phase 10 package moves; no package move is currently justified by this phase.
