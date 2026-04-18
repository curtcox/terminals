---
name: bugword
description: Find and work a bug by its user-supplied token word (e.g. "work the 'photo' bug", "debug bug word sky", "what is the bug for the word corner?"). Reads the server event log and the on-disk bug report JSON, extracts the reproduction signal, then diagnoses and (with confirmation) fixes the underlying issue.
---

# Work a bug by its token word

When the user files a bug from the Flutter client, the server assigns a short memorable **token word** (plus a numeric code) and writes two things:

1. A `bug.report.filed` / `bug.report.autodetected` event in `terminal_server/logs/terminals.jsonl` tagged with `bug_token_word:<word>` and `bug_token_code:<code>`.
2. A durable JSON artifact at `terminal_server/logs/bug_reports/<YYYY-MM-DD>/<report_id>.json`, with a sibling `.screenshot.png` when a screenshot is attached.

The token is the user's handle for the bug — they'll say "work the 'photo' bug" or "what happened with the word sky?". This skill turns that word into a concrete diagnosis.

See [plans/bug-reporting-incident-2026-04-16.md](../../../plans/bug-reporting-incident-2026-04-16.md) for the reporting pipeline background.

## Procedure

### 1. Locate the event(s) for the word

```bash
grep -n "\"bug_token_word\":\"<WORD>\"" terminal_server/logs/terminals.jsonl
```

Each match yields a `report_id`, a `report_path` (relative to repo root), a `correlation_id` of the form `bug:<report_id>`, and the `subject_device_id`.

If there are no matches, also check:

- `bug_code:<WORD>` tag (older clients): `grep '"bug_code:.*-<WORD>"' terminal_server/logs/terminals.jsonl`
- `.tmp/run-local-server.log` and `.tmp/run-local-client.log` if the user was running `make run-local`.
- Any override directory from `$TERMINALS_LOG_DIR`.

If still nothing, stop and tell the user — do not guess a report. The 2026-04-16 incident showed that a missing token usually means the report never reached the server, not that we need to search harder.

### 2. Load the bug report (skip the inline screenshot)

The JSON report contains an inline base64 `screenshot_png` field that is enormous (tens of thousands of tokens) and useless for diagnosis — the image is already on disk at `screenshot_path`. Do **not** `Read` the JSON file directly. Instead, extract the useful fields:

```bash
python3 - <<'PY'
import json, sys
path = "<report_path>"
with open(path) as f:
    d = json.load(f)
print(json.dumps({
    "summary": d.get("summary"),
    "client_context": d.get("report", {}).get("client_context"),
    "server_context": d.get("report", {}).get("server_context"),
    "attachments": [a for a in d.get("report", {}).get("attachments", []) if a.get("kind") != "screenshot_png"],
}, indent=2))
PY
```

Key fields to look at:

- `summary.description` — what the user said went wrong.
- `summary.tags` — includes `bug_word:<word>` and `bug_code:<code>`.
- `client_context.connection.last_status` — often the literal error string.
- `client_context.runtime.recent_logs` — last few client-side log lines before the report.
- `client_context.runtime.active_ui_root` — which screen the user was on.
- `client_context.error_capture` — uncaught Flutter errors, if any.
- `client_context.identity.platform` / `os_version` — needed to judge platform-specific bugs (e.g. `macOS:web` means Flutter Web, not desktop).

### 3. View the screenshot

If the summary references a `screenshot_path`, `Read` the PNG at that path. Claude Code renders it inline.

### 4. Pull the full server-side trace for the bug

```bash
grep "\"correlation_id\":\"bug:<report_id>\"" terminal_server/logs/terminals.jsonl
```

This surfaces every server event emitted while handling the report (`bug.report.filed`, acks, follow-on errors). Also scan log lines in the same time window on the `subject_device_id` for upstream causes.

### 5. Diagnose

Combine description + `last_status` + screenshot + client logs to form a hypothesis. Then locate the offending code — usually under `terminal_client/lib/` for client-only bugs, `terminal_server/internal/` for server bugs, `api/proto/` if the contract is wrong. For client errors mentioning a specific API (e.g. `InternetAddress.anyIPv4`), `grep` that symbol in `terminal_client/lib/` to find the call site.

Report the hypothesis to the user with: the report_id, the one-line user description, the root-cause summary, and the file(s) you'd change. Ask before editing unless the user already said "fix it".

### 6. Fix (only after user confirms)

Follow the project's core rules from [CLAUDE.md](../../../CLAUDE.md):

- Never add scenario-specific behavior to the client.
- New client/server messages go through `api/proto/`, not ad-hoc JSON.
- Keep AI providers behind interfaces.

After the change, run the relevant gate(s): `make client-test` / `make server-test` / `make all-check`. If there's a matching use-case ID for the broken feature, also run `make usecase-validate USECASE=<ID>` (see the `usecase-validate` skill).

## Worked example — word "photo" (2026-04-18)

1. `grep '"bug_token_word":"photo"' terminal_server/logs/terminals.jsonl` → `report_id=bug-20260418t222807.151-44d18db8`, `report_path=logs/bug_reports/2026-04-18/bug-20260418t222807.151-44d18db8.json`.
2. Extracting the report (skipping `screenshot_png`) yields:
   - `summary.description`: "The Scan LAN button reports an error."
   - `client_context.connection.last_status`: `"Discovery error: Unsupported operation: InternetAddress.anyIPv4"`
   - `client_context.identity.platform`: `flutter`, `os_version`: `macOS:web` — i.e. Flutter Web.
3. Screenshot at `logs/bug_reports/2026-04-18/bug-20260418t222807.151-44d18db8.screenshot.png` confirms the error banner on the connect screen.
4. Diagnosis: `InternetAddress.anyIPv4` is from `dart:io`, which isn't available on Flutter Web. Scan-LAN discovery needs a web-safe branch. Next step: grep `terminal_client/lib/` for `InternetAddress` / `anyIPv4` and gate the code path behind `kIsWeb` (or disable the button on web).

This is the shape every invocation of this skill should produce: word → report_id → description + last_status + screenshot → concrete file to change.

## Out of scope

- Filing new bug reports (that's the client UI).
- Modifying the bug-reporting pipeline itself — see [plans/bug-reporting-incident-2026-04-16.md](../../../plans/bug-reporting-incident-2026-04-16.md) and its follow-ups.
- Inventing token words the user didn't supply.
