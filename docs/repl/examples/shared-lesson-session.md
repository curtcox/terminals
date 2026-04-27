# Example: shared-lesson-session

Launch a collaborative lesson using a shared artifact and a generalized session.

```text
artifact create lesson fractions-basics
artifact patch art_1 fractions-level-1
session create lesson art_1
session join sess_1 person:teacher
session join sess_1 person:student
session attach sess_1 device:student-tablet
session members sess_1
artifact show art_1
```

Expected outcome:

- the lesson artifact is durable and queryable,
- both teacher and learner are present in a typed session,
- device attachment ties the shared lesson view to the learner terminal.
