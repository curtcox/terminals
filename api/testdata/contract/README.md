# Protocol Contract Golden Fixtures

This directory contains shared Go-Dart protobuf contract fixtures.

- `manifest.yaml` lists every fixture, expected message type, oneof payload, direction, and assertion file.
- `fixtures/*.textproto` is the human-edited source of truth.
- `fixtures/*.binpb` is the binary protobuf artifact decoded by both Go and Dart tests.
- `expected/*.yaml` contains small semantic assertions shared by both runtimes.

The corpus covers handshake, capabilities, server-driven UI, input events,
transport negotiation, command request compatibility, flow starts, edge
observations with artifacts, typed protocol errors, UI notifications, and
diagnostics bug reports. Deprecated fixtures are parse-only when marked that
way in `manifest.yaml`.

Tests read `.binpb` files only. Regenerate binary fixtures after editing textproto:

```bash
make proto-contract-generate
make proto-contract-test
```

By default, Go and Dart tests resolve this directory relative to their normal working directories. Set `TERMINALS_CONTRACT_FIXTURE_ROOT` to an absolute path for CI/debugging overrides.
