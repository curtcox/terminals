# YAML Scenario Schema

YAML scenarios live under `testdata/` and are executed by the use-case validation
harness (`RunScenarioFile`). They complement Go-authored tests; keep Go tests for
low-level coverage and YAML for readable multi-step stories.

## Top-level fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | yes | Stable scenario identifier (e.g. `t2-timer-reminder`) |
| `usecases` | yes | One or more use-case IDs this scenario validates |
| `clock.start` | no | RFC3339 UTC time; pins the harness fake clock before `StartServer` |
| `terminals` | yes | Map of terminal alias → profile |
| `steps` | yes | Ordered list of step objects (see below) |

## Terminal profile

```yaml
terminals:
  kitchen:
    device_id: kitchen   # optional; defaults to map key
    name: Kitchen        # optional; defaults to device_id
```

## Step types

Each step is a single-key object. Supported keys:

### `connect`

Connect every terminal in `terminals` and wait for session establishment.

```yaml
- connect: {}
```

### `command`

Send a manual intent command (`COMMAND_KIND_MANUAL`).

```yaml
- command:
    terminal: kitchen
    intent: set timer
    arguments:
      duration_seconds: "300"
      label: pasta
```

When `duration_seconds` is set and `fire_unix_ms` is omitted, the executor sets
`fire_unix_ms` from the current synthetic clock plus that duration.

### `says`

Voice input from a terminal. Maps to `COMMAND_KIND_VOICE` with `text` set to
the spoken phrase (same path as harness Go tests; no raw `VoiceAudio` STT in CI).

```yaml
- says:
    terminal: kitchen
    text: "announce: dinner is ready"
```

### `clock_advance`

Advance synthetic time by a Go duration string (`5m`, `1h30m`, etc.).

```yaml
- clock_advance:
    duration: 5m1s
```

### `clock_advance_to`

Advance synthetic time to an absolute RFC3339 instant (no-op if already past).

```yaml
- clock_advance_to:
    time: "2026-05-16T09:05:01Z"
```

### `process_due_timers`

Run `Harness.ProcessDueTimers` at the current synthetic time.

```yaml
- process_due_timers: {}
```

Optional fields:

- `expect_processed`: exact count of timers processed (integer)
- `assert_id`: assertion id when count mismatches (default `process-due-timers`)

### `expect`

Record harness assertions. All checks are optional; at least one should be set.

```yaml
- expect:
    id: T2-done-notification
    description: broadcast emits Timer done after timer fires
    broadcast_contains: "Timer done!"
    terminal: kitchen
    scenario_start: timer_reminder
    route_kind: announcement_audio
    timers_processed: 1
```

### `disconnect`

Disconnect terminal(s). Omit `terminal` to disconnect all connected terminals.

```yaml
- disconnect:
    terminal: kitchen
```

## Example (T2 timer reminder)

See `t2-timer-reminder.yaml`.
