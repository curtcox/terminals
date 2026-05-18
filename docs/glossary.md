# Terminals — Glossary

Domain terms used across the codebase, plans, and docs. One entry per concept: definition, canonical source, and a reference link.

---

**Activation**  
A live running instance of a scenario. A scenario *definition* is a singleton; an *activation* is a live instance with its own ID, resource claims, target devices, and optional resume snapshot. Multiple activations of the same scenario can coexist (e.g., two concurrent timer activations).  
→ [`internal/scenario/`](../terminal_server/internal/scenario/), [`plans/features/scenario-engine.md`](../plans/features/scenario-engine.md)

**Affordance**  
A small UI control (e.g., a corner button) automatically injected by the server into a terminal's UI so the user always has an escape hatch — even when the primary scenario doesn't provide one. Main-layer scenarios that intentionally skip affordance injection must be listed in `affordance_optouts.yaml`.  
→ [`internal/scenario/register.go`](../terminal_server/internal/scenario/register.go)

**Carrier**  
One of the transport protocols the server uses to carry the control stream: gRPC, WebSocket, TCP (length-framed), or HTTP. The server advertises all available carriers over mDNS; clients pick the best one they support.  
→ [`internal/transport/`](../terminal_server/internal/transport/), [`docs/server.md`](server.md#control-transport-carriers)

**Claim**  
A lease on a resource (e.g., `display.<id>.main`, `audio_out.<id>`, `mic.<id>.capture`). Activations acquire claims before using a resource. Higher-priority claims can preempt lower-priority ones; releasing a claim can resume a suspended activation.  
→ [`internal/io/`](../terminal_server/internal/io/), [`plans/features/io-abstraction/plan.md`](../plans/features/io-abstraction/plan.md)

**Capability**  
What a connected client can do — declared by the client on connect as a protobuf snapshot, then updated via deltas. Includes platform type, screen size, sensors, IO endpoints, monitoring tier, and edge operators.  
→ [`api/terminals/capabilities/v1/`](../api/terminals/capabilities/v1/), [`docs/capability-lifecycle.md`](capability-lifecycle.md)

**Control stream**  
The bidirectional gRPC/WebSocket/TCP stream between server and client that carries all session state: `ConnectResponse` messages (SetUI, UpdateUI, capabilities, etc.) and `ConnectRequest` messages (input events, heartbeats, capability updates).  
→ [`api/terminals/control/v1/control.proto`](../api/terminals/control/v1/control.proto)

**Edge runtime / TAR (Terminals Application Runtime)**  
An on-device execution environment that runs TAL apps. The runtime loads a `.tap` archive, validates its manifest, and runs the app's exported activation against a typed capability surface. Distinct from the server-side Go scenario engine.  
→ [`docs/edge-execution-runtime.md`](edge-execution-runtime.md), [`docs/tal-example-kitchen-timer.md`](tal-example-kitchen-timer.md)

**FlowPlan**  
A graph of data sources, sinks, analyzers, mixers, and buffers describing how sensing data should flow through the observation pipeline. The server compiles a FlowPlan for a scenario activation and sends concrete transport messages to the router.  
→ [`docs/observation-plane.md`](observation-plane.md), [`docs/sensing-use-case-flows.md`](sensing-use-case-flows.md)

**Intent / Event**  
A normalized trigger record on the server's intent/event bus. Voice utterances, UI button presses, schedule firings, sensor events, automation commands, and webhook calls all produce `IntentRecord` or `EventRecord` values. The scenario engine's trigger matcher handles all sources uniformly.  
→ [`plans/features/scenario-engine.md`](../plans/features/scenario-engine.md)

**MediaPlan**  
A small topology graph of media sources, sinks, mixers, forks, recorders, and analyzers. A scenario passes a MediaPlan to the IO router, which compiles it to concrete transport stream messages. Replaces ad-hoc `Connect`/`Disconnect` calls.  
→ [`plans/features/io-abstraction/plan.md`](../plans/features/io-abstraction/plan.md)

**mDNS**  
Multicast DNS — used by the server to advertise itself on the local network so Flutter clients can discover it automatically without manual IP entry. The server publishes a TXT record with all carrier endpoints and their priority order.  
→ [`internal/discovery/`](../terminal_server/internal/discovery/), [`docs/discovery-and-connection.md`](discovery-and-connection.md)

**Placement engine**  
The server component that resolves semantic target scopes (e.g., "kitchen", "nearest screen", "all cameras", "background_monitor") into concrete device IDs at activation time. Scenarios never address raw device IDs directly.  
→ [`internal/placement/`](../terminal_server/internal/placement/), [`plans/features/placement.md`](../plans/features/placement.md)

**REPL**  
The server's Read-Eval-Print Loop — a typed command registry exposed to terminal sessions, the `term` CLI, and (via `mcpadapter`) to LLM agents over MCP. Commands are organized by capability; the REPL enforces classification (read-only vs mutating) and optional out-of-band user approval for mutating calls.  
→ [`internal/repl/`](../terminal_server/internal/repl/), [`plans/features/repl-and-shell/plan.md`](../plans/features/repl-and-shell/plan.md)

**Scenario**  
A server-side behavior module. A *scenario definition* registers triggers and a factory for activation instances. An *activation* is a live instance that holds claims and drives a terminal's UI. Examples: `PhotoFrameScenario`, `IntercomScenario`, `TimerReminderScenario`.  
→ [`internal/scenario/`](../terminal_server/internal/scenario/)

**Server-driven UI**  
The UI model where the server composes a tree of fixed primitive widgets (`SetUI`) and sends patches (`UpdateUI`) to the client. The client renders whatever the server describes; it never generates UI from scenario knowledge.  
→ [`api/terminals/ui/v1/ui.proto`](../api/terminals/ui/v1/ui.proto), [`plans/features/server-driven-ui.md`](../plans/features/server-driven-ui.md)

**TAL (Terminals Application Language)**  
A declarative language for writing Terminals apps that compile to `.tap` archives and run inside TAR on a device. TAL apps return `ScenarioResult` operation bundles rather than executing side effects directly.  
→ [`docs/tal-example-kitchen-timer.md`](tal-example-kitchen-timer.md), [`docs/edge-execution-runtime.md`](edge-execution-runtime.md)

**`.tap` archive**  
A packaged TAL application — a zip archive containing a manifest, compiled bytecode, and assets. Validated and loaded by the TAR runtime on a device.  
→ [`internal/apppackage/`](../terminal_server/internal/apppackage/)

**World model**  
An in-memory store of calibrated device geometry (camera intrinsics/extrinsics, mic array, radio bias) and fused entity state (presence, verification history). Used by sensing scenarios to answer localization and occupancy queries without rescanning history from scratch.  
→ [`internal/world/`](../terminal_server/internal/world/)

**Zone / Role**  
Semantic placement metadata assigned to a device by the server operator. A *zone* is a location label (e.g., `kitchen`, `living_room`). A *role* is a functional label (e.g., `background_monitor`, `display`). The placement engine uses these to resolve abstract scenario targets to concrete device IDs.  
→ [`plans/features/placement.md`](../plans/features/placement.md)
