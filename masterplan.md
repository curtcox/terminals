# Terminals — Master Plan

A client/server system where devices on the same network serve as terminals for a single unified system. The server orchestrates all behavior; clients are generic IO surfaces that never need updating as new capabilities are added.

This file is an **index**. Detailed designs and phase plans live in the [`plans/`](plans/) directory so they can be read and executed in relative isolation.

## Vision

Every screen, speaker, microphone, and sensor in the home becomes part of a single system. A Chromebook on the kitchen counter is an intercom. A tablet on the wall is a smart photo frame — until someone says "red alert" and every screen in the house lights up. A phone on the nightstand listens for the dishwasher to stop. The old laptop in the kid's room watches the clock and says "you're going to be late."

None of this requires updating the client app. The Flutter client is a generic terminal — it reports its capabilities and does what the server tells it. All intelligence, all scenarios, all behavior lives on the server. Adding a new scenario means writing server-side code only.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                     Mac mini (Server)                    │
│                                                         │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐  │
│  │  Scenario   │  │  IO Router   │  │  AI Backend   │  │
│  │  Engine     │  │              │  │  (pluggable)  │  │
│  └──────┬──────┘  └──────┬───────┘  └───────┬───────┘  │
│         │                │                  │          │
│  ┌──────┴────────────────┴──────────────────┴───────┐  │
│  │              Device Manager                      │  │
│  │    (registry, capabilities, state, routing)      │  │
│  └──────────────────────┬───────────────────────────┘  │
│                         │                              │
│  ┌──────────────────────┴───────────────────────────┐  │
│  │              Transport Layer                     │  │
│  │        gRPC (control) + WebRTC (media)           │  │
│  └──────────────────────┬───────────────────────────┘  │
│                         │                              │
│  ┌──────────────────────┴───────────────────────────┐  │
│  │              Telephony Bridge                    │  │
│  │          SIP/VoIP (external calls)               │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────┬───────────────────────────────┘
                          │ LAN (mDNS discovery)
          ┌───────────────┼───────────────┐
          │               │               │
     ┌────┴─────┐    ┌────┴─────┐    ┌────┴─────┐
     │  Phone   │    │  Tablet  │    │  Laptop  │
     │ (Flutter)│    │ (Flutter)│    │ (Flutter)│
     └──────────┘    └──────────┘    └──────────┘
```

## Core Rules

1. Never add scenario-specific behavior to the client.
2. Define all client/server messages in protobuf, not ad-hoc JSON.
3. Keep AI providers behind interfaces in server code.
4. Build UIs from shared server-driven primitives.

## Plan Index

### System design

- [plans/architecture-client.md](plans/architecture-client.md) — Flutter client: capability declaration, module layout, platform support.
- [plans/architecture-server.md](plans/architecture-server.md) — Go server: module layout and responsibilities.
- [plans/protocol.md](plans/protocol.md) — gRPC control plane, WebRTC media plane, data channels.
- [plans/discovery.md](plans/discovery.md) — mDNS, manual connect, connection lifecycle, trust model.
- [plans/io-abstraction.md](plans/io-abstraction.md) — IO categories and router primitives (consume, produce, forward, fork, mix, composite, record, analyze).
- [plans/server-driven-ui.md](plans/server-driven-ui.md) — Fixed primitive component set, descriptor format, update/patch/animate.
- [plans/scenario-engine.md](plans/scenario-engine.md) — Scenario interface, activation triggers, priority and preemption.
- [plans/use-case-flows.md](plans/use-case-flows.md) — End-to-end flows for each planned scenario.

### Tooling

- [plans/technology.md](plans/technology.md) — Server, client, and pluggable AI backend technology choices.
- [plans/agent-config.md](plans/agent-config.md) — CLAUDE.md / AGENTS.md layout and contents for every subproject.
- [plans/ci.md](plans/ci.md) — Go, Flutter, and proto quality gates; `Makefile`; CI workflows.

### Development phases

Each phase is a standalone checklist with explicit prerequisites. Execute in order.

- [plans/phase-0-setup.md](plans/phase-0-setup.md) — Repo setup, tooling, and CI.
- [plans/phase-1-foundation.md](plans/phase-1-foundation.md) — Proto, skeletons, server-driven UI hello-world.
- [plans/phase-2-terminal.md](plans/phase-2-terminal.md) — Text terminal (PTY + keyboard forwarding).
- [plans/phase-3-media.md](plans/phase-3-media.md) — WebRTC media plane and IO router.
- [plans/phase-4-comms.md](plans/phase-4-comms.md) — Intercom, PA, multi-window, calls, SIP bridge.
- [plans/phase-5-voice.md](plans/phase-5-voice.md) — AI backends and voice assistant pipeline.
- [plans/phase-6-monitoring.md](plans/phase-6-monitoring.md) — Sound classification, timers, schedule monitoring, red alert.
- [plans/phase-7-polish.md](plans/phase-7-polish.md) — Photo frame, preemption hardening, admin UI, misc IO.

### Adjacent docs

- [usecases.md](usecases.md) — User-story-format use cases (planned + architecture-enabled).
- [next.md](next.md) — The single task currently being worked on, per `CLAUDE.md` convention.

## Key Design Decisions

1. **Client is stateless (except connection state)**. All scenario logic, UI generation, and IO routing lives on the server. The client is a render engine + IO bridge.

2. **gRPC for control, WebRTC for media**. gRPC gives us strong typing, bidirectional streaming, and great codegen. WebRTC gives us battle-tested real-time media with built-in echo cancellation and adaptive bitrate.

3. **Server-driven UI with fixed primitives**. The client has a finite set of UI components it can render. The server composes them. This is the contract that lets the client stay unchanged while the server evolves.

4. **Pluggable AI backends**. The system doesn't couple to any specific AI provider. Interfaces allow swapping between local and cloud implementations based on the user's preference and hardware capability.

5. **Scenario engine with priority preemption**. Real-world use requires graceful handling of competing demands for device IO. Priority-based preemption with suspend/resume handles this cleanly.

6. **Trusted LAN, no auth**. For a home network, mDNS discovery + direct connection with no authentication keeps things simple. If this assumption changes, TLS mutual auth can be added at the transport layer without protocol changes.
