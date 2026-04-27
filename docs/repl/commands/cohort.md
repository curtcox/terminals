# cohort commands

Named cohorts let operators define reusable device selectors and resolve them
to live members at runtime.

## Selectors

`cohort put` accepts a comma-separated selector list. Selectors are combined
with logical AND.

Supported selector keys:

- `id:<device-id>` or `device:<device-id>`
- `zone:<zone>`
- `role:<role>`
- `platform:<platform>`
- `type:<device-type>`
- `state:<connected|disconnected>`
- `mobility:<value>`
- `affinity:<value>`

Example:

```text
cohort put family-screens --selectors zone:kitchen,role:screen,state:connected
```

## Commands

```text
cohort ls
cohort show <name>
cohort put <name> --selectors <selector[,selector...]>
cohort del <name>
```

`cohort show` returns both the stored selector definition and the currently
resolved member list from registered devices.
