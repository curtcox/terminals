---
name: next
description: Answer "what should I work on next?" using `make next`. Wraps `scripts/pick-next-work.py` (priority buckets across plans) and adds three drift signals — un-validated planned use-case IDs, open audits/incidents, and stale `building` plans. Use when the user asks "what's next?", "what should I pick up?", "/next", or wants a refreshed snapshot of work-in-flight. Replaces the old hand-edited `next.md` pointer.
---

# What's Next

Single canonical answer to "what should I work on?". The `next.md` file
no longer exists — the answer is computed on demand from plan frontmatter
and use-case sources, not hand-maintained.

## Procedure

1. Run `make next` (equivalent to `python3 scripts/next.py`).
2. Show the user the **Pick** line and the **Drift signals** section.
3. If the user asks for the JSON form (e.g. for piping into another tool),
   run `python3 scripts/next.py --json`.

## What the report contains

- **Pick** — the top recommendation from `pick-next-work.py`'s four
  priority buckets (needs-attention → in-flight → promote → planned-small).
- **Un-validated planned use-case IDs** — IDs in `usecases/*.md`
  (indexed at `usecases/INDEX.md`) that no plan references via
  `validation: automated:<ID>` and that `scripts/usecase-validate.sh`
  doesn't already wire up.
- **Open audits and incidents** — `kind: audit`/`incident` with
  `status: open`.
- **Stale `building` plans** — plans with `status: building` whose
  `progress.md` (or `last-reviewed:` if no progress.md exists) hasn't
  been touched in ≥ 14 days.

## Constraints

- Never write `next.md`. The file was deleted on purpose; reintroducing it
  reintroduces the drift this skill is meant to eliminate.
- Don't second-guess the pick — if the user disagrees with the
  recommendation, surface the other priority buckets from the same report
  rather than re-ranking by hand.
- The report is read-only. To actually log progress on a plan, use the
  `log-progress` skill.
