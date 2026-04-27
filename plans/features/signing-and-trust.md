---
title: "Signing and Trust"
kind: plan
status: shipped-validated
owner: copilot
validation: manual
last-reviewed: 2026-04-27
---

# Signing and Trust

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.
Depends on [package-format.md](package-format.md) (canonical
signed statements) and is referenced by
[application-distribution.md](application-distribution.md)
(vetting pipeline, install transaction) and
[app-migrations.md](app-migrations.md) (migration authority).
Distinct from [identity-and-audience.md](identity-and-audience.md),
which describes *people and audiences* for application logic, not
*signing keys* for the application supply chain.

## Problem

Distribution assumes a trust store, key roles, voucher scope
ceilings, storage authority, installer server keys, and a
revocation list, and asserts that revocation and rotation have
well-defined consequences on already-installed apps. None of
that existed in the repo before this document. `identity-and-
audience.md` covers a different layer — household people and
audiences for in-app logic, not the keys that authorize package
installation.

This document fixes:

- Key lifecycle (creation, rotation, revocation, compromise).
- Authority scoping for authors, vouchers, publishers, and
  server-local installer/audit keys.
- Storage and registry identity that is stable across key
  rotation (**app_id / author lineage**).
- Consequences on already-installed apps when keys move state.
- A minimum v1 policy schema consumed by the distribution
  vetting pipeline.
- An append-only log chain that covers both trust-store
  mutations and verdict-bundle writes.

## Design Principles

1. **Keys authorize, identities explain.** A key is the unit of
   authority; display names and descriptions are UX.
2. **App lineage is stable; keys rotate.** Storage and verdict
   paths key off a stable app lineage identifier assigned at
   first install, not off the current signing key.
3. **Every key has a lifecycle.** Creation, active use, rotation,
   revocation, archival.
4. **Authority is scoped.** A key's rights are an explicit set
   of roles and, for vouchers, an explicit scope ceiling.
5. **Revocation has observable consequences.** Revoking a key
   moves installed apps into defined states with audit records.
6. **Rotation is adversarial.** The receiving server assumes any
   rotation could be forged by a stolen old key and requires
   additional confirmation before transferring authority.
7. **The trust store is server-local and auditable.** There is
   no global truth. Every server owns its decisions and can
   prove them through a hash-chained log.

## Non-Goals

- No PKI hierarchy or X.509. Keys are raw Ed25519.
- No remote attestation protocol for trust-store sync.
- No policy for humans inside an organization. This document
  describes machine authority, not HR.
- No algorithm agility in v1. All signatures are Ed25519;
  adding another algorithm requires a statement-schema bump
  (see [package-format.md](package-format.md) §2.2).

---

## 1. Keys

### 1.1 Identity of a key

```
key_id = "ed25519:" + base64url(pubkey_bytes)
```

The `key_id` is the **entire** cryptographic identity a verifier
needs. Display names, owner notes, and contact info live in the
trust store (§3) and are not part of the `key_id`.

v1 accepts only the `ed25519:` prefix. A verifier encountering
any other prefix rejects the statement; the prefix is not a
negotiation surface.

### 1.2 Key roles

A key may hold any subset of these roles in a given trust store:

| Role         | May sign                                | Additional authority                                 |
|--------------|-----------------------------------------|------------------------------------------------------|
| `author`     | `role = "author"` statements            | Owns an app lineage (§1.4) for any app it authors.   |
| `voucher`    | `role = "voucher"` statements           | Scope ceiling (§2) constrains what it can vouch for. |
| `publisher`  | `role = "publisher"` statements         | None. Informational.                                 |
| `installer`  | Verdict bundles and install log entries | Server-local. Never signs packages. See §6.          |
| `operator`   | Local trust-store edits                 | Not a package-signature role; listed for completeness. |

A key MAY hold multiple roles, but `installer` keys are always
server-local and SHOULD NOT also be `author` keys — that
separation keeps a compromised package-author key from rewriting
install history.

### 1.3 Key states

Every key known to a server is in exactly one state:

- `candidate` — added to the trust store pending operator
  confirmation; may not authorize anything yet.
- `active` — normal working state.
- `rotated` — superseded by a successor key (§4); statements
  from a rotated key are accepted only if the server observed
  them before the `rotation_accepted_at` moment (§4.4). Signed
  `created_unix` alone does not qualify.
- `revoked` — rejected unconditionally; triggers §5 consequences.
- `archived` — not trusted for new statements, but historical
  statements remain valid for audit.

Transitions are operator-driven (`term apps keys …` — see §7)
and recorded in the signed trust-store log (§3.3).

### 1.4 App lineage

A new concept required to make storage and verdict paths stable
across author-key rotation.

```
app_id_bytes = sha256(first_author_key_id || "\0" || manifest_name)
app_id       = "app:sha256:" + hex(app_id_bytes)
```

The textual form `"app:sha256:<hex>"` is the **only** encoding of
`app_id` used in persisted files, JSON fields, paths, CLI
arguments, and cross-plan references. Bare `"sha256:…"` is
reserved for `package_id` and similar; unprefixed hex is a
format rejection.

- The **first** time a package with a given `manifest_name` is
  installed, the server computes `app_id` from the installing
  author key and records it in the registry.
- On rotation that names this app (§4), the registry records a
  new lineage edge `(old_author_key, new_author_key)` under the
  same `app_id`. The `app_id` itself does not change.
- All server-local paths (store prefixes, verdict bundle
  filenames, archive directories, migration journals) are
  keyed by `app_id`, not by `(name, author_key)`.
- `(manifest_name, current_author_key)` is how a *package* is
  matched to an installed app during vetting; `app_id` is how
  *state* is addressed on disk.

This closes the collision gap flagged in review: two apps
sharing a name prefix get distinct `app_id`s, and rotated
authors do not invalidate existing paths.

---

## 2. Voucher Scope Ceilings

A voucher key is configured in the trust store with a **scope
ceiling** — the most-permissive scope it is allowed to commit
to. A signed statement whose declared `scope` exceeds the
ceiling is rejected even if the signature is cryptographically
valid.

Trust-store entry:

```toml
[key."ed25519:abc…"]
roles   = ["voucher"]
state   = "active"
note    = "Staging-server automated reviewer"

[key."ed25519:abc…".voucher_ceiling]
max_tier        = "quarantine"
allowed_testing = ["sim-only"]
max_expiry_days = 30
```

Ceiling evaluation:

- `max_tier` — signed `scope.tier` ≤ this. Ordering:
  `quarantine < custom < full`.
- `allowed_testing` — `scope.tested_under` ∈ this set.
- `max_expiry_days` — if the signed statement lacks
  `expires_unix`, the server treats it as if expiring at
  `server_observed_at + max_expiry_days * 86400`, using the
  trust-log acceptance time (§3.3), not the signer's
  `created_unix`.

A voucher without a ceiling entry defaults to `max_tier =
"quarantine"`, `allowed_testing = ["sim-only"]`,
`max_expiry_days = 14`. New unknown vouchers start weak.

---

## 3. Trust Store

### 3.1 Contents

A versioned TOML file plus an append-only log:

- **Keys.** One entry per `key_id` with state, roles, scope
  ceiling (for vouchers), and `first_observed_at`.
- **Rotation records.** See §4.
- **Policies.** Named, declarative rules (§6).
- **Server installer keys.** One active key pair used for
  signing verdict bundles and install log entries (§1.2).
- **Lineage map.** `app_id` → ordered list of author keys,
  recorded at install and on each rotation acceptance.

### 3.2 Location

- Primary file: `<server_data>/trust/store.toml`.
- Trust-mutation log: `<server_data>/trust/log.ndjson`.
- Verdict log: `<server_data>/trust/verdicts.ndjson` (normative
  schema in §6.4).
- Verdict bundles: `<server_data>/trust/verdicts/<app_id>/<seq>.json`.
- Mutations go through `term apps keys …` / `term apps
  policy …` commands; direct edits are detected by log-chaining
  and flagged.

### 3.3 Log chain

Every mutation appends one line to `log.ndjson`:

```json
{
  "seq": 42,
  "at": 1714000000,
  "actor": "operator:curt",
  "op": "keys.rotate.accept",
  "args": {"old_key": "ed25519:…", "new_key": "ed25519:…", "names": [...]},
  "prev_hash": "sha256:…",
  "this_hash": "sha256:…",
  "installer_sig": "base64:…"
}
```

`prev_hash` chains each entry to the previous one's `this_hash`;
`this_hash = sha256(canonical_json({seq,at,actor,op,args,prev_hash}))`.
Every entry is signed by the current `installer` key — a gap,
hash mismatch, or invalid `installer_sig` invalidates the store
until an operator reconciles it.

This records *server-observed ordering*, which §1.3 and §4.4
rely on to resist signer-controlled timestamp manipulation.

---

## 4. Key Rotation

Rotation lets an author continue to own an app lineage after
replacing their signing key, without re-consenting every
downstream server per install. The v1 protocol is deliberately
conservative: a rotation signed only by the old key is *not*
sufficient.

### 4.1 Rotation statement

Rotation is a **pair of signed statements** plus operator
acceptance.

First, the outgoing author publishes:

```
OldKeyRotationStatement (CBOR, deterministic):
  schema         : "rotation-stmt/1"
  old_key        : "ed25519:…"
  new_key        : "ed25519:…"
  proposed_at    : <unix seconds, advisory only>
  name_scope     : ["kitchen_timer", "coffee_timer"]
  reason         : <tstr, optional>
  sig_old        : signature by old_key over the above
```

The new-key holder must countersign before the server accepts:

```
NewKeyRotationStatement (CBOR, deterministic):
  schema              : "rotation-stmt/1"
  old_key_stmt_digest : sha256(CBOR of OldKeyRotationStatement)
  new_key             : "ed25519:…"
  accept_at           : <unix seconds, advisory only>
  sig_new             : signature by new_key over the above
```

Both statements carry the `schema` string verbatim; v1 verifiers
reject unknown schemas and reject statements missing `schema`.

A stolen old key alone cannot forge authority transfer: it must
also sign over a new key whose holder has not agreed to the
transfer. If an attacker holds only the old key, forged
rotations trivially fail at the new-key countersignature step.

### 4.2 Operator acceptance

In addition to both signed statements, the receiving server
requires an operator action:

```
term apps keys rotate --accept rotation.cbor
```

Effect: records `rotation_accepted_at` (the trust-log sequence
and wall-clock time at which the operator accepted) in the
lineage map. That value — *not* the signer-controlled
`proposed_at` or `accept_at` — is the authoritative rotation
moment for §1.3 and §5 cutoffs.

Rotation acceptance is a `critical_mutating` REPL operation
(see distribution plan) — AI agents may *propose* it but cannot
execute it through ordinary mutating approval.

### 4.3 Name scope and lineage

`name_scope` is the list of app manifest names whose lineages
transfer:

- For each listed name whose `app_id` is owned by `old_key`,
  append `new_key` to the lineage and mark the edge with
  `rotation_accepted_at`.
- A rotation that omits `name_scope` (or uses `["*"]`) is
  permitted only if an explicit trust-store policy grants
  wildcard rotation to the old key. Wildcard rotation is
  `critical_mutating` and requires operator re-confirmation
  per-name.

### 4.4 Cutoff semantics

After rotation acceptance:

- The old key's state moves to `rotated`.
- Statements signed by the old key are accepted only if they
  were `first_observed_at` a trust-log sequence less than the
  rotation-acceptance sequence. Signer-controlled
  `created_unix` is ignored for this purpose. A stolen old key
  that backdates a statement is rejected because the server
  did not observe it before the rotation.
- Statements signed by the new key are accepted immediately.
- A subsequent rotation of the new key proceeds by the same
  pair-sign-plus-accept protocol.

### 4.5 Voucher key rotation

Voucher keys rotate by the same pair-sign-plus-accept mechanism.
`name_scope` is irrelevant; the rotated voucher's outstanding
vouches remain valid only if they were `first_observed_at` a
trust-log sequence less than the rotation-acceptance sequence.

### 4.6 Racing a rotation under compromise

If the operator suspects the old key is compromised, they
revoke it (§5) *before* accepting any rotation. A revoked key
cannot produce a valid `OldKeyRotationStatement`. If a forged
rotation arrives first and reaches operator acceptance, the
legitimate operator can follow up with a revocation that
supersedes the forged transfer — the lineage map records every
acceptance, so rollback to the pre-forgery state is a single
operator command (`term apps keys rotate --rollback
<seq>`).

---

## 5. Revocation

Author keys and voucher keys have distinct consequence state
machines. Running one into the other was a review blocker;
they are specified separately.

### 5.1 Revoking a key

`term apps keys revoke <key_id> --reason "<text>"` is a
`critical_mutating` operation. It moves the key to state
`revoked`, records a revocation log entry, and runs the
corresponding consequence state machine below **synchronously**
for the immediate protective effects, with any bulk re-vet
work scheduled after.

### 5.2 Author-key revocation — state machine

For every app lineage where the revoked key appears as a
current or most-recent author:

```
  active ──revoke──▶ quarantined-revoked
                       │
                       ├── operator: rotate to new key  ──▶ active (new lineage edge)
                       ├── operator: disable            ──▶ disabled
                       └── operator: uninstall          ──▶ archived
```

`quarantined-revoked` is the default. Until
`plans/quarantine-sandbox.md` lands, v1 implements
`quarantined-revoked` as **`disabled`** (new activations refused,
running activations suspended at the next safe point).
`quarantine-sandbox.md` will later redefine this state to mean
"running under the reduced-permission tier."

Synchronous effects on revocation:

1. Mark every dependent lineage `pending-revet/no-new-activations`
   immediately. This blocks new activations before any async
   work runs.
2. Schedule full re-vet of affected lineages.
3. Log the revocation and affected `app_id` set.

### 5.3 Voucher-key revocation — state machine

A voucher key being revoked does not directly quarantine any
app. Instead:

```
  every install whose Gate 3 admission depended
  on a statement from the revoked voucher
         │
         ├── immediate: mark pending-revet/no-new-activations
         ├── scheduled: re-run Gate 3 against current trust store
         │
         ├── new verdict: pass  ──▶ active (lift pending flag)
         ├── new verdict: warn  ──▶ needs-ack (operator review)
         └── new verdict: block ──▶ disabled (until quarantine-sandbox ships)
```

Voucher revocation never escalates past `disabled` without
operator action. Running activations are suspended only if
Gate 3 re-vet fails.

### 5.4 Compromise response

"Compromise" is not a separate concept from revocation — the
operator revokes with `--reason "compromise"`. The difference:

1. Revoke the compromised key.
2. Force re-vet of every installation that trusted the
   compromised key directly or transitively (author, voucher,
   or rotation predecessor along the lineage).
3. For any lineage where the compromised key was the
   most-recent author, the default consequence floor escalates
   from `quarantined-revoked` to `disabled` (which, pre-
   quarantine-sandbox, are the same thing — but the distinction
   matters once the sandbox lands).
4. The `--races` flag on rotation is refused once a key is in
   state `revoked`, so a compromised key cannot sign its own
   successor after the operator has noticed.

---

## 6. Server Installer and Audit Keys

Distribution's verdict bundles and install log entries need a
server-local signing key. That key has its own lifecycle,
separate from package authors.

### 6.1 Installer key pair

Every server generates and persists one `installer` key pair on
first start. The public key is recorded in `store.toml`; the
private key is kept in a restricted file readable only by the
server process.

- Role: `installer`.
- Scope: signs verdict bundles (distribution §5.8), install
  log entries (§3.3), and verdict-log entries (§6.4). Never
  signs packages or rotation statements.
- Rotation: by operator command
  `term apps keys rotate-installer --new=<key>`. The new
  key is added in state `active`, the old key moves to
  `archived` (not `rotated` — installer keys do not transfer
  via `OldKeyRotationStatement`; they are server-local).
  Future verdict bundles use the new key; historical bundles
  remain verifiable by archived keys.
- Revocation: on compromise, the operator revokes and generates
  a new installer key, then re-signs the most recent verdict-
  log head (§6.4) to re-anchor the chain.

### 6.2 Trust-log and verdict-log binding

`log.ndjson` (trust-store mutations) and `verdicts.ndjson`
(install decisions) are separate hash chains, each signed
per-entry by the current installer key. On installer-key
rotation, the first entry signed by the new key includes the
previous chain head; verifiers walking forward across a
rotation follow the binding.

### 6.3 Compromise response for installer keys

A compromised installer key can, in principle, forge verdict
bundles and trust-log entries *for the future*. The hash chain
makes retroactive rewrites detectable: either the chain breaks
(operator flags at next `term apps keys verify`) or the
attacker has to keep the chain consistent, which requires
knowing every prior entry. Off-site log replication (operator
policy, out of scope for v1) is the mitigation against
silent rewrites.

### 6.4 Verdict log (normative)

This section is the single normative definition of the verdict
log. Distribution and migration plans reference it; they do not
restate the fields.

**Path.** `<server_data>/trust/verdicts.ndjson`. The verdict
bundle file itself lives under
`<server_data>/trust/verdicts/<app_id>/<seq>.json`; the ndjson
record is the tamper-evident index. Retroactive edits to a
stored bundle change its sha256, breaking the chain.

**Entry schema (`verdict-log/1`).** One JSON object per line:

```json
{
  "schema": "verdict-log/1",
  "seq": 112,
  "at": 1714000500,
  "actor": "operator:curt|installer:self",
  "tx_id": "tx:…",
  "app_id": "app:sha256:…",
  "package_id": "sha256:…"|null,
  "decision": "installed"
            | "upgraded"
            | "rolled-back"
            | "uninstalled"
            | "disabled"
            | "enabled"
            | "reconcile-pending"
            | "revet-pass"
            | "revet-warn"
            | "revet-block"
            | "aborted-pre-commit"
            | "aborted-post-commit",
  "verdict_bundle_sha256": "…"|null,
  "prev_hash": "sha256:…",
  "this_hash": "sha256:…",
  "installer_sig": "base64:…"
}
```

Rules:

- `this_hash = sha256(canonical_json({schema, seq, at, actor,
  tx_id, app_id, package_id, decision, verdict_bundle_sha256,
  prev_hash}))`. Canonical JSON is RFC 8785.
- `package_id` is null for decisions that do not name a specific
  package (e.g. `disabled`, `enabled`, standalone `revet-*`).
- `verdict_bundle_sha256` is null for decisions that did not
  produce a verdict bundle (e.g. `enabled` from the disabled
  state). Otherwise it is the sha256 of the bundle file.
- `aborted-pre-commit` records an install transaction that
  rejected before visibility; `aborted-post-commit` records
  rollback after commit. Both still append a chain entry.
- `tx_id` is the install-transaction id (`application-
  distribution.md` §6.a.1). Decisions outside a transaction
  (policy-only mutations, log repair) use `"tx:none"`.
- Every entry is signed by the current installer key; on
  installer-key rotation the first post-rotation entry names the
  new key in the `installer_sig` header fields (out-of-band
  metadata, not repeated here).
- Unknown `schema` strings are rejected at read. Unknown fields
  in a known schema are rejected at read. `v2` readers will
  specify upgrade transforms.

**Verdict bundle file schema (`verdict/1`).** The file under
`trust/verdicts/<app_id>/<seq>.json` is the full gate evidence
referenced by `verdict_bundle_sha256`. Its schema is defined in
[application-distribution.md](application-distribution.md) §5.8;
this plan owns the chain, distribution owns the gate payload.

---

## 7. Operator Surface

All commands are `term apps keys …` (consistent with the
distribution plan's `term apps` namespace). The distribution
plan's operator surface table lists the full set; this
section describes the behaviors unique to trust.

Operations classified as `critical_mutating` (require local
operator confirmation, not proposable by agents via ordinary
approval):

- `term apps keys revoke`
- `term apps keys rotate --accept`
- `term apps keys rotate --emit`
- `term apps keys rotate --rollback`
- `term apps keys rotate-installer`
- `term apps keys confirm --role=author`
- `term apps keys archive <active-author-or-voucher-key>`
  (archiving a key in state `revoked` / `rotated` / `candidate`
  is ordinary `mutating`; archiving an `active` key that could
  authorize future installs is `critical_mutating`)
- `term apps policy set` (for any field referenced in this
  document)
- `term apps enable <app>` when the app's current
  `disabled_reason` is `author-revoked`, `voucher-revet-block`,
  `risk-revet-block`, or `reconcile-pending`. A `disabled` state
  set by operator command or dev-TTL expiry is ordinary
  `mutating` to re-enable.

Ordinary `mutating` (proposable by agents with standard
approval):

- `term apps keys add`
- `term apps keys confirm --role=voucher|publisher`
- `term apps keys archive <non-active-key>`

Read-only:

- `term apps keys ls`, `show`, `log`, `verify`

---

## 8. Minimum v1 Policy Schema

Distribution v1 cannot ship without a concrete policy file.
A full declarative grammar will land in
`plans/distribution-policy-grammar.md` once the surface
stabilizes; the schema below is the v1 minimum and is
authoritative until that plan supersedes it.

File: `<server_data>/trust/policy.toml`.

```toml
policy_schema = "policy/1"
name          = "default"

[gate3]                                   # author / voucher admission
# An install is admitted by gate3 iff ANY of these rules matches.
[[gate3.rule]]
kind        = "trusted_author"            # package has author_key in state "active"
name_filter = "*"                         # glob over manifest_name

[[gate3.rule]]
kind         = "voucher_quorum"
min_count    = 2
min_tier     = "full"                     # voucher scope.tier >= this
testing_in   = ["hardware", "production"]
unique_keys  = true                       # distinct key_ids, not distinct statements
name_filter  = "*"

[[gate3.rule]]
kind        = "quarantine_admit"          # signed by any known key; install goes to quarantine tier
enabled     = false                       # off until quarantine-sandbox ships

[gate5]                                   # AI reviewers
min_active_providers = 2
cooldown_treatment   = "require_revet"    # see distribution plan §5.5
# Active provider set is enumerated below; a change to this set
# (add, remove, model bump, context_scope change, substitute_id)
# increments the policy version.
[[gate5.provider]]
id            = "claude-v1"
model         = "claude-opus-4-7"
context_scope = "public"                  # "public" | "private" — gates SERVER_CONTEXT_PRIVATE
substitute_id = "local-llm-v1"            # used if this provider enters cooldown
[[gate5.provider]]
id            = "codex-v1"
model         = "gpt-5-codex-2026-03"
context_scope = "public"
substitute_id = "local-llm-v1"
[[gate5.provider]]
id            = "local-llm-v1"
model         = "…"
context_scope = "private"
substitute_id = ""                        # terminal fallback; empty substitute_id allowed

[gate7]                                   # risk thresholds
block_above_score = 80
warn_above_score  = 40
[[gate7.weight]]
permission = "telephony"
weight     = 40
[[gate7.weight]]
permission = "http.outbound"
weight     = 30
[[gate7.weight]]
permission = "ai.llm"
weight     = 10

[revoke]
author_default   = "quarantined-revoked"  # implemented as "disabled" until quarantine-sandbox
voucher_default  = "pending-revet"
compromise_floor = "disabled"

[voucher_defaults]                        # used when a voucher key has no ceiling entry
max_tier        = "quarantine"
allowed_testing = ["sim-only"]
max_expiry_days = 14
```

Rules:

- Unknown top-level tables are rejected at load (not ignored).
  Policy authority-bearing surface cannot grow silently.
- `policy_schema = "policy/1"` must match the server's
  supported schema versions; a future `policy/2` will specify
  upgrade transforms.
- A server with no `policy.toml` refuses all non-`--dev`
  installs until one is explicitly accepted.

---

## 9. Interaction with the Vetting Pipeline

A crosswalk for readers of the distribution plan; it does not
re-specify gates.

- **Gate 0 (pre-trust)** uses only
  [package-format.md](package-format.md) rules. No trust store
  is consulted.
- **Gate 3 (author / voucher policy)** consults the trust
  store and §8 policy.
- **Gate 7 (risk)** weighs risk in part by the trust tier
  assigned to the keys that signed the package.
- **Rotation** at install time is transparent to gates — the
  package is signed by the new key, which the trust store
  already knows about.
- **Revocation** synchronously sets
  `pending-revet/no-new-activations` on dependent installs
  (§5.2 / §5.3) regardless of whether async re-vet has run.

---

## 10. Acceptance Criteria

- A server can import an author key, accept a signed package,
  and reject a replayed signature bundle (covered jointly with
  `package-format.md` test vectors).
- Accepting a rotation requires both signed statements plus a
  `critical_mutating` operator command; a rotation signed by
  only the old key is rejected with a specific error.
- A stolen old key that backdates statements cannot land them
  after rotation acceptance — the server-observed log sequence
  is the cutoff, not signer-controlled `created_unix`.
- Revoking an author key immediately marks every dependent
  lineage `pending-revet/no-new-activations`, before any async
  re-vet, and `term apps keys log` shows the synchronous
  effect.
- Revoking a voucher key runs the §5.3 state machine; running
  activations are affected only after re-vet produces a new
  verdict.
- The installer key signs verdict bundles, trust-store entries,
  and verdict-log entries; `term apps keys verify` walks both
  chains and reports any break.
- `policy/1` loads; policy with unknown top-level table is
  rejected.
- An installed `app_id` survives author-key rotation and is
  reused for storage/verdict paths.

---

## Open Questions

- **Trust-store replication.** Multi-server households will
  want a shared trust baseline. Candidate approaches:
  rotation-style signed exports, a read-only peer mirror,
  or a one-way replication channel. Orthogonal to v1.
- **Offline operator confirmations.** `candidate -> active`
  and `critical_mutating` operations require operator input;
  headless servers need either a proxy REPL, a one-shot admin
  command with an out-of-band code, or a pre-authorized
  configuration file.
- **Verdict-log replication.** Same concern as trust-log
  replication; a copy outside the server is the only way to
  detect silent future-direction rewrites by a compromised
  installer key.
