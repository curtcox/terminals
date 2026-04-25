# Kitchen Timer App

This package is the TAL/TAR contract example for use case T1. `main.tal`
mirrors the planned lifecycle contract: activations return state plus host
operations for UI, scheduler, TTS, and bus effects.

The executable implementation today is the Go-side `TimerReminderScenario` in
`terminal_server/internal/scenario`. It renders the countdown through
server-driven UI operations, schedules expiry and 1 Hz ticks, patches remaining
time, supports cancellation, speaks and displays completion, emits
`timer.expired`, and removes due records. The app runtime can load this package
and validate its manifest, but it does not yet interpret the TAL lifecycle hooks
directly.

Supported smoke test:

```bash
cd terminal_server
go run ./cmd/term app test kitchen_timer
```

That command proves the package and declared TAL tests load. It is not yet a
synthetic lifecycle simulation for timer expiry.
