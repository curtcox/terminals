---
title: "API Contract Checklist"
kind: reference
status: active
owner: curtcox
last-reviewed: 2026-05-03
---

# API Contract Checklist

Protobuf is the canonical Terminals client/server contract. Flexible fields are governed by [../docs/protocol-evolution.md](../docs/protocol-evolution.md) and inventoried in [../docs/protocol-extension-registry.md](../docs/protocol-extension-registry.md).

Use this checklist for every change under `api/terminals/**`.

## Schema

- Did this add or change a flexible field?
- Did this add a metadata key, string token, selector value, map key, or JSON shape?
- Can the behavior be represented as typed protobuf instead?
- If the field is flexible, is the registry updated?
- Is the classification correct: `typed_contract`, `constrained_scalar`, `registry_backed_extension`, `transitional_escape_hatch`, `display_debug_string`, or `external_payload`?
- Are unknown values handled predictably?

## Compatibility

- Is the change additive for old clients and servers?
- Are old fields still decoded during the compatibility window?
- Do producers emit both typed and legacy values during migration?
- Do consumers prefer typed values and fall back to legacy values?
- Does `docs/compatibility.md` need an update?

## Validation

- Did `make proto-generate` pass if `.proto` files changed?
- Did `make proto-lint` pass?
- Did `make proto-flex-check` pass?
- Are Go and Dart behavior tests included when client/server semantics are affected?
- Are golden fixtures added or updated when wire compatibility matters?

## Agent Rules

- Do not add scenario-specific behavior to the Flutter client.
- Do not add durable client/server behavior through metadata maps, free-form string tokens, or JSON without a registry entry.
- Prefer enums, messages, `oneof`s, and typed repeated records for stable semantics.
- Keep AI providers behind server-side interfaces.
- Build client UIs from shared server-driven primitives.
