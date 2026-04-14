# Scenario Engine

See [masterplan.md](../masterplan.md) for overall system context.

Scenarios are server-side modules that implement specific behaviors. They are the only place where "what the system does" is defined.

## Scenario Interface

```go
type Scenario interface {
    // Name returns the scenario identifier.
    Name() string

    // Match reports whether this scenario should activate
    // given the current trigger (voice command, schedule, event, etc.).
    Match(trigger Trigger) bool

    // Start activates the scenario on the given set of devices.
    Start(ctx context.Context, env *Environment) error

    // Stop deactivates the scenario, releasing all resources.
    Stop() error
}

type Environment struct {
    Devices    DeviceManager    // Query and command devices
    IO         IORouter         // Route IO streams
    AI         AIBackend        // Speech, vision, LLM, etc.
    Telephony  TelephonyBridge  // External calls
    Storage    StorageManager   // Persistence
    Scheduler  Scheduler        // Timers and reminders
    Broadcast  Broadcaster      // Send to all/subset of devices
}
```

## Scenario Activation

Scenarios activate via triggers:

- **Voice**: User says a wake word + command → STT → scenario matching
- **Schedule**: Cron-like time triggers (e.g., check school schedule at 7:30 AM)
- **Event**: An IO analysis result (e.g., sound classifier detects silence after running water)
- **Manual**: User selects a scenario via the UI
- **Cascade**: One scenario triggers another (e.g., "red alert" stops all other scenarios)

## Scenario Priority and Preemption

Scenarios have priority levels. Higher-priority scenarios can preempt lower-priority ones on a device:

| Priority | Examples                        |
|----------|---------------------------------|
| Critical | Red alert, emergency            |
| High     | Active phone call, intercom, PA |
| Normal   | Terminal session, voice query, multi-window |
| Low      | Photo frame, ambient monitoring |
| Idle     | Clock display, standby screen   |

When a higher-priority scenario needs a device, the lower-priority scenario is suspended (not terminated). When the higher-priority scenario ends, the suspended one resumes.

## Engine Responsibilities

- **Lifecycle**: Start, stop, suspend, resume scenarios across devices.
- **Matching**: Route incoming triggers to the scenarios that declare a match.
- **Conflict resolution**: Enforce priority rules when two scenarios want the same device.
- **State restoration**: Remember what a suspended scenario needed so resume is seamless.
- **Isolation**: A scenario crash must not bring down the engine — scenarios run in a supervised manner.

## Related Plans

- [architecture-server.md](architecture-server.md) — Scenario module layout (`internal/scenario/`).
- [io-abstraction.md](io-abstraction.md) — How scenarios manipulate streams.
- [server-driven-ui.md](server-driven-ui.md) — How scenarios render their UIs.
- [use-case-flows.md](use-case-flows.md) — Concrete scenarios and their flows.
