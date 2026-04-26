# Example: search-household-memory

Use typed search and memory commands to resurface context from prior household activity.

```text
memory remember groceries "milk, eggs, bread"
memory remember school "math worksheet due Tuesday"
search query milk
search related groceries
search timeline memory
search recent memory
memory stream groceries
```

Expected outcome:

- the query and related calls return matching indexed entries,
- timeline/recent calls show memory and message activity in chronological slices,
- memory stream yields scoped memory items for quick recall.
