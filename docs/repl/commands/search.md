# search

Unified search over indexed capability content (messages, board posts, artifacts,
and memory entries), plus timeline and related-subject views.

## Commands

- `search query <text> [--json]`
- `search timeline [scope] [--json]`
- `search related <subject-ref> [--json]`
- `search recent [scope] [--json]`

## Notes

- `scope` is optional and can be a kind (`message`, `board`, `artifact`, `memory`) or
	a short filter term.
- `search timeline` returns chronologically ordered activity records from the shared
	typed recent stream.
- `search related` scores indexed content by overlap with the subject phrase or id.
- `search recent` returns the most recent search-visible activity subset.

## Examples

```text
search query movie night
search timeline
search timeline memory
search related board_post_42
search related milk list
search recent
search recent message
```
