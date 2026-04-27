# Example: resume-multiplayer-session

Resume a paused multiplayer scenario by restoring participants, state, and device attachments.

```text
session ls
session show sess_1
memory stream game:maze-run
artifact show art_2
session join sess_1 person:alex
session join sess_1 person:jamie
session attach sess_1 device:living-room-screen
session members sess_1
```

Expected outcome:

- prior game/session context is visible before resuming,
- players rejoin through typed participant refs,
- attachment rebinds the shared session surface to the active display.
