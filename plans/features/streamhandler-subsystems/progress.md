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
