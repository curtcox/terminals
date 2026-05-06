import test from "node:test";
import assert from "node:assert/strict";
import { EndpointResolution } from "../../src/transport/endpoint_resolution.js";

test("normalizes host endpoints to websocket URLs", () => {
  const result = new EndpointResolution().resolve("127.0.0.1:60738/control");
  assert.equal(result.endpoint, "ws://127.0.0.1:60738/control");
});

test("reports browser discovery limitation", () => {
  assert.match(new EndpointResolution().browserDiscoveryDiagnostic(), /mDNS/);
});
