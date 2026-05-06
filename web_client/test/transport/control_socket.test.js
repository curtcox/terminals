import test from "node:test";
import assert from "node:assert/strict";
import { ControlSocket } from "../../src/transport/control_socket.js";
import { FakeSocket } from "../../src/test_support/fake_socket.js";
import { EnvelopeCodec } from "../../src/transport/envelope_codec.js";
import { encodeMessage } from "../../src/protocol/codec.js";
import { create } from "@bufbuild/protobuf";
import { ConnectResponseSchema, WireEnvelopeSchema } from "../../src/protocol/generated/terminals/control/v1/control_pb.js";

test("opens websocket and sends transport hello", () => {
  FakeSocket.instances = [];
  const states = [];
  const socket = new ControlSocket({ endpoint: "ws://x/control", WebSocketCtor: FakeSocket, codec: new EnvelopeCodec() });
  socket.onStateChange = (state) => states.push(state);
  socket.connect();
  FakeSocket.instances[0].open();
  assert.deepEqual(states, ["connecting", "connected"]);
  assert.equal(FakeSocket.instances[0].sent.length, 1);
});

test("decodes server messages", () => {
  FakeSocket.instances = [];
  const messages = [];
  const socket = new ControlSocket({ endpoint: "ws://x/control", WebSocketCtor: FakeSocket, codec: new EnvelopeCodec() });
  socket.onMessage = (message) => messages.push(message);
  socket.connect();
  const response = create(ConnectResponseSchema, { payload: { case: "helloAck", value: { sessionId: "s1" } } });
  const envelope = create(WireEnvelopeSchema, { payload: { case: "serverMessage", value: response } });
  FakeSocket.instances[0].receive(encodeMessage(WireEnvelopeSchema, envelope));
  assert.equal(messages[0].payload.value.sessionId, "s1");
});
