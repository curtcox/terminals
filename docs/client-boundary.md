# Client Boundary

The Flutter client is a generic terminal runtime. It renders descriptors,
reports capabilities, moves media and input, and keeps the control stream
healthy. Scenario decisions live on the Go server.

## Allowed Client-Owned Behavior

- Connection: discovery, manual endpoints, carrier fallback, reconnect, and
  readiness.
- Permissions: platform prompts and local privacy/capture controls.
- Diagnostics: build metadata, transport status, bug-report capture, and local
  error reporting.
- Accessibility: focus, semantics, text scaling support, and input affordances.
- Rendering: generic protobuf UI descriptors through `terminal_client/lib/ui/`.
- Local I/O bridge: media playback/capture, sensors, edge artifact storage, and
  platform adapters requested by server commands.

## Prohibited Client Behavior

- Branching on scenario names or application package IDs.
- Choosing which scenario, app, or orchestration policy should run.
- Encoding application-specific semantics into renderer primitives.
- Adding ad-hoc JSON contracts instead of protobuf messages.
- Importing server orchestration concepts into `terminal_client/lib/ui/`.
- Selecting AI providers or model behavior from Flutter code.

## Module Rules

- `terminal_client/lib/main.dart` stays a minimal `runApp` entry point.
- `terminal_client/lib/app/` wires dependencies and owns the shell until the
  connection controller split is complete.
- `terminal_client/lib/ui/` renders `terminals.ui.v1.Node` descriptors and emits
  generic `ServerDrivenAction` values only.
- `terminal_client/lib/connection/` may translate generic renderer actions into
  protobuf control or I/O messages.
- `terminal_client/lib/capabilities/` owns capability probing, display metrics,
  generation tracking, and capability message construction.
- `terminal_client/lib/diagnostics/` owns terminal chrome; it may display
  server-provided metadata generically, but must not special-case scenarios.

## Validation

Run the boundary scan before client refactor PRs:

```bash
./scripts/check-client-boundary.sh
```

The scan is intentionally lightweight. It catches obvious scenario-name and
package-ID leakage in production client code. Tests and generated protobuf code
are outside its scope; server-provided fixture strings belong in tests, not in
production client behavior.

