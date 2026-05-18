# usecases/

User-story use cases for the Terminals system. One file per family; IDs are stable and referenced from plan frontmatter.

## ID system

IDs are `<FAMILY><NUMBER>` (e.g., `C1`, `M5`, `PL27`). Family letters:

| Letter | Family |
|--------|--------|
| `AA` | Software Agents and Automation |
| `AB` | Adjacent — Business / Retail / Hospitality |
| `AH` | Adjacent — Home |
| `AO` | Adjacent — Office |
| `B` | Bug Reporting and Diagnostics |
| `C` | Communication |
| `D` | Display and Ambient |
| `I` | System and Infrastructure |
| `M` | Monitoring and Alerts |
| `P` | Terminal and Productivity |
| `PL` | PLATO-Inspired Extensions |
| `S` | Security and Surveillance |

See [INDEX.md](INDEX.md) for the full auto-generated table (do not edit by hand — run `make usecases-index`).

## Validation wiring

A use case can be:

- **Planned** — described here but not yet automated.
- **Automated** — the plan that implements it includes `validation: automated:<ID>` in its frontmatter, and a test runs against `make usecase-validate USECASE=<ID>`.

See [docs/usecase-validation-matrix.md](../docs/usecase-validation-matrix.md) for the full coverage table.

Run one validation:
```bash
make usecase-validate USECASE=C1
```

Run all automated validations:
```bash
make usecase-validate USECASE=all
```

Check for IDs that are described but not yet wired to a test:
```bash
make usecase-wiring-audit
```

## Adding a new use case

1. Pick the right family file (or create a new one for a new family).
2. Assign the next sequential ID for that family.
3. Add a row in user-story format: _As a … I would like to … so that I can …_
4. Run `make usecases-index` to regenerate [INDEX.md](INDEX.md).
5. To implement and automate it: use the `usecase-implement` skill.
