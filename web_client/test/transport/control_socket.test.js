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

test("schedules deterministic reconnect after abnormal close", () => {
  FakeSocket.instances = [];
  const states = [];
  const timers = [];
  const socket = new ControlSocket({
    endpoint: "ws://x/control",
    WebSocketCtor: FakeSocket,
    codec: new EnvelopeCodec(),
    reconnectPolicy: { delayForAttempt: (attempt) => attempt * 10 },
    setTimeoutFn: (callback, delayMs) => {
      timers.push({ callback, delayMs });
      return timers.length;
    },
    clearTimeoutFn: () => {}
  });
  socket.onStateChange = (state, detail) => states.push({ state, detail });
  socket.connect();
  FakeSocket.instances[0].open();
  FakeSocket.instances[0].onclose({ code: 1006, reason: "dropped", wasClean: false });
  assert.equal(states.at(-1).state, "reconnecting");
  assert.equal(states.at(-1).detail.delayMs, 10);
  assert.equal(timers.length, 1);
  timers[0].callback();
  assert.equal(FakeSocket.instances.length, 2);
});

test("explicit close does not schedule reconnect", () => {
  FakeSocket.instances = [];
  const timers = [];
  const socket = new ControlSocket({
    endpoint: "ws://x/control",
    WebSocketCtor: FakeSocket,
    codec: new EnvelopeCodec(),
    setTimeoutFn: (callback, delayMs) => {
      timers.push({ callback, delayMs });
      return timers.length;
    },
    clearTimeoutFn: () => {}
  });
  socket.connect();
  socket.close();
  assert.equal(timers.length, 0);
});
