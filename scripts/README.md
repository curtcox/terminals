# scripts/

Helper scripts used by `make` targets. Prefer the `make` target over invoking scripts directly — the Makefile handles env and ordering.

`lib/` contains helper code sourced by other scripts (not executable directly).

## Side-effecting scripts

These touch ports, files, or running processes — do not call from tests or CI without understanding their blast radius.

| Script | `make` target | What it mutates |
|--------|---------------|-----------------|
| `run-local.sh` | `make run-local` | Starts server + client; writes `.tmp/run-local-*.log` |
| `run-mac.sh` | `make run-mac` | Launches macOS native client |
| `stop-server.sh` | `make stop-server` | Kills running server process(es) |
| `ui-inspect-run.sh` | `make ui-inspect` | Launches clients and takes screenshots into `.tmp/` |
| `usecase-validate.sh` | `make usecase-validate USECASE=X` | Runs an in-process server, writes to `artifacts/` |
| `bug-resolve.py` | `make bug-resolve` | Writes to `terminal_server/bug_reports/` |
| `build-usecase-site.py` | `make usecases-site` | Generates `docs/usecases-site/` |
| `check-ci-gates.sh` | `make ci-status` | Writes `scripts/ci-status.json` |

## Check / validate (read-only)

| Script | `make` target | Purpose |
|--------|---------------|---------|
| `check-ci-gates.sh` | `make ci-status` | Probes GitHub CI gate results |
| `check-client-boundary.sh` | `make client-boundary` | Verifies Flutter client does not import server packages |
| `check-android-client-boundary.sh` | `make android-client-boundary` | Same for Android client |
| `check-proto-flex-fields.py` | (part of `check-fast`) | Ensures proto fields follow flex-field conventions |
| `check-resolved-bugs.py` | `make bug-resolved-check` | Verifies resolved bugs stay resolved |
| `audit-usecase-wiring.py` | `make usecase-wiring-audit` | Checks every use-case ID has a mapped validation |
| `find-oversized-files.py` | `make quality-check` | Flags Go files over the size threshold |
| `assert-server-test-network-probe.sh` | `make server-test-network-probe` | Validates network probe test output |
| `validate-skills.sh` | `make skills-check` | Validates `.claude/skills/` structure |

## Generate (writes to repo)

| Script | `make` target | Writes |
|--------|---------------|--------|
| `generate-usecases-index.py` | `make usecases-index` | `usecases/INDEX.md` |
| `generate-plans-index.py` | `make plans-index` | `plans/INDEX.md` |
| `generate-validation-matrix.py` | `make validation-matrix` | `docs/usecase-validation-matrix.md` |
| `proto-contract-generate.sh` | `make proto-contract-generate` | `api/testdata/` contract snapshots |
| `build-usecase-site.py` | `make usecases-site` | `docs/usecases-site/` |

## Self-tests (scripts that test other scripts)

Files prefixed `test-` or `test_` verify the corresponding script or `make` target.

| Script | Tests |
|--------|-------|
| `test-run-local.sh` | `run-local.sh` bootstrap and launch |
| `test-run-server-target.sh` | `make run-server` target |
| `test-run-client-web-target.sh` | `make run-client-web` target |
| `test-stop-server-target.sh` | `make stop-server` target |
| `test-stop-server-safety.sh` | Stop-server process scope safety |
| `test-mac-e2e.sh` | macOS end-to-end client flow |
| `test-web-client-smoke.sh` | Web client smoke test |
| `test-ui-inspect-run.sh` | UI inspect run script |
| `test-android-client-boundary.sh` | Android boundary check |
| `test-check-client-boundary.sh` | Flutter boundary check |
| `test-development-environment-docs.sh` | Development env doc accuracy |
| `test-server-test-network-probe.sh` | Network probe test helper |
| `test_bug_resolve.py` | `bug-resolve.py` |
| `test_build_usecase_site.py` | `build-usecase-site.py` |

## Priority helpers

| Script | Purpose |
|--------|---------|
| `next.py` | Entry point for `make next` — prints work recommendations |
| `pick-next-work.py` | Priority bucketing across plans and use cases |
| `usecase-validate-helper.py` | Validation harness support logic |
| `server-test-sandbox.sh` | Sandbox detection for server tests |
| `probe_server_test_network.go` | Network probe used by sandbox tests |
| `usecasesite_assets.py`, `usecasesite_models.py` | Site generation support modules |
| `android-client-stop-stuck-workers.py` | Kill stuck Android Gradle workers |
