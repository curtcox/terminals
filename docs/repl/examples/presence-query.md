# presence-query

## Goal

Inspect who is currently reachable in a zone and target follow-up actions to the live audience.

## Script

```text
devices ls
cohort put kitchen-present --selectors zone:kitchen
cohort show kitchen-present
ui broadcast kitchen-present '{"type":"toast","text":"Dinner in 10 minutes"}'
```

## Notes

- Build operational broadcasts from named cohorts rather than hard-coded device IDs.
- Run `cohort show` before fan-out to validate selector resolution.
