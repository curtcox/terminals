# Development Environment Improvement Log

This log captures friction noticed while implementing the kitchen timer.

## 2026-04-24 Kitchen Timer

- `docs/tal-example-kitchen-timer.md` describes a TAL package contract, but the current runtime only loads manifests and stubs activations. It would be easier to implement examples if docs clearly marked which parts are aspirational versus executable today.
- `make usecase-validate USECASE=T1` already existed even though the kitchen timer example called for richer behavior. A small "coverage depth" note in the validation matrix could distinguish smoke coverage from full example parity.
- Timer scheduler entries are opaque string keys, so adding label/duration metadata required key encoding and backward-compatible parsing. A structured scheduler payload would make app development less brittle.
- Scenario starts can broadcast notifications, but they cannot directly return typed UI/TTS/bus operations. That makes TAL-style all-or-nothing operation commits hard to model in the current Go scenario layer.
