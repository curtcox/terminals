---
title: "Package Format — Progress Log"
kind: progress-log
parent: plans/features/package-format/plan.md
---

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

Additional shipped slice in this cycle:

- Added `VerifyPackage(tapBytes, sigBytes)` pre-trust API that validates
  canonical tar structure, parses manifest identity, and verifies
  `tap-sig/1` statement bundles against the computed package hash.
- Implemented statement-level checks for required fields, schema/package
  binding, author-presence minimum acceptance, duplicate
  `(key_id, package_id, nonce)` rejection, and role-specific scope shaping
  (`author`, `voucher`, `publisher`) with unknown authority-bearing keys
  rejected.
- Added deterministic CBOR statement encoding and Ed25519 verification for
  statement signatures, with bundle/parser quotas for v1 boundaries.
- Expanded `terminal_server/internal/apppackage/tap_test.go` with
  `VerifyPackage` success and rejection coverage (unknown voucher scope key,
  missing author statement, duplicate nonce triple).

Additional shipped slice in this cycle:

- Added strict canonical zstd frame validation in pre-trust verification:
  rejects skippable/non-zstd magic, enforces content-size flag present,
  rejects checksum/dictionary-id flags, enforces max window log/size, and
  rejects trailing bytes or concatenated extra frames before decompression.
- Expanded `terminal_server/internal/apppackage/tap_test.go` with explicit
  frame-layer rejection tests for checksum flag, dictionary-id flag, missing
  content-size flag, trailing bytes, multi-frame payloads, skippable frame
  magic, and oversized window descriptors.

Additional shipped slice in this cycle:

- Expanded `.tap.sig` rejection-path coverage in
  `terminal_server/internal/apppackage/tap_test.go` for malformed TOML,
  schema mismatch, bundle `package_id` mismatch, missing statement `nonce`,
  statement manifest mismatch, corrupted Ed25519 signature rejection,
  oversized bundle rejection, and statement-count quota enforcement.
- Added reusable signed-statement test helper(s) to reduce duplication while
  keeping canonical CBOR signing behavior under test.

Additional shipped slice in this cycle:

- Added zstd encoder parity validation in
  `terminal_server/internal/apppackage/tap_test.go` that compares
  `BuildTapFromDir` and pinned `zstd -19 --no-check --format=zstd
  --single-thread` CLI re-encoding against the same canonical tar bytes,
  asserting both verify and produce the same `package_id`.
- Documented current package-format runtime coverage in
  `docs/application-runtime.md` to reflect canonical tar, signature-bundle
  pre-trust checks, and encoder parity testing.
