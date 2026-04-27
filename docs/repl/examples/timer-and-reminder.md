# timer-and-reminder

## Goal

Prototype a lightweight reminder flow by combining keyed state, scheduler intent emission, and UI fan-out from REPL commands.

## Script

```text
store put reminders laundry '{"message":"Move clothes to dryer"}' --ttl 2h
bus emit intent schedule.reminder '{"key":"laundry","after":"30m"}'
handlers on scenario=timer submit --emit event reminder.fire '{"key":"laundry"}'
bus tail --kind event --name reminder.fire --limit 5
```

## Notes

- Persist reminder payloads in `store` so retries and restarts can rehydrate context.
- Use `bus tail` to verify reminder emission before wiring any device-specific UI.
