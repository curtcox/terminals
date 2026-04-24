# Application Migrations

See [masterplan.md](masterplan.md) for overall system context.
Extends [application-runtime.md](application-runtime.md)
(hot reload, per-activation version pinning, optional
`migrate(from_version, state)` export) and
[shared-artifacts.md](shared-artifacts.md) (durable artifacts).
Referenced by [application-distribution.md](application-distribution.md)
(upgrade lifecycle) and [signing-and-trust.md](signing-and-trust.md)
(who may authorize a migration).

## Problem

The distribution plan asserted that migrations "operate on store
snapshots and artifact patches, run in a transaction, and are
idempotent." Review flagged that this is unspecified and unsafe:

- Artifacts are first-class, identity-owned, referenced by other
  apps. A migration that patches them silently can corrupt data
  the migrating app does not own.
- Runtime specifies *per-activation* version pinning and a
  per-package `migrate(from_version, state)` function for
  *durable cross-version resume* — not a bulk data-mutating
  upgrade. The distribution plan conflated the two.
- No executor contract defines what a migration may call, what
  happens on crash mid-run, or how rollback undoes effects
  outside the app's own stores.

This document fixes those gaps. It defines two distinct
migration concepts, both inside the TAR runtime, and specifies
the executor that runs them.

## Design Principles

1. **Migrations are code running under a constrained sub-runtime,
   not a free-form upgrade script.** Permissions narrow during
   migration; they never widen.
2. **Data the app does not own is out of reach.** A migration
   may propose a patch to an artifact it created, but cannot
   reach into artifacts owned by other identities or touch the
   bus, telephony, HTTP, scheduler, or UI.
3. **Atomicity matches the boundary of the owning subsystem.**
   App-scoped stores are transactional. Artifact patches are
   journaled and reversible. Anything else is simply not
   reachable from a migration.
4. **Crash mid-run is a normal case.** Every migration is
   resumable from the last committed step.
5. **Version pinning is preserved.** Pre-upgrade activations
   continue on their old version; migration affects only durable
   state and newly-started activations.

## Non-Goals

- No schema DSL. Migrations are TAL (deterministic) with a
  narrowed module set.
- No automatic inference of the migration path. The author
  supplies forward migrations explicitly.
- No downgrade migrations beyond the minimum required to support
  `apps rollback`; downgrade with data mutation is intentionally
  hard.

---

## 1. Two Kinds of Migration

### 1.1 Activation-state migration (already in runtime)

Runtime defines `migrate(from_version, state)` as an optional
package export. It is called when an *existing* activation's
pinned version is older than the current loaded version and the
activation resumes. This is lightweight: it transforms a single
activation's JSON state dict.

This document does not modify that contract. It is called out
here because distribution review conflated it with §1.2.

### 1.2 Durable-data migration (new)

A durable-data migration transforms **app-owned durable data**
when the app is upgraded across a version boundary declared by
the author. It runs at most once per server per version step,
ordered forward, inside the executor defined in §3.

Durable data means:

- **App-scoped stores.** Declared in `manifest.toml`'s
  `[storage].stores`. Fully owned by the app; freely readable
  and writable from a migration.
- **App-authored artifacts.** Artifacts the app *created* (the
  artifact's `owner` reference is the app's definition). Only
  these may be patched from a migration. Artifacts merely
  referenced or annotated are read-only.

Nothing else is reachable. A migration cannot emit on the bus,
schedule a future trigger, open a UI view, place a call, hit
HTTP, invoke AI, or read presence / placement.

---

## 2. Manifest and Package Layout

A package declares durable-data migrations under `migrate/`:

```text
kitchen_timer/
├── manifest.toml
├── main.tal
└── migrate/
    ├── 0001_v1_to_v2.tal
    └── 0002_v2_to_v3.tal
```

File naming: `<step_number>_<from>_to_<to>.tal`. `<step_number>`
is zero-padded, monotonic, and gapless within a package. The
executor rejects a package with gaps or out-of-order numbering.

### 2.1 Manifest block

```toml
[migrate]
declared_steps = 2                       # sanity check
max_runtime_seconds = 120                # executor kills runaway migrations
checkpoint_every = 500                   # store ops between checkpoints
```

Manifest fields are advisory ceilings, not floors. The executor
enforces them; a migration that exceeds `max_runtime_seconds` is
rolled back to its last checkpoint and the upgrade aborts.

### 2.2 Version window

Each migration file declares a single (from, to) step. The
executor computes the shortest forward path from the installed
version to the target version and runs the files in order. A
missing intermediate step is a package-format error caught at
Gate 1, not at runtime.

---

## 3. Executor Contract

### 3.1 When the executor runs

A durable-data migration runs during `apps install` / `apps upgrade`
**after** the full vetting pipeline passes and **before** the new
package is registered with the scenario engine. This ordering
matters:

1. The executor runs with the old definitions still registered
   so existing activations keep working.
2. On success, the scenario engine swaps to the new definitions
   for new activations only. Existing activations stay pinned to
   the old version, per
   [application-runtime.md](application-runtime.md).
3. On failure, the old package remains the current package; the
   new package is left in staging.

### 3.2 Narrowed module set

Inside migration files, `load(…)` is restricted to:

| Module               | Why                                            |
|----------------------|------------------------------------------------|
| `store`              | Read/write app-scoped KV namespaces.           |
| `artifact.self`      | Patch artifacts *authored by this app*.        |
| `log`                | Structured logs scoped to the migration run.  |
| `migrate.env`        | Versions, checkpoint helpers, abort helper.   |

Everything else (`ui`, `bus`, `scheduler`, `placement`, `ai.*`,
`telephony`, `http`, `presence`, `world`, `claims`, `flow`,
`recent`, `pty`, `observe`) is unavailable — a `load("bus", …)`
inside `migrate/*.tal` fails at compile time with a specific
error.

`artifact.self` is a new host surface distinct from the general
`artifact` module in
[shared-artifacts.md](shared-artifacts.md). Its writes are
filtered by an owner check at the host layer: the artifact's
`owner` must match the migrating app's definition. A package
that tries to patch artifacts it did not author is rejected by
the executor with a structured error.

### 3.3 Journaled effects

The executor maintains a per-run journal:

- `apps/<name>/migrate/<step>/journal.ndjson` — append-only list
  of effects (`store.put`, `store.delete`,
  `artifact.self.patch`) with before/after hashes.
- `apps/<name>/migrate/<step>/checkpoint.json` — last committed
  step number and a logical cursor.

Every `checkpoint_every` effects the executor:

1. Flushes buffered writes to the underlying transactional
   store.
2. Appends journal entries.
3. Updates the checkpoint file with an fsync.

Failure modes:

- **Crash between effect and journal.** On restart, the executor
  sees the effect is not journaled and re-runs the migration
  from the last checkpoint. This is why §3.5 requires idempotent
  migrations.
- **Crash between journal and store commit.** The store
  transaction has not committed, so re-running replays the
  effect safely. Journal entries are idempotent on replay.
- **Crash after store commit but before checkpoint.** The
  checkpoint is behind the store; re-running replays committed
  effects against the now-newer state. Idempotency keeps this
  safe.

### 3.4 Transactional boundary

Store writes are transactional at the subsystem level: all store
effects in a single checkpoint group commit or none do. Artifact
patches are reversible via `artifact.self.patch`'s journal — a
failed migration rewinds artifact patches by applying their
inverse.

**There is no distributed transaction across stores and
artifacts.** Rollback is best-effort for artifacts: if rewinding
an artifact patch itself fails (e.g., the artifact was deleted
by its owner between patch and rollback), the executor logs a
reconciliation task and leaves the artifact alone. The migration
as a whole reports partial rollback, and the operator sees a
`warn` on the install.

This is explicit and documented rather than pretending to a
cross-system atomicity the platform does not provide.

### 3.5 Idempotency requirement

Every migration function MUST be a pure function of inputs
(current store state, current artifact contents) under the
executor's deterministic TAL runtime. The executor does not
verify idempotency statically, but re-runs after any crash — so
a non-idempotent migration will eventually corrupt data, and the
package CI is expected to test it with induced crashes.

### 3.6 Resource limits

- `max_runtime_seconds` (per step, from manifest).
- Hard caps independent of manifest: 100 MB total write volume
  per step, 10⁶ store ops per step, 10⁴ artifact patches per
  step. A migration that exceeds any hard cap is aborted and
  flagged as `block` on retry — an app whose migration needs
  more than this should redesign, not raise the cap.

---

## 4. Authority and Signing

A migration inherits the authority of the package it ships in.
Specifically:

- A migration runs only if the package was installed through the
  normal vetting pipeline, signed by a trusted author (per
  [signing-and-trust.md](signing-and-trust.md)) or quarantined.
- A quarantined install (per the future
  `plans/quarantine-sandbox.md`) MAY still run migrations, but
  the executor applies an additional filter: `artifact.self`
  writes are disabled under quarantine. Stores remain writable
  since they are app-scoped and cannot affect other apps.
- A rotation that transfers `(name, author_key)` to a new key
  does not replay old migrations. The new key's next upgrade
  runs only the migrations between the installed version and
  the new target version.

---

## 5. Downgrade

`apps rollback` installs an older package over a newer one. The
executor handles this by *not* running forward migrations in
reverse. Instead:

- If the older version ships an optional `migrate/downgrade/`
  directory with reverse steps, they are run in reverse order
  under the same executor rules.
- If not, the operator must choose `--archive-data` or `--purge`
  at the rollback command line. `--keep-data` is refused on a
  rollback that spans a version with no reverse migration.

Reverse migrations are optional by design: requiring them would
force authors to implement round-trip for every schema change,
which either makes authors avoid schema changes or ship
half-working reverse paths.

---

## 6. Operator Surface

Additions to the distribution plan's `apps` commands:

```text
apps migrate status <name>                 # current step, last checkpoint, last error
apps migrate retry  <name>                 # restart from last checkpoint
apps migrate abort  <name>                 # roll back to pre-upgrade state
apps migrate logs   <name> [--step=N]      # tail of structured migration logs
```

`apps upgrade` returns a structured result including:

- migration steps planned,
- steps completed,
- final verdict per step (`ok` / `partial-rollback` / `aborted`),
- a pointer to the journal files for post-hoc inspection.

---

## 7. Worked Example

`kitchen_timer` v1 stores completed-timer records in
`store.history`. v2 adds a `label_normalized` field used by a
new search feature.

`migrate/0001_v1_to_v2.tal`:

```python
load("store",        list_keys = "list_keys", get = "get", put = "put")
load("migrate.env",  checkpoint = "checkpoint", abort = "abort")
load("log",          info = "info")

def migrate():
    cursor = None
    count  = 0
    while True:
        page = list_keys(prefix = "history/", after = cursor, limit = 500)
        if len(page) == 0:
            break
        for key in page:
            rec = get(key)
            if "label_normalized" in rec:
                continue                 # idempotent: already migrated
            rec["label_normalized"] = _normalize(rec.get("label", ""))
            put(key, rec)
            count += 1
        cursor = page[-1]
        checkpoint()
    info("history.migrated", records = count)


def _normalize(label):
    return label.strip().lower()
```

Properties that make this migration safe under the executor:

- Touches only `store`, which is app-scoped.
- Pages through work and checkpoints after each page.
- Early-returns on records already carrying `label_normalized`,
  so re-running after a crash is a no-op for completed keys.
- Emits a single structured log at completion; nothing on the
  bus.

A malformed alternative — "also emit a `history.migrated` event
on the bus" — would fail at compile time because `bus` is not
loadable from migration files.

---

## 8. Acceptance Criteria

- A package with gaps in its migration numbering fails Gate 1
  with a specific error.
- A migration that tries to `load("bus")` fails at compile time
  with a specific error.
- A crash injected between `store.put` and the executor's
  checkpoint leaves durable state consistent on restart, and
  re-running the migration reaches the same final state.
- A migration that attempts `artifact.self.patch` on an
  artifact owned by another identity is rejected at the host
  layer with a structured error.
- A rollback across a version with no reverse migration fails
  with `--keep-data` and succeeds with `--archive-data`.
- `apps migrate status` returns last-step and last-error details
  sufficient for `apps migrate retry` or `apps migrate abort`.

---

## Open Questions

- **Long-running migrations without an operator.** The 120-
  second default is appropriate for small household apps but
  wrong for any app with large stores. Per-app ceilings are
  configurable, but the question is whether the executor should
  support chunked execution across server restarts as a first-
  class feature or keep forcing the migration to finish in a
  single run.
- **Migration cost budgets in the risk gate.** Gate 7 of
  distribution could use declared migration size to adjust risk
  scoring. Not specified here; noted so the distribution plan
  can reference it later.
