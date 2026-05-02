---
title: "System and Infrastructure Use Cases"
family: I
ids: [I1, I2, I3, I4, I5, I6, I7, I8, I9, I10, I11]
---

# System and Infrastructure

| # | As a ... | I would like to ... | So that I can ... |
|---|----------|---------------------|-------------------|
| I1 | **Device (client program)** | discover the server automatically via mDNS on the local network | connect without manual configuration on a trusted LAN |
| I2 | **Device (client program)** | fall back to manual server address entry if mDNS fails | still connect when network configuration blocks discovery |
| I3 | **Device (client program)** | report my full capability manifest (screen, mic, camera, sensors, etc.) on connection | let the server know exactly what I can do and never receive commands I can't handle |
| I4 | **Server (scenario engine)** | query the device registry for devices matching a required capability set | route scenarios to appropriate devices automatically |
| I5 | **Server (IO router)** | consume, produce, forward, fork, mix, composite, record, and analyze any IO stream dynamically | orchestrate arbitrarily complex media flows without client changes |
| I6 | **Server (scenario engine)** | preempt lower-priority scenarios and suspend them for later resumption | handle competing demands for device IO gracefully |
| I7 | **Server (AI backend)** | swap between local and cloud AI implementations via configuration | adapt to the user's hardware, cost, and privacy preferences |
| I8 | **CI pipeline (program)** | run `make all-check` to validate the entire codebase (lint, test, proto) | catch regressions before code is merged |
| I9 | **Development agent (Claude Code / Codex)** | read CLAUDE.md and AGENTS.md to understand the project structure and rules | contribute new scenarios and features with full context |
| I10 | **Development agent** | add a new server-side scenario without touching client code | extend system behavior while respecting the thin-client architecture |
| I11 | **Device (client program)** | automatically reconnect and have my previous state restored after a disconnect | resume where I left off without manual intervention |
