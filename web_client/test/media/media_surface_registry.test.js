import test from "node:test";
import assert from "node:assert/strict";
import { MediaSurfaceRegistry } from "../../src/media/media_surface_registry.js";
import { installDomHarness } from "../../src/test_support/dom_test_harness.js";

test("creates media surfaces with stable identifiers", () => {
  const restore = installDomHarness();
  const registry = new MediaSurfaceRegistry();
  const surface = registry.createSurface("video", "video-1", "track-1");
  assert.equal(surface.attributes["data-component-id"], "video-1");
  assert.equal(surface.attributes["data-media-id"], "track-1");
  restore();
});
