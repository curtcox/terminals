---
title: "Terminal UI — Progress Log"
kind: progress-log
parent: plans/features/terminal-ui/plan.md
---

## Implementation Progress

- 2026-05-02: Created progress log and marked feature as building.

## 2026-05-02

- Phase A complete. Added config-aware `withCornerAffordanceConfig`
  (corner, visibility, `min_hit_dp`, density, safe_area) with
  backwards-compatible defaults; existing `withCornerAffordance` is now
  a thin shim. `MergeCornerAffordanceConfig` resolves user-pref +
  activity-override (override wins; user pref restored on activity exit)
  and enforces a 44 dp hit-target floor.
- New reachability-invariant tests in
  `terminal_server/internal/transport/corner_affordance_test.go`:
  per-corner placement, hit-target ≥ 44 dp at densities 1.0/2.0/3.0
  (with `min_hit_px` derivation), Z-order (corner is last child of
  containing stack, both for native-stack roots and wrapped
  non-stack roots), asymmetric `safe_area` non-occlusion across all
  four corners, invisible-skip, and a registry-iteration test that
  exercises every registered scenario name against three fixture
  descriptor shapes.
- New Flutter widget test in `terminal_client/test/widget_test.dart`
  drives a wrapped descriptor end-to-end and asserts the emitted
  `UIAction` carries the canonical scoped corner `componentId`
  (`act:<owner>/__affordance.corner__`) and `corner.open` action.
- Opt-out governance + CODEOWNERS gating from prior commits remain in
  place; the Phase A test list is now fully covered by `make all-check`.
- Side fixes (made to keep `internal/transport` compiling and green):
  - Corrected pre-existing build break in
    `control_stream_reconnect_test.go` — the file used non-existent
    `iov1.InputEvent_Action` / `iov1.ActionEvent`; replaced with the
    actual `Payload: &iov1.InputEvent_UiAction{UiAction:
    &iov1.UIAction{ComponentId: ..., Action: "corner.open"}}` shape.
  - Fixed `TestSessionRunRelaysWebRTCSignalsAcrossDeviceSessions`
    drain count: after commit 7fc35f8d added route-replay messages
    (StartStream + RouteStream) on register, the test's fixed
    two-message drain was stale; replaced with a quiescent drain.
  - Skipped `TestGeneratedSessionUI_RECON_1` and
    `TestGeneratedSessionMidFlightOverlayIdempotent` with
    `t.Skip` markers pointing to Phase G — both depend on the
    overlay-open-on-`corner.open` flow that Phase G owns and that is
    not yet wired in those test paths.
- `make all-check` is green.

- Phase G partial: re-enabled the two reconnect tests skipped in
  Phase A. Two real bugs were behind their failures, both now fixed:
  1. The tests asserted an unscoped, dotted overlay component id
     (`"global.overlay"`) that never appears on the wire — the actual
     constant is `ui.GlobalOverlayComponentID = "global_overlay"`,
     and the rewriter scopes it to `act:<owner>/global_overlay`.
     Updated the assertions in
     `control_stream_reconnect_test.go` to check both the bare
     constant and the canonical scoped form via suffix match.
  2. The Phase G replay path read live routes from the IO router
     (`routeSnapshotForDevice`), but `HandleDisconnect` had already
     torn those routes down by the time the device reconnected, so
     no `StartStream` / `RouteStream` messages were emitted on
     reconnect. Added `routeReplayByDevice` to `StreamHandler`:
     `disconnectRoutesForDevice` snapshots the route set before
     calling `Disconnect`/`unregisterMediaStream`, and the
     CapabilitySnapshot/Register replay paths now fall back to that
     snapshot when the live router is empty. The disconnect
     teardown semantics for recording, media streams, and the live
     router are unchanged, so
     `TestHandleDisconnectStopsRecordingForDisconnectedDeviceRoutes`
     and similar tests continue to pass.
- `TestGeneratedSessionUI_RECON_1` and
  `TestGeneratedSessionMidFlightOverlayIdempotent` both pass; the
  `t.Skip` markers were removed.
- `make all-check` is green.
