## Protocol Checklist

If this PR touches `api/terminals/**` or changes client/server wire behavior:

- [ ] I checked `api/CONTRACTS.md`.
- [ ] I used typed protobuf fields where practical.
- [ ] I updated `docs/protocol-extension-registry.md` for any new metadata key, string token, map key, selector, or JSON shape.
- [ ] I preserved additive compatibility or documented the compatibility impact in `docs/compatibility.md`.
- [ ] I ran `make proto-generate` if `.proto` files changed.
- [ ] I added or updated Go-Dart golden fixtures for changed message semantics.
- [ ] I confirmed no scenario-specific behavior was added to the Flutter client.
- [ ] I ran `make proto-lint` and `make proto-flex-check`.
- [ ] I ran `make proto-contract-test`.
