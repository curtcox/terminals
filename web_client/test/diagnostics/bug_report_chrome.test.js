import test from "node:test";
import assert from "node:assert/strict";
import { createBugReportPayload } from "../../src/diagnostics/bug_report_chrome.js";

test("creates generic browser bug report payload", () => {
  const payload = createBugReportPayload({ connectionPhase: "error", endpoint: "ws://x", diagnostics: [{ message: "boom" }] });
  assert.equal(payload.client, "web");
  assert.equal(payload.diagnostics[0].message, "boom");
});
