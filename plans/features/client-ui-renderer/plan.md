---
title: "Client UI Renderer Refactor"
kind: plan
status: proposed
owner: curtcox
validation: automated
last-reviewed: 2026-05-03
---

# Client UI Renderer Refactor

Target repository path: `plans/features/client-ui-renderer/plan.md`

Extends the completed server-driven UI work in `plans/features/server-driven-ui.md`.
Preserves the architecture rule from `AGENTS.md`, `CLAUDE.md`, and `masterplan.md`:

> Add behavior on the server, not the client. The client remains a generic terminal.

This plan addresses the audit recommendation:

> Split Flutter `main.dart` and create a real `lib/ui` renderer module.

## Problem

`terminal_client/lib/main.dart` has become the composition root, app shell, connection controller, transport diagnostics layer, capability lifecycle manager, server-driven UI renderer, diagnostic chrome host, build metadata display, carrier-selection helper, and platform/runtime policy bridge in one file.

That concentration creates three risks:

1. **Boundary drift.** Scenario-specific behavior can accidentally enter the Flutter client because unrelated client concerns are colocated.
2. **Renderer fragility.** Server-driven UI primitives are harder to extend safely when descriptor rendering is embedded inside app shell and connection code.
3. **Agent/editing risk.** Large mixed-responsibility files are hard for agents and humans to patch without creating unrelated regressions.

The project already has a `terminal_client/lib/ui/` directory, but it is not yet the authoritative renderer module. That directory should become the home for the generic renderer that turns protobuf `terminals.ui.v1.Node` descriptors into Flutter widgets.

## Goals

- Keep the Flutter client generic: no scenario-specific branches, names, package IDs, or behavior in production client code.
- Move server-driven UI rendering into `terminal_client/lib/ui/`.
- Make `main.dart` a minimal entry point.
- Keep `TerminalClientApp` as the stable testable app entry type, but move it out of `main.dart`.
- Split connection/session lifecycle from rendering.
- Split capability lifecycle from rendering and client chrome.
- Split client diagnostic chrome from server-driven content rendering.
- Preserve current behavior during the refactor.
- Improve test locality so renderer, connection, diagnostics, and capability lifecycle can be tested independently.
- Make future UI primitive additions require focused renderer tests instead of broad app-scaffold tests.

## Non-Goals

- No protobuf schema changes in this plan.
- No new server-driven UI primitives unless required to preserve existing behavior.
- No visual redesign of the client.
- No scenario-specific client features.
- No replacement of the current Flutter state-management approach unless a later implementation task creates and justifies a separate plan.
- No Go server orchestration changes.
- No release or distribution changes.

## Design Principles

1. **Renderer is pure generic terminal behavior.** It renders protobuf UI descriptors and emits generic UI actions. It never knows scenario names such as `photo_frame`, `terminal`, `timer`, `red_alert`, or application package IDs.
2. **Client chrome is separate from server content.** Connection status, build metadata, bug-report affordance, transport diagnostics, permission prompts, and privacy/capture indicators are terminal chrome, not server-driven UI content.
3. **Connection lifecycle is not rendering.** Discovery, carrier selection, reconnect, heartbeat, capability snapshot/delta, and incoming response dispatch belong outside `lib/ui`.
4. **Capability lifecycle is not rendering.** Screen metrics, lifecycle deltas, generation tracking, and stale-generation rebaseline behavior belong outside `lib/ui`.
5. **Platform APIs stay behind adapters.** Renderer widgets should not import platform-specific speech, alert, WebRTC, discovery, filesystem, notification, or permission implementations directly.
6. **Every primitive has tests.** A server-driven widget primitive is not complete until there is a focused widget test for rendering and action emission where applicable.
7. **No big-bang rewrite.** Move pure functions and leaf widgets first, then split renderer, chrome, capability, and connection state controllers, then shrink `main.dart`.

## Current State

### Client entry point

`terminal_client/lib/main.dart` currently owns or exposes at least the following responsibilities:

- `TerminalClientApp` composition.
- Control client factory wiring.
- Capability probe factory wiring.
- Media engine and audio playback factory wiring.
- Alert delivery wiring.
- Wake-word controller wiring.
- Media permission probing.
- Build metadata labels and parity notes.
- Control stream diagnostics clipboard text.
- Carrier priority and endpoint resolution.
- Transport error diagnosis.
- Bug-report token word vocabulary and bug-report action prefix.
- Connection lifecycle and reconnect behavior.
- Capability snapshot and delta generation.
- Server response handling.
- UI rendering and client chrome display.

### UI module

`terminal_client/lib/ui/` exists but is not yet the authoritative renderer module.

### Tests

`terminal_client/test/widget_test.dart` already covers many behaviors, including:

- Transport error diagnosis.
- Carrier preference and endpoint resolution.
- Build metadata and server/client parity display.
- Auto-connect behavior.
- Notification delivery.
- Heartbeat behavior.
- Capability snapshot/delta behavior.
- Display geometry updates and debounce.
- Stale capability-generation rebaseline.
- Sensor telemetry.
- Client chrome indicators.

Those tests are valuable and must keep passing during the refactor. As code moves, focused tests should be created next to the relevant subsystem, and the broad widget tests should be reduced to end-to-end smoke coverage rather than deleted wholesale.

## Target Module Layout

The target layout is intentionally conservative. It avoids introducing a large architecture framework while making boundaries explicit.

```text
terminal_client/lib/
  main.dart

  app/
    terminal_client_app.dart
    terminal_client_shell.dart
    client_dependencies.dart
    terminal_client_view_state.dart

  ui/
    server_driven_renderer.dart
    server_driven_action.dart
    server_driven_node_key.dart
    primitive_props.dart
    renderer_policy.dart
    widgets/
      layout_widgets.dart
      text_widgets.dart
      media_widgets.dart
      input_widgets.dart
      overlay_widgets.dart
      device_control_widgets.dart

  connection/
    control_session_controller.dart
    control_response_dispatcher.dart
    carrier_preference.dart
    endpoint_resolution.dart
    transport_diagnostics.dart
    reconnect_policy.dart

  capabilities/
    capability_session.dart
    screen_metrics.dart
    lifecycle_capability_monitor.dart

  diagnostics/
    client_chrome.dart
    build_metadata.dart
    bug_report_chrome.dart
    diagnostic_clipboard.dart

  media/
    existing media files remain here

  discovery/
    existing discovery files remain here

  util/
    existing platform adapters remain here
```

This layout describes the desired end state. During migration, temporary imports back into `main.dart` are acceptable only while a phase is in progress and must be removed by Phase 6.

## Module Contracts

### `main.dart`

`main.dart` should only:

- Import `package:flutter/material.dart`.
- Import `package:terminal_client/app/terminal_client_app.dart`.
- Call `runApp(const TerminalClientApp())`.

Target shape:

```dart
import 'package:flutter/material.dart';
import 'package:terminal_client/app/terminal_client_app.dart';

void main() {
  runApp(const TerminalClientApp());
}
```

### `app/terminal_client_app.dart`

Owns app-level dependency injection and `MaterialApp` wiring.

Responsibilities:

- Define public constructor seams currently on `TerminalClientApp`.
- Build `MaterialApp`.
- Construct `TerminalClientShell` with a dependency bundle.
- Keep test injection seams stable or provide explicit replacements.

Does not:

- Render `uiv1.Node` directly.
- Parse server-driven UI props.
- Decide carrier ordering.
- Generate capability deltas.
- Contain scenario names.

### `app/client_dependencies.dart`

Owns injectable seams and default factories currently defined near the app entry point.

Responsibilities:

- Control client factory.
- Capability probe factory.
- Media engine factory.
- Audio playback factory.
- Alert delivery adapter.
- Wake-word detector factory.
- Media permission probe.
- Time provider.
- Optional test-only screenshot capture seam.

Does not:

- Store live connection state.
- Render widgets.
- Translate server-driven UI actions to protobuf.

### `app/terminal_client_shell.dart`

Owns the visual shell that combines:

- Client chrome.
- Connection controls.
- Diagnostics surfaces.
- Current server-driven content rendered by `ServerDrivenRenderer`.

Does not:

- Implement primitive rendering details.
- Implement carrier selection or reconnect policy.
- Encode/decode protobuf transport envelopes.
- Build capability protobuf messages.

### `app/terminal_client_view_state.dart`

Defines immutable view-state objects consumed by `TerminalClientShell` and `ClientChrome`.

Responsibilities:

- Represent connection phase, diagnostics, build metadata, capture indicators, notification text, and current server-driven root node.
- Keep UI rendering deterministic and testable.

Does not:

- Own timers, stream subscriptions, reconnect policy, or platform effects.

### `ui/server_driven_renderer.dart`

Owns rendering of `uiv1.Node` into Flutter widgets.

Primary API:

```dart
typedef ServerDrivenActionHandler = void Function(ServerDrivenAction action);

typedef MediaSurfaceBuilder = Widget Function(
  BuildContext context,
  String trackId,
);

typedef ImageLoader = Widget Function(
  BuildContext context,
  String url,
);

class ServerDrivenRenderer extends StatelessWidget {
  const ServerDrivenRenderer({
    super.key,
    required this.root,
    required this.onAction,
    this.mediaSurfaceBuilder,
    this.imageLoader,
    this.policy = const RendererPolicy(),
  });

  final uiv1.Node root;
  final ServerDrivenActionHandler onAction;
  final MediaSurfaceBuilder? mediaSurfaceBuilder;
  final ImageLoader? imageLoader;
  final RendererPolicy policy;

  @override
  Widget build(BuildContext context);
}
```

Renderer responsibilities:

- Render each current `uiv1.Node` widget variant.
- Apply generic layout properties.
- Assign stable widget keys for tests and UI patch targeting.
- Emit generic actions for buttons, text inputs, sliders, toggles, dropdowns, and gesture areas.
- Delegate media surfaces and image loading through injected builders where tests or platform behavior need control.
- Apply a single explicit fallback policy for unsupported or malformed nodes.

Renderer must not:

- Open network connections.
- Access discovery.
- Access server connection state.
- Access local filesystem.
- Know scenario/application names.
- Know AI, REPL, MCP, or server orchestration concepts.
- Send protobuf messages directly.

### `ui/server_driven_action.dart`

Defines generic action output from renderer widgets.

```dart
class ServerDrivenAction {
  const ServerDrivenAction({
    required this.componentId,
    required this.action,
    this.value = '',
  });

  final String componentId;
  final String action;
  final String value;
}
```

This maps to `iov1.UIAction` in connection/dispatch code, not inside the renderer.

### `ui/renderer_policy.dart`

Defines renderer fallback behavior.

Initial policy:

- In debug/test mode, unsupported or malformed nodes render a small diagnostic placeholder with a stable key and no scenario-specific text.
- In release mode, unsupported or malformed nodes render `SizedBox.shrink()`.
- Renderer errors must not throw during normal server-driven rendering.

This removes ambiguity around whether malformed nodes render placeholders or empty boxes.

### `ui/primitive_props.dart`

Centralizes parsing of `Node.props`.

Responsibilities:

- Parse colors, alignment, sizing, semantic labels, padding, style strings, and other generic visual props.
- Provide defaults.
- Reject or ignore malformed props consistently.
- Keep map-based proto props from spreading across widget files.

Does not:

- Interpret scenario/application semantics from props.
- Fetch external resources.
- Emit protobuf messages.

### `connection/control_session_controller.dart`

Owns stateful connection lifecycle.

Responsibilities:

- Connect/disconnect.
- Carrier attempt sequence.
- Reconnect schedule.
- Heartbeats.
- Sensor telemetry loop coordination.
- Send hello/capability snapshot on connect.
- Handle stale-generation rebaseline trigger in cooperation with `CapabilitySession`.
- Expose view state and callbacks to the shell.

Does not:

- Render UI primitives.
- Parse visual props.
- Implement widget layouts.

### `connection/control_response_dispatcher.dart`

Translates incoming `ConnectResponse` messages into session state changes and delegated effects.

Responsibilities:

- `SetUI`, `UpdateUI`, and `TransitionUI` handling.
- Notifications.
- WebRTC signal forwarding.
- Start/stop stream dispatch to media engine.
- Play audio dispatch.
- Install/remove bundle dispatch to edge host.
- Bug-report ack handling.
- Control errors.

Does not:

- Render `uiv1.Node`.
- Decide reconnect policy.
- Parse renderer props.

### `connection/carrier_preference.dart`

Moves pure carrier helpers out of `main.dart`:

- `carrierKindFromPriorityName`
- `isCarrierSupportedOnRuntime`
- `buildCarrierPreference`

### `connection/endpoint_resolution.dart`

Moves pure endpoint helpers out of `main.dart`:

- `resolveInitialControlHost`
- `resolvePageHost`
- `resolvePreferredEndpoint`
- `websocketPathFromEndpoint`

### `connection/transport_diagnostics.dart`

Moves transport error and carrier-attempt diagnosis helpers out of `main.dart`:

- `TransportErrorDiagnosis`
- `diagnoseTransportError`
- `classifyCarrierFailure`
- carrier attempt formatting

Clipboard-ready formatting belongs in `diagnostics/diagnostic_clipboard.dart` so transport diagnosis remains independent from UI presentation.

### `capabilities/capability_session.dart`

Owns capability generation and message creation.

Responsibilities:

- Build initial `CapabilitySnapshot`.
- Build `CapabilityDelta`.
- Track generation numbers.
- Emit forced rebaseline snapshot after stale-generation protocol errors.
- Convert probe results and screen metrics to protobuf capabilities.

Does not:

- Render widgets.
- Own app shell state.
- Decide server-driven UI behavior.

### `capabilities/screen_metrics.dart`

Moves `ScreenMetrics`, metrics provider seams, debounce policy, and comparison helpers out of `main.dart`.

### `diagnostics/client_chrome.dart`

Owns terminal-owned chrome widgets:

- Connection status section.
- Build metadata section.
- Transport diagnostics section.
- Client privacy/capture indicators.
- Slot for bug-report affordance supplied by `bug_report_chrome.dart`.

This is allowed client behavior because it supports terminal operation and diagnostics, not scenarios.

### `diagnostics/build_metadata.dart`

Moves:

- `buildMetadataLabel`
- `normalizeBuildValue`
- `buildVersionParityNote`
- `buildServerBuildLine`
- `buildWebConnectionChipLabel`

### `diagnostics/diagnostic_clipboard.dart`

Moves clipboard presentation helpers:

- `buildTransportDiagnosticsClipboardText`
- `buildControlStreamClipboardText`

### `diagnostics/bug_report_chrome.dart`

Moves bug-report diagnostic chrome:

- bug-report action prefix
- token vocabulary
- bug-report UI trigger helpers
- client-side bug-report capture glue

The bug-report module must remain scenario-agnostic. The token vocabulary is diagnostic client chrome, not scenario behavior.

## Renderer Primitive Contract

The renderer must support every currently defined widget in `api/terminals/ui/v1/ui.proto`.

| Proto widget | Renderer behavior | Required test |
| --- | --- | --- |
| `StackWidget` | Renders children in a `Stack` or equivalent generic container. | child order and keys |
| `RowWidget` | Renders children horizontally. | layout smoke test |
| `GridWidget` | Renders children in fixed-column grid. | column count honored |
| `ScrollWidget` | Renders scrollable content by direction. | vertical/horizontal behavior |
| `PaddingWidget` | Applies padding. | padding value honored |
| `CenterWidget` | Centers child content. | child centered structurally |
| `ExpandWidget` | Expands child in flex context. | expansion wrapper present |
| `TextWidget` | Renders text value, style, color. | text and style mapping |
| `ImageWidget` | Renders image URL through injectable loader or `Image.network`. | URL passed to loader |
| `VideoSurfaceWidget` | Delegates to media surface builder by track ID. | builder receives track ID |
| `AudioVisualizerWidget` | Renders generic visualizer placeholder or delegated surface by stream ID. | stream ID preserved |
| `CanvasWidget` | Renders existing draw-ops behavior if already present; otherwise uses renderer fallback policy. | malformed draw ops safe |
| `TextInputWidget` | Renders input and emits value action. | input emits `ServerDrivenAction` |
| `ButtonWidget` | Renders label and emits action. | tap emits action |
| `SliderWidget` | Renders slider and emits value change. | change emits value |
| `ToggleWidget` | Renders switch and emits value change. | toggle emits value |
| `DropdownWidget` | Renders options and emits selection. | selection emits value |
| `GestureAreaWidget` | Captures generic gesture action. | tap emits action |
| `OverlayWidget` | Renders overlay children generically. | overlay child path |
| `ProgressWidget` | Renders progress indicator. | value clamped or used consistently |
| `FullscreenWidget` | Delegates fullscreen intent through generic action/effect channel; no direct platform call in renderer. | delegates without scenario behavior |
| `KeepAwakeWidget` | Delegates keep-awake intent through generic action/effect channel; no direct platform call in renderer. | delegates without scenario behavior |
| `BrightnessWidget` | Delegates brightness intent through generic action/effect channel; no direct platform call in renderer. | delegates without scenario behavior |

## Explicit Boundary Rules

Add or document these rules near the new renderer and in client boundary docs:

1. `terminal_client/lib/ui/**` may import generated UI protobufs and Flutter widgets.
2. `terminal_client/lib/ui/**` may not import discovery, control clients, REPL, MCP, edge package runtime, server concepts, or scenario/application modules.
3. `terminal_client/lib/ui/**` may emit `ServerDrivenAction`, but may not send protobuf messages directly.
4. `terminal_client/lib/connection/**` may translate `ServerDrivenAction` to `iov1.UIAction`.
5. Client chrome may display connection/build/diagnostic state but may not alter server-driven content semantics.
6. Any new UI primitive requires a focused renderer test.
7. Any new client-owned behavior must be classified as one of: connection, permissions, diagnostics, accessibility, rendering, or local I/O bridge.
8. Scenario names and app package IDs are allowed in tests/fixtures only when they are data received from the server, not client-side behavior branches.

## Implementation Phases

### Phase 0: Characterization and Safety Net

Status: completed

Tasks:

- Record current line count and public symbols in `terminal_client/lib/main.dart`.
- Run `flutter test` before refactor and save the passing baseline in the progress log.
- Identify current `uiv1` rendering functions and widget builders in `main.dart`.
- Identify pure helper functions that can move without changing behavior.
- Identify existing tests that should move from `widget_test.dart` into focused test files.

Acceptance criteria:

- Existing `flutter test` passes before code movement.
- A short inventory of moved responsibilities is added to this plan's progress log.
- No production code has changed in this phase except optional comments or TODO markers.

Validation:

```bash
cd terminal_client && flutter test
cd terminal_client && flutter analyze
```

### Phase 1: Move Pure Helpers Out of `main.dart`

Status: completed

Tasks:

- Create `connection/carrier_preference.dart`.
- Move carrier preference helpers and their tests.
- Create `connection/endpoint_resolution.dart`.
- Move endpoint resolution helpers and their tests.
- Create `connection/transport_diagnostics.dart`.
- Move transport error diagnosis and failure-classification helpers.
- Create `diagnostics/build_metadata.dart`.
- Move build metadata and parity helpers.
- Create `diagnostics/diagnostic_clipboard.dart`.
- Move clipboard formatting helpers.
- Update imports.

Acceptance criteria:

- No behavior changes.
- Existing tests are split from `widget_test.dart` into focused unit test files where practical.
- `main.dart` no longer contains carrier ordering, endpoint parsing, transport diagnosis, build metadata, or diagnostic clipboard pure functions.

Suggested test files:

```text
terminal_client/test/connection/carrier_preference_test.dart
terminal_client/test/connection/endpoint_resolution_test.dart
terminal_client/test/connection/transport_diagnostics_test.dart
terminal_client/test/diagnostics/build_metadata_test.dart
terminal_client/test/diagnostics/diagnostic_clipboard_test.dart
```

Validation:

```bash
cd terminal_client && flutter test test/connection test/diagnostics
cd terminal_client && flutter test
cd terminal_client && flutter analyze
```

### Phase 2: Create the Server-Driven Renderer Module

Status: completed

Tasks:

- Create `ui/server_driven_action.dart`.
- Create `ui/renderer_policy.dart`.
- Create `ui/server_driven_renderer.dart`.
- Create `ui/primitive_props.dart`.
- Create `ui/server_driven_node_key.dart`.
- Move current node rendering code from `main.dart` into the renderer module.
- Add stable keys for rendered nodes using node ID and widget kind.
- Ensure renderer emits `ServerDrivenAction` instead of sending protobuf messages.
- Keep media surfaces injectable so renderer does not own WebRTC/media engine details.
- Keep image loading injectable for tests.
- Apply the explicit renderer fallback policy for missing, unsupported, or malformed widget variants.

Acceptance criteria:

- `terminal_client/lib/ui/` contains the complete server-driven renderer.
- `main.dart` does not directly switch on `uiv1.Node` widget variants.
- Renderer tests cover every currently defined proto widget variant.
- Renderer has no scenario-name branches.
- Renderer code does not import connection, discovery, edge runtime, REPL, MCP, or platform adapter modules.

Suggested test file:

```text
terminal_client/test/ui/server_driven_renderer_test.dart
```

Validation:

```bash
cd terminal_client && flutter test test/ui/server_driven_renderer_test.dart
cd terminal_client && flutter test
cd terminal_client && flutter analyze
```

### Phase 3: Split Client Chrome from Server-Driven Content

Status: in progress

Tasks:

- Create `diagnostics/client_chrome.dart`.
- Move connection status display, build metadata display, selectable diagnostics, and privacy/capture indicators into client chrome widgets.
- Create `diagnostics/bug_report_chrome.dart`.
- Move bug-report affordance/token helpers out of `main.dart`.
- Define a small immutable chrome view model or use `TerminalClientViewState` if already introduced.
- Keep chrome separate from `ServerDrivenRenderer` content.

Acceptance criteria:

- Client chrome widgets are tested without a fake control stream.
- Server-driven renderer tests do not need connection diagnostics state.
- Bug-report UI remains available and generic.
- Existing bug-report and diagnostics tests still pass.

Suggested test files:

```text
terminal_client/test/diagnostics/client_chrome_test.dart
terminal_client/test/diagnostics/bug_report_chrome_test.dart
```

Validation:

```bash
cd terminal_client && flutter test test/diagnostics
cd terminal_client && flutter test
cd terminal_client && flutter analyze
```

### Phase 4: Split Capability Lifecycle

Status: in progress

Tasks:

- Create or extend `capabilities/screen_metrics.dart`.
- Move `ScreenMetrics`, `ScreenMetricsProvider`, and display geometry debounce helpers.
- Create `capabilities/capability_session.dart`.
- Move capability snapshot/delta generation and generation tracking.
- Move stale-generation rebaseline logic out of UI shell code.
- Keep platform probing behind `CapabilityProbe` and test seams.

Acceptance criteria:

- Capability snapshot and delta behavior is tested without pumping the full app when possible.
- Existing widget-level capability tests either remain or become higher-level smoke tests.
- `main.dart` does not build protobuf capability messages directly.
- Renderer and diagnostics modules do not import capability-session internals.

Suggested test files:

```text
terminal_client/test/capabilities/capability_session_test.dart
terminal_client/test/capabilities/screen_metrics_test.dart
```

Validation:

```bash
cd terminal_client && flutter test test/capabilities
cd terminal_client && flutter test
cd terminal_client && flutter analyze
```

### Phase 5: Split Connection Session Controller

Status: in progress

Tasks:

- Create `connection/control_session_controller.dart`.
- Move connect/disconnect/reconnect/heartbeat/sensor loop coordination.
- Create `connection/control_response_dispatcher.dart`.
- Move incoming response dispatch into focused handlers.
- Define view-state objects consumed by `TerminalClientShell`.
- Ensure controller exposes deterministic seams for tests.
- Keep existing fake client harness tests passing.

Acceptance criteria:

- Connection lifecycle can be tested without rendering every piece of client chrome.
- Response dispatch can be tested with synthetic `ConnectResponse` messages.
- `TerminalClientShell` consumes controller state and callbacks rather than owning transport policy.
- No renderer code imports control clients.
- No connection code parses primitive visual props.

Suggested test files:

```text
terminal_client/test/connection/control_session_controller_test.dart
terminal_client/test/connection/control_response_dispatcher_test.dart
```

Validation:

```bash
cd terminal_client && flutter test test/connection
cd terminal_client && flutter test
cd terminal_client && flutter analyze
```

### Phase 6: Shrink `main.dart` and Stabilize Public Test Seams

Status: in progress

Tasks:

- Create `app/client_dependencies.dart` if not already created.
- Create `app/terminal_client_app.dart`.
- Create `app/terminal_client_shell.dart`.
- Create `app/terminal_client_view_state.dart` if not already created.
- Move `TerminalClientApp` out of `main.dart`.
- Leave `main.dart` as `runApp(const TerminalClientApp())` only.
- Update tests to import `package:terminal_client/app/terminal_client_app.dart` where direct app construction is needed.
- Keep backward compatibility for tests only if necessary with a short-lived export file; prefer direct app import.

Acceptance criteria:

- `main.dart` is under 25 lines.
- `main.dart` contains no app logic besides `runApp`.
- All public constructor seams used by tests still exist on `TerminalClientApp` or have a clear documented replacement.
- Full client test suite passes.

Validation:

```bash
cd terminal_client && flutter test
cd terminal_client && flutter analyze
cd terminal_client && dart format --set-exit-if-changed .
```

### Phase 7: Boundary Enforcement and Documentation

Status: in progress

Tasks:

- Add `docs/client-boundary.md` or a section in existing client docs.
- Document allowed client-owned behavior categories:
  - connection
  - permissions
  - diagnostics
  - accessibility
  - rendering
  - local I/O bridge
- Document prohibited client behavior:
  - scenario-specific branching
  - server orchestration decisions
  - application package semantics
  - AI provider behavior
- Add a lightweight script to scan `terminal_client/lib` for known scenario names and application IDs.
- Add the script to client CI or docs/process CI.

Acceptance criteria:

- Boundary documentation exists.
- Scenario-name scan exists and is easy to update.
- CI fails on obvious client-side scenario-name leakage, with an allowlist for tests/fixtures only.
- The scan does not block diagnostic vocabulary unless it is used as a behavior branch.

Suggested files:

```text
docs/client-boundary.md
scripts/check-client-boundary.sh
```

Validation:

```bash
./scripts/check-client-boundary.sh
cd terminal_client && flutter test
```

## Test Plan

### Unit tests

- Carrier preference ordering.
- Endpoint resolution.
- Transport error diagnosis.
- Build metadata formatting.
- Diagnostic clipboard formatting.
- Capability generation and generation increments.
- Capability rebaseline after stale-generation error.
- Props parsing defaults and malformed values.
- Renderer fallback policy selection.

### Widget tests

- Every server-driven UI primitive.
- Action emission for input primitives.
- Client chrome state rendering.
- Bug-report affordance rendering.
- Renderer safe fallback for malformed nodes.
- Renderer media surface delegation.

### Integration-style widget tests

- App launches and renders `MaterialApp`.
- Auto-connect starts stream on launch when enabled.
- Notification envelope triggers alert callback only for explicit notifications.
- Register ack updates server build metadata.
- Heartbeats continue while connected.
- Sensor telemetry sends declared signals only.
- Reconnect state still displays diagnostics.
- Server `SetUI` updates server-driven content through `ServerDrivenRenderer`.
- Server `UpdateUI` patches the correct node/component.
- UI action from renderer is translated into `iov1.UIAction` request.

### Negative tests

- Renderer does not throw on missing widget variant.
- Renderer does not throw on unknown props.
- Malformed color/style props fall back safely.
- Empty node IDs still receive deterministic fallback keys.
- Scenario-name scan catches known forbidden tokens in production client code.
- Scenario-name scan permits server-provided test fixture data when explicitly allowlisted.

## Validation Gates

Every PR implementing this plan must pass:

```bash
cd terminal_client && flutter test
cd terminal_client && flutter analyze
cd terminal_client && dart format --set-exit-if-changed .
```

When the boundary script exists:

```bash
./scripts/check-client-boundary.sh
```

Optional full repo gate:

```bash
make client-test
make client-lint
make client-build-web
```

## Migration Strategy

The refactor should proceed by moving code without changing behavior.

1. Move pure helpers first. These are easiest to validate and reduce `main.dart` immediately.
2. Introduce renderer APIs while keeping old call sites temporarily.
3. Add renderer tests before removing old rendering code.
4. Move client chrome into dedicated widgets.
5. Move capability-session logic after pure helpers and before the connection controller split.
6. Move connection controller/response dispatch after renderer, chrome, and capability logic are stable.
7. Shrink `main.dart` last.

Do not combine this refactor with visual redesign, protocol changes, release work, or server behavior changes.

## Review Checklist

Every PR under this plan should answer:

- Does the client remain generic?
- Did any scenario name or application ID enter production `terminal_client/lib` behavior?
- Are server-driven UI primitives still derived from protobuf descriptors?
- Did any renderer code import transport, discovery, REPL, MCP, edge runtime, or server orchestration modules?
- Are new primitive behaviors covered by focused widget tests?
- Did moved tests preserve old behavior?
- Did `flutter analyze`, `flutter test`, and format checks pass?
- Did the PR avoid visual redesign and protocol changes?

## Suggested PR Sequence

### PR 1: Pure helper extraction

Move connection and diagnostics pure helpers out of `main.dart`.

Expected changed files:

- `terminal_client/lib/connection/carrier_preference.dart`
- `terminal_client/lib/connection/endpoint_resolution.dart`
- `terminal_client/lib/connection/transport_diagnostics.dart`
- `terminal_client/lib/diagnostics/build_metadata.dart`
- `terminal_client/lib/diagnostics/diagnostic_clipboard.dart`
- focused tests under `terminal_client/test/connection/` and `terminal_client/test/diagnostics/`

### PR 2: Renderer module skeleton and primitive tests

Create renderer module and move rendering logic.

Expected changed files:

- `terminal_client/lib/ui/server_driven_renderer.dart`
- `terminal_client/lib/ui/server_driven_action.dart`
- `terminal_client/lib/ui/renderer_policy.dart`
- `terminal_client/lib/ui/primitive_props.dart`
- `terminal_client/test/ui/server_driven_renderer_test.dart`

### PR 3: Client chrome split

Move connection/build/diagnostic chrome and bug-report affordance.

Expected changed files:

- `terminal_client/lib/diagnostics/client_chrome.dart`
- `terminal_client/lib/diagnostics/bug_report_chrome.dart`
- `terminal_client/test/diagnostics/client_chrome_test.dart`
- `terminal_client/test/diagnostics/bug_report_chrome_test.dart`

### PR 4: Capability lifecycle split

Move capability generation and screen metrics lifecycle.

Expected changed files:

- `terminal_client/lib/capabilities/capability_session.dart`
- `terminal_client/lib/capabilities/screen_metrics.dart`
- focused capability tests

### PR 5: Connection controller split

Move connection lifecycle and response dispatch.

Expected changed files:

- `terminal_client/lib/connection/control_session_controller.dart`
- `terminal_client/lib/connection/control_response_dispatcher.dart`
- focused connection tests

### PR 6: App shell split and thin `main.dart`

Move app classes and dependency bundle.

Expected changed files:

- `terminal_client/lib/app/terminal_client_app.dart`
- `terminal_client/lib/app/terminal_client_shell.dart`
- `terminal_client/lib/app/client_dependencies.dart`
- `terminal_client/lib/app/terminal_client_view_state.dart`
- `terminal_client/lib/main.dart`

### PR 7: Boundary documentation and scan

Add guardrails.

Expected changed files:

- `docs/client-boundary.md`
- `scripts/check-client-boundary.sh`
- CI or Makefile integration if desired

## Acceptance Criteria for the Whole Plan

- `terminal_client/lib/main.dart` is a minimal entry point under 25 lines.
- `terminal_client/lib/ui/` contains the authoritative server-driven renderer.
- Renderer supports every current `uiv1.Node` widget variant.
- Renderer tests cover every current widget variant.
- UI actions flow as `ServerDrivenAction` from renderer to connection code, then to protobuf `iov1.UIAction`.
- Client chrome is separate from server-driven content.
- Connection lifecycle logic is separate from renderer and chrome.
- Capability lifecycle logic is separate from renderer and chrome.
- No scenario-specific branches exist in production Flutter client code.
- Existing behavior covered by `widget_test.dart` is preserved or moved into focused tests.
- `flutter test`, `flutter analyze`, and `dart format --set-exit-if-changed .` pass.
- Boundary documentation exists.
- Boundary scan exists or is explicitly deferred with a follow-up issue.

## Risks and Mitigations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Refactor changes behavior accidentally | Client reconnect/rendering regressions | Move one responsibility per PR; preserve existing tests before moving code |
| Renderer API becomes too abstract | Slower implementation and harder tests | Keep renderer API small: `root`, `onAction`, optional builders, explicit fallback policy |
| Scenario names are needed for tests | False positives in boundary scan | Allow scenario names in test fixtures only; deny behavior branches in production `lib` code |
| Client chrome and server content become visually inconsistent | UX regression | Keep existing widget output first; defer visual redesign |
| Capability tests become flaky due to widget binding | Slow test feedback | Extract pure capability-session tests with deterministic metrics providers |
| Media surface rendering gets coupled to WebRTC engine | Renderer boundary drift | Use builder injection for media surfaces |
| Bug-report affordance is mistaken for scenario logic | Boundary confusion | Document diagnostics/chrome as allowed client-owned behavior |
| Device-control widgets perform platform side effects from renderer | Boundary drift and test fragility | Delegate fullscreen, keep-awake, and brightness through generic action/effect channel |

## Open Questions

These questions do not block Phase 0 or Phase 1. They should be resolved before or during Phase 2.

- Should `ServerDrivenRenderer` expose one callback for all actions, or typed callbacks per input kind? Initial recommendation: one callback until tests show a need for typed callbacks.
- Should `Node.props` parsing remain hand-written string parsing, or should a generated typed props adapter be introduced later? Initial recommendation: hand-written parser in this plan; generated props belong in a separate protocol/tooling plan.
- Should the boundary scan be denylist-based initially or import-graph-based from the start? Initial recommendation: start denylist-based, then add import-graph checks if false positives stay manageable.
- Should low-level reusable visual widgets remain under `lib/ui/widgets`, or later move to `lib/widgets`? Initial recommendation: keep them under `lib/ui/widgets` until there is non-renderer reuse.

## Implementation Progress

Create `plans/features/client-ui-renderer/progress.md` when work starts.

Suggested initial progress file:

```markdown
# Client UI Renderer Refactor Progress

## 2026-05-02

- Created plan.
- Initial status: proposed.
- Next step: Phase 0 characterization and baseline `flutter test`.
```
