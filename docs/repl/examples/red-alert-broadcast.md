# red-alert-broadcast

## Goal

Send an urgent UI banner to all display-capable terminals and track per-device acknowledgement state without adding scenario-specific server code.

## Script

```text
cohort put all-screens --selectors capability:display
ui broadcast all-screens '{"type":"banner","id":"alert-banner","tone":"critical","title":"Red Alert","text":"Shelter now"}'
handlers on scenario=red_alert submit --run 'store put alert_ack $device acknowledged --ttl 1h'
store watch alert_ack
```

## Notes

- Keep fan-out targeting in `cohort` so operator scope is auditable and reusable.
- Use `store watch` during live drills to confirm acknowledgements as they arrive.
