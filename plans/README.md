# Plans

Each subdirectory under `plans/` holds one plan. A plan directory contains:

- `plan.md` — the authoritative spec (frontmatter: title, kind, status, owner, validation, last-reviewed)
- `progress.md` — running log of completed work (most recent entries first)
- Any supporting files (design notes, sub-plans, etc.)

## Progress-log rollover convention

Progress logs accumulate quickly. To keep them agent-readable:

**When a phase ships:** move the phase entries to
`plans/archive/<plan>/progress-YYYY-MM.md` and leave a one-line pointer
in the live `progress.md`:

```
_Entries before 2026-MM: archived at [plans/archive/<plan>/progress-YYYY-MM.md](...)._
```

**When a plan reaches `shipped-validated`:** move the entire `progress.md`
to `plans/archive/<plan>/progress-YYYY-MM.md` and replace the live file
with a one-line stub pointing to the archive.

The archive path convention is `plans/archive/<plan-name>/progress-<YYYY-MM>.md`
where the date is the most recent month covered by the archived entries.

## Frontmatter fields

| Field | Values | Meaning |
|---|---|---|
| `title` | string | Human-readable plan name |
| `kind` | `plan`, `audit` | `plan` = feature/refactor work; `audit` = investigation or review |
| `status` | see below | Current lifecycle state |
| `owner` | `copilot`, `curtcox`, `unowned`, … | Who is driving the work |
| `validation` | `none`, `manual`, `automated:<ID>` | How the plan's success is verified; `automated:<ID>` links to a use-case ID in `usecases/INDEX.md` |
| `last-reviewed` | `YYYY-MM-DD` | Date the plan was last read and verified as current |

## Status values

| Status | Meaning |
|---|---|
| `proposed` | Not yet started |
| `building` | Actively being implemented |
| `complete` | Implementation done, not yet validated |
| `shipped-validated` | Complete and all automated validations pass |
| `superseded` | Replaced by a newer plan |
| `paused` | Work stopped temporarily |
| `cancelled` | Dropped |
