import test from "node:test";
import assert from "node:assert/strict";
import { selectChromeView } from "../../src/state/view_state.js";

test("normalizes chrome capability summary from protobuf-shaped state", () => {
  const view = selectChromeView({
    connectionPhase: "connected",
    endpoint: "ws://example/control",
    build: { sha: "abc", date: "" },
    serverMetadata: null,
    diagnostics: [{ message: "one" }],
    capabilities: {
      deviceId: "web-client",
      identity: { deviceName: "Browser", platform: "web" },
      screen: { width: 800, height: 600, density: 1.5 },
      pointer: { type: "coarse" },
      keyboard: { physical: true },
      touch: { supported: true },
      speakers: {},
      microphone: {},
      camera: null
    }
  });

  assert.deepEqual(view.capabilities, {
    deviceId: "web-client",
    deviceName: "Browser",
    platform: "web",
    screen: "800x600@1.5",
    pointer: "coarse",
    keyboard: true,
    touch: true,
    audioOutput: true,
    audioInput: true,
    camera: false
  });
  assert.equal(view.diagnosticCount, 1);
});
