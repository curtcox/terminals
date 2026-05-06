import test from "node:test";
import assert from "node:assert/strict";
import { readScreenMetrics, watchScreenMetrics } from "../../src/capabilities/screen_metrics.js";

function createEventTarget() {
  const listeners = new Map();
  return {
    addEventListener(type, handler) {
      listeners.set(type, handler);
    },
    removeEventListener(type, handler) {
      if (listeners.get(type) === handler) listeners.delete(type);
    },
    dispatch(type) {
      listeners.get(type)?.();
    },
    listenerCount() {
      return listeners.size;
    }
  };
}

test("reads viewport preferences and visibility from injected browser objects", () => {
  const document = {
    documentElement: { clientWidth: 640, clientHeight: 480 },
    visibilityState: "hidden"
  };
  const window = {
    devicePixelRatio: 2,
    screen: { orientation: { type: "portrait-primary" } },
    matchMedia(query) {
      return { matches: query.includes("reduced-motion") };
    }
  };

  const metrics = readScreenMetrics({ window, document });
  assert.equal(metrics.viewportWidth, 640);
  assert.equal(metrics.viewportHeight, 480);
  assert.equal(metrics.devicePixelRatio, 2);
  assert.equal(metrics.orientation, "portrait-primary");
  assert.equal(metrics.reducedMotion, true);
  assert.equal(metrics.colorScheme, "light");
  assert.equal(metrics.visibility, "hidden");
});

test("watches resize, orientation, and visibility changes with debounce and cleanup", async () => {
  const windowTarget = createEventTarget();
  const orientationTarget = createEventTarget();
  const documentTarget = createEventTarget();
  const document = {
    ...documentTarget,
    documentElement: { clientWidth: 320, clientHeight: 240 },
    visibilityState: "visible"
  };
  const window = {
    ...windowTarget,
    innerWidth: 800,
    innerHeight: 600,
    devicePixelRatio: 1,
    screen: { orientation: { ...orientationTarget, type: "landscape-primary" } },
    matchMedia: () => ({ matches: false })
  };
  const snapshots = [];

  const stop = watchScreenMetrics((metrics) => snapshots.push(metrics), { window, document, delayMs: 5 });
  window.dispatch("resize");
  window.screen.orientation.dispatch("change");
  document.visibilityState = "hidden";
  document.dispatch("visibilitychange");
  await new Promise((resolve) => setTimeout(resolve, 20));

  assert.equal(snapshots.length, 1);
  assert.equal(snapshots[0].viewportWidth, 800);
  assert.equal(snapshots[0].visibility, "hidden");

  stop();
  assert.equal(window.listenerCount(), 0);
  assert.equal(window.screen.orientation.listenerCount(), 0);
  assert.equal(document.listenerCount(), 0);
});
