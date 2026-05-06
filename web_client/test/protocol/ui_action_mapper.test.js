import test from "node:test";
import assert from "node:assert/strict";
import { mapRendererActionToConnectRequest } from "../../src/protocol/ui_action_mapper.js";

test("maps renderer action to UIAction input payload", () => {
  const request = mapRendererActionToConnectRequest(
    { componentId: "button-1", action: "start", value: 7 },
    { deviceId: "web-client" }
  );
  assert.equal(request.payload.case, "input");
  assert.equal(request.payload.value.deviceId, "web-client");
  assert.equal(request.payload.value.payload.value.componentId, "button-1");
  assert.equal(request.payload.value.payload.value.action, "start");
  assert.equal(request.payload.value.payload.value.value, "7");
});

test("rejects malformed renderer actions", () => {
  assert.throws(() => mapRendererActionToConnectRequest({ action: "tap" }), /componentId/);
});
