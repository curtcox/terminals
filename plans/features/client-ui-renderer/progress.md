# Client UI Renderer Refactor Progress

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
