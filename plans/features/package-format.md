---
title: "Package Format"
kind: plan
status: building
owner: copilot
validation: none
last-reviewed: 2026-04-26
---

# Package Format

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.
Extends [application-runtime.md](application-runtime.md) (on-disk
package layout). Referenced by
[application-distribution.md](application-distribution.md) (how
packages move between servers) and
[signing-and-trust.md](signing-and-trust.md) (who may sign what).

## Problem

[application-runtime.md](application-runtime.md) specifies a
package as a *directory on disk*. Distribution needs a *file* that
can move between servers and be reasoned about cryptographically.
The file format has to make three things unambiguous:

1. **What bytes are "the package."** Every server and every
   reviewer must compute the same `package_id` from the same
   source tree, regardless of which encoder they used.
2. **What each signature actually asserts.** A signature over a
   hash alone is too weak: it cannot distinguish author from
   voucher, cannot bind a voucher to its review conditions, and
   cannot prevent reuse of an old signature against a new
   package.
3. **Where trust begins.** Before any trust store is consulted,
   the package must be proven well-formed and self-consistent
   through a pre-trust verifier — including decompression,
   canonical tar validation, and manifest extraction.

Both the statement semantics and the pre-trust ordering were
flagged as blockers in review; this document fixes them.

## Design Principles

1. **Canonicalization of the trust boundary.** A package is a
   byte sequence produced by a fully specified pipeline. The
   `package_id` commits to that sequence and nothing else.
2. **One pre-trust pipeline.** `verify_package()` is the single
   function every gate consults before trust. Manifest contents
   are not read until the archive has been canonicalized.
3. **Every signature signs a statement, not a hash.** The
   statement names its own role, scope, key, timestamps, nonce,
   and the package it refers to.
4. **Schema-versioned statements.** Future roles and fields
   extend the schema with an explicit bump. v1 verifiers reject
   unknown authority-bearing fields rather than ignoring them.
5. **Ed25519 only in v1.** Algorithm agility is deferred behind a
   schema bump.

## Non-Goals

- No transport format. `.tap` / `.tap.sig` travel over whatever
  `PackageSource` carries them (see distribution plan).
- No trust policy — that lives in
  [signing-and-trust.md](signing-and-trust.md). This document
  specifies what can be verified *about* a statement, not
  whether to accept it.

---

## 1. Package File (`.tap`)

### 1.1 Two-layer canonicalization

A package is specified as two independently canonical forms.
The inner canonical tar is authority-bearing; the outer zstd
frame is transport compression.

```
canonical_tar         = deterministic POSIX ustar (§1.3)
tap_bytes             = canonical_zstd_frame(canonical_tar)    (§1.2)
package_id            = "sha256:" + hex(sha256(canonical_tar))
```

The file on disk is `<name>-<version>.tap` and its contents are
`tap_bytes`. `package_id` commits to the **canonical tar bytes**,
not to the compressed frame. This choice means:

- Signatures remain valid across any encoder that produces the
  same canonical tar even if the zstd frame bytes differ
  (different compression library versions, level tuning,
  block-size decisions).
- `verify_package()` (§3) decompresses first, validates the
  canonical tar, then hashes it. An attacker cannot forge a
  `package_id` by feeding a different `.tap`: canonical-tar
  validation fails first.
- Frame-level properties (§1.2) are still enforced — a `.tap`
  that is not a canonical zstd frame is a Gate 0 rejection —
  but frame encoding no longer needs bit-for-bit parity across
  implementations.

### 1.2 Canonical zstd frame

v1 `.tap` files MUST use exactly one canonical zstd frame with
the following constraints:

| Property                        | Value                                            |
|---------------------------------|--------------------------------------------------|
| Frame format                    | zstd single frame (RFC 8878), magic `0x28B52FFD` |
| Compression level               | 19                                               |
| Window log                      | 23 (8 MiB)                                       |
| Block size                      | encoder default at level 19; not constrained     |
| Content Size flag               | **set**; decompressed size stored in frame header |
| Content checksum flag           | **unset** (integrity comes from `package_id` over the canonical tar, not from zstd) |
| Dictionary ID flag              | **unset**                                        |
| Single Segment flag             | encoder default                                  |
| Number of frames                | exactly 1; skippable frames are rejected         |
| Trailing bytes                  | forbidden; bytes after the frame terminator are a format error |

Pinned encoder reference: `zstd` command-line tool version ≥
1.5.x built from upstream, invoked as `zstd -19 --no-check
--format=zstd --single-thread`. A Go implementation (used by the
server) MUST be tested to produce byte-identical output for the
golden vectors in §4.

If two encoder implementations ever diverge on a source tree
that conforms to §1.3, the zstd pin is tightened in a `.tap/2`
schema bump. v1 treats encoder divergence as a bug in the
non-matching encoder.

### 1.3 Canonical tar

The decompressed tar stream MUST satisfy every rule below. A
non-conforming tar is rejected by `verify_package()` before any
signature is examined.

**Member ordering.** Entries appear in lexicographic order of
their full path under the package root. No duplicate paths. No
empty-directory entries.

**Paths.**

- Relative, rooted at the package name: `kitchen_timer/main.tal`.
- UTF-8 NFC normalized.
- No `.`, no `..`, no leading `/`.
- No symlinks. No hardlinks. No device, FIFO, or socket entries.
- No component longer than 255 bytes. Total path ≤ 4096 bytes.
- Case-sensitive; two entries differing only in case are
  rejected.

**Types.** Only `regtype` (regular file) is allowed. Directories
are implicit.

**Modes.** File mode is exactly `0644`.

**Owner / group.** `uid = gid = 0`, `uname = gname = ""`.

**Times.** `mtime = 0`, `atime = 0`, `ctime = 0`.

**Sizes.** `size` matches the actual byte length of the payload.

**Headers.** No PAX extended headers. No GNU long-name or
long-link headers.

**Padding.** Standard 512-byte block padding. Two zero blocks at
end of archive. No trailing bytes after the end marker.

**Magic / version.** `magic = "ustar\0"`, `version = "00"`.

**Checksum.** Computed per POSIX ustar. Mismatches reject the
archive.

### 1.4 Required contents

Every `.tap` MUST contain:

- `<name>/manifest.toml` (exactly one).
- `<name>/main.tal` (exactly one).

Every `.tap` MAY contain any subset of:

- `<name>/lib/**/*.tal`
- `<name>/tests/**/*.tal`
- `<name>/kernels/**/*.wasm`
- `<name>/models/**/*`
- `<name>/assets/**/*`
- `<name>/migrate/**/*.tal` (see
  [app-migrations.md](app-migrations.md))

No other top-level directories are permitted. Unknown top-level
entries cause rejection in §3.

---

## 2. Signature Bundle (`.tap.sig`)

### 2.1 Outer shape

A TOML file named `<name>-<version>.tap.sig`. Append-only:
additional statements accumulate over the package's lifetime as
vouchers arrive.

```toml
schema     = "tap-sig/1"
package_id = "sha256:…"

[[statement]]
role             = "author"
key_id           = "ed25519:…"
created_unix     = 1714000000
manifest_name    = "kitchen_timer"
manifest_version = "0.1.0"
nonce            = "base64url:…"          # 16 random bytes, required
scope            = { }
sig              = "base64:…"

[[statement]]
role             = "voucher"
key_id           = "ed25519:…"
created_unix     = 1714100000
manifest_name    = "kitchen_timer"
manifest_version = "0.1.0"
nonce            = "base64url:…"
scope            = { tier = "quarantine", reviewed = ["manifest", "tal", "tests"], tested_under = "sim-only" }
sig              = "base64:…"
```

Every TOML `[[statement]]` MUST carry every field the signed
statement carries, including `nonce`. Missing `nonce` is a
format rejection; a verifier cannot reconstruct the signed
bytes without it.

**Parser quotas** (v1 minimum; operator policy may raise):

- File size ≤ 1 MiB.
- Statement count ≤ 64.
- Any string field length ≤ 8 KiB.
- TOML parse depth ≤ 16.

Exceeding any quota is a format rejection.

### 2.2 Canonical signed statement

Each `[[statement]]` commits to a **canonical signed statement**
— a byte string that is stable across encoders. The signature
is over that byte string, never over `package_id` alone.

The statement is a CBOR map with fixed, sorted integer keys.
Schema:

| Key | Field                | Type   | Notes                                         |
|-----|----------------------|--------|-----------------------------------------------|
| 0   | `schema_version`     | uint   | Matches outer `schema`. v1 value: `1`.        |
| 1   | `package_id`         | bstr   | Raw 32-byte sha256 of the **canonical tar** (§1.1). |
| 2   | `role`               | tstr   | `"author"`, `"voucher"`, `"publisher"`.       |
| 3   | `key_id`             | tstr   | `"ed25519:…"` in v1.                          |
| 4   | `created_unix`       | uint   | Signer-asserted. Advisory; §3.5 binds authoritative time to server-observed receipt per [signing-and-trust.md](signing-and-trust.md). |
| 5   | `manifest_name`      | tstr   | From the package's manifest.                  |
| 6   | `manifest_version`   | tstr   | From the package's manifest.                  |
| 7   | `scope`              | map    | Role-specific; see §2.3.                      |
| 8   | `nonce`              | bstr   | 16 random bytes. Required.                    |

Rules:

- CBOR is encoded deterministically per RFC 8949 §4.2. The
  verifier deterministically re-encodes the CBOR map from the
  parsed TOML statement and verifies the Ed25519 signature over
  those re-encoded bytes. The signer and verifier therefore
  agree on a single canonical byte string for the signed
  statement; any encoder-level ambiguity fails closed because
  the signature check fails.
- `package_id` (CBOR key 1) is the raw 32-byte sha256 of the
  **canonical tar** (§1.1), not of the outer zstd frame and not
  the `"sha256:…"` string. Length is fixed at 32; any other
  length is a format rejection.
- `manifest_name` and `manifest_version` are redundant with the
  package contents but signed explicitly so that a verifier can
  reject a bundle whose statements name a different manifest
  than the `.tap` actually contains.
- `nonce` is per-statement. Two statements with equal
  `(key_id, package_id, nonce)` are a format rejection.

### 2.3 Role-specific scope

Every statement has a `scope` map. v1 verifiers **reject** any
unknown key in an authority-bearing scope — unknown keys cannot
silently carry new authority. This reverses the earlier draft's
"unknown keys are ignored" rule.

**`role = "author"`**

```
scope = { }   # reserved; authors assert authorship, not conditions.
```

Authors do not narrow their claim. An author either signed the
package or did not. A non-empty `scope` on an author statement
is a format rejection in v1.

**`role = "voucher"`**

```
scope = {
    tier         : "full" | "quarantine" | "custom"
    reviewed     : [ list of values in {"manifest","tal","tests","kernels","models","assets"} ]
    tested_under : "sim-only" | "hardware" | "production"
    notes        : <tstr, optional, ≤ 2 KiB>
    expires_unix : <uint, optional>
}
```

The voucher commits to the exact conditions under which review
occurred. A vouch under `tier = "quarantine"` cannot be
laundered into a full-tier install.

`expires_unix`, when present, is advisory — trust policy decides
if an expired vouch still counts. A vouch without
`expires_unix` is bounded by the voucher's ceiling in
[signing-and-trust.md](signing-and-trust.md) §2.

**`role = "publisher"`**

```
scope = {
    via : "<hostname or peer id, ≤ 256 bytes>"
}
```

Publisher signatures record re-hosting events. No gate currently
acts on them; they exist so a server can audit how a package
reached it.

### 2.4 Append semantics (no ordering authority)

Statements are appended to `.tap.sig` as vouchers arrive.
**Statement order is not authority-bearing.** A previous draft
claimed statement-reordering would fail verification; that was
wrong — only the per-statement signature is bound. Trust policy
(§9 crosswalk in [signing-and-trust.md](signing-and-trust.md))
consumes the unordered set and may weight statements by
server-observed receipt time recorded in the trust log.

Appending a voucher:

1. Recompute `sha256(<tap>)` and confirm it equals the bundle's
   `package_id`. A bundle whose outer `package_id` no longer
   matches its `.tap` is rejected outright.
2. Validate every existing statement per §3.
3. Append a new `[[statement]]` block. Reordering is harmless
   to signature verification but is not a supported editing
   operation; toolchains SHOULD preserve insertion order to
   match server-observed log ordering.

### 2.5 Minimum acceptance (field-level; trust is out of scope)

For a `.tap` + `.tap.sig` pair to reach any policy gate at all,
`verify_package()` must succeed (§3). In addition:

- At least one statement has `role = "author"`. Unsigned
  packages are not accepted.
- All statement (`key_id`, `package_id`, `nonce`) triples are
  distinct.
- No statement is rejected under §2.3 scope rules.

Trust decisions — which keys, which roles, which scopes — are
deferred to [signing-and-trust.md](signing-and-trust.md).

---

## 3. `verify_package()` — The Pre-Trust Pipeline

`verify_package(tap_bytes, sig_bytes) → (package_id,
manifest, statements)` is the single entry point for
Gate 0 of the distribution pipeline. Every later gate runs
*only* after it succeeds.

```python
def verify_package(tap_bytes, sig_bytes):
    # ---- Layer 1: outer frame ---------------------------------
    assert_canonical_zstd_frame(tap_bytes)          # §1.2
    canonical_tar = zstd_decompress(tap_bytes)      # stream length ≤ window_log bound
    # ---- Layer 2: canonical tar -------------------------------
    members = parse_tar(canonical_tar)
    assert_canonical_tar(members)                   # §1.3 (paths, types, modes, times…)
    assert_required_contents(members)               # §1.4
    # ---- Layer 3: manifest extraction -------------------------
    manifest_bytes = members["<name>/manifest.toml"]
    manifest       = parse_toml(manifest_bytes)
    # ---- Layer 4: package identity ----------------------------
    pkg_id = sha256(canonical_tar)                  # §1.1 — authority hash is over tar, not frame
    # ---- Layer 5: signature bundle ----------------------------
    bundle = parse_toml(sig_bytes)                  # quotas from §2.1
    assert bundle["schema"] == "tap-sig/1"
    assert bundle["package_id"] == "sha256:" + hex(pkg_id)
    seen_triples = set()
    author_seen  = False
    statements   = []
    for stmt in bundle["statement"]:
        assert_required_fields(stmt)                # incl. nonce
        cbor = canonical_cbor(stmt, pkg_id)         # §2.2
        verify_signature(stmt["key_id"], cbor, stmt["sig"])
        assert stmt["manifest_name"]    == manifest["name"]
        assert stmt["manifest_version"] == manifest["version"]
        assert_scope_shape(stmt["role"], stmt["scope"])   # §2.3
        triple = (stmt["key_id"], pkg_id, stmt["nonce"])
        assert triple not in seen_triples
        seen_triples.add(triple)
        if stmt["role"] == "author":
            author_seen = True
        statements.append(stmt)
    assert author_seen                              # §2.5
    return pkg_id, manifest, statements
```

Properties this pipeline guarantees:

- **Zstd frame well-formedness is proven before decompression
  runs on the result.** Decompression is bounded by the window
  log and the frame's content-size field; bombs are rejected.
- **Manifest contents are never read until canonicalization
  passes.** A tar with two `manifest.toml` entries is rejected
  at §1.3 before the parser picks one.
- **Statements are validated against the manifest bytes
  extracted from the same canonical archive the
  `package_id` was computed over.** A bundle that names a
  different manifest version is rejected.
- **Duplicate-statement replay is rejected** by the
  `(key_id, package_id, nonce)` uniqueness check. A voucher
  quorum cannot be inflated by replaying the same signed
  statement twice.

Anything failing §3 is a *pre-trust rejection* with no key
lookup and no policy decision. The `.tap` is staged for audit,
not installed.

---

## 4. Test Vectors

The repo MUST ship golden vectors exercising every rejection
path, under `terminal_server/apps/tap/testdata/`. The full
vector set covers:

**Happy path.**

- `good_v1.tap` / `good_v1.tap.sig` — canonical, signed by a
  test author key; round-trips; `package_id` is stable across
  rebuilds.
- `reencoded_frame.tap` — re-compressed with a different zstd
  implementation over the same canonical tar as `good_v1.tap`.
  Its bytes differ from `good_v1.tap` but its `package_id`
  (sha256 over the canonical tar, §1.1) MUST match and
  `good_v1.tap.sig` MUST verify against it without
  modification. This vector falsifies any implementation that
  accidentally hashes the outer frame.

**Zstd-layer rejections (§1.2).**

- `zstd_checksum_flag.tap` — content-checksum flag set.
- `zstd_dict_flag.tap` — dictionary flag set.
- `zstd_multiframe.tap` — two frames concatenated.
- `zstd_trailing_bytes.tap` — bytes after frame terminator.
- `zstd_window_too_large.tap` — window log > 23.
- `zstd_missing_content_size.tap` — content-size flag unset.

**Tar-layer rejections (§1.3 / §1.4).**

- `dup_path.tap` — two entries with identical paths.
- `dup_manifest.tap` — two `manifest.toml` entries.
- `symlink.tap` — symlink entry.
- `hardlink.tap` — hardlink entry.
- `case_collision.tap` — `main.tal` and `Main.tal`.
- `path_traversal.tap` — entry named `kitchen_timer/../other`.
- `absolute_path.tap` — entry named `/kitchen_timer/main.tal`.
- `pax_header.tap` — PAX extended header present.
- `mtime_nonzero.tap` — non-zero mtime.
- `mode_nonstandard.tap` — file mode != 0644.
- `unknown_top_level.tap` — entry under `kitchen_timer/secrets/`.
- `missing_main_tal.tap` — no `main.tal`.
- `missing_manifest.tap` — no `manifest.toml`.

**Bundle-layer rejections (§2).**

- `malformed_toml.tap.sig` — TOML parse error.
- `schema_mismatch.tap.sig` — wrong `schema`.
- `package_id_mismatch.tap.sig` — bundle `package_id` does not
  match the `.tap`.
- `no_author.tap.sig` — only voucher statements.
- `missing_nonce.tap.sig` — `nonce` field omitted.
- `duplicate_nonce.tap.sig` — two statements with the same
  `(key_id, package_id, nonce)`.
- `rolled_voucher.tap.sig` — statement text rewritten (role
  author→voucher) without re-signing; CBOR verification fails.
- `scope_edited.tap.sig` — voucher with `scope.tier` changed
  post-sign; CBOR verification fails.
- `unknown_scope_key.tap.sig` — voucher with `scope.new_field =
  "yes"` on v1 verifier; must reject.
- `wrong_manifest_name.tap.sig` — statement names a manifest
  other than the one in the archive.
- `replayed_sig.tap.sig` — signature bundle from a different
  package with matching key; must fail because `package_id` in
  the signed statement mismatches.
- `invalid_signature.tap.sig` — statement with a corrupted
  signature byte.
- `oversized_bundle.tap.sig` — file > 1 MiB.
- `too_many_statements.tap.sig` — statement count > 64.

A change to this document that does not come with a
test-vector diff is incomplete. The vectors exist to make
`verify_package()` falsifiable.

---

## 5. Schema Compatibility

`tap-sig/1` is the v1 statement schema. A server MUST:

- Accept only `tap-sig/1` statements in v1.
- Reject a bundle whose outer `schema` is unknown.
- Persist the outer `schema` field alongside the installed app
  so that upgrade transforms can be written when `tap-sig/2`
  ships.

Schema bumps are reserved for changes that alter *signed
authority surface* — adding a signed field, adding a new role,
or changing canonicalization. Non-authority changes (parser
limits, error codes) do not bump the schema.

---

## Open Questions

- **Encoder parity testing.** §1.2 pins a reference zstd CLI
  invocation; the Go server needs a CI check that the same
  source tree produces byte-identical `.tap` under both
  encoders. The test is straightforward but needs a home.
- **Maximum package size.** Not fixed here beyond the 1 MiB
  bundle cap. Distribution has to enforce a size cap at fetch
  time; the choice is deployment-specific and belongs in the
  distribution plan or operator policy, not the format spec.
- **Algorithm agility.** Ed25519-only in v1. A future
  `tap-sig/2` will add an `alg` key (CBOR key 9) and widen the
  `key_id` prefix set. Until then, non-`ed25519:` prefixes are
  rejected at Gate 0.

## Implementation Progress (2026-04-26)

This plan moved to `building` with an initial shipped slice in server code:

- Added `terminal_server/internal/apppackage` with deterministic `.tap` build
  and pre-trust `.tap` verification APIs.
- Implemented canonical tar validation guards for sorted archive paths,
  duplicate/case-colliding paths, path traversal rejection, required
  `manifest.toml` and `main.tal`, and top-level directory allow-list.
- Implemented package identity as `sha256` over decompressed canonical tar
  bytes (`package_id`), aligning pre-trust identity with this plan.
- Added unit tests in `terminal_server/internal/apppackage/tap_test.go` for
  deterministic builds and representative rejection paths.

Remaining work includes full `.tap.sig` statement verification, strict zstd
frame flag enforcement, and comprehensive golden-vector coverage described in
 this plan.
