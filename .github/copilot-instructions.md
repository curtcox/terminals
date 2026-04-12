# Copilot Instructions

## Architecture constraints

1. Keep client generic. Do not add scenario-specific behavior in Flutter.
2. Define communication in protobuf under `api/proto`.
3. Keep server orchestration and scenarios in Go under `terminal_server/internal`.
4. Prefer small, testable units and add tests with behavior changes.

