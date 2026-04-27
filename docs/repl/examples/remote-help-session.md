# Example: remote-help-session

Create a live help session, add participants, and grant temporary control.

```text
session create support kitchen-tablet
session ls
session join sess_1 person:dad
session join sess_1 person:tech
session attach sess_1 device:kitchen-tablet
session control request sess_1 person:tech remote
session control grant sess_1 person:tech person:dad remote
session members sess_1
session control revoke sess_1 person:tech person:dad
session detach sess_1 device:kitchen-tablet
```

Expected outcome:

- both participants appear in the session roster,
- control grant/revoke events are visible in session state,
- device attachment is explicit and reversible through typed commands.
