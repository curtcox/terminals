# Signing and Trust

See [masterplan.md](masterplan.md) for overall system context.
Depends on [package-format.md](package-format.md) (canonical
signed statements) and is referenced by
[application-distribution.md](application-distribution.md)
(vetting pipeline) and
[app-migrations.md](app-migrations.md) (migration authority).
Distinct from [identity-and-audience.md](identity-and-audience.md),
which describes *people and audiences* for application logic, not
*signing keys* for the application supply chain.

## Problem

Distribution assumes a trust store, key roles, voucher weights,
storage authority tied to an author key, and a revocation list.
None of that is defined anywhere in the repo today.
`identity-and-audience.md` covers a different layer —
household people and audiences for in-app logic, not the keys
that authorize package installation.

The distribution review called this out as a blocker: the plan
invents a key model inline and never addresses rotation,
revocation of an already-installed app, or compromise response.
This document fixes that gap.

## Design Principles

1. **Keys authorize, identities explain.** A key is the unit of
   authority; display names and descriptions are UX.
2. **Every key has a lifecycle.** Creation, active use,
   rotation, revocation, and archival are explicit states — not
   implicit from whether the key appears in a trust file.
3. **Authority is scoped.** A key's rights are a set of allowed
   roles and, for vouchers, an allowed scope ceiling.
4. **Revocation has observable consequences.** Revoking a key
   triggers defined actions on already-installed apps; it is not
   just a bit that prevents future installs.
5. **Rotation is a first-class operation, not a reinstall.** An
   author who rotates keys can continue to own the same
   `(name, author_key)` storage scope via a signed, auditable
   transition.
6. **The trust store is server-local and auditable.** There is
   no global truth. Every server owns its decisions.

## Non-Goals

- No PKI hierarchy or X.509. Keys are raw Ed25519 by default.
- No remote attestation protocol for trust-store sync — that
  belongs with discovery.
- No policy for humans inside an organization. This document
  describes machine authority, not HR.

---

## 1. Keys

### 1.1 Identity of a key

```
key_id = "ed25519:" + base64url(pubkey_bytes)
```

The `key_id` is the **entire** cryptographic identity a verifier
needs. Display names, owner notes, and contact info are held
separately in the trust store (§3) and are not part of the
`key_id`.

Alternative algorithms are allowed in the future via a new prefix
(`"bls12-381:"`, `"mldsa44:"`) and a statement-schema bump (see
`package-format.md` open questions). This document specifies
Ed25519 concretely; everywhere it says "signature," read "the
algorithm indicated by the `key_id` prefix."

### 1.2 Key roles

A key may hold any subset of these roles in a given trust store:

| Role         | May sign                               | Additional authority                                   |
|--------------|----------------------------------------|--------------------------------------------------------|
| `author`     | `role = "author"` statements           | Owns `(name, author_key)` storage scope for any app it authors. |
| `voucher`    | `role = "voucher"` statements          | Scope ceiling (§2) constrains what it can vouch for.   |
| `publisher`  | `role = "publisher"` statements        | None. Informational.                                   |
| `operator`   | Local trust-store edits and overrides  | Not a package-signature role; listed for completeness. |

A single key MAY hold multiple roles (an author who also vouches
for others, an operator who also authors). Roles are additive.

### 1.3 Key states

Every key known to a server is in exactly one state:

- `candidate` — added to the trust store pending operator
  confirmation; may not authorize anything yet.
- `active` — normal working state.
- `rotated` — superseded by a successor key (§4); accepted only
  for statements `created_unix ≤ rotation_time`.
- `revoked` — rejected unconditionally; triggers §5 consequences.
- `archived` — not trusted for new statements, but historical
  statements remain valid for audit.

Transitions are operator-driven (`apps keys …` — see §6) and
recorded with a signed trust-store log entry (§3.3).

---

## 2. Voucher Scope Ceilings

A voucher key is configured in the trust store with a **scope
ceiling** — the most-permissive scope it is allowed to commit to
in its signed statements. A signed statement whose declared
`scope` exceeds the ceiling is rejected even if the signature is
cryptographically valid.

Example trust-store entry:

```toml
[key."ed25519:abc…"]
roles   = ["voucher"]
state   = "active"
note    = "Staging-server automated reviewer"

[key."ed25519:abc…".voucher_ceiling]
max_tier    = "quarantine"
allowed_testing = ["sim-only"]
max_expiry_days = 30
```

The scope ceiling closes the voucher-laundering path flagged
during review: a voucher created under sim-only conditions
cannot assert `tier = "full"` or `tested_under = "production"`
without the trust store allowing it explicitly.

Ceiling evaluation:

- `max_tier` — the signed `scope.tier` must be no more permissive
  than this. Tier ordering: `quarantine < custom < full`.
- `allowed_testing` — `scope.tested_under` must be in this set.
- `max_expiry_days` — if the signed statement lacks
  `expires_unix`, the server treats it as if expiring at
  `created_unix + max_expiry_days * 86400`.

A voucher without a ceiling entry is treated as `max_tier =
"quarantine"`, `allowed_testing = ["sim-only"]`, `max_expiry_days
= 14`. New unknown vouchers start weak.

---

## 3. Trust Store

### 3.1 Contents

A server's trust store is a versioned TOML file plus an
append-only log. It holds:

- **Keys.** One entry per `key_id` with state, roles, and
  (for voucher keys) scope ceiling.
- **Revocation list.** Revoked `key_id`s with revocation time
  and reason.
- **Rotation records.** See §4.
- **Policies.** Named, declarative rules the vetting pipeline
  consults (e.g., "accept unknown author if ≥2 trusted vouchers
  at `tier ≥ full`"). Policy language is specified in the
  distribution plan's Gate 3.

### 3.2 Location and authority

- Primary file: `<server_data>/trust/store.toml`.
- Append-only log: `<server_data>/trust/log.ndjson`.
- Mutations go through `apps keys …` / `apps policy …` REPL
  commands; direct edits are detected by log-chaining and
  flagged.

### 3.3 Log chain

Every mutation appends one line to `log.ndjson`:

```json
{"seq": 42, "at": 1714000000, "actor": "operator:curt", "op": "keys.rotate", "args": {...}, "prev_hash": "sha256:…", "this_hash": "sha256:…"}
```

`prev_hash` chains each entry to the previous one's `this_hash`.
A gap or mismatch invalidates the store until an operator
reconciles it. This gives after-the-fact evidence of tampering
without preventing operator edits.

---

## 4. Key Rotation

Rotation lets an author continue to own an app's storage scope
after replacing their signing key, without re-consenting every
downstream server per install.

### 4.1 Rotation statement

The outgoing author publishes a rotation statement signed by the
*old* key:

```
RotationStatement (CBOR, deterministic):
  schema_version : 1
  old_key        : "ed25519:…"
  new_key        : "ed25519:…"
  rotation_time  : <unix seconds>
  name_scope     : ["kitchen_timer", "coffee_timer"]  # app names covered
  reason         : <tstr, optional>
  sig            : signature by old_key over the above
```

`name_scope` is the explicit list of application names whose
`(name, author_key)` ownership transfers. A rotation that omits
`name_scope` (or uses `["*"]`) is permitted only if the trust
store's policy allows wildcard rotation for that key.

Rotation statements travel out-of-band relative to any single
package — they are trust-store records, not `.tap.sig` content.
They are delivered to a server through `apps keys rotate --accept
<statement>` (operator-acknowledged) or through the discovery
layer when the rotation is signed by a key the server already
trusts for rotation.

### 4.2 Effects

When a server accepts a rotation:

1. The old key's state moves to `rotated` with
   `rotation_time = <statement time>`.
2. The new key is added in state `active` with the same roles.
3. For every listed app name, ownership of
   `(name, author_key)` storage scope transitions to `new_key`.
   The old `(name, old_key)` tuple remains a valid *historical*
   owner for audit but can no longer install upgrades.
4. Future installs of the same app name must be signed by the
   new key; packages signed by the old key after `rotation_time`
   are rejected with a specific error ("signed by rotated key
   after rotation_time").

Rotation is an upgrade event for the app: the next install on
the new key runs the full vetting pipeline and any required
migrations.

### 4.3 Rotation of voucher keys

Voucher keys rotate by the same mechanism, but `name_scope` is
irrelevant. A rotated voucher's outstanding vouches remain valid
if created before `rotation_time`. New vouches must be signed by
the new key.

---

## 5. Revocation

### 5.1 Revoking a key

`apps keys revoke <key_id> --reason "<text>"` moves the key to
state `revoked` and records a revocation entry.

### 5.2 Effect on already-installed apps

Revocation is not only a forward-facing filter. For every
installed app whose author key is revoked, the server takes one
of these actions per policy:

- **`quarantine`** (default). Running activations are allowed to
  finish, no new activations are started, and the app is flagged
  in `apps ls` as `quarantined-revoked`. An operator may review
  and either re-sign (via rotation) or uninstall.
- **`disable`.** Suspend running activations and refuse new
  ones. Data is retained per the data-retention policy (see
  distribution plan's follow-on).
- **`uninstall`.** Treat the revocation as an implicit
  `apps uninstall --archive-data`.

Default policy is `quarantine`; server operators may raise the
floor per risk tier. Revocation consequences run inside a single
transaction so a crash cannot leave the app partly-revoked.

### 5.3 Effect on vouches

Revoking a voucher key invalidates all of its outstanding
vouches. The distribution pipeline automatically re-vets every
installed app whose acceptance depended on a vouch from the
revoked key (see distribution plan §6). If the re-vet fails,
§5.2 actions apply.

### 5.4 Compromise response

"Compromise" is not a separate concept from revocation — the
operator revokes the key with `--reason "compromise"`. The
difference is procedural:

1. Revoke compromised key.
2. Force re-vet of every installation that trusted the
   compromised key directly or transitively (as author,
   voucher, or rotation predecessor).
3. For any app where the compromised key was the author on the
   *most recent* install, default action escalates from
   `quarantine` to `disable` regardless of per-app policy.
4. A new key may then be published and accepted by operators
   manually — a compromised key cannot sign a rotation statement
   pointing at its successor, by construction.

---

## 6. Operator Surface

All commands are available over MCP to Claude and Codex with
mutating approval. The distribution plan's `apps keys` commands
expand to:

```text
# inspection
apps keys ls [--role=…] [--state=…]
apps keys show <key_id>

# additions
apps keys add <key_id> --role=author [--note="…"]
apps keys add <key_id> --role=voucher --ceiling-tier=quarantine \
    --ceiling-testing=sim-only --ceiling-expiry-days=30 [--note="…"]

# state transitions
apps keys confirm <key_id>                  # candidate -> active
apps keys revoke  <key_id> --reason="…" [--on-installed=quarantine|disable|uninstall]
apps keys archive <key_id>

# rotation
apps keys rotate --accept <rotation_statement_file>
apps keys rotate --emit  --old=<key_id> --new=<key_id> --names=kitchen_timer,coffee_timer

# audit
apps keys log [--since=…] [--actor=…]
apps keys verify                            # walk the log chain
```

`apps policy` commands live alongside and are covered in the
distribution plan.

---

## 7. Interaction with the Vetting Pipeline

This section is a crosswalk for readers of the distribution plan;
it does not re-specify gates.

- **Gate 2 (signatures)** uses only
  [package-format.md](package-format.md) rules. No trust store
  is consulted yet.
- **Gate 3 (author / voucher policy)** consults the trust store
  built here: key state, roles, voucher ceilings, and named
  policies.
- **Gate 7 (risk)** weighs risk in part by the trust tier
  assigned to the keys that signed the package.
- **Rotation** at install time is transparent to gates — the
  package is signed by the new key, which the trust store
  already knows about. Nothing special happens at Gate 3.
- **Revocation** bypasses gates entirely: the scheduled
  consequences of §5.2 run immediately on revocation, regardless
  of whether the app would otherwise pass re-vet.

---

## 8. Acceptance Criteria

- A server can import an author key, accept a signed package,
  and reject a replayed signature bundle — covered by
  `package-format.md` test vectors plus trust-store state.
- Rotating an author key preserves `(name, author_key)` storage
  scope exactly when the rotation statement names the app;
  storage scope does not transfer when the rotation omits it.
- Revoking an author key quarantines that author's installed
  apps within one REPL command, with a log entry describing
  every affected app.
- Re-vetting every installed app after a voucher revocation
  terminates in bounded time and produces fresh verdicts.
- Log chain verification (`apps keys verify`) detects any
  out-of-band edit to `store.toml` or `log.ndjson`.

---

## Open Questions

- **Trust-store replication.** Multi-server households will want
  a shared trust baseline. Sync is orthogonal to this spec but
  will need a design; candidate approaches are rotation-style
  signed exports or a read-only peer mirror.
- **Offline operator confirmations.** `candidate -> active`
  requires operator input; what does that look like on a
  headless server? Either proxy through a REPL over another
  terminal, or ship a one-shot admin command with an out-of-band
  code.
- **Quorum vouches.** Some deployments may want to require M-of-N
  vouchers from specific keys for a given tier. Policy language
  in the distribution plan needs to express that; this document
  already carries the primitives (ceilings + named policies).
