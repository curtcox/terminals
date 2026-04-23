---
name: ui-inspect
description: Visually inspect the Flutter web client and the native macOS client to find UI problems (layout, branding, rendering, inconsistencies between platforms). Use when the user asks to "check the UI", "look at the client", "find UI problems", "compare web vs macOS", or any request that requires *seeing* rendered terminal client output. Produces screenshots plus a written punch-list.
---

# Visually inspect the web and macOS clients

This skill is the repeatable procedure for actually looking at the Flutter clients. Automated lint/test tooling cannot catch layout, branding, and cross-platform rendering differences — someone has to load the app and look. This skill is that "someone."

## When to use it

- "Are there any UI problems with the clients?"
- "Does the web client look right?"
- "Compare the macOS and web UI."
- Any request that requires verifying the *rendered* UI, not the Dart source.

## Prerequisites (check before running)

1. No Terminals server/client already listening on the default ports (`lsof -nP -iTCP:60739 -sTCP:LISTEN` should be empty).
2. Computer-use MCP tools are available (`mcp__computer-use__*`). If they aren't loaded, load them via `ToolSearch { query: "computer-use", max_results: 30 }`.
3. The user is on macOS with Xcode command line tools (`xcodebuild -version` succeeds). Required for the native client build.
4. Chrome is installed (`open -a "Google Chrome" http://localhost/` resolves).

## Procedure

### 1. Start both clients in the background (in parallel)

Each `run-local.sh` invocation spawns its *own* server on the next available port, so they don't collide. Run them in parallel — the macOS Flutter build takes 1–2 minutes.

```bash
# Web: server + `flutter run -d web-server`
RUN_LOCAL_OPEN_BROWSER=false ./scripts/run-local.sh --skip-bootstrap --platform web-server \
  > /tmp/run-local-web.out 2>&1 &

# macOS: server + native .app build and launch
RUN_LOCAL_OPEN_BROWSER=false ./scripts/run-local.sh --skip-bootstrap --platform macos \
  > /tmp/run-local-macos.out 2>&1 &
```

Run these via `Bash(run_in_background=true)`. Poll each log until you see:

- Web ready: `lib/main.dart is being served at` in `/tmp/run-local-web.out` (~20s)
- macOS ready: `Press Ctrl+C to stop both processes.` line printed *after* "Building macOS client..." in `/tmp/run-local-macos.out` (~1–2 min)

The web client URL is printed as `Browser client URL: http://localhost:<port>` (default 60739). Capture that port — you will need it.

### 2. Request computer-use access

Two request_access calls are needed because the name heuristic on the native app resolves a pre-existing Chrome PWA by default:

```
mcp__computer-use__request_access(
  apps=["Google Chrome", "com.example.terminalClient"],
  reason="Inspect the Terminals web client (in browser) and macOS client UI to identify UI problems."
)
```

Tier notes:
- `Google Chrome` → tier **read** (can screenshot, cannot click/type). That's fine here — we only need to *see* the web client; Flutter auto-connects and renders on first paint.
- `com.example.terminalClient` → tier **click** (native macOS app is misclassified as "terminal-like" because of the name; left-click works, typing does not). Also fine — we are inspecting, not typing into fields.

If the first call returns "macOS Accessibility and Screen Recording are now both granted. Call request_access again immediately", do exactly that — the first call is priming the permissions dialog, the second shows the app picker.

### 3. Capture the web client

```bash
open -a "Google Chrome" http://localhost:<port>/
```

Then `mcp__computer-use__screenshot`. The Flutter app takes ~5–8 seconds to compile and render in debug mode on first load; if the page is blank, wait ~8 seconds and re-screenshot.

If Chrome is on a non-primary monitor, the screenshot tool will name the monitor it captured. Use `mcp__computer-use__switch_display` with the name of the monitor Chrome is on.

### 4. Capture the macOS client

The native app is already running (launched by `run-local.sh`). Just:

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
./scripts/stop-server.sh || true
pkill -f 'flutter run -d web-server' || true
pkill -f 'terminal_client/build/macos/Build/Products/Debug/terminal_client' || true
```

Or let the user close the native window — the `run-local.sh` monitor loop exits when the client dies.

## Failure modes and recovery

- **Chrome-in-Chrome MCP reports "not connected":** that's the faster path (DOM-aware clicks, real text extraction). Use it if available. If not, fall back to this procedure — `open -a "Google Chrome"` + `computer-use screenshot` is enough to *see* the UI, which is the job.
- **Port 60739 already in use:** something from a previous session. `./scripts/stop-server.sh` and `pkill -f run-local` first, or set `TERMINALS_CLIENT_WEB_PORT` to a free port.
- **macOS build fails "Xcode build system has crashed":** `run-local.sh` already auto-retries up to 3 times. If it still fails, check `.tmp/run-local-client-diagnostics.log` — that script writes a full `xcodebuild` log there on failure.
- **Tab title shows truncated "termin..." only:** that *is* the UI issue (tab title is `terminal_client`), not a tooling problem. Flag it in the punch list.

## Why this skill exists

Without it, each UI-inspection request re-derives the same workflow: which ports, which flags, which MCP, which display, how to get past the Flutter-web compile delay, how to handle the "com.example.terminalClient is a terminal" tier misclassification. The procedure is short but error-prone when reconstructed from memory, and the user has asked for UI checks often enough that it's worth writing down.
