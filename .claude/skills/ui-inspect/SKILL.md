---
name: ui-inspect
description: Visually inspect the Flutter web client and the native macOS client to find UI problems (layout, branding, rendering, inconsistencies between platforms). Use when the user asks to "check the UI", "look at the client", "find UI problems", "compare web vs macOS", or any request that requires *seeing* rendered terminal client output. Produces screenshots plus a written punch-list.
---

# Visually inspect the web and macOS clients

This skill is the repeatable procedure for actually looking at the Flutter clients. Automated lint/test tooling cannot catch layout, branding, and cross-platform rendering differences — someone has to load the app and look. This skill is that "someone."

The startup/teardown workflow is wrapped in `scripts/ui-inspect-run.sh`. Prefer the scripted path. Fall back to the manual procedure only if the script can't run (e.g. no shell access). The manual steps are kept in the appendix.

## When to use it

- "Are there any UI problems with the clients?"
- "Does the web client look right?"
- "Compare the macOS and web UI."
- Any request that requires verifying the *rendered* UI, not the Dart source.

## Prerequisites (check before running)

1. No Terminals server/client already listening on the default ports (`lsof -nP -iTCP:60739 -sTCP:LISTEN` should be empty). If one is running, call `./scripts/ui-inspect-run.sh stop || ./scripts/stop-server.sh` first.
2. You are on macOS with Xcode command line tools (`xcodebuild -version` succeeds). Required for the native client build.
3. Chrome is installed (`open -a "Google Chrome" http://localhost/` resolves).
4. For screenshotting: either the computer-use MCP tools are available (`mcp__computer-use__*`), or you have a fallback (see "Tool-API compatibility" below). Load computer-use schemas if deferred: `ToolSearch { query: "computer-use", max_results: 30 }`.

## Procedure

### 1. Start both clients via the helper script

```bash
./scripts/ui-inspect-run.sh start
```

The helper starts the web flavor first, waits for `Browser client URL:` in its log, then starts the macOS flavor and waits for `Press Ctrl+C to stop both processes.`. Running them **sequentially** (not in parallel) is deliberate — the two `run-local.sh` invocations share `.tmp/run-local-*.log` and a parallel rotation races the `mv file.N file.N+1` chain with errors like `mv …run-local-server.log.2 …log.3: No such file or directory`.

On success the script prints a block like:

```
[ui-inspect] READY
[ui-inspect]   web url : http://localhost:60739
[ui-inspect]   web pid : 12345
[ui-inspect]   mac pid : 12346
[ui-inspect]   web log : /…/.tmp/ui-inspect/web.log
[ui-inspect]   mac log : /…/.tmp/ui-inspect/macos.log
[ui-inspect]   state   : /…/.tmp/ui-inspect/state
```

Capture the web URL and both PIDs from that output — you'll use them for the capture step and for cleanup.

Useful knobs:
- `UI_INSPECT_READY_TIMEOUT=360 ./scripts/ui-inspect-run.sh start` — bump from default 240s if macOS build is slow.
- `UI_INSPECT_SKIP_MACOS=1 …` / `UI_INSPECT_SKIP_WEB=1 …` — start only one flavor.

### 2. Request computer-use access (if computer-use MCP is available)

Two `request_access` calls are sometimes needed because the name heuristic on the native app resolves a pre-existing Chrome PWA by default:

```
mcp__computer-use__request_access(
  apps=["Google Chrome", "com.example.terminalClient"],
  reason="Inspect the Terminals web client (in browser) and macOS client UI to identify UI problems."
)
```

Tier notes:
- `Google Chrome` → tier **read** (can screenshot, cannot click/type). Fine — we only need to *see* the web client; Flutter auto-connects and renders on first paint.
- `com.example.terminalClient` → tier **click** (misclassified as "terminal-like" by name; left-click works, typing does not). Fine — we inspect, we don't type.

If the first call returns "macOS Accessibility and Screen Recording are now both granted. Call request_access again immediately", do exactly that — the first call is priming the permissions dialog, the second shows the app picker.

### 3. Capture the web client

```bash
open -a "Google Chrome" <WEB_URL-printed-by-step-1>
```

Then `mcp__computer-use__screenshot`. The Flutter app takes ~5–8 seconds to compile and render in debug mode on first load; if the page is blank, wait ~8 seconds and re-screenshot.

If Chrome is on a non-primary monitor, the screenshot tool will name the monitor it captured. Use `mcp__computer-use__switch_display` with that monitor's name.

### 4. Capture the macOS client

The native app is already running (launched by `run-local.sh` via step 1). Just:

```
mcp__computer-use__switch_display(display="<primary-monitor-name>")
mcp__computer-use__screenshot()
```

### 5. Compare and produce the punch list

For each screenshot, note:

- **Branding:** browser tab title, window title, menu bar title, bundle identifier. `terminal_client` / `com.example.terminalClient` / "A new Flutter project." in `index.html` meta are Flutter scaffold defaults, not product strings.
- **Layout:** padding/margins (does the content hit the window edge?), overflow (does the FAB cover content?), misplaced accessory widgets (e.g. a "Scan LAN" button sitting *between* two form fields).
- **State consistency:** does the same connection state render the same way on web and macOS? The web auto-connects and shows a "Connected" chip; the macOS build has been observed to still show the "Scan LAN" action button under the *same* auto-connect config — that's a real inconsistency worth flagging.
- **Button states:** disabled/enabled accuracy (Disconnect enabled while already connected is fine; Connect Stream enabled while connected is a bug).
- **Debug artifacts visible in the product UI:** the red DEBUG ribbon is expected in debug-mode Flutter, but raw "Control Stream: Responses: 8 / Media routes: 0 / Active streams: 0" counters on a user-facing screen are not.

Zoom in on anything small using `mcp__computer-use__zoom region=[x0,y0,x1,y1]` — the full-screen screenshot is heavily downsampled and text styling (e.g. an underline inside a button) is hard to see otherwise.

### 6. Clean up when done

```bash
./scripts/ui-inspect-run.sh stop
```

This reads the recorded PIDs from `.tmp/ui-inspect/state`, kills each process tree (descendants first, then parent, then `SIGKILL` after 10s), and finishes by running `./scripts/stop-server.sh` as a fallback port sweep. If the state file is missing/stale it falls straight through to `stop-server.sh`.

## Tool-API compatibility

Not every environment exposes the full computer-use toolkit. The skill has been run under two shapes:

### Full computer-use MCP (preferred)

Tools: `request_access`, `screenshot`, `switch_display`, `zoom`, `left_click`, `type`, `scroll`, `open_application`, …

Use the steps above verbatim.

### Reduced toolkit (`get_app_state`, `click`, …) — no `request_access` / `screenshot` / `switch_display` / `zoom`

If the environment only exposes `get_app_state` and `click`-style tools (no `request_access`, `screenshot`, `switch_display`, `zoom`), substitute as follows. Always check what's actually in the deferred-tool list — don't assume. `ToolSearch { query: "computer-use", max_results: 30 }` reveals which variant is available.

| Task | Full API | Reduced API fallback |
| --- | --- | --- |
| Grant access | `request_access(apps=[...])` | Skip — the reduced toolkit typically doesn't gate on access; proceed to the capture step. |
| See rendered UI | `screenshot()` | `get_app_state()` returns a structured snapshot of the focused window. For pixel-accurate checks, use the Chrome-in-Chrome MCP (`mcp__Claude_in_Chrome__*`) for the web client, and fall back to asking the user to attach a screenshot for the macOS window if neither tool can capture it. |
| Pick a monitor | `switch_display(display=name)` | `get_app_state()` reports which display the focused window is on; re-focus with `click(target=<window>)` or `open_application(...)`. If the app is on the wrong display and you can't move it programmatically, ask the user to drag it to the primary display once. |
| Zoom into a region | `zoom(region=[x0,y0,x1,y1])` | Open the web client in Chrome-in-Chrome and use `read_page` / `get_page_text` / element-level inspection — text styling is exposed in the DOM without needing a magnified raster. |
| Click an element | `left_click(x, y)` | `click(target=...)` — pass the accessibility label / id from `get_app_state` instead of pixel coordinates. |

Pseudocode for the reduced-API capture step:

```
state = get_app_state()              # Chrome foreground, window on display D
# no request_access needed
# no screenshot() — inspect via get_app_state() or Chrome MCP
chrome_state = get_app_state()       # for browser tab URL / title checks
# for the macOS client, focus it and snapshot its state:
open_application(name="terminal_client")
mac_state = get_app_state()
```

Write punch-list items from the structured state (window title, element labels, visible text, geometry) rather than from pixel screenshots. Flag anything you cannot verify because the API is unavailable — don't silently skip it.

### Chrome MCP alternative

If `mcp__Claude_in_Chrome__*` is connected, prefer it for the web client regardless of which computer-use flavor is available. It's DOM-aware, text is extractable directly, and clicks target elements by id/selector. The macOS client still needs computer-use (or the user) for capture.

## Environment and access notes

### Sandboxed execution may block bind/build

Running under a restricted sandbox, you may see:

- `listen tcp 127.0.0.1:<port>: bind: operation not permitted` — localhost port binding is disallowed.
- `xcodebuild: error: ... permission denied` — native build is disallowed.
- `pkill: operation not permitted` / `ps: operation not permitted` — process listing and signaling is restricted.

All three block the workflow. When blocked:

1. Tell the user what was blocked and the command that failed.
2. Ask them to re-run the failing step in a non-sandboxed terminal: `./scripts/ui-inspect-run.sh start`.
3. Once it's up, resume from step 2 (access) or step 3 (capture) as if you had started it yourself. The state file in `.tmp/ui-inspect/state` gives you the URL and PIDs regardless of who ran `start`.

### Deterministic cleanup even without `pkill`

`ui-inspect-run.sh stop` uses the PIDs it recorded in `.tmp/ui-inspect/state`. It sends SIGTERM to each process tree via `pgrep -P` and, if that's unavailable, just SIGTERM to the parent PID. It then invokes `./scripts/stop-server.sh`, which sweeps listening ports via `lsof` and sends SIGTERM/SIGKILL on PIDs found there. Between those two paths you get teardown without needing `pkill -f …` patterns.

If both paths are blocked (e.g. sandbox disallows signaling processes you didn't spawn), ask the user to run `./scripts/ui-inspect-run.sh stop` themselves.

## Failure modes and recovery

- **Chrome-in-Chrome MCP reports "not connected":** that's the faster path (DOM-aware clicks, real text extraction). Use it if available. If not, fall back to `open -a "Google Chrome"` + `computer-use screenshot` — enough to *see* the UI.
- **Port already in use:** `./scripts/ui-inspect-run.sh stop` first (it calls `stop-server.sh`), then retry. Or set `TERMINALS_CLIENT_WEB_PORT` to a free port.
- **macOS build fails "Xcode build system has crashed":** `run-local.sh` already auto-retries up to 3 times. If it still fails, check `.tmp/run-local-client-diagnostics.log` — that script writes a full `xcodebuild` log there on failure.
- **Tab title shows truncated "termin..." only:** that *is* the UI issue (tab title is `terminal_client`), not a tooling problem. Flag it in the punch list.
- **`ui-inspect-run.sh start` says "a previous ui-inspect session is still running":** run `./scripts/ui-inspect-run.sh stop` first, then retry.
- **Readiness marker never arrives:** inspect `.tmp/ui-inspect/web.log` and `.tmp/ui-inspect/macos.log`. The script prints both paths on failure.

## Appendix: manual fallback (no helper script)

Only use this if `scripts/ui-inspect-run.sh` is unavailable for some reason (fresh checkout, shell-less environment, etc.). **Do not run the two `run-local.sh` invocations in parallel** — they race on shared log rotation.

```bash
# Web first. Wait for "Browser client URL:" before moving on.
RUN_LOCAL_OPEN_BROWSER=false ./scripts/run-local.sh --skip-bootstrap --platform web-server \
  > /tmp/run-local-web.out 2>&1 &
WEB_PID=$!
until grep -q 'Browser client URL:' /tmp/run-local-web.out 2>/dev/null; do sleep 1; done

# Then macOS. Wait for "Press Ctrl+C to stop both processes.".
RUN_LOCAL_OPEN_BROWSER=false ./scripts/run-local.sh --skip-bootstrap --platform macos \
  > /tmp/run-local-macos.out 2>&1 &
MACOS_PID=$!
until grep -q 'Press Ctrl+C to stop both processes.' /tmp/run-local-macos.out 2>/dev/null; do sleep 1; done

echo "web pid=$WEB_PID  macos pid=$MACOS_PID"
```

Cleanup (in order — most deterministic first, last-resort patterns last):

```bash
kill "$WEB_PID" "$MACOS_PID" 2>/dev/null || true
./scripts/stop-server.sh || true
# Only if the above didn't finish the job:
pkill -f 'flutter run -d web-server' 2>/dev/null || true
pkill -f 'terminal_client/build/macos/Build/Products/Debug/terminal_client' 2>/dev/null || true
```

## Why this skill exists

Without it, each UI-inspection request re-derives the same workflow: which ports, which flags, which MCP, which display, how to get past the Flutter-web compile delay, how to handle the "com.example.terminalClient is a terminal" tier misclassification, how to avoid the log-rotation race, how to clean up without `pkill`. The procedure is short but error-prone when reconstructed from memory, and the user has asked for UI checks often enough that it's worth writing down.
