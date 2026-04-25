---
title: "Bug Reporting and Diagnostics"
kind: plan
status: planned
owner: unowned
validation: none
last-reviewed: 2026-04-25
---

# Bug Reporting and Diagnostics

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context. This plan defines how any user, on any device, at any time, can file a bug report against the system — including bugs in devices whose own input mechanisms are broken. The mechanism is **modality-agnostic** (no single input device required) and **cross-device** (a bug can be filed about device A from device B).

## Goals

- Any person at any terminal at any time can file a bug report.
- No single IO modality — keyboard, microphone, camera, touchscreen, pointer — is required to file a report.
- A bug can be reported *about* device A from device B, so devices with no working input are still reportable.
- The server captures the full context of the reporter device and the subject device sufficient to diagnose and fix the bug.
- Reports are stored durably on the server and correlatable with the existing event log.

## Non-Goals

- Shipping reports off-box. Reports are strictly local-filesystem + admin HTTP in this phase; remote triage (Sentry, GitHub Issues, Slack) is a later adapter concern behind an interface and must not shape the on-disk format.
- Replacing the event log. The event log remains authoritative for server traces; bug reports reference it via `correlation_id`.
- Automated bug triage / deduplication / severity scoring.

## Architectural Fit

Follows the project's core rules (see [CLAUDE.md](../../CLAUDE.md)):

1. **No scenario-specific client code.** Every on-device entry point is built from the existing `ui/v1` primitives and the existing `InputEvent` / `SensorData` / voice streams. The client adds only a generic "collect my context" capability, not bug-flow logic.
2. **All wire contracts in protobuf.** New package `api/terminals/diagnostics/v1/diagnostics.proto`.
3. **Pluggable server-side.** Intake adapters (SIP, email, webhook) and downstream sinks (issue trackers) sit behind Go interfaces in a new `internal/diagnostics/bugreport` package.
4. **Server-driven UI.** The "Report a bug" affordance is composed by the server into every scenario's UI tree; the client just renders it.

## Mechanisms — Multi-Modal, Multi-Source

The server guarantees that **at least one on-device mechanism** is available for every live device based on its declared capabilities (see `capabilities.proto`), and that **at least one off-device mechanism** exists for every device regardless of local hardware state. Any single mechanism is sufficient to file a complete report.

### On-device (subject reports itself)

| # | Mechanism | Required capability | Where it lives |
|---|-----------|---------------------|----------------|
| 1 | Server-driven "Report a bug" button | Screen + tap/click/pointer | A shared `withBugReportAffordance` scenario wrapper injects a button into every root UI tree. |
| 2 | Reserved long-press gesture | Touchscreen | A transparent `gestureArea` overlay primitive matches a three-finger long-press. |
| 3 | Shake intent | Accelerometer | Scenario engine subscribes to `SensorData`; a shake pattern fires an `intent.bug_report.open`. |
| 4 | Keyboard shortcut | Physical keyboard | Terminal scenario intercepts `Ctrl+Alt+B` before forwarding to PTY; other scenarios install an equivalent handler. |
| 5 | Voice intent | Microphone + wake word | Existing voice pipeline maps "report a bug" / "file a bug on the kitchen screen" to `intent.bug_report.open` with an optional subject resolved via the placement engine. |

### Off-device (third-party reports about a broken subject)

| # | Mechanism | Reporter requirement | Subject requirement |
|---|-----------|----------------------|---------------------|
| 6 | QR code on idle/standby UI | Any device with a camera or a QR-scanning app | Subject must render *something* — the QR encodes `/bug?device=<subject_id>`. |
| 7 | NFC tag | Any NFC-capable phone | None — tag is passive; encodes the same URL. |
| 8 | Admin dashboard `/admin/bugs/new` | Any browser on LAN | None — subject chosen from a device dropdown including recently-disconnected devices. |
| 9 | SIP bug line (reserved extension) | Any phone (internal or external SIP) | None — caller speaks the report; existing voice pipeline transcribes. |
| 10 | `POST /bug/intake` JSON endpoint + email-in adapter | Any system with network access | None — subject named by id, zone, or free-text description. |

### Server-initiated

| # | Mechanism | Trigger |
|---|-----------|---------|
| 11 | Autodetect | Heartbeat timeout, never-registered, repeated reconnect loop, or unhandled `ControlError` → server files a `suspected_failure` pending report with last-known state. Any user can confirm with one tap from any mechanism above. |

## Client-Side Information Collected

A new `diagnostics.v1.ClientContext` is collected **at the moment of report**, redacted and size-capped before send. Only the reporter device contributes this; the subject device's equivalent state is filled in server-side (see below).

### Identity and build

- `device_id`, `device_name`, `device_type`, `platform`
- Client build version, git SHA, build timestamp
- OS version, locale, timezone, device system-clock offset vs server (derived from existing `ClockSample`)

### Capability manifest

- The full `DeviceCapabilities` proto the device last reported (screen, keyboard, pointer, touch, speakers, microphone, camera, sensors, connectivity, battery, edge)

### Runtime state

- Active scenario and activation ids this device is attached to
- Current UI root (serialized `ui.v1.Node`) exactly as last set by the server
- Last N (cap 32) UI updates / patches / transitions applied
- Last N (cap 32) `UIAction` messages the user sent
- Active streams, routes, recent WebRTC signals (data already tracked in `terminal_client/lib/main.dart`)
- Last N (cap 200) console / debug log lines from a ring buffer

### Connection health

- Time of last successful heartbeat, current reconnect attempt count, last connection status string
- Recent `ControlError`s received from the server
- Client-side async-writer drop counters (if any)

### Hardware state

- Battery level and charging flag
- Connectivity online flag, last known RTT-to-server
- Most recent accelerometer / ambient-light snapshot (if available)
- Screen dimensions, density, orientation, `devicePixelRatio`

### Error capture

- Most recent caught `FlutterError` with stack trace, via a `FlutterError.onError` hook installed in client bootstrap

### User-supplied content

- Free-text description (optional)
- Short audio clip (optional, microphone only, default-off)
- Screenshot (optional, via `RenderRepaintBoundary`, default-off)
- Structured tag set: `dead_screen`, `no_audio`, `unresponsive`, `wrong_content`, `lost_connection`, `input_ignored`, `ui_glitch`, `other`

### Redaction rules

Before send, the client strips:
- Any `KeyEvent.text` content from the UI-action history (the user's typed input).
- Any `TouchEvent` coordinate streams older than 30 s.
- Any URL query parameters in the UI-tree snapshot that match a configurable deny pattern (secrets/tokens).

## Server-Side Enrichment

On arrival, a new `internal/diagnostics/bugreport` package enriches the report with facts the reporter cannot know:

- Subject device's `device.Manager` entry — id, name, type, platform, placement (zone/roles), current `State`, last heartbeat.
- Subject's current active activations and current server-sent UI root.
- Last N `eventlog` events for the subject device id, filtered to the window `[report.timestamp - 5 min, report.timestamp]`.
- `subject_offline=true` flag if the subject is not currently registered or heartbeat-stale.
- New `correlation_id` threaded through `eventlog.WithCorrelation`; every follow-up event for this subject in a post-report window inherits it.
- Cross-link to any existing `bug.report.autodetected` record for the same subject within a recency window (dedup).

## Wire Contract

New package: `api/terminals/diagnostics/v1/diagnostics.proto`.

- `BugReport` — `report_id`, `reporter_device_id`, `subject_device_id` (may equal reporter or be empty for "unknown"), `source` enum (SCREEN_BUTTON, GESTURE, SHAKE, KEYBOARD, VOICE, QR, NFC, ADMIN, SIP, WEBHOOK, AUTODETECT, OTHER), `description`, repeated `tags`, `timestamp`, `client_context` (the `ClientContext` above), optional inline `screenshot_png` and `audio_wav` bytes *or* opaque blob refs.
- `BugReportAck` — persisted `report_id`, server-assigned `correlation_id`, `status` enum (FILED, MERGED_WITH_AUTODETECT, REJECTED).
- `ClientContext` — identity, capabilities, runtime state, connection health, hardware state, error capture, tags. All collections bounded.

Control-plane integration (`terminals.control.v1`):
- Adds `BugReport bug_report = N` to `ConnectRequest.payload`.
- Adds `BugReportAck bug_report_ack = M` to `ConnectResponse.payload`.

HTTP intake (admin + external):
- `POST /bug/intake` accepts the JSON form of `BugReport` (protobuf JSON mapping).
- `GET /bug?device=<id>` serves a mobile-friendly intake form that submits to the same endpoint.

## Server Storage

- Durable JSON per report: `logs/bug_reports/<YYYY-MM-DD>/<report_id>.json`. Binary attachments as siblings: `<report_id>.screenshot.png`, `<report_id>.audio.wav`.
- Structured event emitted via existing `eventlog.Emit`:
  - `bug.report.filed` — every user-filed report.
  - `bug.report.autodetected` — every server-generated pending report.
  - `bug.report.confirmed` — when a user confirms an autodetected report.
  - Attrs include `report_id`, `correlation_id`, `reporter_device_id`, `subject_device_id`, `source`, `tags`, summary counters. Large fields are **not** duplicated into the event log; the event carries a pointer `report_path` to the JSON file.

## Admin Surfaces

New routes in `internal/admin`:

- `GET /admin/bugs` — list, filter by subject / reporter / source / tag / time window / confirmed-or-pending.
- `GET /admin/bugs/<id>` — full context, screenshot, audio, linked event-log trace (by `correlation_id`), subject snapshot at report time.
- `GET /admin/bugs/new?device=<id>` — third-party intake form (same page served by `GET /bug?device=<id>` for mobile clients; identical submission path).
- `GET /admin/api/bugs` — JSON list.
- `POST /admin/api/bugs/<id>/confirm` — mark an autodetected report confirmed.

## Driving Example — "This terminal screen isn't working"

Concrete current state: the Flutter web client at `http://localhost:58998/` is unreachable (connection refused; nothing listening on that port). The subject device has never completed control-plane registration this run. The mechanisms degrade as follows:

| Mechanism | Works? | Why |
|---|---|---|
| On-screen button on the broken terminal | No | Nothing rendering. |
| Reserved gesture or keyboard shortcut on the broken terminal | No | No process to intercept input. |
| QR code on the broken terminal | No | Nothing rendering; nothing to scan. |
| NFC tag on the broken device | Yes | Passive; phone reads the URL; opens `/bug?device=web-server-*` served by admin HTTP. |
| Voice on another device ("file a bug on the laptop screen") | Yes | Placement engine resolves the subject; independent of the subject's health. |
| Admin dashboard from a phone or laptop | Yes | Subject selected from the recently-disconnected list. |
| SIP bug line | Yes | Requires only a phone. |
| `POST /bug/intake` / email-in | Yes | Independent of any local hardware. |
| Server autodetect | Yes | Subject never registered → `bug.report.autodetected` filed with "client never connected" tag; any user confirms with one tap from any channel. |

At least five independent paths succeed with the subject fully dead, and the stored report contains: reporter's full `ClientContext`, subject's server-side last-known record (here: "never registered; last known at mDNS host X"), `subject_offline=true`, autodetect linkage, and a `correlation_id` threaded into the event log. This is the validation: the worst case — the device you would normally file the report *on* is itself the broken one — is covered by multiple non-overlapping mechanisms.

## Phased Rollout

Each phase is independently mergeable and independently useful.

1. **Core pipeline.** `diagnostics.v1` proto; server handler; durable store at `logs/bug_reports/`; `bug.report.filed` event; `/admin/bugs` list and `/admin/bugs/<id>` detail. No client UI yet; a `POST /bug/intake` is sufficient to smoke-test end to end.
2. **Client context capture.** Implement `ClientContext` collection in `terminal_client/lib`: capabilities, version, current UI root, recent-log ring buffer, recent UI-action ring buffer, connection health, battery, `FlutterError` hook. Gated behind an explicit "collect and send" request.
3. **On-device entry points** — server-composed, no scenario-specific client code. Adds the `withBugReportAffordance` scenario wrapper, `intent.bug_report.open` intent, keyboard shortcut, shake pattern, voice phrase, and three-finger-long-press gesture.
4. **Third-party reporting.** `/admin/bugs/new` form; `GET /bug?device=<id>` short-URL page; `POST /bug/intake` JSON endpoint; server-side enrichment from `device.Manager`, scenario runtime, and `eventlog`. QR code on every standby/photo-frame card.
5. **Autodetection and dead-device fallback.** Heartbeat-timeout / never-registered / reconnect-loop detectors file `suspected_failure` pending reports; SIP bug-line extension; email-in adapter behind an interface; NFC tag URL doc.

## Security and Privacy Considerations

- The admin HTTP and `/bug/intake` endpoints inherit the existing trusted-LAN assumption. If that assumption is tightened (see masterplan §10), add TLS mutual auth at the transport layer; the protocol does not change.
- Redaction rules above keep typed input, touch streams, and secret-like URL parameters out of stored context.
- Audio and screenshot collection is opt-in per report, never automatic.
- Stored reports inherit the filesystem trust boundary of the existing event log.

## Open Questions

- Should we surface a live-tail of `bug.report.*` events in the admin dashboard the way `/admin/logs` does for the event log? (Low-cost addition; defer to phase 1 if cheap.)
- Max inline attachment size vs. forcing a separate upload round-trip? Default to a 2 MB inline cap and a later "request full artifact" round-trip for anything larger.
- Retention policy for `logs/bug_reports/`? Defer to the same rotation model as the event log, but with a longer default horizon.

## Incident Addendum (2026-04-16)

See [bug-reporting-incident-2026-04-16.md](../incidents/2026-04-16-bug-reporting.md) for the full report from local testing and iterative fixes.

Summary of observed failure:

- Multiple user-submitted bug reports produced no durable server-side evidence:
  - no `bug.report.filed` events in `terminal_server/logs/terminals.jsonl`
  - no `bug_token_word` / `bug_token_code` matches
  - no `logs/bug_reports/<date>/<report_id>.json` artifacts
- User-facing status suggested success too early (before confirmed server ack), creating false confidence.

Summary of mitigations implemented in code:

- Ack-driven state handling: reports are "filed" only after `BugReportAck`; otherwise they become "not confirmed" on timeout/stream close.
- Offline queue and replay: reports created while disconnected are queued and replayed after connection.
- Auto-connect on submit: a queued report triggers stream connect automatically instead of requiring manual connect.
- Correlation token enrichment: token word/code are embedded in tags/source hints and propagated to server event attributes.
- Optional input UX: description/tags remain optional; reference token is presented as text + audio + QR.
- Expanded automatic source hints: connection, UI, stream counts, and queue/pending counters are captured at submit time.
