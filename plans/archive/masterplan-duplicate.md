---
title: "Terminals — Master Plan"
kind: plan
status: superseded
owner: unowned
validation: none
last-reviewed: 2026-04-25
---

# Terminals — Master Plan (superseded duplicate)

> **Superseded.** The canonical master plan is [`/masterplan.md`](../../masterplan.md). This file is a stale fork preserved here for history; its internal links were broken before the reorg and have not been updated.

A client/server system where devices on the same network serve as terminals for a single unified system. The server orchestrates all behavior; clients are generic IO surfaces that never need updating as new capabilities are added.

This file is an **index**. Detailed designs and phase plans live in the [`plans/`](plans/) directory so they can be read and executed in relative isolation.

## Vision

Every screen, speaker, microphone, and sensor in the home becomes part of a single system. A Chromebook on the kitchen counter is an intercom. A tablet on the wall is a smart photo frame — until someone says "red alert" and every screen in the house lights up. A phone on the nightstand listens for the dishwasher to stop. The old laptop in the kid's room watches the clock and says "you're going to be late."

None of this requires updating the client app.

The Flutter client is a generic terminal — it reports its capabilities and current capability state and does what the server tells it. All intelligence, all scenarios, all behavior lives on the server. Adding a new scenario means writing server-side code only.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│ Mac mini (Server)                                      │
│                                                         │
│  ┌─────────────┐   ┌──────────────┐   ┌───────────────┐ │
│  │ Scenario    │   │ IO Router    │   │ AI Backend    │ │
│  │ Engine      │   │              │   │ (pluggable)   │ │
│  └──────┬──────┘   └──────┬───────┘   └───────┬───────┘ │
│         │                 │                   │         │
│  ┌──────┴─────────────────┴───────────────────┴───────┐ │
│  │ Device Manager                                      │ │
│  │ (registry, capabilities, state, routing)            │ │
│  └──────────────────────┬──────────────────────────────┘ │
│                         │                                │
│  ┌──────────────────────┴──────────────────────────────┐ │
│  │ Transport Layer                                      │ │
│  │ gRPC (control) + WebRTC (media)                      │ │
│  └──────────────────────┬──────────────────────────────┘ │
│                         │                                │
│  ┌──────────────────────┴──────────────────────────────┐ │
│  │ Telephony Bridge                                     │ │
│  │ SIP/VoIP (external calls)                            │ │
│  └──────────────────────────────────────────────────────┘ │
└─────────────────────────┬───────────────────────────────┘
                          │
                  LAN (mDNS discovery)
        ┌─────────────────┼─────────────────┐
        │                 │                 │
   ┌────┴─────┐      ┌────┴─────┐      ┌────┴─────┐
   │ Phone    │      │ Tablet   │      │ Laptop   │
   │ (Flutter)│      │ (Flutter)│      │ (Flutter)│
   └──────────┘      └──────────┘      └──────────┘
```

## Core Rules

1. Never add scenario-specific behavior to the client.
2. Define all client/server messages in protobuf, not ad-hoc JSON.
3. Keep AI providers behind interfaces in server code.
4. Build UIs from shared server-driven primitives.
5. Treat terminal capabilities as live state, not static registration metadata.

## Plan Index

### System design

- [plans/architecture-client.md](plans/architecture-client.md) — Flutter client: capability discovery, runtime capability monitoring, module layout, platform support.
- [plans/architecture-server.md](plans/architecture-server.md) — Go server: module layout and responsibilities.
- [plans/protocol.md](plans/protocol.md) — gRPC control plane, capability lifecycle, WebRTC media plane, data channels.
- [plans/discovery.md](plans/discovery.md) — mDNS, manual connect, connection lifecycle, trust model.
- [plans/io-abstraction.md](plans/io-abstraction.md) — IO categories, resources and claims, capability compilation, media-plan topology graph compiled by the router.
- [plans/placement.md](plans/placement.md) — Zones, roles, and the placement engine that turns semantic targets into concrete devices.
- [plans/server-driven-ui.md](plans/server-driven-ui.md) — Fixed primitive component set, descriptor format, update/patch/animate.
- [plans/scenario-engine.md](plans/scenario-engine.md) — Scenario definitions vs activations, intent/event triggers, claim-driven preemption, scenario recipes.
- [plans/use-case-flows.md](plans/use-case-flows.md) — End-to-end flows for each planned scenario.
- [plans/application-runtime.md](plans/application-runtime.md) — Runtime model for app/session lifecycles and server orchestration.
- [plans/edge-execution.md](plans/edge-execution.md) — Edge execution model and on-device/off-device execution boundaries.
- [plans/observation-plane.md](plans/observation-plane.md) — Telemetry, sensing signals, and observation pipeline.
- [plans/world-model-calibration.md](plans/world-model-calibration.md) — World-model calibration strategy and feedback loops.
- [plans/sensing-use-case-flows.md](plans/sensing-use-case-flows.md) — End-to-end sensing-centric use-case flows.
- [plans/bug-reporting.md](plans/bug-reporting.md) — Modality-agnostic, cross-device bug reporting with full client/subject context capture.
- [plans/capability-lifecycle.md](plans/capability-lifecycle.md) — Capability model, handshake, runtime updates, and server reactions.

### Tooling

- [plans/technology.md](plans/technology.md) — Server, client, and pluggable AI backend technology choices.
- [plans/agent-config.md](plans/agent-config.md) — CLAUDE.md / AGENTS.md layout and contents for every subproject.
- [plans/ci.md](plans/ci.md) — Go, Flutter, and proto quality gates; `Makefile`; CI workflows.

### Development phases

Each phase is a standalone checklist with explicit prerequisites. Execute in order.

- [plans/phase-0-setup.md](plans/phase-0-setup.md) — Repo setup, tooling, and CI.
- [plans/phase-1-foundation.md](plans/phase-1-foundation.md) — Proto, skeletons, server-driven UI hello-world, capability lifecycle foundations.
- [plans/phase-2-terminal.md](plans/phase-2-terminal.md) — Text terminal (PTY + keyboard forwarding).
- [plans/phase-3-media.md](plans/phase-3-media.md) — WebRTC media plane and IO router.
- [plans/phase-4-comms.md](plans/phase-4-comms.md) — Intercom, PA, multi-window, calls, SIP bridge.
- [plans/phase-5-voice.md](plans/phase-5-voice.md) — AI backends and voice assistant pipeline.
- [plans/phase-6-monitoring.md](plans/phase-6-monitoring.md) — Sound classification, timers, schedule monitoring, red alert.
- [plans/phase-6b-edge-sensing.md](plans/phase-6b-edge-sensing.md) — Edge sensing expansion phase.
- [plans/phase-7-polish.md](plans/phase-7-polish.md) — Photo frame, preemption hardening, admin UI, misc IO.
- [plans/phase-capability-lifecycle.md](plans/phase-capability-lifecycle.md) — Cross-cutting checklist for capability disclosure, runtime updates, and claim invalidation.

### Adjacent docs

- [usecases.md](usecases.md) — User-story-format use cases (planned + architecture-enabled).
- [next.md](next.md) — The single task currently being worked on, per `CLAUDE.md` convention.

## Key Design Decisions

1. **Client is stateless (except connection state)**. All scenario logic, UI generation, and IO routing lives on the server. The client is a render engine + IO bridge.
2. **gRPC for control, WebRTC for media**. gRPC gives us strong typing, bidirectional streaming, and great codegen. WebRTC gives us battle-tested real-time media with built-in echo cancellation and adaptive bitrate.
3. **Server-driven UI with fixed primitives**. The client has a finite set of UI components it can render. The server composes them. This is the contract that lets the client stay unchanged while the server evolves.
4. **Pluggable AI backends**. The system doesn't couple to any specific AI provider. Interfaces allow swapping between local and cloud implementations based on the user's preference and hardware capability.
5. **Activations are the unit of execution**. A scenario definition is a singleton; a scenario *activation* is a live instance with its own ID, claims, targets, and resume snapshot. Multiple timers, terminal sessions, or calls coexist cleanly. See [plans/scenario-engine.md](plans/scenario-engine.md).
6. **Resource-level preemption via claims**. Activations claim specific resources (main screen, overlay, speaker, mic, camera, PTY) rather than whole devices. Higher-priority claims suspend lower ones; releases resume the suspended activation with exactly the claims it had. PA can take speakers without hiding the photo frame; a voice reply can overlay without replacing the terminal. See [plans/io-abstraction.md](plans/io-abstraction.md#resource-claims).
7. **Semantic placement**. A placement engine turns "kitchen", "nearest screen", "all cameras" into concrete device sets. Scenarios never target raw device IDs; zones and roles are server-assigned metadata. See [plans/placement.md](plans/placement.md).
8. **Typed intents and events**. Voice, UI actions, schedules, webhooks, classifier events, and automation agents all produce the same `Intent`/`Event` records on one bus. One matcher handles every trigger source. See [plans/scenario-engine.md](plans/scenario-engine.md#triggers-intents-and-events).
9. **Declarative media topology**. Scenarios hand the IO router a `MediaPlan` — a small graph of sources, sinks, mixers, forks, analyzers, recorders — and the router compiles it to concrete transport messages. No stream-kind magic strings. See [plans/io-abstraction.md](plans/io-abstraction.md#media-topology-plans-not-connects).
10. **Capabilities are first-class runtime state**. Clients send a full capability snapshot on connect and send explicit deltas whenever capabilities change. The server treats that stream as the source of truth for routing, placement, claims, and UI composition. See [plans/capability-lifecycle.md](plans/capability-lifecycle.md).
11. **Trusted LAN, no auth**. For a home network, mDNS discovery + direct connection with no authentication keeps things simple. If this assumption changes, TLS mutual auth can be added at the transport layer without protocol changes.
