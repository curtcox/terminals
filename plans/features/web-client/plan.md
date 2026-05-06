---
title: "HTML/JavaScript Web Client"
kind: plan
status: implemented
owner: curtcox
validation: automated
last-reviewed: 2026-05-05
---

# HTML/JavaScript Web Client

Target repository path: `plans/features/web-client/plan.md`

Extends the core architecture rule from `masterplan.md`, `docs/server.md`, and the completed client renderer refactor:

> Add behavior on the server, not the client. The client remains a generic terminal.

This plan adds a non-Flutter browser client implemented with plain HTML, CSS, and JavaScript. The web client must connect to the existing Terminals server over the browser-friendly control carrier, render the existing server-driven UI protocol, forward generic input and capability events, and remain scenario-agnostic.

## Problem Statement

The repository currently has a Flutter client that can be built for web, but a Flutter web build is still a Flutter application: it requires the Flutter toolchain, ships the Flutter runtime, and keeps browser-specific behavior behind Flutter's abstraction layer. That is appropriate for a cross-platform terminal, but it is heavier than necessary for a browser-first terminal surface and makes low-level browser validation harder.

A small HTML/JavaScript client would provide a second browser implementation of the same client contract. It should be useful for smoke tests, protocol compatibility checks, browser-specific diagnostics, and simple deployment where a static web directory is preferable to a Flutter build artifact.

The risk is architectural drift. A JavaScript client can easily become a parallel product with ad-hoc JSON messages, browser-only scenario behavior, or UI semantics that diverge from the Flutter renderer. This plan prevents that by treating the HTML/JavaScript client as another generic terminal renderer over the same protobuf control protocol and the same closed server-driven UI primitive set.

## Goals

- Add a plain HTML/CSS/JavaScript web client under a dedicated source tree.
- Reuse the existing server control protocol and browser-friendly WebSocket carrier.
- Decode and encode protobuf messages generated from `api/terminals/**` rather than defining ad-hoc JSON contracts.
- Render every current `terminals.ui.v1.Node` primitive supported by the Flutter server-driven renderer.
- Forward generic UI actions as existing `terminals.io.v1.UIAction` payloads.
- Declare browser-backed capabilities honestly and conservatively.
- Provide deterministic unit tests for protocol mapping, renderer behavior, action emission, and fallback handling.
- Provide a static development server workflow and Make targets for local use and CI.
- Keep the client scenario-agnostic and free of server orchestration policy.
- Preserve the existing Flutter client and its web build path.

## Non-Goals

- No replacement of the Flutter client.
- No removal of `make client-build-web` or the Flutter web build.
- No new scenario-specific client behavior.
- No new server-driven UI primitives unless a separate protocol plan adds them.
- No protocol fork for JavaScript.
- No ad-hoc JSON control stream.
- No migration from protobuf to REST for the control plane.
- No implementation of server orchestration, placement, claims, scenario matching, AI behavior, REPL behavior, or app runtime behavior in the browser.
- No support for browser features that cannot be backed by explicit web platform APIs.
- No bundled frontend framework unless a later plan justifies it.

## Design Principles

1. **One protocol, two clients.** The JavaScript client is a second implementation of the existing terminal contract, not a new client/server API.
2. **Plain browser primitives first.** Use standard HTML, CSS, JavaScript modules, WebSocket, WebRTC, Canvas, Web Audio, and browser permissions APIs before adding dependencies.
3. **Server-driven UI remains closed.** The browser renderer supports the same fixed primitive family as the Flutter renderer: layout, content, input, overlay, and system-control descriptors.
4. **No scenario knowledge.** Production web-client code must not branch on scenario names, application package IDs, use-case names, or server module names.
5. **Typed protobuf at the boundary.** WebSocket payloads are binary protobuf envelopes. JSON may appear only as explicitly documented compatibility data inside existing protobuf fields.
6. **Browser capability honesty.** The web client declares only what the browser and user-granted permissions actually support.
7. **Small modules, explicit contracts.** Transport, protobuf conversion, renderer, capabilities, diagnostics, and media bridges are separate modules with focused tests.
8. **Cross-client parity where it matters.** Rendering, action emission, capability declaration, metadata handling, and fallback behavior should match the Flutter client's protocol semantics even when DOM details differ.
9. **Safe fallback.** Unsupported or malformed server-driven nodes render a deterministic diagnostic placeholder in development/test and a non-disruptive empty node in production.
10. **Static deployability.** The built client should be a static artifact that can be hosted by the server admin surface, a local static server, or any ordinary file host.

## Current State

### Existing client architecture

`terminal_client/lib/main.dart` is already a thin Flutter entry point that imports `TerminalClientApp` and calls `runApp(const TerminalClientApp())`. The completed client renderer refactor plan split the Flutter client into focused app, UI, connection, capability, diagnostics, media, discovery, and utility modules.

The Flutter client depends on Flutter, gRPC, protobuf, multicast DNS, WebRTC, WebSocket channel support, audio playback, text-to-speech, local notifications, and platform storage helpers. The existing client therefore remains the full-featured cross-platform terminal.

### Existing server contract

The server owns scenario behavior. `docs/server.md` describes the Go server as the system orchestrator and lists `terminal_server/internal/transport` as the home of gRPC, WebSocket, TCP, and HTTP control carriers. Browser clients should use the WebSocket carrier because browsers do not expose arbitrary TCP sockets and do not support the same gRPC transport model as native clients without a web-specific layer.

The server already advertises and serves a WebSocket control listener via `TERMINALS_CONTROL_WS_HOST` and `TERMINALS_CONTROL_WS_PORT`. `TERMINALS_CONTROL_WS_ALLOWED_ORIGINS` controls cross-origin browser websocket upgrades. Local same-origin and loopback-origin development flows are supported without that setting.

The logical control plane is carrier-neutral. gRPC is preferred where available, and WebSocket is the browser-friendly fallback using binary protobuf envelopes. mDNS TXT metadata advertises per-carrier endpoints and priority, but browsers cannot use mDNS directly without help, so the initial JavaScript client should support manual endpoint entry and optional bootstrap configuration.

### Existing server-driven UI contract

Server-driven UI is a transport-level contract. The server emits `SetUI`, `UpdateUI`, and `TransitionUI` messages carrying `terminals.ui.v1.Node` descriptors. The closed widget contract includes layout primitives (`stack`, `row`, `grid`, `scroll`, `padding`, `center`, `expand`), content primitives (`text`, `image`, `video_surface`, `audio_visualizer`, `canvas`), input primitives (`text_input`, `button`, `slider`, `toggle`, `dropdown`, `gesture_area`), and overlay/system primitives (`overlay`, `progress`, `fullscreen`, `keep_awake`, `brightness`).

The JavaScript client must implement that same closed set and must not add browser-only descriptor semantics.

### Build and validation state

The root `Makefile` already has client and protocol gates, including `client-build-web`, `client-test`, `client-lint`, `client-boundary`, `proto-generate`, `proto-contract-test`, `development-docs-test`, and `all-check`. A new HTML/JavaScript client should add targets without weakening those existing gates.

## Target Layout

The target layout keeps the HTML/JavaScript client outside `terminal_client/` so it is clearly not part of the Flutter app.

```text
web_client/
  README.md
  package.json
  package-lock.json
  index.html
  src/
    main.js
    app.js
    config.js
    state/
      store.js
      view_state.js
    transport/
      control_socket.js
      envelope_codec.js
      reconnect_policy.js
      endpoint_resolution.js
      transport_diagnostics.js
    protocol/
      generated/
        terminals/
          control/v1/*.js
          io/v1/*.js
          ui/v1/*.js
      codec.js
      ui_action_mapper.js
      capability_mapper.js
      metadata_mapper.js
    ui/
      renderer.js
      node_key.js
      primitive_props.js
      renderer_policy.js
      actions.js
      dom/
        layout.js
        content.js
        input.js
        overlay.js
        system.js
      styles/
        base.css
        renderer.css
        diagnostics.css
    capabilities/
      browser_capabilities.js
      screen_metrics.js
      permission_probe.js
      lifecycle_monitor.js
    media/
      webrtc_engine.js
      audio_player.js
      media_surface_registry.js
    diagnostics/
      client_chrome.js
      build_metadata.js
      diagnostic_clipboard.js
      bug_report_chrome.js
    test_support/
      fake_socket.js
      fixtures.js
      dom_test_harness.js
  test/
    transport/
    protocol/
    ui/
    capabilities/
    media/
    diagnostics/
  scripts/
    generate-protos.sh
    serve-dev.mjs
    check-boundary.mjs
    build.mjs
  dist/
    .gitkeep
```

The generated JavaScript protobuf tree should be treated as build output if generation is fast and deterministic in CI. If generator installation is expensive or unstable, generated files may be committed, but `scripts/generate-protos.sh --check` must prove they are current.

## Module Contracts

### `web_client/index.html`

Owns the static shell only.

Responsibilities:

- Load `src/main.js` as an ES module.
- Provide root containers for app chrome, server-driven content, diagnostics, and modal prompts.
- Include minimal static metadata and accessibility landmarks.

Does not:

- Contain scenario logic.
- Inline protocol constants.
- Inline large renderer behavior.

### `src/main.js`

Composition entry point.

Responsibilities:

- Load configuration.
- Create the application store.
- Create transport, capability, renderer, media, and diagnostics dependencies.
- Mount the app into DOM roots.
- Start auto-connect only when configured.

Does not:

- Decode protobuf messages directly.
- Switch on UI widget variants directly.
- Own reconnect policy implementation.

### `src/app.js`

Owns the browser terminal shell.

Responsibilities:

- Coordinate client chrome, connection controls, diagnostics, and current server-driven root node.
- Subscribe to transport state changes.
- Pass server-driven content to the renderer.
- Pass renderer actions to the protocol action mapper and transport send path.
- Request browser permissions only through capability/media modules.

Does not:

- Interpret scenario-specific data.
- Build server UI descriptors.
- Implement transport envelope parsing.

### `src/state/store.js`

Small observable state container.

Responsibilities:

- Hold connection phase, selected endpoint, build metadata, capability state, current UI root, media surfaces, and diagnostics.
- Provide deterministic state transitions for tests.
- Avoid framework-specific global state.

Does not:

- Open sockets.
- Touch DOM directly.
- Parse protobuf bytes.

### `src/transport/control_socket.js`

Owns the WebSocket carrier.

Responsibilities:

- Open and close WebSocket connections.
- Send and receive binary protobuf frames.
- Expose connection lifecycle events.
- Apply reconnect policy.
- Surface close codes and errors to diagnostics.
- Respect browser origin/CORS constraints.

Does not:

- Interpret UI descriptors.
- Build DOM nodes.
- Decide scenario behavior.

### `src/transport/envelope_codec.js`

Owns carrier envelope framing.

Responsibilities:

- Encode outbound `ConnectRequest` envelopes.
- Decode inbound `ConnectResponse` envelopes.
- Validate message kind expectations for the WebSocket carrier.
- Preserve unknown protobuf fields according to generated runtime behavior where supported.

Does not:

- Convert UI actions to DOM events.
- Branch on scenario names.

### `src/transport/endpoint_resolution.js`

Owns browser endpoint selection.

Responsibilities:

- Resolve endpoint from query string, local storage, injected config, or manual input.
- Normalize `ws://` and `wss://` endpoints.
- Avoid mDNS assumptions in browser-only code.
- Provide explicit diagnostics when discovery is unavailable in a browser.

Does not:

- Probe arbitrary LAN ports.
- Use non-browser APIs.

### `src/protocol/generated/**`

Generated JavaScript protobuf bindings.

Responsibilities:

- Represent `api/terminals/**` messages in JavaScript.
- Be regenerated by `scripts/generate-protos.sh`.
- Stay aligned with Go and Dart generated bindings.

Does not:

- Contain hand edits.

### `src/protocol/codec.js`

Thin wrapper around generated protobuf bindings.

Responsibilities:

- Provide stable encode/decode APIs for tests and transport.
- Centralize binary/text fixture helpers.
- Hide generator-specific details from app modules.

Does not:

- Define protocol semantics not present in `.proto`.

### `src/protocol/ui_action_mapper.js`

Maps renderer actions to existing protobuf input messages.

Responsibilities:

- Convert `{ componentId, action, value }` renderer actions into `terminals.io.v1.UIAction` inside the appropriate `ConnectRequest` payload.
- Preserve action token strings exactly as the server emitted them.
- Validate required fields and surface malformed action diagnostics.

Does not:

- Decide what an action means.
- Rewrite action names based on browser UI.

### `src/protocol/capability_mapper.js`

Maps browser capability probes to existing capability protobuf messages.

Responsibilities:

- Declare display, pointer, keyboard, audio output, audio input, camera, screen metrics, WebRTC, notification, and wake/keep-awake support only when detectable.
- Include browser/runtime metadata useful for diagnostics without creating scenario behavior.
- Track permission state as capability deltas when the browser exposes it.

Does not:

- Claim background capability unless the browser can actually sustain it.
- Infer capability from user agent alone when feature probing is available.

### `src/ui/renderer.js`

Authoritative HTML/JavaScript server-driven UI renderer.

Primary API:

```javascript
export class ServerDrivenRenderer {
  constructor({ rootElement, onAction, mediaSurfaceRegistry, imageLoader, policy }) {}
  render(rootNode) {}
  patch(componentId, node) {}
  transition(componentId, transition) {}
  clear() {}
}
```

Renderer responsibilities:

- Render each current `terminals.ui.v1.Node` widget variant into DOM.
- Apply generic layout and style props through `primitive_props.js`.
- Assign stable DOM attributes for tests, such as `data-node-id`, `data-widget-kind`, and `data-component-id`.
- Emit generic renderer actions from input primitives.
- Delegate video/audio media surfaces to `media_surface_registry.js`.
- Delegate image loading through `imageLoader` where tests need control.
- Apply deterministic fallback behavior for unsupported or malformed nodes.

Renderer must not:

- Open sockets.
- Send protobuf messages directly.
- Import server modules.
- Import scenario modules.
- Know scenario names or application package IDs.

### `src/ui/primitive_props.js`

Centralizes prop parsing.

Responsibilities:

- Parse colors, text style tokens, sizes, padding, alignment, grid counts, scroll direction, semantic labels, and accessibility labels.
- Provide default values.
- Ignore or downgrade malformed props consistently.
- Keep map-style proto props from spreading across renderer modules.

Does not:

- Interpret application-specific props.
- Fetch external resources.

### `src/ui/renderer_policy.js`

Defines renderer fallback behavior.

Initial policy:

- Development/test builds render unsupported or malformed nodes as small diagnostic placeholders with stable DOM attributes.
- Production builds render unsupported or malformed nodes as inert empty elements.
- Renderer exceptions are captured and surfaced to diagnostics instead of breaking the full terminal shell.

### `src/capabilities/browser_capabilities.js`

Owns browser feature detection.

Responsibilities:

- Probe display, pointer, keyboard, touch, viewport, device pixel ratio, WebRTC, media devices, notifications, wake lock, fullscreen, clipboard, and visibility APIs.
- Request permissions only when user action or server instruction requires them.
- Emit capability snapshots and deltas through `capability_mapper.js`.

Does not:

- Assume platform capabilities from user agent strings when direct feature detection exists.
- Store sensitive permission grants outside browser-managed state.

### `src/capabilities/screen_metrics.js`

Owns screen and viewport telemetry.

Responsibilities:

- Track viewport size, device pixel ratio, orientation where available, reduced-motion preference, color-scheme preference, and visibility state.
- Debounce resize/orientation events.
- Emit generic capability deltas.

Does not:

- Decide placement or target selection.

### `src/media/webrtc_engine.js`

Owns browser WebRTC bridge.

Responsibilities:

- Handle server-provided WebRTC signaling messages.
- Attach remote tracks to media surfaces by track or stream ID.
- Capture local microphone/camera streams only after permission is granted.
- Surface browser media errors to diagnostics.

Does not:

- Implement media routing policy.
- Decide which devices receive which streams.

### `src/diagnostics/client_chrome.js`

Owns browser client chrome.

Responsibilities:

- Render connection state, selected endpoint, server metadata, browser capability status, transport diagnostics, and permission prompts.
- Keep chrome separate from server-driven content.

Does not:

- Interpret server-driven UI content.
- Replace server content with browser-specific scenario UI.

## Renderer Primitive Contract

The JavaScript renderer must support every current widget variant accepted by the server-driven UI contract.

| Proto widget | DOM behavior | Required test |
| --- | --- | --- |
| `StackWidget` | Renders positioned/layered children in a stack container. | child order and stable node attributes |
| `RowWidget` | Renders children in a horizontal flex row. | flex direction and child order |
| `GridWidget` | Renders children in CSS grid with fixed or parsed columns. | column count honored |
| `ScrollWidget` | Renders scrollable content by typed direction first and legacy string fallback. | vertical/horizontal behavior |
| `PaddingWidget` | Applies padding to child container. | padding values honored |
| `CenterWidget` | Centers child content. | centering class/style present |
| `ExpandWidget` | Expands child in flex context. | expansion style present |
| `TextWidget` | Renders text value, style, color, and accessibility labels. | text and style mapping |
| `ImageWidget` | Renders URL through injected loader or safe `img` element. | URL passed through safely |
| `VideoSurfaceWidget` | Delegates to media surface registry. | track ID preserved |
| `AudioVisualizerWidget` | Renders generic visualizer placeholder or delegated stream surface. | stream ID preserved |
| `CanvasWidget` | Renders typed draw operations when present and legacy JSON only per protocol registry behavior. | typed ops, malformed legacy JSON safe |
| `TextInputWidget` | Renders input and emits value action. | input/change/submit emits action |
| `ButtonWidget` | Renders button and emits configured action. | click emits action |
| `SliderWidget` | Renders range input and emits value changes. | value emitted consistently |
| `ToggleWidget` | Renders checkbox/switch and emits boolean value. | toggle emits value |
| `DropdownWidget` | Renders select/options and emits selected value. | selection emits value |
| `GestureAreaWidget` | Captures generic pointer/click action. | click/tap emits action |
| `OverlayWidget` | Renders overlay children above base content. | overlay layer order |
| `ProgressWidget` | Renders progress indicator with safe value handling. | value clamp/default |
| `FullscreenWidget` | Delegates fullscreen request to browser API through generic effect path. | delegates without scenario behavior |
| `KeepAwakeWidget` | Delegates wake-lock request to browser API when available. | unsupported API safe fallback |
| `BrightnessWidget` | Renders diagnostic fallback or delegates only if a safe browser capability exists. | unsupported API safe fallback |

## Explicit Boundary Rules

Add these rules to the web client README and boundary check:

1. `web_client/src/ui/**` may import generated UI protobuf bindings and DOM helper modules.
2. `web_client/src/ui/**` may not import transport sockets, server orchestration concepts, scenario names, placement, claims, REPL, MCP, or app runtime modules.
3. `web_client/src/ui/**` may emit generic renderer actions, but may not send protobuf messages directly.
4. `web_client/src/protocol/**` may translate renderer actions to protobuf messages.
5. `web_client/src/transport/**` may move protobuf envelopes over WebSocket, but may not interpret scenario semantics.
6. Browser permission prompts belong in capabilities, media, or diagnostics modules, not in renderer primitives.
7. New UI primitives require a protocol plan, generated bindings, renderer support, and focused tests.
8. Scenario names and application IDs are allowed in tests and fixtures only when they are data received from the server.

## Implementation Phases

### Phase 0: Inventory and Generator Decision

Status: completed

Tasks:

- Inventory the current protobuf generation workflow under `api/`.
- Choose the JavaScript protobuf generator and runtime.
- Verify generated JavaScript can encode/decode the same control, IO, and UI messages used by Go and Dart contract tests.
- Decide whether generated JS is committed or generated during CI.
- Record the generator command in `web_client/scripts/generate-protos.sh`.
- Add a small README section explaining why this client is separate from the Flutter client.

Acceptance criteria:

- Generator choice is documented.
- `web_client/scripts/generate-protos.sh` can generate JS bindings from `api/terminals/**`.
- A check mode fails when generated JS is stale, if generated files are committed.
- No production source exists yet that hand-defines protocol message shapes.

Validation:

```bash
cd web_client && npm install
cd web_client && npm run proto:generate
cd web_client && npm run proto:check
make proto-contract-test
```

### Phase 1: Static Shell and Build/Test Harness

Status: completed

Tasks:

- Create `web_client/index.html`.
- Create `web_client/src/main.js`, `src/app.js`, and base CSS.
- Create `package.json` with scripts for test, lint, build, serve, and proto generation.
- Add DOM test harness using a lightweight runner such as Node's built-in test runner plus jsdom, or another explicitly documented minimal dependency.
- Add static build script that copies `index.html`, CSS, JS modules, and generated bindings into `web_client/dist`.
- Add root Make targets for web-client test, lint, build, and serve.

Acceptance criteria:

- The web client can render an empty terminal shell from static files.
- The test runner executes in CI without a browser GUI.
- Build output is static and does not require a Node server at runtime.
- Existing Flutter client targets are unchanged.

Validation:

```bash
cd web_client && npm test
cd web_client && npm run lint
cd web_client && npm run build
python3 -m http.server 60740 --directory web_client/dist
```

### Phase 2: WebSocket Control Transport

Status: completed

Tasks:

- Implement `transport/control_socket.js`.
- Implement `transport/envelope_codec.js`.
- Implement `transport/reconnect_policy.js`.
- Implement `transport/endpoint_resolution.js`.
- Support endpoint selection from query string, persisted local preference, injected static config, and manual input.
- Send initial hello/register request through the existing protobuf envelope.
- Decode register ack, server metadata, set UI, update UI, transition UI, notification, WebRTC signal, and control error responses.
- Surface transport diagnostics in client chrome state.

Acceptance criteria:

- The client can connect to `ws://host:50054` or `wss://host:port` when the server allows the origin.
- Binary protobuf frames are used for control messages.
- Transport errors produce actionable diagnostics.
- Reconnect behavior is deterministic and testable.
- Browser mDNS absence is represented as a diagnostic, not as a failure.

Validation:

```bash
cd web_client && npm test -- transport
make server-test
make proto-contract-test
```

### Phase 3: Browser Capability Snapshot and Delta

Status: completed

Tasks:

- Implement browser feature probes.
- Implement capability snapshot mapping to existing protobuf capability messages.
- Implement screen metrics tracking and debounced deltas.
- Track page visibility and permission state where browser APIs expose it.
- Add conservative fallbacks for unsupported APIs.
- Add tests for common browser capability profiles.

Acceptance criteria:

- The web client declares display, keyboard, pointer/touch, WebRTC, media capture, notification, wake lock, fullscreen, clipboard, and visibility capabilities only when supported.
- Resize and orientation changes produce debounced capability updates.
- Permission denial does not break the control session.
- Capabilities are protocol-compatible with server expectations.

Validation:

```bash
cd web_client && npm test -- capabilities
make proto-contract-test
```

### Phase 4: Server-Driven DOM Renderer

Status: completed

Tasks:

- Implement `ui/renderer.js`.
- Implement `ui/primitive_props.js`.
- Implement `ui/renderer_policy.js`.
- Implement DOM modules for layout, content, input, overlay, and system primitives.
- Add stable DOM attributes for node ID, widget kind, and component ID.
- Implement full-tree `SetUI`, targeted `UpdateUI`, and transition hint handling.
- Emit generic renderer actions for input primitives.
- Add focused renderer tests for every primitive.

Acceptance criteria:

- Renderer supports every current server-driven UI primitive.
- Renderer tests cover every primitive and malformed-node fallback.
- UI actions are emitted as generic action objects and are not sent directly by the renderer.
- Renderer modules do not import transport code.
- Renderer modules contain no scenario-specific branches.

Validation:

```bash
cd web_client && npm test -- ui
cd web_client && npm run boundary
```

### Phase 5: Action, Metadata, and Protocol Compatibility

Status: completed

Tasks:

- Implement `protocol/ui_action_mapper.js`.
- Implement `protocol/metadata_mapper.js`.
- Add JS protocol contract fixtures for register ack metadata, typed server metadata, set UI, update UI, transition UI, UI action, and unknown metadata behavior.
- Ensure typed-first and legacy-fallback behavior matches the protocol evolution rules.
- Add cross-language fixture coverage where practical.
- Add tests that JavaScript decodes the same fixture bytes as Go and Dart for selected messages.

Acceptance criteria:

- Renderer actions become existing protobuf `UIAction` requests.
- Metadata parsing prefers typed fields and falls back to documented legacy map keys.
- Unknown metadata keys are ignored or preserved according to the protocol registry.
- JS protocol tests fail on schema drift.

Validation:

```bash
cd web_client && npm test -- protocol
make proto-contract-test
```

### Phase 6: Browser Media Bridge

Status: completed

Tasks:

- Implement `media/webrtc_engine.js`.
- Implement `media/media_surface_registry.js`.
- Implement `media/audio_player.js`.
- Attach remote video/audio tracks to renderer-delegated surfaces.
- Handle WebRTC signal type enum-first and legacy fallback behavior.
- Request local media permissions only through explicit user action or server-directed generic media flow.
- Add tests with fake RTCPeerConnection and fake media tracks.

Acceptance criteria:

- Server-directed media surfaces render in the correct DOM slots.
- WebRTC signaling uses the existing control protocol.
- Permission denial is reported as diagnostics and capability deltas, not as scenario behavior.
- Media modules do not own routing policy.

Validation:

```bash
cd web_client && npm test -- media
make server-test
```

### Phase 7: Diagnostics, Boundary Guardrails, and Documentation

Status: completed

Tasks:

- Implement browser client chrome.
- Implement diagnostic clipboard text.
- Implement generic bug-report affordance if the existing protocol path is available to the client.
- Add `web_client/scripts/check-boundary.mjs`.
- Document web-client boundaries in `web_client/README.md`.
- Link the new plan from `masterplan.md` and any generated plan index workflow.
- Add root Make targets and optional CI integration.

Acceptance criteria:

- Browser diagnostics show endpoint, transport status, server metadata, build metadata, and browser capability status.
- Boundary scan fails on scenario-name leakage in production `web_client/src/**`.
- README explains setup, connection, origin configuration, validation, and deployment.
- Plan index generation includes this plan.

Validation:

```bash
cd web_client && npm run boundary
cd web_client && npm test -- diagnostics
make development-docs-test
python3 ./scripts/generate-plans-index.py --check
```

### Phase 8: End-to-End Smoke Flow

Status: completed

Tasks:

- Add a local smoke script that starts the server and static web client when the environment permits loopback networking.
- Connect the web client to the server over WebSocket.
- Validate hello/register, capability snapshot, SetUI render, UI action submit, and server response handling.
- Add a deterministic fixture mode for CI environments that cannot open loopback listeners.
- Document manual smoke-test steps for real browsers.

Acceptance criteria:

- A developer can run one command to exercise the browser client against a local server in a permissive environment.
- CI can still validate renderer/protocol behavior without real networking.
- Smoke logs capture enough detail to diagnose WebSocket origin and protobuf decode failures.

Validation:

```bash
make web-client-smoke-test
make web-client-test
make web-client-build
```

## Acceptance Criteria

- `web_client/` exists with a plain HTML/CSS/JavaScript implementation.
- The web client uses generated protobuf bindings from `api/terminals/**`.
- The web client connects through the existing WebSocket control carrier.
- The web client renders every current server-driven UI primitive.
- The web client emits existing protobuf UI action requests for input primitives.
- Browser capability snapshots and deltas are conservative, tested, and protocol-compatible.
- WebRTC/media behavior is bridged through existing signaling and media surface descriptors.
- Diagnostics are separated from server-driven content.
- Boundary checks prevent production scenario-specific behavior.
- Existing Flutter client behavior and targets remain unchanged.
- Root Make targets cover web-client test, lint, build, boundary, and smoke validation.

## Validation Commands

Minimum per-PR validation after the web client exists:

```bash
cd web_client && npm test
cd web_client && npm run lint
cd web_client && npm run build
cd web_client && npm run boundary
make proto-contract-test
make development-docs-test
```

Root targets to add:

```bash
make web-client-test
make web-client-lint
make web-client-build
make web-client-boundary
make web-client-smoke-test
```

Optional full repo gate after CI integration:

```bash
make all-check
```

## Test Plan

### Unit tests

- Endpoint resolution from query string, config, local storage, and manual entry.
- WebSocket reconnect policy and close-code classification.
- Protobuf envelope encode/decode.
- Register ack metadata typed-first and legacy fallback.
- UI action mapping.
- Capability mapping for browser feature profiles.
- Screen metrics debounce and visibility changes.
- Primitive prop parsing and malformed value fallback.
- Renderer fallback policy.
- Diagnostic clipboard formatting.

### DOM renderer tests

- Every current server-driven UI primitive.
- Stable `data-node-id`, `data-widget-kind`, and `data-component-id` attributes.
- SetUI replaces the root tree.
- UpdateUI patches only the targeted component.
- TransitionUI applies transition hints without changing server semantics.
- Input primitives emit generic renderer actions.
- Unsupported and malformed nodes do not break the app shell.
- Media surface widgets delegate to the media registry.

### Protocol compatibility tests

- JavaScript decodes selected golden fixtures also decoded by Go and Dart.
- Unknown metadata behavior matches the registry.
- Typed enum fields are preferred over legacy strings where current protocol migrations define both.
- Legacy fields remain readable during compatibility windows.
- Generated JavaScript protobuf output is current.

### Browser/media tests

- Fake WebSocket transport receives and sends binary payloads.
- Fake RTCPeerConnection handles offer, answer, and ICE candidate flows.
- Remote media tracks attach to registered DOM surfaces.
- Permission denial updates diagnostics and capability state.
- Notification and wake-lock unsupported cases are safe.

### Negative tests

- Renderer does not throw on missing widget variant.
- Renderer does not throw on unknown props.
- Malformed colors, sizes, padding, directions, and canvas payloads fall back safely.
- WebSocket text frames are rejected or diagnosed if binary protobuf is required.
- Scenario-name boundary scan catches forbidden tokens in production source.
- Browser mDNS absence does not block manual WebSocket connection.

## Migration Strategy

This is an additive client. Migration means adding a new browser implementation beside the Flutter client, not moving users off Flutter.

1. Build the JavaScript client as an independent static artifact under `web_client/`.
2. Keep the Flutter client as the primary cross-platform implementation.
3. Use the same protobuf schema and WebSocket carrier so the server does not need scenario-specific branches for JavaScript clients.
4. Add protocol compatibility tests before enabling broad smoke tests.
5. Add renderer coverage before exposing the web client as a recommended workflow.
6. Add optional server static hosting only after the static build is stable.
7. Document feature gaps explicitly rather than silently over-declaring browser capabilities.

No server protocol migration is expected. If implementation reveals missing protocol affordances, create a separate protocol plan and keep this plan blocked on that additive change.

## Review Checklist

Every PR under this plan should answer:

- Does the web client remain a generic terminal?
- Does it use generated protobuf bindings rather than hand-written message shapes?
- Does it connect through the existing WebSocket carrier?
- Did any production source branch on a scenario name, package ID, or use-case name?
- Are server-driven UI primitives rendered from descriptors only?
- Did renderer code avoid importing transport and protocol send paths?
- Are browser capabilities declared only when feature probes or permission APIs support them?
- Are fallback and unsupported-browser cases tested?
- Did `npm test`, `npm run lint`, `npm run build`, `npm run boundary`, and `make proto-contract-test` pass?
- Did the PR avoid changing the Flutter client except where shared docs or Make targets require it?

## Suggested PR Sequence

### PR 1: Web client skeleton and protobuf generation

Expected changed files:

```text
web_client/README.md
web_client/package.json
web_client/index.html
web_client/src/main.js
web_client/src/app.js
web_client/scripts/generate-protos.sh
web_client/scripts/build.mjs
web_client/test/protocol/*
Makefile
```

### PR 2: WebSocket transport and endpoint diagnostics

Expected changed files:

```text
web_client/src/transport/control_socket.js
web_client/src/transport/envelope_codec.js
web_client/src/transport/reconnect_policy.js
web_client/src/transport/endpoint_resolution.js
web_client/src/transport/transport_diagnostics.js
web_client/test/transport/*
```

### PR 3: Capability snapshot and browser lifecycle

Expected changed files:

```text
web_client/src/capabilities/browser_capabilities.js
web_client/src/capabilities/screen_metrics.js
web_client/src/capabilities/permission_probe.js
web_client/src/capabilities/lifecycle_monitor.js
web_client/src/protocol/capability_mapper.js
web_client/test/capabilities/*
```

### PR 4: Server-driven DOM renderer

Expected changed files:

```text
web_client/src/ui/renderer.js
web_client/src/ui/primitive_props.js
web_client/src/ui/renderer_policy.js
web_client/src/ui/actions.js
web_client/src/ui/dom/*.js
web_client/src/ui/styles/*.css
web_client/test/ui/*
```

### PR 5: Action mapping and protocol contract fixtures

Expected changed files:

```text
web_client/src/protocol/ui_action_mapper.js
web_client/src/protocol/metadata_mapper.js
web_client/src/protocol/codec.js
web_client/test/protocol/*
api/testdata/envelopes/*
Makefile
```

### PR 6: Browser media bridge

Expected changed files:

```text
web_client/src/media/webrtc_engine.js
web_client/src/media/audio_player.js
web_client/src/media/media_surface_registry.js
web_client/test/media/*
```

### PR 7: Diagnostics, boundary scan, and docs

Expected changed files:

```text
web_client/src/diagnostics/client_chrome.js
web_client/src/diagnostics/build_metadata.js
web_client/src/diagnostics/diagnostic_clipboard.js
web_client/src/diagnostics/bug_report_chrome.js
web_client/scripts/check-boundary.mjs
web_client/README.md
masterplan.md
Makefile
```

### PR 8: End-to-end smoke validation

Expected changed files:

```text
web_client/scripts/serve-dev.mjs
scripts/test-web-client-smoke.sh
Makefile
.github/workflows/*
```

## Whole-Plan Acceptance Criteria

- `plans/features/web-client/plan.md` is linked from the plan index or `masterplan.md`.
- `web_client/` builds to a static artifact in `web_client/dist`.
- The JavaScript client uses generated protobuf bindings from `api/terminals/**`.
- The JavaScript client connects to the existing WebSocket control carrier with binary protobuf envelopes.
- The JavaScript renderer supports every current server-driven UI primitive.
- `SetUI`, `UpdateUI`, and `TransitionUI` are handled in the browser renderer.
- Input primitives emit existing protobuf `UIAction` messages through the transport layer.
- Browser capability snapshots and deltas are conservative and tested.
- WebRTC signaling and media surfaces use existing protocol messages.
- Client chrome and diagnostics are separate from server-driven content.
- Boundary checks prevent scenario-specific production behavior.
- JavaScript protocol contract tests cover representative golden fixtures shared with Go and Dart.
- The existing Flutter client still passes its current validation gates.
- `make web-client-test`, `make web-client-lint`, `make web-client-build`, `make web-client-boundary`, `make proto-contract-test`, and `make development-docs-test` pass.
