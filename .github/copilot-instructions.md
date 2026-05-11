# Copilot Instructions

## Engineering priorities (apply before adding features)

1. Prioritize clearer specs, better design, and better code over more features.
2. Fix existing bugs in the area you are touching before adding new behavior.
3. Add missing tests and enable missing static-analysis / lint checks before extending a unit.
4. Encode every new invariant or fixed bug class as a CI check (test, lint rule, `make` target in a workflow).
5. Favor simplicity over backward compatibility. Change shapes, rename, and update all callers rather than adding shims or deprecated aliases.

## Architecture constraints

1. Keep client generic. Do not add scenario-specific behavior in Flutter.
2. Define communication in protobuf under `api/terminals/`.
3. Keep server orchestration and scenarios in Go under `terminal_server/internal`.
4. Prefer small, testable units and add tests with behavior changes.

