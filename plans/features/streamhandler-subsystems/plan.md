---
id: streamhandler-subsystems
title: Split StreamHandler into Focused Server Subsystems
status: proposed
owner: server architecture
created: 2026-05-02
last-reviewed: 2026-05-02
target-area: terminal_server/internal/transport
source: audit-top-recommendation
---

# Split `StreamHandler` into Focused Server Subsystems

## Placement in the Repo

Put this plan at:

```text
plans/features/streamhandler-subsystems/plan.md
```

Create a companion progress log at:

```text
plans/features/streamhandler-subsystems/progress.md
```

This belongs under `plans/features/` because it changes the server architecture in support of future features, even though it is primarily a refactor. Keeping it in a dedicated directory also matches the existing `plans/features/<name>/plan.md` and `progress.md` pattern.

## Required Repo Updates

After adding this plan, update the following files:

1. `masterplan.md`
   - Add a link under **System design** or **Tooling**:
     - `plans/features/streamhandler-subsystems/plan.md` — Refactor the server control stream handler into focused, testable subsystems.
   - Prefer **System design** because the work changes server package boundaries and control-plane ownership.

2. `plans/INDEX.md`
   - Regenerate if it is generated:
     ```bash
     make plans-index
     ```
   - If it is manually maintained, add this plan with the same status/frontmatter conventions used by nearby plans.

3. `plans/features/streamhandler-subsystems/progress.md`
   - Start with an initial entry:
     ```markdown
     # StreamHandler Subsystems Progress

     - 2026-05-02: Created plan. Status: proposed. No implementation started.
     ```

4. `AGENTS.md` and `CLAUDE.md`
   - No architectural rule change is required.
   - Optional one-line addition under build/check guidance:
     - For StreamHandler refactors, follow `plans/features/streamhandler-subsystems/plan.md` and keep behavior server-side.

5. `SKILLS.md`
   - No update is required unless a repo-local skill is later created for this workflow.

6. `Makefile`
   - No immediate target is required.
   - If the repo later adds a docs/plans validation target, ensure this plan is included through `make plans-index` or equivalent.

## Summary

`terminal_server/internal/transport/control_stream.go` has become a high-risk coordination point. It handles transport-neutral message dispatch, device capability lifecycle, UI replay, route replay, command dispatch, media stream state, voice buffering, diagnostics, recording, WebRTC signaling, menu overlay state, and terminal/REPL session support.

This plan splits that behavior into focused, server-side collaborators while preserving the external control stream API and keeping the Flutter client generic.

The refactor must be incremental. Each phase should preserve behavior, add focused tests, and keep `StreamHandler` compiling and passing the existing transport test suite.

## Architectural Rule

This plan must preserve the project rule:

> Add behavior on the server, not the client. The Flutter client remains a generic terminal.

Do not move scenario behavior, routing decisions, UI policy, or command interpretation into `terminal_client`.

## Goals

1. Reduce `StreamHandler` to control-stream dispatch, metrics, and response assembly.
2. Move mutable lifecycle state into named, testable collaborators.
3. Preserve current protobuf contracts.
4. Preserve current client behavior.
5. Preserve current transport carrier behavior.
6. Improve reconnect, replay, malformed-message, and command authorization testability.
7. Keep each implementation PR small enough to review safely.
8. Maintain `go test ./...` and `go test -race ./...` throughout the work.

## Non-Goals

1. Do not change `.proto` files.
2. Do not change Flutter client code.
3. Do not redesign scenario runtime.
4. Do not redesign IO router.
5. Do not redesign REPL or MCP.
6. Do not introduce authentication, pairing, or transport security changes.
7. Do not remove deprecated protocol fields.
8. Do not move packages out of `transport` until extraction has stabilized.

## Core Problem

`StreamHandler` currently mixes three kinds of responsibility:

1. **Transport dispatch**
   - Interpret `ClientMessage`.
   - Return `ServerMessage`.
   - Track protocol metrics and protocol errors.

2. **Session/lifecycle state**
   - Capability baseline and deltas.
   - Last known UI per device.
   - Route replay after disconnect.
   - Active media stream state.
   - Voice audio buffers.
   - Recent command audit buffer.

3. **Domain policy**
   - UI resume decisions.
   - Menu overlay input policy.
   - Command validation.
   - Voice command pipeline behavior.
   - Bug report intake.
   - Scenario transition defaults.

The desired direction is:

```text
StreamHandler = dispatch and composition
Collaborators = lifecycle state and domain-specific behavior
```

## Proposed Final Shape

Keep the initial extraction under `terminal_server/internal/transport` to avoid import cycles and broad package churn.

```text
terminal_server/internal/transport/
  control_stream.go                  # high-level dispatch only
  capability_lifecycle.go            # hello/register/snapshot/delta lifecycle
  ui_session_state.go                # UI replay, overlays, resume state
  route_replay.go                    # reconnect route replay
  command_dispatcher.go              # command validation and scenario dispatch
  voice_pipeline.go                  # voice buffering and finalization
  media_control_state.go             # media streams, WebRTC, recording hooks
  diagnostics_intake.go              # bug report filing and acknowledgement
```

Later package moves are allowed only after the extracted files have stable tests and no import-cycle pressure.

## Design Principles

1. Prefer concrete structs first. Add interfaces only where needed for tests or dependency seams.
2. Each collaborator owns its own mutex if it owns mutable maps or slices.
3. Do not call slow or blocking dependencies while holding a collaborator mutex.
4. Return immutable result structs or copied slices/maps.
5. Keep protocol error mapping stable.
6. Keep `StreamHandler` public methods stable unless a separate migration plan exists.
7. Preserve existing test names and behavior where possible.
8. Add characterization tests before moving high-risk code.
9. Avoid introducing generic abstractions that do not remove complexity.
10. Keep scenario-specific defaults behind policies so they can later move out of `transport`.

## Subsystem Boundaries

### 1. `CapabilityLifecycle`

#### Owns

- `Hello` handling.
- Deprecated `Register` compatibility path.
- Capability snapshot application.
- Capability delta application.
- Auto-create-on-snapshot behavior.
- Capability generation validation.
- Capability invalidation calculation.
- Capability acknowledgement assembly.

#### Does Not Own

- UI replay after registration.
- Route replay after registration.
- Scenario capability-change effects.
- Transport carrier negotiation.

#### Initial Location

```text
terminal_server/internal/transport/capability_lifecycle.go
```

#### Target Shape

```go
type CapabilityLifecycle struct {
    control *ControlService
}

type CapabilityResult struct {
    DeviceID          string
    Messages          []ServerMessage
    IsInitialBaseline bool
    BeforeCaps        map[string]string
    AfterCaps         map[string]string
}

func NewCapabilityLifecycle(control *ControlService) *CapabilityLifecycle

func (c *CapabilityLifecycle) HandleHello(
    ctx context.Context,
    req HelloRequest,
) ([]ServerMessage, error)

func (c *CapabilityLifecycle) HandleRegister(
    ctx context.Context,
    req RegisterRequest,
) (CapabilityResult, error)

func (c *CapabilityLifecycle) HandleSnapshot(
    ctx context.Context,
    req CapabilitySnapshotRequest,
) (CapabilityResult, error)

func (c *CapabilityLifecycle) HandleDelta(
    ctx context.Context,
    req CapabilityDeltaRequest,
) (CapabilityResult, error)
```

The exact result fields may change during implementation. Avoid exposing repository-internal device record types unless that keeps the code simpler.

#### Required Tests

- `Hello` returns `HelloAck`.
- `Register` returns backward-compatible `RegisterAck`.
- First snapshot can establish a baseline.
- Snapshot for a missing device follows existing compatibility behavior.
- Delta updates generation and capabilities.
- Stale delta returns the current protocol violation behavior.
- Capability invalidations match current behavior.
- Malformed device IDs preserve current error mapping.

### 2. `UISessionState`

#### Owns

- Last `SetUI` by device.
- UI replay after registration/reconnect.
- UI host event index by device.
- Main UI activation by device.
- Menu overlay state.
- Menu overlay input policy.
- Multi-window resume state.
- UI action ownership if it remains tightly coupled to UI replay.

#### Does Not Own

- Scenario runtime execution.
- IO routing.
- Protobuf UI schema.
- Client rendering behavior.

#### Initial Location

```text
terminal_server/internal/transport/ui_session_state.go
```

#### Target Shape

```go
type UISessionState struct {
    // private mutex and maps
}

type UIReplayInput struct {
    DeviceID        string
    InitialUI       *ui.Descriptor
    ExistingRoutes  []ServerMessage
    HasOverlay      bool
}

type UIReplayResult struct {
    Messages []ServerMessage
}

func NewUISessionState(policy ScenarioTransitionPolicy) *UISessionState

func (s *UISessionState) RememberSetUI(deviceID string, messages []ServerMessage)
func (s *UISessionState) ReplayAfterRegistration(input UIReplayInput) UIReplayResult
func (s *UISessionState) CaptureMultiWindowResume(deviceID, priorScenario string)
func (s *UISessionState) RestoreMultiWindowResume(deviceID string) (*ui.Descriptor, *UITransition, bool)
```

#### Policy Boundary

Keep scenario-specific transition defaults out of `StreamHandler` behind:

```go
type ScenarioTransitionPolicy interface {
    EnterTransitionForScenario(name string) (UITransition, bool)
}
```

The first implementation may preserve current behavior. Moving that policy into scenario/app runtime can be a later phase.

#### Required Tests

- First registration returns initial UI.
- Reconnect replays previous `SetUI`.
- Overlay replay is appended when overlay state exists.
- Multi-window resume restores exactly once.
- Menu overlay state is per-device.
- UI ownership checks preserve current behavior.
- Returned descriptors are copied where mutation would be unsafe.

### 3. `RouteReplayStore`

#### Owns

- Route snapshots captured at disconnect.
- Reconnect route replay fallback.
- Conversion of routes into `StartStream` and `RouteStream` message pairs.
- Clearing replay state.

#### Does Not Own

- IO router route calculation.
- WebRTC signaling.
- Recording lifecycle.

#### Initial Location

```text
terminal_server/internal/transport/route_replay.go
```

#### Target Shape

```go
type RouteReplayStore struct {
    // private mutex and map
}

func NewRouteReplayStore() *RouteReplayStore

func (s *RouteReplayStore) Capture(deviceID string, routes []iorouter.Route)
func (s *RouteReplayStore) MessagesForDevice(deviceID string, liveRoutes []iorouter.Route) []ServerMessage
func (s *RouteReplayStore) Clear(deviceID string)
```

#### Required Tests

- Live route snapshot is preferred over captured replay.
- Captured replay is used when live routes are empty.
- Replay emits `StartStream` and `RouteStream`.
- Replay preserves `origin=route_delta`.
- Replay preserves `webrtc_mode=server_managed`.
- Replays are isolated by device.
- Clear removes one device only.

### 4. `CommandDispatcher`

#### Owns

- Command request validation.
- Command action normalization.
- Command kind normalization.
- Required field validation.
- Scenario start/stop dispatch.
- Recent command audit events.
- Command result message assembly.

#### Does Not Own

- Voice audio buffering.
- STT.
- UI input routing.
- REPL command execution.

#### Initial Location

```text
terminal_server/internal/transport/command_dispatcher.go
```

#### Target Shape

```go
type CommandDispatcher struct {
    runtime     *scenario.Runtime
    recentLimit int
    // private mutex and recent events
}

func NewCommandDispatcher(runtime *scenario.Runtime, recentLimit int) *CommandDispatcher

func (d *CommandDispatcher) Handle(ctx context.Context, req CommandRequest) ([]ServerMessage, error)
func (d *CommandDispatcher) Recent() []CommandEvent
```

#### Required Tests

- Invalid action maps to current error code.
- Invalid kind maps to current error code.
- Missing device ID maps to current error code.
- Missing voice text maps to current error code.
- Missing intent where required maps to current error code.
- Start command dispatches to runtime.
- Stop command dispatches to runtime.
- Runtime error records failed command event.
- Request ID is preserved in responses.

### 5. `DiagnosticsIntake`

#### Owns

- Bug report intake availability check.
- Bug report filing.
- Bug report acknowledgement assembly.
- Bug report intake error propagation.

#### Does Not Own

- Screenshot capture.
- Client bug token UX.
- Event-log storage internals.

#### Initial Location

```text
terminal_server/internal/transport/diagnostics_intake.go
```

#### Target Shape

```go
type DiagnosticsIntake struct {
    // private mutex if setter remains mutable
    intake BugReportIntake
}

func NewDiagnosticsIntake(intake BugReportIntake) *DiagnosticsIntake

func (d *DiagnosticsIntake) SetIntake(intake BugReportIntake)
func (d *DiagnosticsIntake) HandleBugReport(
    ctx context.Context,
    report *diagnosticsv1.BugReport,
) (ServerMessage, error)
```

#### Required Tests

- Nil intake returns existing unavailable error behavior.
- Successful intake returns `BugReportAck`.
- Intake error maps to current response behavior.
- Context cancellation propagates.

### 6. `VoicePipeline`

#### Owns

- Voice audio buffers by device.
- Live audio fan-out to `DeviceAudioPublisher`.
- Final audio handling.
- Buffer clearing.
- Wake-word dedupe dependency if it is currently part of this path.
- Conversion from final audio/STT result into command dispatch input.

#### Does Not Own

- Command validation.
- Scenario start/stop dispatch after command creation.
- Media route planning.

#### Initial Location

```text
terminal_server/internal/transport/voice_pipeline.go
```

#### Target Shape

```go
type VoicePipeline struct {
    // private mutex and buffers
}

func NewVoicePipeline(opts VoicePipelineOptions) *VoicePipeline

func (p *VoicePipeline) SetDeviceAudioPublisher(pub DeviceAudioPublisher)

func (p *VoicePipeline) HandleAudio(
    ctx context.Context,
    req VoiceAudioRequest,
) (VoicePipelineResult, error)
```

#### Required Tests

- Non-final audio buffers by device.
- Audio is published to `DeviceAudioPublisher`.
- Final audio triggers current finalization behavior.
- Final audio clears the device buffer.
- Empty final audio preserves current behavior.
- Buffers are isolated by device.
- Context cancellation is respected.
- A follow-up issue exists for max buffer size if no bound is implemented in this refactor.

### 7. `MediaControlState`

#### Owns

- Media stream state by stream ID.
- `StreamReady` handling.
- WebRTC signal engine delegation.
- Recording manager hooks.
- Stream cleanup on stop/disconnect.

#### Does Not Own

- IO router route decisions.
- Route replay storage.
- Voice command buffering.

#### Initial Location

```text
terminal_server/internal/transport/media_control_state.go
```

#### Target Shape

```go
type MediaControlState struct {
    // private mutex and stream map
}

func NewMediaControlState(recording recording.Manager, webrtc WebRTCSignalEngine) *MediaControlState

func (m *MediaControlState) SetRecordingManager(recording.Manager)
func (m *MediaControlState) SetWebRTCSignalEngine(WebRTCSignalEngine)

func (m *MediaControlState) HandleStreamReady(ctx context.Context, req StreamReadyRequest) ([]ServerMessage, error)
func (m *MediaControlState) HandleWebRTCSignal(ctx context.Context, req WebRTCSignalRequest, deviceID string) ([]ServerMessage, error)
func (m *MediaControlState) CleanupDevice(ctx context.Context, deviceID string) []ServerMessage
```

#### Required Tests

- Stream ready marks stream state.
- WebRTC signal engine response becomes relay message.
- Missing WebRTC engine preserves existing behavior.
- Recording manager hooks are called on stream lifecycle transitions.
- Cleanup removes only affected streams.
- Race tests pass under concurrent stream updates.

## Implementation Phases

### Phase 0 — Baseline and Characterization

#### Objective

Make current behavior observable before moving code.

#### Tasks

1. Run:
   ```bash
   cd terminal_server && go test ./...
   cd terminal_server && go test -race ./...
   ```

2. List existing tests that instantiate:
   - `NewStreamHandler`
   - `NewStreamHandlerWithRuntime`
   - `HandleMessage`
   - disconnect/reconnect helpers.

3. Add characterization tests where behavior is high-risk but under-specified:
   - reconnect with prior UI,
   - reconnect with route replay,
   - stale capability generation,
   - bug report unavailable,
   - command validation errors.

4. Add field grouping comments to `StreamHandler`:
   - capability lifecycle,
   - UI session state,
   - route replay,
   - command dispatch,
   - voice pipeline,
   - media control,
   - diagnostics,
   - metrics/transport dispatch.

#### Acceptance Criteria

- Baseline tests pass.
- Race tests pass or existing failures are documented.
- High-risk behavior has characterization coverage.
- No production behavior changes.

### Phase 1 — Centralize Constructors

#### Objective

Remove duplicated initialization before moving fields.

#### Tasks

1. Add private helper:

   ```go
   func newStreamHandler(control *ControlService, runtime *scenario.Runtime) *StreamHandler
   ```

2. Rewrite both public constructors to call it.

3. Add constructor invariant tests for:
   - maps initialized,
   - default limits set,
   - terminal manager initialized,
   - REPL session service initialized,
   - no-op recording manager initialized,
   - UI ownership tracker initialized,
   - wake-word dedupe initialized,
   - menu policy initialized.

#### Acceptance Criteria

- No duplicated constructor initialization remains.
- Existing tests pass.
- No behavior changes.

### Phase 2 — Extract `CapabilityLifecycle`

#### Objective

Move hello/register/snapshot/delta handling out of the main switch.

#### Tasks

1. Create `capability_lifecycle.go`.
2. Move capability invalidation helpers.
3. Move hello/register/snapshot/delta application logic.
4. Keep UI replay and route replay outside this collaborator.
5. Update `HandleMessage` capability branches to delegate.
6. Add focused tests.

#### Acceptance Criteria

- Capability branches are smaller and delegate to `CapabilityLifecycle`.
- Capability tests pass.
- Existing control stream tests pass.
- No protobuf/client changes.

### Phase 3 — Extract `RouteReplayStore`

#### Objective

Isolate reconnect route replay behavior.

#### Tasks

1. Create `route_replay.go`.
2. Move replay map and helper methods.
3. Move route-to-message conversion.
4. Update registration/reconnect handling to call the store.
5. Add focused route replay tests.

#### Acceptance Criteria

- `StreamHandler` no longer owns `routeReplayByDevice`.
- Route replay tests pass.
- Existing reconnect tests pass.

### Phase 4 — Extract `UISessionState`

#### Objective

Isolate UI replay, overlay, and resume behavior.

#### Tasks

1. Create `ui_session_state.go`.
2. Move UI state maps.
3. Move UI replay helpers.
4. Move multi-window resume helpers.
5. Move menu overlay state helpers where practical.
6. Add transition policy boundary.
7. Add focused UI session tests.

#### Acceptance Criteria

- `StreamHandler` no longer owns UI replay maps.
- UI replay tests pass.
- Overlay and multi-window behavior remain unchanged.
- Scenario-specific transition logic is behind a policy boundary.

### Phase 5 — Extract `CommandDispatcher`

#### Objective

Isolate command validation, runtime dispatch, and recent command audit state.

#### Tasks

1. Create `command_dispatcher.go`.
2. Move command validation helpers.
3. Move recent command event buffer.
4. Move scenario start/stop dispatch wrappers.
5. Update command branch in `HandleMessage`.
6. Add table-driven command tests.

#### Acceptance Criteria

- `StreamHandler` no longer owns recent command event state.
- Command branch delegates to `CommandDispatcher`.
- Existing command tests pass.
- New validation tests pass.

### Phase 6 — Extract `DiagnosticsIntake`

#### Objective

Make bug-report handling independently testable.

#### Tasks

1. Create `diagnostics_intake.go`.
2. Move bug report intake field and setter behavior.
3. Keep `StreamHandler.SetBugReportIntake` as a delegating compatibility method.
4. Update bug report branch in `HandleMessage`.
5. Add focused diagnostics tests.

#### Acceptance Criteria

- Bug-report branch is a small delegation.
- Existing bug report behavior is unchanged.
- Nil intake and intake error paths are tested.

### Phase 7 — Extract `VoicePipeline`

#### Objective

Isolate voice audio buffering and finalization.

#### Tasks

1. Create `voice_pipeline.go`.
2. Move voice audio buffer map.
3. Move device audio publisher field and setter behavior.
4. Keep `StreamHandler.SetDeviceAudioPublisher` as a delegating compatibility method.
5. Move wake-word dedupe only if it is part of the voice audio path.
6. Add voice pipeline tests.
7. Add a follow-up task for bounded buffer size if not implemented.

#### Acceptance Criteria

- `StreamHandler` no longer owns voice audio buffers.
- Existing voice tests pass.
- Per-device buffering and final clearing are tested.
- Race tests pass.

### Phase 8 — Extract `MediaControlState`

#### Objective

Isolate media stream state, WebRTC delegation, and recording hooks.

#### Tasks

1. Create `media_control_state.go`.
2. Move media stream map.
3. Move recording manager field and setter behavior.
4. Move WebRTC engine field and setter behavior.
5. Keep existing public setter methods as delegating methods.
6. Add media state tests.

#### Acceptance Criteria

- `StreamHandler` no longer owns media stream map, recording manager, or WebRTC engine directly.
- Existing WebRTC/media tests pass.
- Race tests pass.

### Phase 9 — Clean Up `HandleMessage`

#### Objective

Make `HandleMessage` read like dispatch.

#### Tasks

1. Group message cases:
   - session/capability,
   - input/commands,
   - media,
   - voice,
   - telemetry/observations,
   - diagnostics.

2. Replace repeated error-response construction with a helper if it reduces duplication.

3. Keep metrics increments either:
   - in `HandleMessage`, or
   - in a small `ProtocolMetrics` helper.

4. Add comments documenting ownership boundaries.

#### Acceptance Criteria

- `HandleMessage` is significantly shorter and easier to scan.
- Each subsystem has focused tests.
- `go test ./...` passes.
- `go test -race ./...` passes.

### Phase 10 — Optional Package Moves

#### Objective

Move stable collaborators out of `transport` only if doing so reduces coupling.

#### Rules

1. Do not move a collaborator until it has tests.
2. Do not move a collaborator if it causes import cycles.
3. Do not move a collaborator if it forces broad public interfaces.
4. Prefer keeping transport-message-specific collaborators in `transport`.

#### Candidate Moves

| Transport File | Possible Future Package | Move Only If |
|---|---|---|
| `ui_session_state.go` | `internal/sessionui` | It can return domain effects instead of `ServerMessage`. |
| `voice_pipeline.go` | `internal/voice` | STT/wake-word behavior is separable from transport messages. |
| `diagnostics_intake.go` | `internal/diagnostics` | Bug report acknowledgement no longer needs transport types. |
| `command_dispatcher.go` | `internal/scenario/command` | Command dispatch can use scenario-domain inputs/outputs. |
| `media_control_state.go` | `internal/media` | Recording/WebRTC state can avoid transport response types. |

## Test Commands

Run after every implementation phase:

```bash
cd terminal_server && go test ./...
cd terminal_server && go test -race ./...
```

Run targeted tests during each phase:

```bash
cd terminal_server && go test ./internal/transport -count=1
```

Run broader repo checks before merging the final phase:

```bash
make server-test
make all-check
```

If `make all-check` is known to include platform-dependent client work that is unavailable locally, document the skipped parts in the PR.

## Concurrency Requirements

1. Each collaborator that owns mutable maps or slices must own its lock.
2. A collaborator must not expose mutable internal maps or slices directly.
3. Copy slices and maps before returning them.
4. Do not call slow dependencies while holding locks.
5. Avoid nested locks across collaborators.
6. If nested locks are unavoidable, document lock ordering in code.
7. Context-aware operations must accept `context.Context`.
8. Setter methods must be safe, but production wiring should still happen before streams start.

## Error Handling Requirements

1. Preserve existing sentinel errors.
2. Preserve existing control error codes.
3. Preserve `errors.Is` behavior where callers rely on it.
4. Avoid changing user-visible error text unless required and tested.
5. Add tests before changing error mapping.
6. Keep protocol violations explicit.

## Logging and Metrics Requirements

1. Keep protocol metrics visible from `StreamHandler`.
2. Avoid noisy logs for normal reconnect and replay paths.
3. Add warning logs only for unexpected or suspicious state transitions.
4. Do not add logging inside tight loops unless rate-limited.
5. Prefer structured event fields consistent with the existing event taxonomy.

## PR Plan

Use one phase per PR unless a phase is too large.

| PR | Title | Scope | Expected Size |
|---:|---|---|---|
| 1 | `refactor(server): centralize StreamHandler initialization` | Phase 1 | Small |
| 2 | `test(server): characterize StreamHandler reconnect and capability behavior` | Remaining Phase 0 gaps | Small/Medium |
| 3 | `refactor(server): extract capability lifecycle` | Phase 2 | Medium |
| 4 | `refactor(server): extract route replay store` | Phase 3 | Medium |
| 5 | `refactor(server): extract UI session state` | Phase 4 | Medium/Large |
| 6 | `refactor(server): extract command dispatcher` | Phase 5 | Medium |
| 7 | `refactor(server): extract diagnostics intake` | Phase 6 | Small |
| 8 | `refactor(server): extract voice pipeline` | Phase 7 | Medium |
| 9 | `refactor(server): extract media control state` | Phase 8 | Medium |
| 10 | `refactor(server): simplify StreamHandler dispatch` | Phase 9 | Medium |
| 11 | `refactor(server): move stable subsystems to domain packages` | Phase 10, optional | Medium/Large |

Each PR should include:

- Behavior-change statement.
- Tests added.
- Commands run.
- Race-test result.
- Rollback note.
- Updated progress log entry.

## Progress Log Format

Append entries to:

```text
plans/features/streamhandler-subsystems/progress.md
```

Use this format:

```markdown
## 2026-05-02 — Phase 1

Status: complete

Changes:
- Centralized StreamHandler constructor initialization.
- Added constructor invariant test.

Validation:
- `cd terminal_server && go test ./internal/transport -count=1`
- `cd terminal_server && go test ./...`
- `cd terminal_server && go test -race ./...`

Notes:
- No behavior changes intended.
```

## Definition of Done

This plan is complete when:

1. `StreamHandler` no longer directly owns:
   - capability lifecycle state,
   - route replay state,
   - UI replay/overlay/resume maps,
   - recent command event state,
   - voice audio buffers,
   - media stream map,
   - bug report intake field.

2. `StreamHandler.HandleMessage` reads as high-level dispatch.

3. Each extracted subsystem has focused unit tests.

4. Existing transport integration tests pass.

5. `cd terminal_server && go test ./...` passes.

6. `cd terminal_server && go test -race ./...` passes.

7. No protobuf changes were required.

8. No Flutter client changes were required.

9. `plans/features/streamhandler-subsystems/progress.md` records each completed phase.

10. `masterplan.md` and `plans/INDEX.md` link to this plan.

## Success Metrics

### Immediate

- `control_stream.go` is shorter.
- `StreamHandler` has fewer direct mutable fields.
- New test files exist for extracted collaborators.
- Constructor initialization is centralized.

### Near-Term

- Reconnect behavior can be tested without constructing the full handler.
- Capability lifecycle can be tested without UI replay setup.
- Route replay can be tested without scenario runtime setup.
- Command validation can be tested without media or UI state.
- Bug report handling can be tested without full stream setup.

### Long-Term

- Future server-side scenario work does not require editing unrelated control stream state.
- Agent-assisted changes touch smaller files.
- Race-test failures are easier to localize.
- Local multi-device failure modes are easier to diagnose.

## Risks and Mitigations

| Risk | Severity | Mitigation |
|---|---:|---|
| Behavior changes during extraction | High | Add characterization tests first; keep one phase per PR. |
| New races from lock changes | High | Collaborator-owned locks; run race tests after every phase. |
| Import cycles from premature package moves | Medium | Keep initial files under `transport`; move packages only in Phase 10. |
| Over-abstracted interfaces | Medium | Use concrete structs first; add interfaces only for test seams. |
| Reconnect regression | High | Prioritize route replay and UI replay characterization tests. |
| Scenario-specific policy remains in transport | Medium | Hide it behind policy interfaces and move later. |
| Setter methods mutate live dependencies unexpectedly | Medium | Make setters safe; document startup-only production use. |
| PRs become too large | Medium | Split by phase and update progress log after each merge. |

## Open Questions

Resolve these during implementation, not before starting Phase 1:

1. Should `UISessionState` return `ServerMessage` values initially, or should it return domain-level UI effects?
2. Should `RouteReplayStore` convert routes to `ServerMessage`, or should `StreamHandler` do that conversion?
3. Should voice finalization directly call command dispatch, or return a `CommandRequest` for `CommandDispatcher`?
4. Should command dispatch remain under `transport` or eventually move closer to `scenario`?
5. Should photo-frame transition/default behavior move to app runtime during Phase 4 or later?
6. Should recording/WebRTC dependencies be immutable after server startup?
7. Should metrics stay in `StreamHandler` permanently or move to a small `ProtocolMetrics` collaborator?

## Recommended First PR

### Title

```text
refactor(server): centralize StreamHandler initialization
```

### Scope

- Add private `newStreamHandler(control, runtime)` helper.
- Rewrite `NewStreamHandler`.
- Rewrite `NewStreamHandlerWithRuntime`.
- Add constructor invariant tests.
- Add field grouping comments.

### Behavior Change

None intended.

### Commands

```bash
cd terminal_server && go test ./internal/transport -count=1
cd terminal_server && go test ./...
cd terminal_server && go test -race ./...
```

## Recommended Second PR

### Title

```text
test(server): characterize StreamHandler reconnect and capability behavior
```

### Scope

Add or confirm tests for:

- initial capability snapshot,
- stale capability delta,
- reconnect with previous UI,
- reconnect with route replay,
- bug report unavailable path,
- invalid command request.

### Behavior Change

None intended.

### Commands

```bash
cd terminal_server && go test ./internal/transport -count=1
cd terminal_server && go test ./...
```

## Recommended Third PR

### Title

```text
refactor(server): extract capability lifecycle
```

### Scope

- Add `capability_lifecycle.go`.
- Move hello/register/snapshot/delta lifecycle logic.
- Keep UI replay and route replay in `StreamHandler`.
- Add focused lifecycle tests.
- Update capability branches in `HandleMessage`.

### Behavior Change

None intended.

### Commands

```bash
cd terminal_server && go test ./internal/transport -run 'Capability|ControlStream' -count=1
cd terminal_server && go test ./...
cd terminal_server && go test -race ./...
```
