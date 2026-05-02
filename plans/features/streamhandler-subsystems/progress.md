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
