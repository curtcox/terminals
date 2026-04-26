# Client Architecture

The Flutter client is a generic terminal runtime. It does not contain
scenario-specific behavior. Scenario orchestration, policy, and planning stay
on the Go server.

## Design Contract

The client is responsible for:

1. Discovering or manually connecting to a server.
2. Opening a control stream and identifying itself.
3. Sending a capability snapshot baseline.
4. Publishing capability deltas as runtime conditions change.
5. Executing server-issued IO/UI commands.
6. Streaming input, sensor, and media data as directed.

The server is responsible for deciding what to run and where to route it.

## Runtime Shape

Current implementation centers on:

- `terminal_client/lib/main.dart` for connection lifecycle, capability runtime,
  and server-driven UI execution.
- `terminal_client/lib/capabilities/probe.dart` for platform capability probing
  and endpoint enumeration.
- `terminal_client/lib/connection/` for carrier-specific control clients and
  reliability wrappers.
- `terminal_client/lib/media/` and `terminal_client/lib/io/` for media and
  sensor/input execution.

The architecture remains generic even where code is physically concentrated,
and is covered by widget/integration-style tests in `terminal_client/test/`.

## Capability Lifecycle

The client treats capabilities as live state.

- On initial registration it emits `CapabilitySnapshot` with generation
  tracking.
- On runtime changes it emits `CapabilityDelta` with reason strings and fresh
  generations.
- Display geometry changes (size, orientation, DPR, safe area) are coalesced
  with a debounce before delta emission.
- Privacy toggles withdraw and restore sensitive capabilities (microphone,
  camera) via deltas.
- Capability updates are generation-ordered and synchronized with server
  acknowledgements.

## Display and Audio Modeling

- Display metadata is attached at runtime and includes width/height, density,
  orientation, and safe-area insets.
- Audio input and output are modeled separately, including endpoint lists.
- Camera endpoints are modeled independently from audio endpoints.

This allows the server to compile endpoint-scoped claims and media plans based
on currently available resources instead of static assumptions.

## Validation Evidence

Client widget tests cover snapshot/delta emission and runtime geometry/privacy
change behavior (see `terminal_client/test/widget_test.dart`).

Repository-wide quality and compatibility gates are run via `make all-check`.

## Related References

- `docs/discovery-and-connection.md`
- `docs/server.md`
- `docs/client-web.md`
- `docs/client-macos.md`
- `plans/features/protocol.md`
- `plans/features/io-abstraction.md`
