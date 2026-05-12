---
title: "Terminal UI Use Cases"
family: UI
ids: [UI1, UI2, UI3, UI4, UI5, UI6, UI7, UI8, UI9, UI10]
---

# Terminal UI

End-to-end validation for [plans/features/terminal-ui/plan.md](../../plans/features/terminal-ui/plan.md) (Phase I). IDs `UI1`–`UI10` map to the plan’s former working names `UI-IDLE-1`, `UI-CORNER-1`, `UI-CORNER-2`, `UI-PRIV-1`, `UI-PRIV-2`, `UI-WAKE-1`, `UI-WAKE-2`, `UI-ROT-1`, `UI-RECON-1`, and `UI-INVARIANT-1`.

| # | As a … | I would like to … | So that … |
|---|--------|-------------------|-----------|
| UI1 | **Resident** | see an idle tablet show server-driven ambient UI with a reachable corner menu affordance | I can open the system menu from any idle screen |
| UI2 | **Resident** | tap the corner affordance and open the menu overlay | I can reach apps and settings without leaving the underlying screen |
| UI3 | **Resident** | have default overlay routing keep background audio live while pointer goes to the menu | ambient audio continues while I navigate the overlay |
| UI4 | **Resident** | toggle privacy mode and have mic/camera capability withdrawn atomically on the wire | capture stops cleanly at the capability cutover |
| UI5 | **Resident** | stay in privacy mode without wake-word streaming or client chrome privacy indicators | privacy is expressed only through capabilities and server UI |
| UI6 | **Resident** | speak a wake phrase on one terminal and have the server react | voice intents work on a single device |
| UI7 | **Resident** | have duplicate wake detections from two nearby terminals dedupe to one intent | the room does not get double-triggered |
| UI8 | **Resident** | rotate or resize the display and emit capability deltas the server can use | layout and overlays stay coherent across geometry changes |
| UI9 | **Resident** | reconnect while the menu overlay is open and see both layers restored | transient network loss does not strand UI state |
| UI10 | **Operator** | rely on CI to enforce the corner affordance reachability invariant for every registered main-layer scenario | skipped wrappers cannot ship without review |
