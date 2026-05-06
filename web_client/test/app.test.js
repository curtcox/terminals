import test from "node:test";
import assert from "node:assert/strict";
import { create } from "@bufbuild/protobuf";
import { TerminalWebClientApp } from "../src/app.js";
import { createStore } from "../src/state/store.js";
import { ConnectResponseSchema } from "../src/protocol/generated/terminals/control/v1/control_pb.js";

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
