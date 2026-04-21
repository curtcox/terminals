# Search and Memory Plan

See `repl-capability-closure.md` for the capability-closure rule this plan satisfies.

## Design Principle

Searchable memory is a platform service, not an app-by-app afterthought. If the system is expected to recover prior notes, route users to relevant history, or surface what happened while someone was away, the index and retrieval layer must be typed and reusable.

## Goals

- unified query across messages, boards, artifacts, records, and selected logs or activity streams,
- timeline views,
- related-subject retrieval,
- simple topic resurfacing for repeated household discussions,
- a REPL-visible memory surface that humans and agents can use.

## Data Sources

Initial indexable sources:

- room and direct messages,
- board posts and replies,
- artifacts and annotations,
- acknowledgement records,
- selected activity events,
- app-defined structured records that opt in.

## TAL Host Modules

Add `search` and optionally `memory`.

Suggested functions:

- `search.query(scope, text, filters)`
- `search.timeline(filters)`
- `search.related(subject_ref)`
- `search.recent(scope, window)`
- `memory.stream(filters)`
- `memory.link(subject_a, subject_b)`

If `memory` is not separate, fold these into `search`.

## Services

### SearchService

- `Query`
- `Timeline`
- `Related`
- `Recent`
- `Suggest`

### MemoryService` (optional)

- `GetActivityStream`
- `LinkSubjects`
- `ListLinks`

## REPL Surface

Add `search` and `memory` command groups.

Examples:

```text
search 'movie night'
search --scope messages oven
search --scope boards groceries
search related board_post_42
search timeline --since 24h

memory show household
memory links board_post_42
```

`timeline`, `related`, and `recent` are subcommands of `search`,
not top-level REPL groups.

## Topic Resurfacing

Apps should be able to ask for prior related content when a similar topic comes up again.

Examples:

- a groceries board post resurfaces prior list patterns,
- a repeated family discussion finds prior decisions,
- lesson review shows the learner's recent wrong answers,
- an alert investigation shows related recent signals and acknowledgements.

## Use Cases Enabled

This plan directly supports:

- searching notes, messages, and announcements,
- building a chronological household activity stream,
- surfacing relevant older notes when similar topics recur,
- reviewing learner progress and prior mistakes,
- recovering contextual history while away.

## Acceptance Criteria

- all core durable content types are queryable through one typed service,
- REPL can search and render timeline-oriented results in human and machine formats,
- TAL can retrieve related historical context deterministically,
- apps can opt structured records into indexing without custom search backends.
