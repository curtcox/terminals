---
name: log-progress
description: Append a dated entry to a plan's progress log and bump its `last-reviewed:` date. Use when the user wants to record completed work against a plan, e.g. "log progress on the protocol plan", "/log-progress for app-migrations: shipped X", "record what just landed against the io-abstraction plan", or asks to bump a plan's last-reviewed/status after shipping. Optional `--status <value>` rewrites the plan's status frontmatter.
---

# Log Progress Against a Plan

Standardizes how completed work gets recorded so progress logs stay
machine-readable and `last-reviewed` stays accurate. The operation is a pure
append — it never rewrites or rewords existing entries.

## Inputs

- **Plan reference** (required): either a path
  (`plans/features/protocol/plan.md`) or a fuzzy title match (`"protocol"`,
  `"app migrations"`).
- **Summary** (required): one paragraph (or a list of bullets) describing what
  just landed. Use the bullets the user provided verbatim where possible —
  this is their record, not yours to rewrite.
- **`--status <value>`** (optional): when present, rewrite the plan's
  `status:` frontmatter field. Valid values:
  `planned`, `building`, `shipped-untested`, `shipped-buggy`,
  `shipped-validated`, `superseded`, `archived`.

## Procedure (in order)

### 1. Resolve the plan path

If the user gave a path, verify it exists. If they gave a title:

```bash
grep -l -i -E "^title: \"[^\"]*<query>[^\"]*\"" \
  plans/features/**/plan.md plans/features/*.md plans/phases/*.md 2>/dev/null
```

(Match `title:` frontmatter case-insensitive substring.)

- **No match** → stop and report. Do not guess.
- **Exactly one match** → proceed.
- **Multiple matches** → list candidates and ask the user to pick.

### 2. Locate the progress sink

Check in order:

1. **Sibling `progress.md`** (the F3 shape) — preferred. Path:
   sibling of the plan file (e.g. `plans/features/protocol/progress.md`).
2. **Inline `## Implementation Progress` heading** in the plan itself —
   fall back.
3. **Neither exists** — create a sibling `progress.md` with frontmatter:

   ```yaml
   ---
   title: "<Plan Title> — Progress Log"
   kind: progress-log
   parent: <plan path relative to repo root>
   ---
   ```

   Then append a one-line pointer at the bottom of the plan body:

   ```markdown
   The dated implementation log lives in [`progress.md`](progress.md) (append-only).
   ```

**Stop and report** (don't guess) if the plan is in a non-standard shape —
e.g. multiple `## Implementation Progress` headings, or an inline block in a
plan that *also* has a sibling `progress.md`.

### 3. Append the dated entry

Read today's date from the `# currentDate` line in the conversation context.
Never hardcode a date.

- If the most recent `## YYYY-MM-DD` (or `## Implementation Progress (YYYY-MM-DD)`
  for the inline form) heading is **not** today's date, append a new
  `## <today>` heading first.
- Then append the user's summary as bullet(s) under that heading.

For the inline-progress fallback, match the existing heading style of the
target file (some use `## Implementation Progress (YYYY-MM-DD)` per day,
some have a single `## Implementation Progress` with dated bullets — mirror
what's already there).

### 4. Bump `last-reviewed:`

In the plan's YAML frontmatter, set `last-reviewed:` to today's date. This is
the only frontmatter field this skill touches unless `--status` was passed.

### 5. (Optional) Transition status

If `--status <value>` was passed, rewrite the `status:` frontmatter field of
the plan. Reject any value not in the list above.

### 6. Refresh the plans index

```bash
make plans-index
```

This regenerates [plans/INDEX.md](../../../plans/INDEX.md) so the new
`last-reviewed` (and any status change) shows up.

### 7. Show the diff and confirm

Run `git diff -- <plan> <progress sink> plans/INDEX.md` and show the user the
result. Confirm before considering the operation done.

## Constraints (non-negotiable)

- **Pure append.** Never rewrite, reword, or reorder existing progress
  entries. Existing bullets are immutable history.
- **Scope.** Touch only: the targeted plan file, its `progress.md` (creating
  if needed), and `plans/INDEX.md` (via `make plans-index`). Nothing else.
- **No date guessing.** Always read today's date from the `# currentDate`
  context line.
- **Stop on ambiguity.** If the plan's progress shape is non-standard (see
  step 2), report and stop rather than guess.

## Reporting

When done, tell the user:

- The plan path and the progress sink that was written to.
- The new dated entry (one-line summary).
- The new `last-reviewed:` value, and the new `status:` if changed.
- Confirmation that `make plans-index` ran clean.
