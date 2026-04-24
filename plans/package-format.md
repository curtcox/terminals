# Package Format

See [masterplan.md](masterplan.md) for overall system context.
Extends [application-runtime.md](application-runtime.md) (on-disk
package layout). Referenced by
[application-distribution.md](application-distribution.md) (how
packages move between servers) and
[signing-and-trust.md](signing-and-trust.md) (who may sign what).

## Problem

[application-runtime.md](application-runtime.md) specifies a package
as a *directory on disk*. Distribution needs a *file* that can move
between servers and be reasoned about cryptographically. The file
format has to make two things unambiguous:

1. **What bytes are "the package."** Every server and every
   reviewer must compute the same `package_id` from the same
   source tree.
2. **What each signature actually asserts.** A signature that
   commits only to "the package bytes" is too weak: it cannot
   distinguish an author claim from a voucher claim, cannot bind
   a voucher to the review conditions that were true when it was
   issued, and cannot prevent reuse of an old signature against a
   new package.

Both were flagged as blockers during review of the distribution
plan; this document fixes them before any trust machinery is
layered on top.

## Design Principles

1. **Canonicalization is content, not convention.** A package is
   a byte sequence produced by a fully specified tar canonicalizer.
   Two authors who build from the same source tree produce the
   same bytes.
2. **Every signature signs a statement, not a hash.** The statement
   names its own role, scope, key, timestamps, and the package
   it refers to. A hash alone is ambiguous.
3. **Statements are schema-versioned.** Future roles and fields
   extend the schema without breaking old verifiers.
4. **Static validation before trust.** A package that fails
   canonicalization or statement-schema validation is rejected
   before any key is even looked up.

## Non-Goals

- No compression scheme selection — this document fixes one.
- No transport format — `.tap` / `.tap.sig` travel over whatever
  `PackageSource` carries them (see distribution plan).
- No trust policy — that lives in
  [signing-and-trust.md](signing-and-trust.md). This document
  only specifies what can be verified *about* a signature, not
  whether to accept it.

---

## 1. Package File (`.tap`)

### 1.1 Outer shape

A package is a POSIX ustar archive, compressed with zstd at a
fixed level, with filename `<name>-<version>.tap`.

```
<name>-<version>.tap = zstd(level=19, deterministic)( canonical_tar )
```

zstd level 19 is fixed so a rebuild is byte-identical. The zstd
frame has no dictionary and no checksum flag (checksumming is
done at the tar + signature layer instead, so it is not
duplicated inside the compressor).

### 1.2 Canonical tar

The tar stream MUST satisfy every rule below. A non-conforming
tar is rejected at Gate 1 before any signature is examined.

**Member ordering.** Entries appear in lexicographic order of
their full path under the package root. No duplicate paths. No
empty directories as standalone entries (implicit from files).

**Paths.**

- Relative, rooted at the package name: `kitchen_timer/main.tal`.
- UTF-8 NFC normalized.
- No `.`, no `..`, no leading `/`.
- No symlinks. No hardlinks. No device, FIFO, or socket entries.
- No component longer than 255 bytes. Total path ≤ 4096 bytes.
- Case-sensitive; two entries differing only in case are
  rejected (protects extractors on case-insensitive filesystems).

**Types.** Only `regtype` (regular file) is allowed. Directories
are implicit.

**Modes.** File mode is exactly `0644`. The executable bit is
never carried in the archive; TAL modules do not need it, and
kernel blobs are loaded by path, not executed directly.

**Owner / group.** `uid = gid = 0`, `uname = gname = ""` (empty
strings, not `"root"`).

**Times.** `mtime = 0`, `atime = 0`, `ctime = 0`.

**Sizes.** `size` matches the actual byte length of the payload.

**Headers.** No PAX extended headers. No GNU long-name or
long-link headers. If a path exceeds the 100/155 ustar split, the
build fails rather than emitting a `L`/`K` extension header.
(Paths this long are rejected under the path rules above; this
is a belt-and-braces rule so no extension-header parsing is ever
required by a reader.)

**Padding.** Standard 512-byte block padding. Two zero blocks at
end of archive. No trailing bytes after the end marker.

**Magic / version.** `magic = "ustar\0"`, `version = "00"`.

**Checksum.** Computed per POSIX ustar. Mismatches reject the
archive.

### 1.3 Required contents

Every `.tap` MUST contain:

- `<name>/manifest.toml` — the package manifest defined in
  [application-runtime.md](application-runtime.md).
- `<name>/main.tal` — the top-level TAL module.

Every `.tap` MAY contain any subset of:

- `<name>/lib/**/*.tal`
- `<name>/tests/**/*.tal`
- `<name>/kernels/**/*.wasm`
- `<name>/models/**/*`
- `<name>/assets/**/*`
- `<name>/migrate/**/*.tal` (see
  [app-migrations.md](app-migrations.md))

No other top-level directories are permitted. Unknown top-level
entries cause Gate 1 rejection — this is the lever future
extensions pull rather than silently tolerating junk.

### 1.4 Package identity

```
package_id = "sha256:" + hex(sha256(<tap bytes>))
```

The hash covers the **compressed** outer file. This is the only
identifier that crosses server boundaries. `name` and `version`
inside the manifest are metadata, not identity.

### 1.5 Builder contract

`term app pack <name>` produces `<name>-<version>.tap`. The
builder:

1. Validates the source tree against §1.3.
2. Emits the canonical tar per §1.2.
3. Compresses per §1.1.
4. Records the produced `package_id` in an adjacent
   `<name>-<version>.tap.buildinfo` (unsigned, for reproducibility
   audits; not part of the package and not consumed at install).

A second invocation on the same clean source tree produces a
byte-identical `.tap`. CI for this repo should check that
property on every app package change.

---

## 2. Signature Bundle (`.tap.sig`)

### 2.1 Outer shape

A TOML file named `<name>-<version>.tap.sig`. Unlike the package
itself, the signature bundle is **append-only**: additional
signatures accumulate over the package's lifetime as vouchers
arrive.

```toml
schema = "tap-sig/1"
package_id = "sha256:…"

[[statement]]
role         = "author"
key_id       = "ed25519:…"
created      = "2026-04-24T12:34:56Z"
manifest_name    = "kitchen_timer"
manifest_version = "0.1.0"
scope        = { }
sig          = "base64:…"

[[statement]]
role         = "voucher"
key_id       = "ed25519:…"
created      = "2026-04-25T09:00:00Z"
manifest_name    = "kitchen_timer"
manifest_version = "0.1.0"
scope        = { tier = "quarantine", reviewed = ["manifest", "tal", "tests"], tested_under = "sim-only" }
sig          = "base64:…"
```

### 2.2 Canonical signed statement

Each `[[statement]]` commits to a **canonical signed statement**
— a byte string that is stable across encoders. The signature is
over that byte string, never over `package_id` alone.

The statement is a CBOR map with fixed, sorted integer keys.
Integers (not strings) keep the encoding compact and stable.
The schema:

| Key | Field                | Type   | Notes                                         |
|-----|----------------------|--------|-----------------------------------------------|
| 0   | `schema_version`     | int    | Matches outer `schema`. Current value: `1`.   |
| 1   | `package_id`         | bstr   | Raw 32-byte sha256 of the `.tap`.             |
| 2   | `role`               | tstr   | `"author"`, `"voucher"`, `"publisher"`.       |
| 3   | `key_id`             | tstr   | Key identifier per signing-and-trust.md.      |
| 4   | `created_unix`       | int    | UTC seconds.                                  |
| 5   | `manifest_name`      | tstr   | From the package's manifest.                  |
| 6   | `manifest_version`   | tstr   | From the package's manifest.                  |
| 7   | `scope`              | map    | Role-specific; see §2.3.                      |
| 8   | `nonce`              | bstr   | 16 random bytes. Prevents signature reuse.    |

Rules:

- CBOR is encoded deterministically per RFC 8949 §4.2.
- `package_id` is the raw hash bytes, not the `sha256:…` string,
  so the statement cannot be made to "float" between formats.
- `manifest_name` and `manifest_version` are redundant with the
  package contents but signed explicitly so that a verifier can
  reject a bundle whose statements name a different manifest than
  the `.tap` actually contains.
- `nonce` prevents an attacker from replaying an identical
  statement CBOR twice to inflate voucher counts.

### 2.3 Role-specific scope

Every statement has a `scope` map. Unknown keys MUST be ignored
by older verifiers; unknown *values* in a known key MUST be
treated as unrecognized (policy may then choose to warn or
block).

**`role = "author"`**

```
scope = { }   # reserved; authors assert authorship, not conditions.
```

Authors do not narrow their claim. An author either signed the
package or did not.

**`role = "voucher"`**

```
scope = {
    tier         : "full" | "quarantine" | "custom"
    reviewed     : ["manifest", "tal", "tests", "kernels", "models", "assets"]
    tested_under : "sim-only" | "hardware" | "production"
    notes        : <tstr, optional, bounded to 2 KiB>
    expires_unix : <int, optional>
}
```

The voucher commits to the exact conditions under which review
occurred. A vouch under `tier = "quarantine"` cannot be
laundered into a full-tier install: the signed statement itself
carries the scope. This closes the voucher-laundering path
called out in review.

`expires_unix`, when present, is advisory — a verifier MAY refuse
old vouches per policy. A vouch without `expires_unix` is open-
ended; trust-store policy is where expiry floors are applied (see
[signing-and-trust.md](signing-and-trust.md)).

**`role = "publisher"`**

```
scope = {
    via : "<hostname or peer id>"
}
```

Publisher signatures record re-hosting events. No gate currently
acts on them; they exist so a server can audit how a package
reached it.

### 2.4 Append semantics

Appending a voucher:

1. Recompute `sha256(<tap>)` and confirm it equals the bundle's
   `package_id`. A bundle whose outer `package_id` no longer
   matches its `.tap` is rejected outright.
2. Validate every existing statement's canonical encoding and
   signature. A malformed older statement makes the whole bundle
   invalid — a verifier never accepts a bundle piecemeal.
3. Append a new `[[statement]]` block. Do not reorder or remove
   existing blocks.

A `.tap.sig` file that has been tampered with — statements
reordered, role changed, scope edited — fails §2.2 verification
because the CBOR statement bytes no longer match the signature.
Role and scope cannot be modified in transit without breaking the
signature, which is the property the distribution review flagged
as missing.

### 2.5 Minimum acceptance

For a `.tap` + `.tap.sig` pair to reach any policy gate at all,
all of the following MUST hold:

- The `.tap` satisfies §1.
- The bundle satisfies §2.1 / §2.4.
- `schema` is a version this server understands.
- Every statement's CBOR parses, its signature verifies against
  the named key, and its `manifest_name` / `manifest_version`
  match the `.tap`'s `manifest.toml`.
- At least one statement has `role = "author"`. (Unsigned
  packages are not accepted by this format; a server that wants
  to accept unsigned code must do so explicitly outside the
  `.tap` path.)

Trust decisions — which keys, which roles, which scopes — are
deferred to [signing-and-trust.md](signing-and-trust.md).

---

## 3. Verification Pseudocode

```python
def verify_package(tap_bytes, sig_bytes):
    assert_canonical_tar(tap_bytes)             # §1.2
    pkg_id = sha256(tap_bytes)                  # §1.4
    bundle = parse_toml(sig_bytes)              # §2.1
    assert bundle["schema"] == "tap-sig/1"
    assert bundle["package_id"] == "sha256:" + hex(pkg_id)
    manifest = read_manifest(tap_bytes)         # §1.3
    author_seen = False
    for stmt in bundle["statement"]:
        cbor = canonical_cbor(stmt, pkg_id)     # §2.2
        verify_signature(stmt["key_id"], cbor, stmt["sig"])
        assert stmt["manifest_name"]    == manifest["name"]
        assert stmt["manifest_version"] == manifest["version"]
        if stmt["role"] == "author":
            author_seen = True
    assert author_seen                          # §2.5
    return pkg_id, bundle["statement"]
```

Every gate in the distribution plan runs *after* this function
succeeds. A failure here is a pre-trust rejection with no key
lookup and no policy decision.

---

## 4. Test Vectors

The repo MUST ship golden vectors exercising every rejection
path, under `terminal_server/apps/tap/testdata/`:

- `good_v1.tap` / `good_v1.tap.sig` — canonical, signed by a
  test author key, round-trips.
- `dup_path.tap` — two entries with identical paths.
- `symlink.tap` — symlink entry; must reject.
- `case_collision.tap` — `main.tal` and `Main.tal`.
- `pax_header.tap` — PAX extended header present.
- `mtime_nonzero.tap` — non-zero mtime.
- `rolled_voucher.tap.sig` — author signature has been rewritten
  as voucher; statement CBOR signature must fail.
- `scope_edited.tap.sig` — voucher with `scope.tier` changed
  post-sign; must fail.
- `replayed_sig.tap.sig` — signature bundle from a different
  package with matching key; must fail because `package_id` in
  the signed statement mismatches.
- `wrong_manifest_name.tap.sig` — statement names a manifest
  other than the one in the archive.

These vectors exist to make the canonicalizer and verifier
falsifiable. A change to this document that does not come with a
test-vector diff is incomplete.

---

## Open Questions

- **Ed25519 only, or an alg agreement field?** This spec assumes
  Ed25519 by the `key_id = "ed25519:…"` prefix and no separate
  algorithm field in the statement. If we want BLS or PQC
  signatures later, add an `alg` key to the statement schema
  (key 9) and bump `schema_version`.
- **Maximum package size.** Not fixed here. Distribution has to
  enforce a size cap at fetch time; the choice is deployment-
  specific and belongs in the distribution plan or operator
  policy, not the format spec.
