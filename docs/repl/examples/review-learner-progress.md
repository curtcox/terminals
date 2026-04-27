# Example: review-learner-progress

Review learner performance by combining lesson artifacts, memory notes, and typed search.

```text
artifact ls
artifact show art_1
memory remember learner:mia "Completed fractions worksheet level 2"
memory remember learner:mia "Needs help with word problems"
search query learner:mia
search timeline learner:mia
memory recall word problems
```

Expected outcome:

- artifact + memory entries provide a unified learning trail,
- `search query` and `search timeline` surface recent progress context,
- recall calls return the targeted intervention notes.
