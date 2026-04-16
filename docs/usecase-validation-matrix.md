# Use Case Validation Matrix

This matrix maps `usecases.md` IDs to current automated validation coverage.

Primary command:

```bash
make usecase-validate USECASE=<ID>
# or
make usecase-validate USECASE=all
```

## Automated IDs

| ID | Scenario | Validation Command | Primary Evidence |
|---|---|---|---|
| C1 | Intercom (2-way / route-stop / fan-out) | `make usecase-validate USECASE=C1` | `internal/transport` generated+wire integration tests |
| C3 | PA mode | `make usecase-validate USECASE=C3` | PA relay, voice start, voice stop alias tests |
| C5 | Internal video call | `make usecase-validate USECASE=C5` | `TestGeneratedSessionInternalVideoCallStartSetUIAndHangupFlow` |
| D1 | Photo frame idle rotation | `make usecase-validate USECASE=D1` | photo-frame config + heartbeat rotation tests |
| M1 | "Tell me when X stops" audio monitor | `make usecase-validate USECASE=M1` | silence classifier integration test |
| M3 | Red alert broadcast | `make usecase-validate USECASE=M3` | generated+wire red alert integration tests |
| M4 | Stand down / stop red alert | `make usecase-validate USECASE=M4` | generated+wire voice stop/stand-down tests |
| S1 | Show all cameras | `make usecase-validate USECASE=S1` | generated+wire voice show-all-cameras tests |
| S2 | Focus one camera audio | `make usecase-validate USECASE=S2` | generated+wire focus-action routing tests |
| S3 | Mixed multi-camera audio overview | `make usecase-validate USECASE=S3` | generated+wire multi-window audio mix tests |
| P1 | Terminal session UI transitions | `make usecase-validate USECASE=P1` | generated+wire terminal transition tests |
| T1 | Timer firing | `make usecase-validate USECASE=T1` | due-timer loop + transport timer tests |

## Planned / Not Yet Automated

The following planned IDs currently do not have a dedicated `make usecase-validate USECASE=<ID>` mapping yet:

`C2`, `C4`, `C6`, `V1`, `V2`, `V3`, `T2`, `T3`, `T4`, `M2`, `M5`, `D2`, `D3`, `P2`, `I1`-`I11`.

Use `make all-check` as the baseline repository gate while dedicated use-case mappings are added.
