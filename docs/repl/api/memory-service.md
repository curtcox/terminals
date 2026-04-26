# Memory Service

Typed operations for higher-level household and activity memory built on indexed search content.

## SearchService

- `Query(q)`
	- API: `GET /admin/api/search?q=<text>`
	- Returns ranked content matches across message, board, artifact, and memory records.
- `Timeline(scope)`
	- API: `GET /admin/api/search/timeline?scope=<kind-or-filter>`
	- Returns timeline-oriented activity items derived from typed recent activity.
- `Related(subject)`
	- API: `GET /admin/api/search/related?subject=<subject-ref-or-text>`
	- Returns indexed items related to a reference id or topic phrase.
- `Recent(scope)`
	- API: `GET /admin/api/search/recent?scope=<kind-or-filter>`
	- Returns a bounded newest-first slice suitable for quick resurfacing.

## MemoryService

- `Remember(scope, text)`
	- API: `POST /admin/api/memory/remember`
- `Recall(query)`
	- API: `GET /admin/api/memory?q=<text>`
- `Stream(scope)`
	- API: `GET /admin/api/memory/stream?scope=<scope-or-filter>`

The optional MemoryService can remain a standalone conceptual service while sharing
storage/indexing behavior with SearchService.
