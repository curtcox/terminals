# Application Distribution

See [masterplan.md](masterplan.md) for overall system context.
Extends [application-runtime.md](application-runtime.md) (TAR/TAL package
format and lifecycle) and [repl-capability-plan.md](repl-capability-plan.md)
(authoring substrate). Related: [scenario-engine.md](scenario-engine.md),
[shared-artifacts.md](shared-artifacts.md), [capability-lifecycle.md](capability-lifecycle.md),
[identity-and-audience.md](identity-and-audience.md).

Worked example used throughout: [docs/tal-example-kitchen-timer.md](../docs/tal-example-kitchen-timer.md).

## Problem

[application-runtime.md](application-runtime.md) defines *what* an app package
is (a directory with `manifest.toml`, TAL modules, kernels, models, assets)
and *how* it runs (TAR + scenario engine). It does not say how apps come
into existence, how they move between servers, or how a server decides it
is safe to load one written elsewhere.

This plan covers the end-to-end application supply chain:

1. **Author** — a user develops a new app on one Terminals server, using
   the REPL, Claude, and Codex.
2. **Publish** — the app is turned into a signed, versioned, portable
   package.
3. **Discover** — another server finds the package. Discovery itself is
   orthogonal and pluggable; this plan only fixes the interfaces.
4. **Vet** — the receiving server inspects the package through multiple
   independent gates before any code is loaded.
5. **Install** — TAR registers the app, provisions persistent storage,
   and activates it.
6. **Evolve** — versions, migrations, rollback, and uninstall.

## Design Principles

1. **The server is always in charge of its own load order.** No remote
   party can cause an app to be loaded, only *offered*. Discovery,
   transport, and author reputation never grant load authority.
2. **Vetting is layered, independent, and explicit.** Any single gate
   can block installation. Gates produce machine-readable verdicts
   that are persisted alongside the installed package.
3. **Package identity is content-addressed.** Two packages with the
   same bytes are the same package, on any server, forever.
4. **Author identity is cryptographic, not social.** Names are UX;
   keys are authority.
5. **Persistence is the app's contract, not the package's.** App code
   is replaceable; app data outlives it and is versioned separately.
6. **Claude and Codex are reviewers and authors, not signers.** AI
   analysis is input to a human or policy decision, never the
   decision itself.

## Non-Goals

- No design for the discovery layer (public registry, gossip, QR
  hand-off, git remotes). Discovery plugs into the interfaces below.
- No app-store economics (payment, licensing, DRM).
- No sandbox design beyond what TAR already specifies in
  [application-runtime.md](application-runtime.md); this plan relies
  on the permission/capability model already defined there.
- No cross-server *running* activations. Sharing transfers packages,
  not live state. Federated activations are out of scope.

---

## 1. Authoring on a Terminals Server

A Terminals server is a self-sufficient development environment for
TAL apps. The author never leaves the system to write, test, or ship
an app — every step is a REPL command or an AI-assisted edit through
the MCP adapter.

### 1.1 The REPL loop

The REPL (see [repl-and-shell.md](repl-and-shell.md) and
[repl-capability-plan.md](repl-capability-plan.md)) is the primary
authoring surface. The relevant command families are:

| Area             | Example commands                                    |
|------------------|-----------------------------------------------------|
| Package scaffold | `app new <name>`, `app describe <name>`             |
| Edit             | `app edit <name>` (opens the package tree)          |
| Simulate         | `sim run <name>`, `sim trigger`, `sim advance`      |
| Activate locally | `app install --dev <name>`, `app reload <name>`     |
| Inspect          | `activations ls`, `observe tail`, `logs tail`       |
| Package          | `app pack <name>`, `app sign <name>`                |

The kitchen-timer example runs end-to-end under `sim run
kitchen_timer` with synthetic triggers; no device or speaker is
needed to write the app. This is the inner loop: edit → `sim run`
→ `app reload` → drive from a real device.

`--dev` installs bypass the full vetting pipeline described in §5,
but still cannot exceed the manifest's declared permissions. Dev
mode is a property of the *install*, not the package.

### 1.2 Claude and Codex as authoring assistants

Both Claude Code and Codex connect to the server through the
Terminals MCP adapter (see `docs/repl/agents/claude-code-setup.md`
and `docs/repl/agents/codex-setup.md`). Through MCP they can:

- call every `read_only` REPL command directly,
- propose `mutating` REPL actions that prompt the user before
  running,
- read TAL source, tests, and manifests in `terminal_server/apps/`,
- run `sim run` and read structured test output.

An authoring session with Claude or Codex looks like:

1. The author states an intent in natural language
   ("a timer that speaks when it expires").
2. The agent inspects existing apps and TAL host modules to reuse
   patterns (the kitchen-timer walkthrough is the canonical
   example).
3. The agent drafts `manifest.toml`, `main.tal`, and a
   `tests/*.tal` file.
4. The agent runs `sim run` through MCP and iterates on failures.
5. The agent proposes `app install --dev`; the author confirms.

The agents are deliberately on *both* sides of the supply chain: at
authoring time they help the author satisfy the checks the
installing server will later run (§5.3 explicitly runs an AI-assisted
review on the receiving side, unconditionally, regardless of who
wrote the code). The author's agent is advisory; the receiver's
agent is adversarial.

### 1.3 Claude and Codex as runtime providers

TAL's `ai.*` host modules (`ai.tts`, `ai.chat`, `ai.complete`, …)
are backed by the kernel's AI adapter interface. Claude and Codex
can be registered as providers for the appropriate families:

- Claude: `ai.chat`, `ai.complete`, `ai.review` (text, tool use).
- Codex: `ai.complete`, `ai.review` (code-focused completions).

Provider selection is a server-policy decision, not an app concern.
An app that declares `permissions = ["ai.chat"]` gets whatever
provider the server has configured for that family; swapping Claude
for a local model never requires an app update. This preserves the
rule in [CLAUDE.md](../CLAUDE.md): AI providers live behind
interfaces in server code.

---

## 2. Package Format and Versioning

### 2.1 Package as a directory (already defined)

`application-runtime.md` fixes the on-disk layout:

```text
<app_name>/
├── manifest.toml
├── main.tal
├── lib/…
├── tests/…
├── kernels/…   # wasm operator kernels
├── models/…    # weights, ONNX, etc.
└── assets/…
```

### 2.2 Package as a file (new)

For transport, a package is a deterministic tar of that directory
plus a sidecar `package.sig`:

```text
<name>-<version>.tap        # tar of the package directory
<name>-<version>.tap.sig    # detached signature bundle
```

`.tap` = "Terminals Application Package." The tar is deterministic
(sorted entries, zeroed mtimes, fixed uid/gid, no extended
attributes) so the file hash is stable across authors who build
from the same sources.

**Package identity.** `package_id = sha256(<tap file>)`. This is the
only identifier that crosses server boundaries. `name` and
`version` are human-facing metadata inside the manifest; they do
not grant identity.

### 2.3 Signature bundle (`package.sig`)

A TOML document with one or more signatures over the package hash:

```toml
package_id = "sha256:…"

[[signature]]
role     = "author"
key_id   = "ed25519:…"
sig      = "base64:…"
created  = "2026-04-24T12:34:56Z"

[[signature]]
role     = "voucher"
key_id   = "ed25519:…"
sig      = "base64:…"
scope    = "reviewed+tested"
created  = "2026-04-25T09:00:00Z"
```

Roles (extensible, but these three are defined here):

- `author` — the identity that produced the source. Required.
- `voucher` — a third party attesting they reviewed or tested
  the package. Optional; multiple allowed. §5.2 defines how a
  server treats vouchers.
- `publisher` — the server that packaged the file. Optional, used
  when a server re-hosts something it did not author.

Keys bind to identities through the existing identity scheme in
[identity-and-audience.md](identity-and-audience.md). A signature
with an unknown key is never *invalid* per se; it is simply
low-trust input that the vetting pipeline weighs accordingly.

### 2.4 Version scheme

Three distinct numbers, each with its own semantics:

| Field                  | Meaning                                                 |
|------------------------|---------------------------------------------------------|
| `version`              | Semver of the app itself. Breaking change = major bump. |
| `language = "tal/1"`   | TAL dialect. Breaking dialect change = major bump.      |
| `requires_kernel_api`  | Host module contract (`"1.x"` means ≥1.0 <2.0).         |

Versioning rules:

- `version` is **monotonic per `(author_key, name)`**. The
  receiving server refuses to install a lower version over a
  higher one without explicit `--allow-downgrade`.
- A major bump in `version` implies a *possibly* incompatible
  persistence schema. See §3.3.
- `requires_kernel_api` is a hard gate. If the installing server
  cannot satisfy the range, the package is rejected at load time
  with a structured error — never silently partially loaded.

---

## 3. Persistence

Apps persist two distinct things. They are managed by different
subsystems, survive different events, and version independently.

### 3.1 Activation state (ephemeral-durable)

Already specified in [application-runtime.md](application-runtime.md):

- Lives inside the activation's state dict.
- Must be JSON-serializable.
- Snapshotted on every successful commit.
- Restored by `resume` after a suspend or crash.
- Discarded when the activation terminates (`done = True` or
  explicit `stop`).

Kitchen-timer's `remaining`, `status`, and `target` fields are
this kind of state. On uninstall or upgrade, running activations
are given a chance to `suspend` and then discarded — activation
state is not a long-term record.

### 3.2 App-owned durable data

Apps that need data to outlive their activations declare it in the
manifest:

```toml
[storage]
artifact_kinds = ["note", "checklist"]   # from shared-artifacts.md
stores         = ["preferences"]         # namespaced KV
```

- **Artifacts** (see [shared-artifacts.md](shared-artifacts.md))
  are versioned, auditable documents owned by identities. Apps
  read and write them through the `artifact` host module. They
  are first-class citizens of the server, not of any one app
  install — reinstalling an app does not delete them.
- **Stores** are app-scoped KV namespaces. Each declared store
  gets a private prefix that only the installed app with the
  matching `(name, author_key)` can read or write.

Storage ownership rules:

- Scoping key is `(name, author_key)`, not `package_id`. Upgrading
  from v1.0.0 to v1.1.0 keeps the same scope.
- A package signed by a *different* author key is, by definition,
  a different app even if `name` matches. It gets a fresh scope.
  This is what prevents name-squatting from hijacking data.
- Uninstall offers three modes: `--keep-data` (default),
  `--archive-data` (compressed snapshot retained), `--purge`.

### 3.3 Migrations across versions

A package may ship a `migrate/` directory with forward migrations
between its own versions:

```text
my_app/
└── migrate/
    ├── 0001_v1_to_v2.tal
    └── 0002_v2_to_v3.tal
```

Each migration is a pure TAL function operating on a JSON snapshot
of the app's stores and on artifact patches. Migrations:

- run after vetting passes and before the new app is registered
  with the scenario engine,
- run inside a transaction; a failure rolls back the install,
- must be idempotent (re-running yields the same output), because
  a crash mid-upgrade will re-run the migration at next start,
- never cross author keys; there is no cross-author upgrade path.

Downgrade migrations are optional and discouraged; if absent,
`--allow-downgrade` requires `--purge` or `--archive-data`.

---

## 4. Sharing Between Servers

Sharing transfers `.tap` + `.tap.sig` pairs between servers.
Everything else — how they are advertised, fetched, or mirrored —
is discovery, which this plan leaves pluggable.

### 4.1 The discovery interface

```go
type PackageSource interface {
    // List candidate packages matching a query.
    List(ctx context.Context, q PackageQuery) ([]PackageHandle, error)

    // Fetch one candidate. Returns tap bytes and signature bundle.
    Fetch(ctx context.Context, h PackageHandle) (Package, error)
}
```

The server ships with at least two built-in implementations;
additional sources are loaded as plugins:

- **File source** — a local directory of `.tap` files.
- **Peer source** — another Terminals server reachable over the
  existing transport; the peer exposes a read-only catalog
  endpoint.

A public registry, a mirror-through-Git scheme, a gossip layer
over the bus, or an offline USB-stick workflow are all future
`PackageSource` implementations. None of them change the rest of
this document.

### 4.2 Offering vs. installing

A `PackageSource` can only *offer* packages. It can never cause
one to be installed. The install command takes a handle from a
source and runs it through the pipeline in §5:

```text
apps offer <source> <query>        # list candidates
apps fetch <source> <handle>       # download to staging
apps vet <staged>                  # run gates, produce report
apps install <staged> [--voucher=…]
```

`apps fetch` writes to a staging directory and never modifies the
live registry. Staged packages are garbage-collected if not
installed within a configurable TTL.

---

## 5. Install-Time Vetting

Every non-`--dev` install runs a pipeline of independent gates.
Gates are ordered cheapest-first so a rejection short-circuits.
Each gate produces a structured verdict (`pass`, `warn`, `block`)
with evidence; the verdict set is persisted with the install and
available via `apps describe`.

```text
staged package
   │
   ▼
(1) Manifest gate      ── static, <100ms
(2) Signature gate     ── crypto check
(3) Author/voucher gate── policy lookup
(4) Static analysis    ── permissions vs. code
(5) AI-assisted review ── Claude/Codex, structured prompt
(6) Conflict/redundancy── vs. already-installed apps
(7) Risk analysis      ── capability impact, blast radius
   │
   ▼
verdict set → install decision
```

Any gate returning `block` stops the pipeline. `warn` does not
stop, but an install with any `warn` requires explicit operator
acknowledgment (or policy-level auto-accept for trusted sources).

### 5.1 Gate 1 — Manifest

Purely static checks on `manifest.toml`:

- All required fields present and well-typed.
- `language` and `requires_kernel_api` supported by this server.
- `permissions` is a subset of the known permission set.
- `exports` names are not already registered by another app.
- Referenced files (kernels, models, assets) exist in the tarball.

### 5.2 Gate 2 — Signatures

For each `[[signature]]` in `package.sig`:

- Signature is cryptographically valid over `package_id`.
- The key is parseable and not on the revocation list.
- Timestamps are plausible relative to the server's clock.

An unsigned package is not an error at this gate — it is data fed
to Gate 3, which decides what unsigned means under current policy.

### 5.3 Gate 3 — Author and voucher policy

The server maintains a trust store mapping keys → roles and
weights. Typical policies:

- "Only install packages with an `author` signature from a key in
  my trust store."
- "Accept unknown `author` if at least two `voucher` signatures
  from trusted reviewers are present."
- "Accept any signed package into a quarantine tier, where
  `placement.read`, `ai.tts`, and `bus.emit` are allowed but
  `telephony.*` and `scheduler` are not."

Policy is expressed declaratively (not TAL — this runs before TAR
can load anything) and is itself versioned and auditable. The
default policy is conservative: an author key must be trusted or
a human operator must approve the install.

### 5.4 Gate 4 — Static analysis

TAL is deterministic and small, which makes static analysis
tractable. The analyzer checks:

- Every `load(…)` call names a module covered by a declared
  permission. (This is also enforced at TAR load time; catching
  it here gives a cleaner rejection.)
- No calls into removed or deprecated host APIs.
- No suspicious patterns: unbounded `scheduler.every`, recursive
  bus emission without a guard, writes to stores not declared in
  `[storage]`.
- Tests exist and name the public definitions; packages without
  tests are `warn`, not `block`, by default.

### 5.5 Gate 5 — AI-assisted review

The server runs a code review pass using its configured
`ai.review` provider (Claude, Codex, or another). The prompt is
structured, not free-form, and asks for:

- a plain-English summary of what the app does,
- a list of side effects the app can cause, grouped by host
  module,
- any mismatch between the manifest `description` and actual
  behavior,
- any pattern the reviewer considers risky *for this specific
  deployment* (the prompt includes the installing server's
  declared zones, device roles, and already-installed apps),
- a verdict: `pass` / `warn` / `block` with reasons.

The AI verdict is treated as one input, not the final answer. A
`block` from the AI reviewer escalates to a human; a `pass`
never *bypasses* human review required by policy. The exact
review prompt and the reviewer's response are persisted verbatim
with the install, so decisions are auditable after the fact.

### 5.6 Gate 6 — Conflict and redundancy

Before registering, TAR simulates the install against the current
registry:

- **Export conflicts.** Two apps cannot claim the same export
  name. `kitchen_timer` v2 from the same author replaces
  v1 (upgrade path); from a different author it is rejected as a
  conflict.
- **Trigger overlap.** If the new app's `match(req)` would
  accept requests currently owned by an installed app, the
  reviewer is shown both apps' match conditions and asked to
  choose, or to install with a priority override.
- **Redundancy.** If the new app's declared capabilities and
  triggers are a strict subset of an installed app's, the gate
  emits a `warn` with the overlap. This prevents silently
  accumulating three timers that all respond to `timer.set`.

The conflict analysis uses only manifest metadata and static
`match` analysis; it never executes TAL from the candidate
package.

### 5.7 Gate 7 — Risk analysis

A capability-weighted assessment of blast radius:

- Which host modules does this app touch, and which are
  irreversible (`telephony.place_call`, `ai.tts` on shared
  speakers, `bus.emit` patterns that other apps subscribe to)?
- Does the install introduce a new permission class to this
  server (e.g., first app ever to request `telephony`)?
- How many zones / devices does `placement` access imply?
- What is the failure mode if the app crashes in `handle`?

The output is a risk report attached to the install decision.
Policies can auto-block above a threshold; by default this gate
produces `warn` only.

### 5.8 Install decision

After all gates run:

- **All `pass`** → install proceeds.
- **Any `warn`** → install requires acknowledgment; under policy
  auto-accept, proceed and record that the warn was auto-accepted.
- **Any `block`** → install aborted; the full verdict set is
  retained so the author (or operator) can see why and fix it.

The complete verdict bundle is signed by the installing server's
key and stored with the package. Later `apps describe <name>`
returns it. Re-running `apps vet` against an already-installed
package is always allowed and produces a fresh verdict — useful
after a policy or AI-model change.

---

## 6. Upgrade, Rollback, Uninstall

### 6.1 Upgrade

An upgrade is an install of version `v_new` where `(name, author_key)`
matches an existing install at `v_old`:

1. Run the full vetting pipeline against `v_new`. Prior verdicts for
   `v_old` do not short-circuit this — policy or models may have
   moved.
2. On pass: `suspend` running activations, run migrations from
   `v_old` → `v_new`, atomically swap the registered definitions,
   then `resume`.
3. On failure at any step: the old version remains live; the new
   package is left in staging for inspection.

### 6.2 Rollback

`apps rollback <name>` reinstalls the most recent previous version
that is still on disk. Rollback *is* an install and runs the full
pipeline again. If the rolled-back version lacks a reverse
migration, the operator must choose `--archive-data` or `--purge`.

### 6.3 Uninstall

`apps uninstall <name> [--keep-data|--archive-data|--purge]`:

1. Stop or suspend all activations.
2. Unregister definitions from the scenario engine.
3. Apply the data policy.
4. Remove the package directory; retain the `.tap`, `.tap.sig`,
   and verdict bundle in an archive for later audit.

---

## 7. Operator Surface

The REPL commands that make this plan usable. All are available
over MCP for Claude and Codex, with mutating commands gated by
the existing approval flow.

```text
# authoring
app new <name>
app edit <name>
app pack <name>
app sign <name> [--role=author]

# sharing
apps source add <kind> <config>
apps source ls
apps offer <source> [--query=…]
apps fetch <source> <handle>

# vetting and install
apps vet <staged>
apps describe <staged|installed>
apps install <staged> [--voucher=…] [--allow-warn]
apps upgrade <name> <staged>
apps rollback <name>
apps uninstall <name> [--keep-data|--archive-data|--purge]

# trust
apps keys add <key> [--role=…]
apps keys revoke <key>
apps policy show
apps policy set <path> <value>
```

---

## 8. Worked Example (Kitchen Timer, End-to-End)

A compressed trace using the example in
[docs/tal-example-kitchen-timer.md](../docs/tal-example-kitchen-timer.md).

1. **Author on Server A.**
   - Operator: "build me a kitchen timer" (via Claude over MCP).
   - Agent scaffolds `kitchen_timer/` with `manifest.toml`,
     `main.tal`, and `tests/kitchen_timer_test.tal`.
   - `sim run kitchen_timer` passes.
   - `app install --dev kitchen_timer` registers it locally.
   - Real voice trigger on a kitchen tab verifies end-to-end.

2. **Publish on Server A.**
   - `app pack kitchen_timer` → `kitchen_timer-0.1.0.tap`.
   - `app sign kitchen_timer --role=author` →
     `kitchen_timer-0.1.0.tap.sig` with the operator's author key.

3. **Discover on Server B.**
   - Server B has a `file` source pointing at a synced folder, or
     a `peer` source for Server A. Discovery itself is not
     specified further here.
   - `apps offer file "kitchen_timer*"` lists the candidate.

4. **Vet on Server B.**
   - Manifest gate: `ai.tts`, `ui.*`, `scheduler`, `bus.emit`,
     `placement.read` all known → `pass`.
   - Signatures: author signature valid; author key unknown →
     policy-dependent.
   - Author/voucher: policy requires either a trusted author or a
     trusted voucher. Server A's author key is imported with
     `apps keys add` → `pass`. Alternatively, a reviewer on
     Server B signs a voucher after reading the code.
   - Static analysis: all `load(…)`s covered; test exists.
   - AI review: Claude summarizes the app as "a one-activation
     timer that patches a UI, speaks on expiry, and emits
     `timer.expired` on the bus" — matches manifest → `pass`.
   - Conflict/redundancy: no other app matches
     `timer.set` / `timer.start` → `pass`.
   - Risk: `ai.tts` on shared kitchen speaker flagged `warn`;
     operator acknowledges.

5. **Install on Server B.**
   - `apps install kitchen_timer-0.1.0.tap` registers the
     definition, provisions no stores (none declared), attaches
     the verdict bundle, and the app is live.

6. **Upgrade.**
   - Server A ships `0.2.0` with snoozing. Server B repeats
     discovery + vet + install. Because `(name, author_key)`
     matches, the existing install is upgraded in place. No
     migration needed (activation state dict is forward-
     compatible; no stores were declared).

---

## Open Questions

- **Voucher discovery.** Vouchers are signatures on a package
  hash; how do they travel? Same source as the package, a
  separate endpoint, or bus messages? Left to the discovery layer.
- **AI-review provider independence.** Should Gate 5 *always* run
  a second reviewer with a different provider (Claude and Codex
  both, or Claude and a local model) to reduce single-model
  blind spots? Worth piloting.
- **Quarantine tier.** Should TAR support a reduced-permission
  runtime tier for provisional installs, separate from `--dev`?
  This plan describes it in §5.3 as a policy option but does not
  specify its implementation; that belongs in a follow-on
  amendment to [application-runtime.md](application-runtime.md).
- **Artifact ownership on uninstall.** Artifacts are first-class
  and may be referenced by other apps or users. The default
  `--keep-data` preserves them; a future plan should specify
  reference counting so `--purge` is safe.
