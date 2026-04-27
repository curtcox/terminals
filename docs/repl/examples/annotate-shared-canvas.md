# Example: annotate-shared-canvas

Create a canvas artifact, collaborate in a session, and inspect version history after annotation.

```text
artifact create canvas kitchen-layout
session create collab art_1
session join sess_1 person:mom
session join sess_1 person:dad
session attach sess_1 device:kitchen-display
artifact patch art_1 kitchen-layout-v2-with-notes
artifact history art_1
artifact show art_1
```

Expected outcome:

- the shared canvas lives as a durable artifact,
- a collaborative session coordinates active participants/devices,
- artifact history shows the annotation update as a new revision.
