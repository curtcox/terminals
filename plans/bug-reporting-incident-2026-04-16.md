# Bug Report Incident: Bug Reporting Reliability (2026-04-16)

## Incident Status

Mitigated in code on 2026-04-16. Follow-up hardening items remain open.

## Scope

This incident tracks failures in the bug-reporting feature itself during local development runs of the Flutter web client and Go server.

## Reported Problem

User filed several bug reports and provided time-derived token words/codes (for example `sky` and its paired code). The system could not find corresponding records.

## Expected Behavior

- On submit, each report should produce:
  - a server-side `bug.report.filed` event
  - searchable token attributes (`bug_token_word`, `bug_token_code`)
  - a durable report artifact under `${TERMINALS_LOG_DIR}/bug_reports/<YYYY-MM-DD>/<report_id>.json` (default local path during `make run-local`: `terminal_server/logs/bug_reports/...`)
- Client status should only indicate success after server ack.

## Actual Behavior

- No matching token traces were found in server logs.
- No persisted `bug_reports` directory or report JSON artifacts were found.
- Client could show optimistic progress states that did not guarantee server persistence.

## Reproduction Notes (from this thread)

1. Start local stack with `make run-local`.
2. Submit bug report from web UI.
3. Attempt to find token in:
   - `.tmp/run-local-client.log`
   - `.tmp/run-local-server.log`
   - `terminal_server/logs/terminals.jsonl`
   - `terminal_server/logs/bug_reports/<YYYY-MM-DD>/`
4. Observe missing token matches and missing persisted report files.

## Root Causes

1. **Success semantics too early**
   - Client state did not strictly enforce "filed only after ack".
2. **Disconnected submit drop risk**
   - Submissions while not fully connected could be missed unless users manually connected and retried.
3. **Operational observability gap**
   - When ingestion failed, there was no durable artifact to inspect, and token search gave no result.

## Fixes Implemented

1. **Ack-gated completion**
   - Report becomes "filed" only after `BugReportAck`.
   - Timeout and stream-close paths mark report as "not confirmed".
2. **Offline queue + replay**
   - Bug reports created while disconnected are queued.
   - Queue flushes automatically when stream connects.
3. **Auto-connect on offline submit**
   - Submitting while disconnected now triggers connect flow automatically.
4. **Token observability**
   - Token word/code included in client tags and source hints.
   - Server emits token attributes in structured `bug.report.*` events.
5. **Additional automatic metadata**
   - Host/port, status, connection state, UI root, stream/route counts, and queue/pending counts are captured.
6. **User input policy**
   - Description/tags remain optional.
   - Token presented in text, audio, and QR formats.

## Remaining Gaps / Follow-ups

1. Add an automated integration test that:
   - submits a report with stream initially disconnected
   - validates eventual ack
   - validates presence of token fields in server logs
   - validates on-disk JSON artifact creation
2. Improve admin lookup ergonomics for token word/code:
   - current workaround: filter by `tag` using `bug_word:<word>` or `bug_code:<code>`
   - follow-up: add dedicated `bug_token_word` / `bug_token_code` filter inputs
3. Add a startup health check warning when the server process is not alive while the client is running.

## Verification Checklist

- [ ] Submit report while connected; verify ack and token in logs.
- [ ] Submit report while disconnected; verify auto-connect, queue flush, ack, and token in logs.
- [ ] Confirm `terminal_server/logs/bug_reports/<date>/<report_id>.json` exists (or `${TERMINALS_LOG_DIR}/bug_reports/...` when overridden).
- [ ] Confirm no "filed" UI state appears without ack.
