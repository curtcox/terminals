---
title: "Compatibility"
kind: reference
status: active
owner: curtcox
last-reviewed: 2026-05-03
---

# Compatibility

This document tracks protocol compatibility windows and migration notes.

Current compatibility policy:

- Protocol migrations are additive first.
- Producers emit both typed and legacy fields during a migration window.
- Consumers prefer typed fields and fall back to legacy fields while old clients or servers are supported.
- Deprecated fields remain decodable until their documented removal criteria are met.
- Flexible fields are governed by [protocol-evolution.md](protocol-evolution.md) and [protocol-extension-registry.md](protocol-extension-registry.md).

## Open Windows

No typed protocol replacement window is currently open.

## Pending Migrations

The protocol extension registry identifies transitional escape hatches that should be reviewed after 2026-06-15, including server metadata, stream kind, WebRTC signal type, input actions, flow state, flow args, observation attributes, and canvas draw operations.
