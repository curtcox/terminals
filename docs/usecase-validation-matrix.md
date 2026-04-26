# Use Case Validation Matrix

This matrix maps `usecases.md` IDs to current automated validation coverage.

Primary command:

```bash
make usecase-validate USECASE=<ID>
# or
make usecase-validate USECASE=all
```

## Automated IDs

Coverage depth labels:

- `Smoke`: proves a narrow server loop or command path.
- `Transport`: proves generated/wire control-plane behavior.
- `Scenario`: proves scenario matching and server-side side effects.
- `Contract`: proves app/package/runtime contract surfaces.
- `Simulation`: proves lifecycle behavior against synthetic time/events.
- `Full`: covers trigger, placement, UI, scheduling, side effects, and expiry/cancel/resume behavior.

| ID | Scenario | Validation Command | Primary Evidence | Coverage Depth |
|---|---|---|---|---|
| B1 | On-device bug reporting modality parity | `make usecase-validate USECASE=B1` | `internal/transport` bug-report input action tests for screen/gesture/shake/keyboard/voice | Scenario |
| B2 | Cross-device bug filing for unavailable subject | `make usecase-validate USECASE=B2` | `internal/diagnostics/bugreport` cross-device subject offline coverage test | Scenario |
| B3 | Diagnostics autodetect + merge | `make usecase-validate USECASE=B3` | `internal/diagnostics/bugreport` autodetect merge service test | Scenario |
| B4 | Admin bug report visibility (`/admin/bugs`) | `make usecase-validate USECASE=B4` | `internal/admin` bug intake/list/detail + tag filter tests | Scenario |
| B5 | SIP bug-line intake | `make usecase-validate USECASE=B5` | `internal/admin` JSON intake SIP source + transcript hints test | Scenario |
| C1 | Intercom (2-way / route-stop / fan-out) | `make usecase-validate USECASE=C1` | `internal/transport` generated+wire integration tests | Transport |
| C3 | PA mode | `make usecase-validate USECASE=C3` | PA relay, voice start, voice stop alias tests | Transport |
| C5 | Internal video call | `make usecase-validate USECASE=C5` | `TestGeneratedSessionInternalVideoCallStartSetUIAndHangupFlow` | Transport |
| D1 | Photo frame idle rotation | `make usecase-validate USECASE=D1` | photo-frame config + heartbeat rotation tests | Scenario |
| M1 | "Tell me when X stops" audio monitor | `make usecase-validate USECASE=M1` | silence classifier integration test | Scenario |
| M2 | "Tell me when the dryer beeps" audio monitor | `make usecase-validate USECASE=M2` | runtime audio monitor detection test for `dryer_beep` | Scenario |
| M3 | Red alert broadcast | `make usecase-validate USECASE=M3` | generated+wire red alert integration tests | Transport |
| M4 | Stand down / stop red alert | `make usecase-validate USECASE=M4` | generated+wire voice stop/stand-down tests | Transport |
| S1 | Show all cameras | `make usecase-validate USECASE=S1` | generated+wire voice show-all-cameras tests | Transport |
| S2 | Focus one camera audio | `make usecase-validate USECASE=S2` | generated+wire focus-action routing tests | Transport |
| S3 | Mixed multi-camera audio overview | `make usecase-validate USECASE=S3` | generated+wire multi-window audio mix tests | Transport |
| P1 | Terminal session UI transitions | `make usecase-validate USECASE=P1` | generated+wire terminal transition tests | Transport |
| PL1 | Live room text chat | `make usecase-validate USECASE=PL1` | `internal/capability` message room/thread/unread acknowledgement lifecycle test | Contract |
| PL8 | Shared live collaborative session | `make usecase-validate USECASE=PL8` | `internal/capability` session join/leave plus control request/grant/revoke tests | Contract |
| PL20 | Reusable visual templates | `make usecase-validate USECASE=PL20` | artifact template save/apply plus durable patch/history capability tests | Contract |
| T1 | Timer firing | `make usecase-validate USECASE=T1` | due-timer loop; transport `run_due_timers`; kitchen timer package smoke test; future TAL simulation coverage | Smoke |

## Planned / Not Yet Automated

The following planned IDs currently do not have a dedicated `make usecase-validate USECASE=<ID>` mapping yet:

`C2`, `C4`, `C6`, `V1`, `V2`, `V3`, `T2`, `T3`, `T4`, `M5`, `D2`, `D3`, `P2`, `I1`-`I11`, `PL2`-`PL7`, `PL9`-`PL19`, `PL21`-`PL27`.

Use `make all-check` as the baseline repository gate while dedicated use-case mappings are added.
