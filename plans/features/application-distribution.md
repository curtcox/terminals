---
title: "Application Distribution"
kind: plan
status: planned
owner: unowned
validation: none
last-reviewed: 2026-04-25
---

# Application Distribution

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.
Extends [application-runtime.md](application-runtime.md) (TAR/TAL
package format and lifecycle) and
[repl-capability-plan.md](repl-capability/plan.md) (authoring
substrate). Depends on [package-format.md](package-format/plan.md)
(canonical `.tap`, signed statements, `verify_package`
pipeline), [signing-and-trust.md](signing-and-trust.md) (keys,
trust store, installer key, `app_id` lineage, revocation,
rotation, policy schema), and [app-migrations.md](app-migrations/plan.md)
(migration executor, drain, reconciliation). Related:
[scenario-engine.md](scenario-engine.md),
[shared-artifacts.md](shared-artifacts.md),
[capability-lifecycle.md](capability-lifecycle.md),
[identity-and-audience.md](identity-and-audience.md).

Worked example used throughout:
[docs/tal-example-kitchen-timer.md](../../docs/tal-example-kitchen-timer.md).

## Problem

[application-runtime.md](application-runtime.md) defines *what*
an app package is and *how* it runs. It does not say how apps
come into existence, move between servers, or get loaded safely
on a server that did not author them.

This plan covers the end-to-end supply chain:

1. **Author** — develop an app on one Terminals server with
   the REPL, Claude, and Codex.
2. **Publish** — produce a signed, versioned, portable package.
3. **Discover** — a second server finds the package. Discovery
   itself is orthogonal and pluggable.
4. **Vet** — the receiving server runs an ordered pipeline of
   independent gates, starting from a pre-trust canonicalization
   gate (Gate 0) that runs before anything reads the manifest.
5. **Install** — TAR registers the app, provisions durable
   storage, and activates it, all inside an install transaction
   that revalidates trust and registry epochs at commit.
6. **Evolve** — upgrade with version-pinned activations,
   rollback, uninstall, and automatic re-vetting as conditions
   change.

Three sibling plans own the load-bearing mechanics so this plan
stays focused on orchestration:

- [package-format.md](package-format/plan.md) owns the `.tap` + signed
  statement wire format and the `verify_package` pre-trust
  pipeline.
- [signing-and-trust.md](signing-and-trust.md) owns keys,
  `app_id` lineage, voucher scope ceilings, rotation,
  revocation, the installer key, the verdict-log hash chain,
  the `critical_mutating` operation tier, and the v1 policy
  schema.
- [app-migrations.md](app-migrations/plan.md) owns the migration
  executor, drain semantics, the `reconcile_pending` state,
  and the artifact-patch boundary.

This document references them rather than re-deriving their
contracts.

## Design Principles

1. **The server is always in charge of its own load order.** No
   remote party can cause an app to be loaded, only *offered*.
   Discovery, transport, and author reputation never grant load
   authority.
2. **Canonicalize before trusting.** Untrusted bytes are
   canonicalized and content-addressed before any trust,
   policy, or reviewer reads them. This is Gate 0 (§5.0) and
   it has no operator override.
3. **Vetting is layered, independent, and explicit.** Any single
   gate can block installation. Gates produce machine-readable
   verdicts that are persisted alongside the installed package
   in a hash-chained, installer-signed verdict log.
4. **Signed statements, not signed hashes.** Authority over a
   package is always tied to a fully-qualified signed statement
   per [package-format.md](package-format/plan.md).
5. **Activation version pinning is preserved.** Upgrades never
   migrate live activations to a new version — runtime already
   specifies that existing activations stay pinned to the
   version that created them. Incompatible migrations require
   drain first, per [app-migrations.md](app-migrations/plan.md) §3.1.
6. **Installed apps can be re-vetted.** Trust state, policies,
   installed-app topology, and terminal capabilities all
   change; the server re-vets installs on material changes, not
   only at install time. Re-vets are keyed by `app_id` lineage,
   so key rotation does not lose verdict history.
7. **Claude and Codex are reviewers and authors, not signers.**
   AI analysis is input to a human or policy decision, never
   the decision itself. All reviewer output is untrusted text
   with at most enumerated machine-actionable fields.
8. **Every change to state that could broaden authority is
   `critical_mutating`.** Voucher add, key revoke, policy set,
   migration abort-to-baseline, and reconciliation resolution
   are not approvable by AI agents under ordinary mutating
   approval. See [signing-and-trust.md](signing-and-trust.md)
   §7.

## Non-Goals

- No design for the discovery layer (public registry, gossip,
  QR hand-off, Git remotes). Discovery plugs into the
  interfaces defined here.
- No app-store economics (payment, licensing, DRM).
- No quarantine sandbox specification. v1 treats the
  `quarantined` trust level as **disabled with data retained**
  (see §6.4) because the sandbox is the subject of a follow-on
  plan. An operator that wants to run something they do not
  trust must either wait for the sandbox or explicitly promote
  trust.
- No data retention policy for app-owned data beyond uninstall
  modes — fully defined in a follow-on plan.
- No cross-server *running* activations. Sharing transfers
  packages, not live state.
- No full policy grammar beyond the v1 minimum schema
  referenced in §5.9 — that belongs to a follow-on grammar RFC.

---

## 1. Authoring on a Terminals Server

A Terminals server is a self-sufficient development environment
for TAL apps. The author never leaves the system to write, test,
or ship an app — every step is a REPL command or an AI-assisted
edit through the MCP adapter.

### 1.1 Command namespaces

Two REPL namespaces split the authoring and registry concerns
cleanly, and both operators and AI agents must recognize the
split:

- `term app <verb> <name>` — per-package operations on one
  named app (inherited from
  [application-runtime.md](application-runtime.md)): `new`,
  `check`, `test`, `load`, `reload`, `logs`, `trace`, `pack`,
  `sign`, `rollback`.
- `term apps <verb>` — registry-wide distribution operations
  that span packages or installs: sources, offers, fetches,
  vet/install/upgrade, keys, policy, migration control,
  reconciliation.

The namespaces are different words on purpose (singular vs.
plural). Operators and agents must not improvise between them.

### 1.2 The authoring inner loop

| Area             | Commands                                            |
|------------------|-----------------------------------------------------|
| Package scaffold | `term app new <name>`                               |
| Validate         | `term app check <name>`                             |
| Simulate         | `term sim run <name>`, `term sim trigger/advance`   |
| Activate locally | `term app load <name>`, `term app reload <name>`    |
| Inspect          | `term activations ls`, `term observe tail`          |
| Package          | `term app pack <name>`, `term app sign <name>`      |

Kitchen-timer runs end-to-end under `term sim run kitchen_timer`
with synthetic triggers; no real device or speaker is needed.

### 1.3 `--dev` installs and their limits

`term app load` supports dev-mode installs for authoring. Dev
installs:

- skip Gates 3 (author/voucher policy), 5 (AI-assisted review),
  6 (conflict/redundancy), and 7 (risk). They still run Gate 0
  (canonical format), Gate 1 (manifest), and Gate 4 (static
  analysis) so a malformed app fails fast.
- refuse permissions from an explicit `dev_dangerous` set:
  `telephony`, `http.outbound`, `pty`, `ai.llm` with external
  providers, `bus.emit` into reserved namespaces, and any
  permission that would broadcast beyond the author's own
  devices. Attempting to load with any of these causes dev
  install to fail; the operator is told to use the full
  pipeline.
- are scoped to the current operator session's identity and
  decay after a configurable TTL (default 24 hours). A
  surviving dev install after TTL is `disabled` on next server
  start (per §6.4; not quarantined, since the sandbox is
  deferred).
- are tagged visibly in `term apps ls` as `dev` with the
  operator's identity attached.
- use a synthetic `app_id` derived from the dev session, so they
  cannot collide with or accidentally migrate data owned by a
  later production install.

Dev installs are a development affordance, not a trust bypass.
An authoring agent (Claude or Codex over MCP) can *propose* a
dev install; the operator must confirm.

### 1.4 Claude and Codex as authoring assistants

Both Claude Code and Codex connect to the server through the
Terminals MCP adapter (see `docs/repl/agents/claude-code-setup.md`
and `docs/repl/agents/codex-setup.md`). Through MCP they can:

- call every `read_only` REPL command directly,
- propose `mutating` REPL actions that prompt the operator,
- read TAL source, tests, and manifests in
  `terminal_server/apps/`,
- run `term sim run` and read structured test output.

They cannot approve their own `critical_mutating` operations
(keys revoke, policy set, migration abort-to-baseline,
reconciliation resolution). See
[signing-and-trust.md](signing-and-trust.md) §7.

A typical authoring session:

1. The author states an intent in natural language.
2. The agent inspects existing apps and TAL host modules to
   reuse patterns (kitchen-timer is the canonical example).
3. The agent drafts `manifest.toml`, `main.tal`, and
   `tests/*.tal`.
4. The agent runs `term sim run` through MCP and iterates on
   failures.
5. The agent proposes `term app load --dev`; the operator
   confirms.

The agents are on *both* sides of the supply chain: at
authoring time they help the author satisfy the checks the
installing server will later run; at install time a *separate*
agent invocation runs Gate 5 adversarially. The author's agent
is advisory; the receiver's agent is adversarial and its output
is treated as untrusted text (§5.5).

### 1.5 Claude and Codex as runtime providers

[application-runtime.md](application-runtime.md) defines three
TAL-visible AI permissions: `ai.stt`, `ai.tts`, `ai.llm`. Claude
and Codex register with the kernel's AI adapter as providers
for `ai.llm`; Claude additionally for `ai.tts` where the
deployment uses its voice. Provider selection is a server-policy
decision; an app that declares `permissions = ["ai.llm"]` gets
whatever provider the server has configured for that family,
and swapping Claude for a local model never requires an app
update.

Gate 5 (§5.5) invokes the same providers through a *kernel-
internal* call that is not exposed as a TAL permission. Apps
cannot call the reviewer surface; only TAR and the distribution
pipeline can.

---

## 2. Package Format and Versioning

Package file format, signature bundle, canonicalization, and
signed-statement schema are all specified in
[package-format.md](package-format/plan.md). This plan references
that format but does not restate it.

Summary:

- `.tap` = canonical-zstd(canonical POSIX ustar) of the source
  directory. Both frame and tar are pinned to a deterministic
  encoding; `verify_package` (Gate 0) rejects anything else.
- `package_id = sha256(<tap bytes>)` — the only identifier that
  crosses server boundaries.
- `<name>-<version>.tap.sig` is an append-only TOML bundle of
  `[[statement]]` blocks. Each statement is a deterministically-
  encoded CBOR map binding role, key, scope, timestamps,
  `manifest_name`, `manifest_version`, a required random nonce,
  and the `package_id`. Signatures cover that full statement,
  not the hash alone. Roles: `author`, `voucher`, `publisher`.
  v1 signing is Ed25519-only; non-`ed25519:` key prefixes are
  rejected at Gate 0.

### 2.1 Version scheme

Three distinct numbers, each with its own semantics:

| Field                  | Meaning                                                 |
|------------------------|---------------------------------------------------------|
| `version`              | Semver of the app itself. Breaking change = major bump. |
| `language = "tal/1"`   | TAL dialect. Breaking dialect change = major bump.      |
| `requires_kernel_api`  | Host module contract (`"1.x"` means ≥1.0 <2.0).         |

Rules:

- `version` is **monotonic per `app_id`** (see
  [signing-and-trust.md](signing-and-trust.md) §1.4). The
  receiving server refuses to install a lower version over a
  higher one without explicit `--allow-downgrade`.
- A major bump in `version` MAY require a durable-data
  migration (see [app-migrations.md](app-migrations/plan.md)). The
  manifest declares migrations and their compatibility /
  drain_policy; the executor enforces them.
- `requires_kernel_api` is a hard gate. If the installing
  server cannot satisfy the range, the package is rejected at
  Gate 1 with a structured error.

### 2.2 What changes across a version bump

- **Source and assets** travel in the `.tap`.
- **Manifest permissions** may grow or shrink. A permission
  *added* in a new version re-triggers Gate 3 policy (may the
  author claim this new permission?) and Gate 7 risk analysis.
- **Declared migrations** determine whether an upgrade touches
  durable data, requires drain, or can run in-place
  (compatible + none).

### 2.3 Schema versioning

Every persisted envelope this plan references carries a
`schema` string:

| Schema            | Owned by                                   | Purpose                                         |
|-------------------|--------------------------------------------|-------------------------------------------------|
| `tap-sig/1`       | [package-format.md](package-format/plan.md) §2  | Signature bundle outer TOML.                    |
| `rotation-stmt/1` | [signing-and-trust.md](signing-and-trust.md) §4.1 | Pair-signed rotation statements (CBOR).  |
| `policy/1`        | [signing-and-trust.md](signing-and-trust.md) §8   | Trust / vetting policy file.              |
| `verdict-log/1`   | [signing-and-trust.md](signing-and-trust.md) §6.4 | Verdict-log ndjson entries.               |
| `verdict/1`       | this plan §5.8                             | Per-install verdict bundle JSON file.          |
| `install-tx/1`    | this plan §6.a.5                           | Install transaction journal entries.            |
| `drain-intent/1`  | [app-migrations.md](app-migrations/plan.md) §3.1.1 | Drain journal entries (§6.5).              |

v1 readers reject unknown schema strings at parse time, and
reject unknown fields within a known schema. v1 writers only
emit these strings. Upgrades to a `/2` schema require a
co-ordinated plan change and a migration, not a silent bump.

---

## 3. Persistence

Four distinct durable surfaces interact with an app. Each has
its own owner and its own survival rules across reinstall and
upgrade. All server-local paths below are keyed by `app_id`
(the lineage-stable identifier from
[signing-and-trust.md](signing-and-trust.md) §1.4), never by
manifest name — this prevents name-squatting and keeps data
associated with the original author lineage across rotation.

### 3.1 Activation state

Already specified in
[application-runtime.md](application-runtime.md):

- Lives inside the activation's state dict; JSON-serializable.
- Snapshotted on every successful commit.
- Restored by `resume` after suspend or crash.
- Discarded when the activation terminates.

Runtime pins every activation to the version of the package
that created it. Upgrades and uninstalls do not migrate live
activations to a new version; see §6.1.

### 3.2 App-scoped stores

Declared in `manifest.toml`:

```toml
[storage]
artifact_kinds = ["note", "checklist"]
stores         = ["preferences", "history"]
```

- Private KV namespaces, one per declared store.
- Scoping key is `app_id`. Two packages with the same manifest
  name but distinct lineages get distinct `app_id`s and
  therefore distinct store namespaces.

### 3.3 App-authored artifacts

Artifacts (see [shared-artifacts.md](shared-artifacts.md)) are
identity-owned, cross-referenceable, and outlive any one app
install. An app may read, create, and annotate artifacts per
its permissions. Artifacts created by an app record
`owner_app_id` so migration and uninstall decisions stay bound
to lineage, not to the current author key.

Uninstall offers three data modes:

- `--keep-data` (default). Stores are retained in-place;
  authored artifacts are retained and remain reachable by
  their owners.
- `--archive-data`. Stores are snapshotted to an archive file;
  authored artifacts are retained but marked as "orphaned by
  app" in `artifact.list`.
- `--purge`. Stores are deleted; *authored artifacts are not
  deleted* — they belong to their identity owners. The
  follow-on `plans/data-retention.md` specifies owner
  notification and safe deletion.

### 3.4 Distribution registry (server-local)

Installing and managing apps produces its own durable state,
owned by the distribution subsystem rather than by any app:

- **Installed-app records.** Manifest, `package_id`, signed-
  statement bundle, trust snapshot at install time, `app_id`.
- **Verdict bundles.** Every gate's output, the verbatim AI
  review prompts and responses, and the final decision —
  signed by the installing server's installer key (per
  [signing-and-trust.md](signing-and-trust.md) §6) and
  appended to the hash-chained verdict log (§6.a).
- **Staging directory.** Fetched but not-yet-installed
  packages, garbage-collected after a TTL.
- **Archive.** Retired `.tap`, `.tap.sig`, verdict bundles,
  and (for `--archive-data`) store snapshots.
- **Trust store and log chain.** Owned jointly with
  [signing-and-trust.md](signing-and-trust.md) but physically
  co-located under `<server_data>/trust/`.

Paths. All occurrences of `<app_id>` are the full
`"app:sha256:<hex>"` textual form specified in
[signing-and-trust.md](signing-and-trust.md) §1.4.

```
<server_data>/
├── apps/
│   ├── <app_id>/                     # live package directory
│   ├── <app_id>.tap                  # installed source
│   ├── <app_id>/stores/              # §3.2
│   ├── <app_id>/drain/intents.ndjson # §6.5 (schema drain-intent/1)
│   ├── <app_id>/migrate/<run_id>/    # see app-migrations.md §3.3
│   └── <app_id>/install-tx/<tx_id>/  # §6.a.5 (schema install-tx/1)
├── staging/
├── archive/
│   └── <app_id>-<version>-<timestamp>/
└── trust/                            # verdict log, verdict bundles,
                                      # trust store, policy — see
                                      # signing-and-trust.md §3.2, §6.4
```

Verdict bundles and the verdict log live under `trust/`, not
`verdicts/`, to keep all installer-signed artifacts under one
tree. `name_index/` (an advisory manifest-name → app_id alias
table) lives under `trust/name_index/`.

The verdict bundle file schema is defined in §5.8. The
verdict log entry schema and path are normative in
[signing-and-trust.md](signing-and-trust.md) §6.4; this plan
does not restate them.

---

## 4. Sharing Between Servers

Sharing transfers `.tap` + `.tap.sig` pairs between servers.
How they are advertised, fetched, or mirrored is discovery,
which this plan leaves pluggable — but the *peer* discovery
case runs over the existing gRPC transport and needs a typed
contract.

### 4.1 The `PackageSource` interface

```go
type PackageSource interface {
    List(ctx context.Context, q PackageQuery) ([]PackageHandle, error)
    Fetch(ctx context.Context, h PackageHandle) (Package, error)
    Describe() SourceInfo
}
```

Built-in implementations:

- **`file`** — local directory of `.tap` files.
- **`peer`** — another Terminals server reachable over the
  existing control plane; schema in §4.2.

Additional sources (public registry, gossip over the bus,
Git remotes, USB-stick workflows) are future `PackageSource`
implementations; none of them change the rest of this document.

### 4.2 Peer source protobuf

Servers expose a read-only catalog service on the existing gRPC
control plane:

```protobuf
service TerminalCatalog {
  rpc ListPackages(ListPackagesRequest) returns (ListPackagesResponse);
  rpc GetPackageMeta(GetPackageMetaRequest) returns (PackageMeta);
  rpc FetchPackage(FetchPackageRequest) returns (stream PackageChunk);
  rpc FetchSignatures(FetchSignaturesRequest) returns (SignatureBundle);
}

message ListPackagesRequest {
  string name_prefix     = 1;
  string language        = 2;        // e.g. "tal/1"
  string page_token      = 3;
  uint32 page_size       = 4;
}

message ListPackagesResponse {
  repeated PackageMeta items = 1;
  string next_page_token     = 2;
}

message PackageMeta {
  string package_id     = 1;         // "sha256:…"
  string name           = 2;
  string version        = 3;
  string language       = 4;
  string kernel_api     = 5;         // requires_kernel_api
  uint64 size_bytes     = 6;
  repeated string permissions = 7;
  uint32 statement_count = 8;        // hint; real bundle via FetchSignatures
}

message FetchPackageRequest { string package_id = 1; }

message PackageChunk {
  bytes data          = 1;
  uint64 offset       = 2;
  bool final          = 3;
}

message FetchSignaturesRequest { string package_id = 1; }

message SignatureBundle { bytes toml = 1; }

enum CatalogError {
  CATALOG_OK              = 0;
  CATALOG_NOT_FOUND       = 1;
  CATALOG_UNAUTHORIZED    = 2;
  CATALOG_SIZE_LIMIT      = 3;
  CATALOG_RATE_LIMITED    = 4;
  CATALOG_INTEGRITY_FAIL  = 5;
}
```

Rules:

- `FetchPackage` streams chunks; the client computes `sha256`
  across the stream and compares it to the requested
  `package_id`. Mismatch returns `CATALOG_INTEGRITY_FAIL` and
  discards bytes.
- Size limits, rate limits, and authorization are operator-
  configurable per peer source and surfaced through the error
  enum rather than transport-level signals.
- `SignatureBundle.toml` is the raw bundle; the client does not
  trust it until [package-format.md](package-format/plan.md)
  verification (Gate 0) passes. Parse quotas (1 MiB file, 64
  statements, 8 KiB strings, depth 16) from
  [package-format.md](package-format/plan.md) §3 apply.

### 4.3 Peer catalog MITM defense

A `peer` source discovers packages by `name`, but `name` is
attacker-controlled: a compromised peer (or a peer-routing
MITM) can advertise `"kitchen_timer 9.9.9"` signed by a key
the operator once trusted for some other app. The installing
server must never install a package just because the name
matches something it has heard of.

The `peer` source MUST therefore pin both:

1. The expected `package_id` of the specific version, OR
2. The expected `app_id` AND an expected author `key_id` that
   the operator already trusts for this `app_id`.

The source configuration captures this:

```toml
[[source]]
name        = "server-a-lan"
kind        = "peer"
endpoint    = "servera.local:7443"
expected_ca = "fingerprint:…"         # TLS pin, orthogonal
pins = [
  { app_id = "app:…", author_key = "ed25519:…" },
]
```

`term apps offer <source>` filters the advertised list by the
pins. An advertised package that does not match a pin is
visible with a warning but cannot be `term apps fetch`-ed
without `--unpinned` (interactive confirmation, logged in the
trust log). `--unpinned` is `critical_mutating`.

Pins are operator-entered out of band (QR code, paper,
pre-shared file). This plan does not specify pin distribution;
it specifies the enforcement.

### 4.4 Offer-only semantics

A `PackageSource` can only *offer* packages. The install
command takes a handle from a source and runs the pipeline
in §5:

```text
term apps offer <source> [--query=…]                 # list candidates
term apps fetch <source> <handle> [--unpinned]       # --unpinned is †; bypasses §4.3 pins
term apps vet   <staged>                             # run gates, produce report
term apps install <staged> [--voucher=…] [--allow-warn]
```

`term apps fetch` writes to the staging directory and never
modifies the live registry. Staged packages are garbage-
collected if not installed within a configurable TTL.

---


Normative detail for **§5–§8** (install-time vetting through upgrade, rollback, re-vetting, and operator surface) lives in [application-distribution-install-vetting.md](application-distribution-install-vetting.md).

## 9. Worked Example (Kitchen Timer, End-to-End)

Using [docs/tal-example-kitchen-timer.md](../../docs/tal-example-kitchen-timer.md):

1. **Author on Server A.**
   - Operator: "build me a kitchen timer" via Claude over MCP.
   - Agent scaffolds `kitchen_timer/` with `manifest.toml`,
     `main.tal`, and `tests/kitchen_timer_test.tal`.
   - `term sim run kitchen_timer` passes.
   - `term app load --dev kitchen_timer` registers it locally
     (dev mode; `ai.tts` on a kitchen speaker is in the dev-
     allowed set; `telephony` would not be).
   - Real voice trigger on a kitchen tab verifies end-to-end.

2. **Publish on Server A.**
   - `term app pack kitchen_timer` →
     `kitchen_timer-0.1.0.tap` (canonical per
     [package-format.md](package-format/plan.md)).
   - `term app sign kitchen_timer --role=author` →
     `kitchen_timer-0.1.0.tap.sig` with the operator's author
     key. The package's `app_id` is established at first author
     signing.

3. **Discover on Server B.**
   - Server B has a `file` source or a `peer` source for
     Server A. If `peer`, the source is pinned to Server A's
     `app_id` + author `key_id` per §4.3.
   - `term apps offer file --query=kitchen_timer*` lists the
     candidate; `term apps fetch` stages it.

4. **Vet on Server B.**
   - Gate 0 (canonical format): zstd frame, tar layout,
     statement CBOR + Ed25519 signatures, `package_id`
     computed → `pass`.
   - Gate 1 (manifest): `ai.tts`, `ui.*`, `scheduler`,
     `bus.emit`, `placement.read` all known → `pass`.
   - Gate 2 (package-format extended): manifest name/version
     bindings in every statement match → `pass`.
   - Gate 3 (author/voucher): Server A's author key was
     imported via `term apps keys add ed25519:… --role=author`;
     policy accepts → `pass`. (Alternative: a Server B
     reviewer signs a voucher after reading the code; voucher
     scope ceiling satisfied.)
   - Gate 4 (static): all `load(…)`s covered; `match` uses
     declarative equality / membership on `req.kind` and
     `req.action` → `pass`. No `migrate/` shipped.
   - Gate 5 (AI review): Claude and Codex both summarize
     "one-activation timer that patches a UI, speaks on
     expiry, and emits `timer.expired` on the bus"; no
     prompt-injection findings; no permission mismatch →
     `pass`. Provider set, model IDs, and prompt template
     SHA-256 recorded in the verdict bundle.
   - Gate 6 (conflict): no other app accepts `timer.set` /
     `timer.start` → `pass`.
   - Gate 7 (risk): `ai.tts` on shared kitchen speaker
     flagged → `warn`; operator acks.

5. **Install on Server B.**
   - `term apps install kitchen_timer-0.1.0.tap --allow-warn`
     enters the install transaction: per-app lock acquired,
     trust/registry/capability epochs recorded in
     `epochs_at_vet`. `prepare_new` unpacks the canonical tar
     into a staging directory, then atomically renames it to
     `apps/<app_id>/versions/<package_id>/` (immutable). At
     `commit`, the journal commit entry is fsynced, the
     verdict bundle file is written under
     `<server_data>/trust/verdicts/<app_id>/<seq>.json`, the
     `verdict-log/1` entry is appended to
     `<server_data>/trust/verdicts.ndjson` (installer-signed,
     hash-chained), and finally `apps/<app_id>/current` is
     atomically flipped via `rename(2)` to point at
     `versions/<package_id>/`. The scenario engine re-reads
     `current` and the app is live. No stores provisioned
     (none declared).

6. **Upgrade to 0.2.0 (snoozing).**
   - Server A ships 0.2.0. Server B re-discovers, re-fetches,
     re-vets. Because `app_id` matches, this is an upgrade. No
     migration declared; no durable-data work. The install
     transaction re-checks trust, registry, and capability
     epochs at commit; all unchanged → proceeds. A fresh
     immutable `versions/<package_id_v2>/` is written, then
     `current` is re-pointed.
   - Activations running on 0.1.0 continue under the old
     definitions (still loaded from the 0.1.0 version
     directory); new activations resolve `current` to 0.2.0.
     `term apps ls` shows `kitchen_timer 0.2.0 (also 0.1.0
     draining: 2 activations)`.
   - Ten minutes later the last 0.1.0 activation expires; 0.1.0
     definitions unload. The 0.1.0 version directory stays on
     disk (immutable, re-usable by rollback) until archive.

7. **Author key rotation on Server A.**
   - Server A's operator rotates the author key per
     [signing-and-trust.md](signing-and-trust.md) §4
     (pair-signed rotation + operator acceptance on Server B).
   - Server B accepts the rotation statement with
     `term apps keys rotate --accept <file>` (a
     `critical_mutating` operation).
   - `kitchen_timer`'s `app_id` is unchanged; next upgrade's
     packages signed by the new key install normally and reuse
     the same `apps/<app_id>/...` tree.

8. **Voucher revoked on Server B.**
   - Suppose the install had relied on a voucher rather than a
     trusted author. `term apps keys revoke
     ed25519:voucher-…` (a `critical_mutating` op) fires an
     auto-re-vet of every install that depended on that
     voucher. `kitchen_timer` re-runs Gate 3; if the remaining
     voucher set no longer meets quorum, the install moves to
     `disabled` (§6.4) until another voucher is added or the
     author key becomes trusted directly.

---

## Follow-On Plans

- **`plans/quarantine-sandbox.md`.** Specifies the reduced-
  permission runtime tier referenced in §5.3 and §6.4: exact
  permission subset, promotion/demotion rules, lifetime, and
  how activations observe the difference. Amends
  [application-runtime.md](application-runtime.md). Until it
  ships, v1 treats "quarantined" as "disabled, data retained."
- **`plans/data-retention.md`.** Specifies retention classes,
  owner notification, legal holds, and safe deletion for
  app-authored artifacts and archived stores. Extends §3 and
  §6.3 beyond the three uninstall modes.
- **`plans/distribution-policy-grammar.md`.** Full declarative
  grammar extending the v1 schema in
  [signing-and-trust.md](signing-and-trust.md) §8. Deferred
  until the surface stabilizes across quarantine-sandbox and
  data-retention.

---

## Open Questions

- **Dev-install decay at restart.** §1.3 disables surviving
  dev installs at server start. Is that the right behavior for
  multi-operator servers where another operator might rely on
  a running dev install? Possibly add a per-operator "pin dev
  install" flag.
- **Reviewer regression fixtures.** Gate 5's DoS mitigation
  relies on a daily known-good corpus. Who owns that corpus
  and where does it live — shipped with the server, or a
  local curated set?
- **Peer catalog authorization.** §4.2/§4.3 pin package
  identity out of band but do not specify the auth envelope on
  the gRPC calls themselves. That belongs in
  `plans/distribution-policy-grammar.md` or in the existing
  identity layer.
- **Registry-epoch scope.** Should registry-epoch bumps be
  per-namespace (authoring vs. distribution) or global as
  specified in §6.a? The simple global counter is cheap at
  today's scales but may become a bottleneck.
