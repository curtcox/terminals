import test from "node:test";
import assert from "node:assert/strict";
import { EndpointResolution } from "../../src/transport/endpoint_resolution.js";

test("normalizes host endpoints to websocket URLs", () => {
  const result = new EndpointResolution().resolve("127.0.0.1:60738/control");
  assert.equal(result.endpoint, "ws://127.0.0.1:60738/control");
  assert.equal(result.source, "manual");
});

test("resolves endpoint from query, injected config, and storage in priority order", () => {
  const storage = new Map([["terminals.controlWsEndpoint", "stored.example/control"]]);
  const resolver = new EndpointResolution({
    location: { search: "?ws=query.example/control" },
    storage: {
      getItem: (key) => storage.get(key),
      setItem: (key, value) => storage.set(key, value)
    },
    injected: { controlWsEndpoint: "config.example/control" }
  });
  assert.deepEqual(resolver.resolve(), {
    endpoint: "ws://query.example/control",
    source: "query",
    diagnostics: []
  });
  assert.equal(new EndpointResolution({
    location: { search: "" },
    storage: { getItem: (key) => storage.get(key) },
    injected: { controlWsEndpoint: "config.example/control" }
  }).resolve().source, "config");
  assert.equal(new EndpointResolution({
    location: { search: "" },
    storage: { getItem: (key) => storage.get(key) },
    injected: {}
  }).resolve().source, "storage");
});

test("persists normalized manual endpoints", () => {
  const values = new Map();
  const resolver = new EndpointResolution({
    storage: {
      getItem: (key) => values.get(key),
      setItem: (key, value) => values.set(key, value)
    }
  });
  resolver.persist("ws://127.0.0.1:60738/control");
  assert.equal(values.get("terminals.controlWsEndpoint"), "ws://127.0.0.1:60738/control");
});

test("diagnoses invalid websocket endpoints", () => {
  const result = new EndpointResolution().resolve("http://example.invalid/control");
  assert.equal(result.endpoint, "");
  assert.match(result.diagnostics[0], /Invalid WebSocket endpoint/);
});

test("reports browser discovery limitation", () => {
  assert.match(new EndpointResolution().browserDiscoveryDiagnostic(), /mDNS/);
});
