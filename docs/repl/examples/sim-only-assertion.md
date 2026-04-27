# sim-only-assertion

## Goal

Validate authored UI behavior against a virtual device without touching
hardware.

## Script

```text
sim device new sim-kitchen --caps display,keyboard
ui push sim-kitchen '{"type":"stack","children":[{"type":"text","text":"hello"}]}' --root sim-root
sim input sim-kitchen banner submit ack
sim ui sim-kitchen
sim expect sim-kitchen ui hello --within 2s
sim record sim-kitchen --duration 5s
sim device rm sim-kitchen
```

## Notes

- Use `sim ui` to inspect both the latest snapshot and synthetic input history.
- For preflight checks in CI, place commands into a script and run
  `scripts dry-run <path>` before execution, then `scripts run <path>` for the
  operational run.
