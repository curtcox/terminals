import test from "node:test";
import assert from "node:assert/strict";
import { createClientChrome } from "../../src/diagnostics/client_chrome.js";
import { installDomHarness, createElement } from "../../src/test_support/dom_test_harness.js";

test("renders endpoint, build metadata, server metadata, and capability status", () => {
  const cleanup = installDomHarness();
  try {
    const rootElement = createElement("section");
    const diagnosticsElement = createElement("aside");
    const chrome = createClientChrome({ rootElement, diagnosticsElement });

    chrome.render({
      connectionPhase: "connected",
      endpoint: "ws://127.0.0.1:50054/control",
      build: { sha: "client-sha", date: "2026-05-05T12:00:00Z" },
      serverMetadata: { build: { sha: "server-sha", dateRfc3339: "2026-05-05T12:01:00Z" } },
      capabilities: {
        deviceId: "web-client",
        identity: { deviceName: "Test Browser", platform: "web" },
        screen: { width: 1200, height: 800, density: 2 },
        pointer: { type: "mouse" },
        keyboard: { physical: true },
        touch: { supported: false },
        speakers: {},
        microphone: null,
        camera: null
      },
      diagnostics: [{ level: "warning", message: "origin rejected" }]
    });

    assert.equal(rootElement.children[0].children[0].value, "ws://127.0.0.1:50054/control");
    const summaryText = rootElement.children[1].children
      .flatMap((item) => item.children.map((child) => child.textContent))
      .join(" ");
    assert.match(summaryText, /client-sha/);
    assert.match(summaryText, /server-sha/);
    assert.match(summaryText, /Test Browser/);
    assert.match(summaryText, /1200x800@2/);
    assert.equal(diagnosticsElement.children[0].textContent, "origin rejected");
  } finally {
    cleanup();
  }
});
