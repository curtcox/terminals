---
title: "Client UI Renderer — Progress Log (May 2026)"
kind: progress-log
parent: plans/features/client-ui-renderer/plan.md
archived: 2026-05-18
---

# Client UI Renderer Refactor Progress

## 2026-05-03

- Closed out the client UI renderer refactor plan after verifying the completed
  module split: `main.dart` is now a thin entry point, the authoritative
  server-driven renderer lives under `terminal_client/lib/ui/`, client chrome,
  capabilities, diagnostics, and connection helpers have focused modules and
  tests, and boundary documentation plus scenario/import scanning are in place.
- Updated `plans/features/client-ui-renderer/plan.md` to mark the overall plan
  and Phases 3 through 7 as completed.

Validation:

```bash
./scripts/check-client-boundary.sh
./scripts/test-check-client-boundary.sh
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/dart format --set-exit-if-changed .
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test
```

Notes:

- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  `flutter analyze` and `flutter test` completed successfully.
- Full Flutter test result: 226 tests passed.

## 2026-05-03

- Continued Phase 5 by moving server-driven UI response derivation for
  `SetUI`, `UpdateUI`, and `TransitionUI` out of `TerminalClientShell` and
  into `terminal_client/lib/connection/control_response_dispatcher.dart`.
- Added typed dispatcher updates for active root replacement, UI event logging
  metadata, transition hints, and transition default-duration derivation.
- Updated the shell to apply dispatcher-produced state while keeping widget
  animation construction and local chrome state in the shell.
- Added focused dispatcher tests for set, patch, and transition response
  handling.

Validation:

```bash
./scripts/check-client-boundary.sh
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/dart format lib/connection/control_response_dispatcher.dart lib/app/terminal_client_shell.dart test/connection/control_response_dispatcher_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/connection/control_response_dispatcher_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/widget_test.dart --plain-name "handles transition_ui responses"
```

Notes:

- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.
- A first targeted widget-test attempt ran in parallel with `flutter analyze`
  and then hit sandboxed DNS for pub.dev advisories; rerunning the targeted
  widget test with approved network access passed.

## 2026-05-03

- Continued Phase 5 by moving sensor telemetry control-request construction
  out of `TerminalClientShell` and into
  `terminal_client/lib/connection/control_session_controller.dart`.
- Added focused controller coverage for telemetry generated from registered
  battery capabilities and for skip cases when no generic telemetry signals are
  available.
- The shell still owns the timer/lifecycle loop and send counters, while the
  protobuf envelope construction now lives with the other connection-session
  request helpers.

Validation:

```bash
./scripts/check-client-boundary.sh
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/dart format --set-exit-if-changed lib/connection/control_session_controller.dart lib/app/terminal_client_shell.dart test/connection/control_session_controller_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/connection/control_session_controller_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
```

Notes:

- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.

## 2026-05-03

- Continued Phase 5 by moving generic renderer-action protobuf translation
  into `terminal_client/lib/connection/control_session_controller.dart`.
- Added `buildUiActionInputRequest` so `ServerDrivenAction` values flow from
  the renderer into `iov1.UIAction` outside the app shell.
- Added `buildKeyInputRequest` for terminal key input request construction,
  further reducing protobuf request assembly in
  `terminal_client/lib/app/terminal_client_shell.dart`.
- Preserved shell-owned local intercepts for privacy toggles and bug-report
  actions before forwarding ordinary UI actions through the connection helper.
- Added focused controller tests for UI action and key input request building.

Validation:

```bash
./scripts/check-client-boundary.sh
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/dart format --set-exit-if-changed lib/connection/control_session_controller.dart lib/app/terminal_client_shell.dart test/connection/control_session_controller_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/connection/control_session_controller_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/connection
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
```

Notes:

- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.
- An earlier mixed `test/diagnostics test/capabilities test/connection` attempt
  was blocked by sandboxed DNS while pub tried to update package advisories
  before running tests; the focused and full `test/connection` runs passed
  afterward.

## 2026-05-03

- Continued Phase 5 by moving pure carrier target resolution, carrier label
  formatting, gRPC discovered-port parsing, and carrier endpoint display labels
  out of `TerminalClientShell` and into
  `terminal_client/lib/connection/control_session_controller.dart`.
- Moved application launch, playback artifact query, and playback metadata
  query command request construction into the same controller helper module.
- Added focused controller coverage for carrier labels, endpoint-to-target
  resolution, endpoint labels, discovered gRPC port fallback behavior,
  application launch requests, and playback diagnostics requests.

Validation:

```bash
./scripts/check-client-boundary.sh
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/dart format lib/connection/control_session_controller.dart lib/app/terminal_client_shell.dart test/connection/control_session_controller_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/connection/control_session_controller_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/widget_test.dart --plain-name "sends system and playback debug commands and renders diagnostics data"
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/widget_test.dart --plain-name "open application queues launch until register ack"
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
```

Notes:

- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.
- One sandboxed targeted widget test attempt failed on DNS for pub.dev advisory
  lookup; rerunning with approved network access passed.

## 2026-05-03

- Continued Phase 5 by moving system diagnostics command request construction
  out of `TerminalClientShell` and into
  `terminal_client/lib/connection/control_session_controller.dart`.
- Added focused controller coverage for runtime status, device status, and
  scenario-registry query request builders so the shell no longer hand-builds
  those protobuf command envelopes.

Validation:

```bash
./scripts/check-client-boundary.sh
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/dart format lib/connection/control_session_controller.dart lib/app/terminal_client_shell.dart test/connection/control_session_controller_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/connection/control_session_controller_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/widget_test.dart --plain-name "sends system and playback debug commands and renders diagnostics data"
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
```

Notes:

- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.
- A first widget-test filter used the wrong test name and ran no tests; the
  exact diagnostics test name above passed.

## 2026-05-03

- Continued Phase 6 by adding
  `terminal_client/lib/app/terminal_client_view_state.dart` for generic
  shell-level view decisions.
- Replaced the client-side `terminal_root` fullscreen branch with a generic
  server-declared `client_chrome=hidden` root prop.
- Updated the server terminal descriptor to request hidden client chrome through
  that generic prop, and updated widget coverage to assert the generic path.
- Added server descriptor coverage for the generic hidden-chrome prop.
- Tightened the client boundary scan so `terminal_root` cannot re-enter
  production Flutter client code as a behavior token.

Validation:

```bash
./scripts/check-client-boundary.sh
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/app/terminal_client_view_state_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/widget_test.dart --plain-name "server root can hide client chrome"
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test
cd terminal_server && go test ./internal/ui
cd terminal_server && GOCACHE=/tmp/terminals-go-build go test ./internal/transport
```

Notes:

- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.
- Initial sandboxed Flutter test attempts failed on DNS for pub.dev advisory
  lookup; rerunning with approved network access passed.
- The first transport test run hit the sandbox loopback bind restriction in
  websocket tests; rerunning with approved loopback access passed.

## 2026-05-03

- Continued Phase 6 by splitting the stateful client shell out of
  `terminal_client/lib/app/terminal_client_app.dart` into
  `terminal_client/lib/app/terminal_client_shell.dart`.
- Promoted the former private `_ControlStreamScaffold` to the public
  `TerminalClientShell` while preserving the existing dependency injection
  seams and behavior.
- Reduced `terminal_client/lib/app/terminal_client_app.dart` to MaterialApp
  wiring plus constructor seam forwarding.

Validation:

```bash
./scripts/check-client-boundary.sh
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/dart format lib/app/terminal_client_app.dart lib/app/terminal_client_shell.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/widget_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test
```

Notes:

- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.

## 2026-05-03

- Continued Phase 3 by moving bug-report action prefix, token vocabulary,
  token/QR generation, local report ID sanitization, receipt state, and
  queued/pending bug-report value types into
  `terminal_client/lib/diagnostics/bug_report_chrome.dart`.
- Updated the app shell to consume those diagnostics-owned types while keeping
  screenshot capture, speech announcement, queue flushing, and transport side
  effects local for later controller extraction.
- Added focused bug-report chrome tests for deterministic token generation and
  local report ID sanitization.

Validation:

```bash
./scripts/check-client-boundary.sh
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/dart format lib/app/terminal_client_app.dart lib/diagnostics/bug_report_chrome.dart test/diagnostics/bug_report_chrome_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/diagnostics/bug_report_chrome_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/diagnostics
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/widget_test.dart
```

Notes:

- The first focused test attempt hit sandboxed DNS for pub.dev advisories;
  rerunning with approved network access passed.
- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.

## 2026-05-03

- Continued Phase 5 by moving synchronous media/control response derivation
  into `terminal_client/lib/connection/control_response_dispatcher.dart`.
- Added a typed dispatcher update for stream-start acknowledgement IDs and
  generic start/stop/route/WebRTC notification text.
- Updated the app shell to consume those dispatcher-derived values while
  keeping media engine, WebRTC, and edge-host side effects local for later
  controller slices.
- Expanded focused dispatcher coverage for stream start, stream stop, route,
  and WebRTC signal response derivation.

Validation:

```bash
./scripts/check-client-boundary.sh
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/dart format lib/connection/control_response_dispatcher.dart test/connection/control_response_dispatcher_test.dart lib/app/terminal_client_app.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/connection/control_response_dispatcher_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/connection
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/widget_test.dart
```

Notes:

- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.

## 2026-05-03

- Continued Phase 5 by moving playback-artifact ID extraction and
  diagnostics-to-application-intent derivation into
  `terminal_client/lib/connection/control_response_dispatcher.dart`.
- Updated the app shell to consume those dispatcher helpers while keeping the
  mutable text controllers and launch UI state local for a later shell split.
- Expanded focused dispatcher coverage for deterministic playback artifact
  selection and generic application-intent ordering.

Validation:

```bash
./scripts/check-client-boundary.sh
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/dart format --set-exit-if-changed lib/connection/control_response_dispatcher.dart test/connection/control_response_dispatcher_test.dart lib/app/terminal_client_app.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/connection/control_response_dispatcher_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/connection
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/widget_test.dart
```

Notes:

- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.

## 2026-05-03

- Continued Phase 5 by creating
  `terminal_client/lib/connection/control_session_controller.dart` for the
  first pure connection-session primitives.
- Moved reconnect delay calculation, carrier attempt diagnostics formatting,
  and resolved connection target data out of
  `terminal_client/lib/app/terminal_client_app.dart`.
- Added focused controller tests and removed the reconnect-delay unit check from
  the broad widget test file.

Validation:

```bash
./scripts/check-client-boundary.sh
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/dart format lib/app/terminal_client_app.dart lib/connection/control_session_controller.dart test/connection/control_session_controller_test.dart test/widget_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/connection/control_session_controller_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/connection
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/widget_test.dart
```

Notes:

- The first focused test attempt hit sandboxed DNS for pub.dev advisories;
  rerunning with approved network access passed.
- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.

## 2026-05-03

- Continued Phase 4 by adding a stateful `CapabilitySession` abstraction for
  registered capability snapshots, generation numbers, accepted ack generation,
  and signature-based change detection.
- Updated `TerminalClientApp` to delegate bootstrap snapshots, capability
  deltas, forced stale-generation rebaselines, privacy capability withdrawal,
  ack tracking, reconnect reset, and stop reset through `CapabilitySession`.
- Expanded focused capability-session coverage for bootstrap generation,
  unchanged suppression, ack-aware generation advancement, forced
  republishing, and reset behavior.

Validation:

```bash
./scripts/check-client-boundary.sh
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/dart format --set-exit-if-changed lib/capabilities/capability_session.dart test/capabilities/capability_session_test.dart lib/app/terminal_client_app.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/capabilities/capability_session_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/widget_test.dart
```

Notes:

- The first focused capability test attempt hit sandboxed DNS for pub.dev
  advisories; rerunning with approved network access passed.
- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.

## 2026-05-03

- Continued Phase 5 by moving pure command-result diagnostics classification
  into `terminal_client/lib/connection/control_response_dispatcher.dart`.
- Moved register-ack metadata extraction and server build normalization into
  the response dispatcher module.
- Updated `TerminalClientApp` to consume typed dispatcher results while keeping
  mutable app-shell state and side effects in place for the next controller
  slices.
- Moved pure flow-plan bundle ID extraction and play-audio source/byte labeling
  into the response dispatcher module.
- Expanded focused dispatcher tests for diagnostics request IDs, diagnostic
  notification fallbacks, ignored command results, register metadata, flow-plan
  bundle IDs, and play-audio labeling.

Validation:

```bash
./scripts/check-client-boundary.sh
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/dart format --set-exit-if-changed lib/connection/control_response_dispatcher.dart test/connection/control_response_dispatcher_test.dart lib/app/terminal_client_app.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/connection/control_response_dispatcher_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/connection
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
```

Notes:

- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.

## 2026-05-03

- Started Phase 5 by creating
  `terminal_client/lib/connection/control_response_dispatcher.dart`.
- Moved pure incoming-response status labeling out of
  `terminal_client/lib/app/terminal_client_app.dart`.
- Moved `UpdateUI` tree patching out of the app shell so it can be tested
  without pumping the full Flutter app.
- Added focused dispatcher tests under
  `terminal_client/test/connection/control_response_dispatcher_test.dart`.
- Left media effects, register handling, notifications, and stream lifecycle in
  the app shell for the next controller/dispatcher slices.

Validation:

```bash
./scripts/check-client-boundary.sh
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/dart format --set-exit-if-changed lib/connection/control_response_dispatcher.dart test/connection/control_response_dispatcher_test.dart lib/app/terminal_client_app.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/connection/control_response_dispatcher_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/connection
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
```

Notes:

- The first focused test attempt hit sandboxed DNS for pub.dev advisories;
  rerunning with approved network access passed.
- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.

## 2026-05-03

- Started Phase 7 by adding `docs/client-boundary.md` with the allowed
  client-owned behavior categories, prohibited scenario/application behavior,
  and module boundary rules.
- Added `scripts/check-client-boundary.sh` to scan production Flutter client
  code for obvious scenario-name and package-ID leakage, excluding generated
  protobuf output.
- Wired the boundary scan into `make all-lint` through a new
  `client-boundary` target.
- Removed the remaining production client special case for the
  `photo_frame_asset_base_url` register metadata key; register metadata remains
  visible through the existing generic diagnostics data surface.

Validation:

```bash
./scripts/check-client-boundary.sh
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/dart format --set-exit-if-changed lib/app/client_dependencies.dart lib/app/terminal_client_app.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/widget_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test
```

Notes:

- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.

## 2026-05-03

- Continued Phase 6 by extracting public dependency seams and default app
  factories from `terminal_client/lib/app/terminal_client_app.dart` into
  `terminal_client/lib/app/client_dependencies.dart`.
- Preserved existing test imports by re-exporting the dependency seam module
  from `terminal_client/lib/app/terminal_client_app.dart`.
- Kept connection/session behavior in the current app shell for the later
  Phase 5 controller split.

Validation:

```bash
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/dart format --set-exit-if-changed lib/app/client_dependencies.dart lib/app/terminal_client_app.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/widget_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test
```

Notes:

- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.
- A first formatter attempt without the local `HOME` completed formatting but
  failed while writing Flutter telemetry outside the workspace; rerunning with
  the local environment passed.

## 2026-05-03

- Started Phase 3 by extracting terminal-owned diagnostic chrome widgets from
  `terminal_client/lib/app/terminal_client_app.dart` into
  `terminal_client/lib/diagnostics/client_chrome.dart`.
- Extracted bug-report affordance and receipt panel presentation into
  `terminal_client/lib/diagnostics/bug_report_chrome.dart`.
- Added focused widget tests for build metadata, connection phase, diagnostics,
  transport status, bug-report button, and bug receipt presentation under
  `terminal_client/test/diagnostics/`.
- Kept connection lifecycle, screenshot capture, bug-report queueing, and
  server action dispatch in the app shell for later Phase 5 work.

Validation:

```bash
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/diagnostics
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test
```

Notes:

- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.

## 2026-05-03

- Continued Phase 2 by hardening `ServerDrivenRenderer` key generation for
  nodes without explicit IDs. Anonymous nodes now receive deterministic
  traversal-path keys instead of identity-hash-based keys.
- Updated scroll rendering to honor `ScrollWidget.direction = "horizontal"`.
- Expanded focused renderer coverage in
  `terminal_client/test/ui/server_driven_renderer_test.dart` across all current
  `uiv1.Node` widget variants, including fallback policy behavior and generic
  action emission for controls.

Validation:

```bash
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/ui/server_driven_renderer_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test
```

Notes:

- Flutter/pub again printed the hosted advisory decode warning for `http`, but
  validation completed successfully.

## 2026-05-03

- Continued Phase 4 by extracting screen metric types and display geometry
  helper functions from `terminal_client/lib/app/terminal_client_app.dart`
  into `terminal_client/lib/capabilities/screen_metrics.dart`.
- Extracted pure capability-session helpers for capability signatures, display
  metadata projection, and stale-generation error detection into
  `terminal_client/lib/capabilities/capability_session.dart`.
- Added focused screen metric tests under
  `terminal_client/test/capabilities/screen_metrics_test.dart`.
- Added focused capability-session tests under
  `terminal_client/test/capabilities/capability_session_test.dart`.
- Updated the broad widget smoke test to import the capability screen metrics
  module directly while it still covers app-level capability lifecycle behavior.

Validation:

```bash
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/capabilities
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/widget_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
```

Notes:

- Flutter/pub again printed the hosted advisory decode warning for `http`.
- The first focused test attempt failed under sandboxed networking while
  checking pub.dev advisories; rerunning with approved network access passed.

## 2026-05-03

- Found that earlier work had already moved `main.dart` to a minimal
  `runApp(const TerminalClientApp())` entry point.
- Found an existing `terminal_client/lib/ui/` renderer module and focused
  renderer test coverage.
- Continued Phase 1 by extracting diagnostics/build metadata helpers from
  `terminal_client/lib/app/terminal_client_app.dart` into
  `terminal_client/lib/diagnostics/build_metadata.dart`.
- Extracted diagnostic clipboard formatting helpers into
  `terminal_client/lib/diagnostics/diagnostic_clipboard.dart`.
- Added focused diagnostics tests under `terminal_client/test/diagnostics/`.

Validation:

```bash
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/diagnostics
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
```

Notes:

- Flutter/pub printed a hosted advisory decode warning for `http`, but both
  commands completed successfully.

## 2026-05-03

- Continued Phase 7 by tightening `scripts/check-client-boundary.sh` beyond
  scenario/package token scanning. It now also rejects imports from
  `terminal_client/lib/ui/**` into client subsystems such as connection,
  discovery, diagnostics, edge, media, platform utilities, and app shell code.
- Added `scripts/test-check-client-boundary.sh` to exercise both failure modes:
  renderer subsystem imports and scenario-token leakage.
- Wired the boundary checker regression test into `make all-test` through a
  new `client-boundary-test` target.
- Updated `docs/client-boundary.md` to document the import-boundary scan.

Validation:

```bash
./scripts/check-client-boundary.sh
./scripts/test-check-client-boundary.sh
```

## 2026-05-03

- Continued Phase 1 by extracting carrier preference helpers from
  `terminal_client/lib/app/terminal_client_app.dart` into
  `terminal_client/lib/connection/carrier_preference.dart`.
- Extracted endpoint resolution helpers into
  `terminal_client/lib/connection/endpoint_resolution.dart`.
- Extracted transport error diagnosis and carrier failure classification into
  `terminal_client/lib/connection/transport_diagnostics.dart`.
- Added focused connection tests under `terminal_client/test/connection/`.
- Updated the broad widget test to import the new connection helper modules
  directly while it still carries legacy smoke coverage.

Validation:

```bash
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/connection
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter test test/widget_test.dart
cd terminal_client && HOME=/Users/curtcox/me/terminals/.home PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache ../.sdk/flutter/bin/flutter analyze
```

Notes:

- Flutter/pub again printed a hosted advisory decode warning for `http`, but
  all validation commands completed successfully.
