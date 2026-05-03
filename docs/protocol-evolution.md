---
title: "Protocol Evolution"
kind: reference
status: active
owner: curtcox
last-reviewed: 2026-05-03
---

# Protocol Evolution

Terminals keeps the client/server contract in protobuf. Flexible fields are allowed only when they are deliberate, documented, and tested.

## Rule of Thumb

Stable machine-readable behavior belongs in typed protobuf fields, enums, messages, `oneof`s, or repeated typed records.

Acceptable exceptions are:

- constrained scalars, such as URI strings, RFC3339 timestamps, SHA-256 digests, semantic versions, and media types
- registry-backed extensions with documented keys or token namespaces
- transitional escape hatches with owners, review dates, and migration paths
- display/debug strings that code does not branch on
- external payloads governed by a separate standard or versioned format

## Classification

Use exactly one classification for every flexible protocol field.

| Classification | Meaning | Required Documentation |
|---|---|---|
| `typed_contract` | Semantics are already represented directly in protobuf. | Only needed when a static check would otherwise flag the field. |
| `constrained_scalar` | A scalar uses a constrained external format. | Format, validation, malformed-value behavior, producer, consumer. |
| `registry_backed_extension` | Unknown keys or values are intentionally part of the design. | Owner, namespace or keys, producer, consumer, unknown behavior, validation, promotion trigger. |
| `transitional_escape_hatch` | Temporary flexible field while semantics stabilize. | Owner, review date, current consumers, migration path, tests, rule blocking new durable dependencies. |
| `display_debug_string` | Human-readable text only. | Confirmation that code must not branch on content. |
| `external_payload` | Payload is defined by an external standard or separately versioned format. | External format, size limits, malformed-payload behavior. |

The registry lives in [protocol-extension-registry.md](protocol-extension-registry.md).

## Maps

Maps are registries, not junk drawers. Every protocol map must document:

- owner
- allowed keys or namespace
- producer and consumer
- value format
- unknown-key behavior
- validation behavior
- promotion trigger for keys that become durable

If a map key becomes required for normal client/server behavior, add a typed protobuf field and keep the map only as a compatibility fallback during the migration window.

## String Selectors

A string selector is any string field whose value is compared against a fixed or semi-fixed set of values.

For each selector, choose one path:

- replace it with an enum
- keep it as a registry-backed extension
- keep it as a constrained scalar if it follows an external format
- mark it as a transitional escape hatch with an active migration path

Do not add undocumented string tokens to protocol messages.

## JSON

JSON embedded in protobuf must have:

- schema version
- allowed top-level shape
- unknown-field behavior
- size limit
- malformed JSON behavior
- migration path to typed protobuf or a documented reason to keep JSON

New durable JSON shapes should be rejected during review unless protobuf cannot reasonably model the data.

## Compatibility

Protocol migrations are additive first:

1. Add the typed replacement with a new field number.
2. Emit both old and new fields from producers.
3. Read the typed field first in consumers.
4. Fall back to the legacy field while old producers are supported.
5. Add cross-language tests for typed, legacy, and both-field payloads.
6. Deprecate the old field only after fallback behavior is tested.
7. Remove old behavior only after the compatibility window closes.

Unknown values must fail predictably. Consumers must ignore, preserve, reject with a typed protocol error, or downgrade to a documented safe default. They must not silently reinterpret unknown values.

## Reviewer Checklist

- Is stable machine-readable behavior represented as typed protobuf where practical?
- Is every new map, token, selector, or JSON shape listed in the registry?
- Does the registry entry document owner, producer, consumer, unknown behavior, validation, tests, and target state?
- Is the change additive for existing clients and servers?
- Do consumers prefer typed replacements and fall back to legacy fields during migration?
- Are Go and Dart contract tests or golden fixtures updated when behavior crosses the client/server boundary?
- Did `make proto-generate`, `make proto-lint`, and `make proto-flex-check` run?

## Agent Checklist

Before changing `api/terminals/**`, agents must:

- read [api/CONTRACTS.md](../api/CONTRACTS.md)
- avoid adding ad-hoc JSON, new metadata keys, or new string tokens without a registry update
- prefer typed protobuf additions for durable behavior
- update the registry and tests in the same change as any flexible protocol behavior
- leave scenario-specific behavior out of the Flutter client
