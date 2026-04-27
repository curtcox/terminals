# Example: pin-family-bulletin

Create and pin a household bulletin, then record acknowledgement for one actor.

```text
board ls family
board pin family "Trash pickup is tomorrow at 7am"
board ls family
identity ack record bulletin:family-trash --actor person:mom --mode read
identity ack show bulletin:family-trash
```

Expected outcome:

- a pinned board entry appears in the `family` board timeline,
- acknowledgement data is attached to the pinned bulletin subject,
- `identity ack show` reflects actor + mode for follow-up reporting.
