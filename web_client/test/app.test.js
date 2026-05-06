import test from "node:test";
import assert from "node:assert/strict";
import { create } from "@bufbuild/protobuf";
import { TerminalWebClientApp } from "../src/app.js";
import { createStore } from "../src/state/store.js";
import { ConnectResponseSchema } from "../src/protocol/generated/terminals/control/v1/control_pb.js";
import { mapBrowserProbeToCapabilities } from "../src/protocol/capability_mapper.js";

test("stores normalized register ack metadata from server messages", () => {
  const store = createStore();
  const app = new TerminalWebClientApp({
    store,
    config: {},
    endpointResolution: {},
    transport: {},
    renderer: {},
    chrome: {}
  });
  const response = create(ConnectResponseSchema, {
    payload: {
      case: "registerAck",
      value: {
        metadata: {
          server_build_sha: "legacy-sha",
          server_build_date: "legacy-date"
        },
        serverMetadata: {
          build: { sha: "typed-sha", dateRfc3339: "2026-05-03T14:00:00Z" }
        }
      }
    }
  });

  app.handleServerMessage(response);

  assert.deepEqual(store.getState().serverMetadata, {
    build: {
      sha: "typed-sha",
      dateRfc3339: "2026-05-03T14:00:00Z"
    },
    photoFrameAssetBaseUrl: "",
    legacyMetadata: {
      server_build_sha: "legacy-sha",
      server_build_date: "legacy-date"
    }
  });
});

test("diagnoses unresolved endpoint without opening transport", () => {
  const store = createStore();
  let connected = false;
  const app = new TerminalWebClientApp({
    store,
    config: {},
    endpointResolution: {
      resolve: () => ({ endpoint: "", diagnostics: ["No WebSocket endpoint configured"] })
    },
    transport: {
      connect: () => {
        connected = true;
      }
    },
    renderer: {},
    chrome: {}
  });

  assert.equal(app.connect(""), null);
  assert.equal(connected, false);
  assert.deepEqual(store.getState().diagnostics, [{ level: "error", message: "No WebSocket endpoint configured" }]);
});

test("sends logical hello and capability snapshot after transport hello ack", () => {
  const sent = [];
  const capabilities = mapBrowserProbeToCapabilities({
    deviceId: "browser-1",
    deviceName: "Test Browser",
    viewportWidth: 800,
    viewportHeight: 600,
    devicePixelRatio: 2,
    keyboard: true,
    pointerType: "mouse"
  });
  const store = createStore();
  const app = new TerminalWebClientApp({
    store,
    config: { build: { sha: "abc123" } },
    endpointResolution: {},
    transport: {
      sendConnectRequest: (request) => sent.push(request)
    },
    renderer: {},
    chrome: {},
    capabilityProvider: () => capabilities
  });

  app.handleServerMessage({ payload: { case: "transportHelloAck", value: { sessionId: "s1" } } });

  assert.equal(sent.length, 2);
  assert.equal(sent[0].payload.case, "hello");
  assert.equal(sent[0].payload.value.deviceId, "browser-1");
  assert.equal(sent[0].payload.value.clientVersion, "web-client/abc123");
  assert.equal(sent[1].payload.case, "capabilitySnapshot");
  assert.equal(sent[1].payload.value.generation, 1n);
  assert.equal(sent[1].payload.value.capabilities.screen.width, 800);
  assert.equal(store.getState().capabilities.deviceId, "browser-1");
});
