# terminal_client Agent Notes

## Language and Style

- Keep Flutter code platform-neutral where possible.
- Treat client as generic IO/render surface.
- Avoid scenario-specific logic.

## Commands

```bash
flutter pub get
flutter analyze
flutter test
dart format --set-exit-if-changed .
```

## Guardrails

- Do not introduce scenario behavior in client widgets.
- Keep all new client/server data contracts in protobuf.
- Prefer thin adapters around transport and capability detection.

