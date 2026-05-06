import test from "node:test";
import assert from "node:assert/strict";
import { decodeMessage, encodeMessage } from "../../src/protocol/codec.js";
import { create } from "@bufbuild/protobuf";
import { ConnectResponseSchema } from "../../src/protocol/generated/terminals/control/v1/control_pb.js";

test("encodes and decodes binary message bytes", () => {
  const message = create(ConnectResponseSchema, { payload: { case: "helloAck", value: { sessionId: "s1" } } });
  const bytes = encodeMessage(ConnectResponseSchema, message);
  assert.ok(bytes instanceof Uint8Array);
  assert.equal(decodeMessage(ConnectResponseSchema, bytes).payload.value.sessionId, "s1");
});
