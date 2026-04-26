# Bug Reporting and Diagnostics

Durable reference for the implemented local bug-reporting pipeline.

## Scope

The current implementation provides local-only reporting and diagnostics capture:

- Wire contract in [api/terminals/diagnostics/v1/diagnostics.proto](../api/terminals/diagnostics/v1/diagnostics.proto).
- Control-plane transport via `bug_report` request and `bug_report_ack` response payloads.
- Durable server persistence under `terminal_server/logs/bug_reports/<YYYY-MM-DD>/`.
- Correlated server events (`bug.report.*`) in the event log.
- Admin and public intake HTTP surfaces.

## Endpoints

- `POST /bug/intake`
- `GET /bug?device=<id>`
- `GET /admin/bugs`
- `GET /admin/bugs/<report_id>`
- `GET /admin/api/bugs`
- `GET /admin/api/bugs/<report_id>`
- `POST /admin/api/bugs/<report_id>/confirm`

## Stored Artifacts

For each report:

- JSON record: `terminal_server/logs/bug_reports/<date>/<report_id>.json`
- Optional screenshot: `terminal_server/logs/bug_reports/<date>/<report_id>.screenshot.png`
- Optional audio clip: `terminal_server/logs/bug_reports/<date>/<report_id>.audio.wav`

## Runtime Behavior

- Client collects a bounded `ClientContext` snapshot at submit time.
- Reports are ack-driven and support queue/replay on reconnect.
- Server enriches reports with subject snapshot and recent event tail.
- Autodetected reports can merge with user-filed reports for the same subject in a short dedup window.

## Validation

Primary tests:

- `terminal_server/internal/diagnostics/bugreport/service_test.go`
- `terminal_server/internal/admin/bugs_test.go`
- `terminal_server/internal/transport/control_stream_test.go`
- `terminal_client/test/widget_test.dart`

Repository gates:

- `make all-check`
