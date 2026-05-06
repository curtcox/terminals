import test from "node:test";
import assert from "node:assert/strict";
import { mapBrowserProbeToCapabilities } from "../../src/protocol/capability_mapper.js";

test("maps browser probe to conservative capabilities", () => {
  const caps = mapBrowserProbeToCapabilities({
    viewportWidth: 800,
    viewportHeight: 600,
    devicePixelRatio: 2,
    touch: true,
    maxTouchPoints: 5,
    fullscreen: true,
    audioOutput: true
  });
  assert.equal(caps.screen.width, 800);
  assert.equal(caps.screen.touch, true);
  assert.equal(caps.touch.maxPoints, 5);
  assert.equal(caps.speakers.channels, 2);
  assert.equal(caps.microphone, undefined);
});
