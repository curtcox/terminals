import test from "node:test";
import assert from "node:assert/strict";
import { WebRTCEngine, normalizeSignal } from "../../src/media/webrtc_engine.js";

test("normalizes typed WebRTC signal enum before legacy strings", () => {
  assert.deepEqual(normalizeSignal({
    streamId: "stream-1",
    signalType: "answer",
    signalTypeEnum: 1,
    payload: "{\"sdp\":\"v=0-offer\"}"
  }), {
    streamId: "stream-1",
    signalType: "offer",
    payload: "{\"sdp\":\"v=0-offer\"}"
  });
});

test("handles offer and emits protocol-shaped answer", async () => {
  const calls = [];
  const localSignals = [];
  const peer = {
    async setRemoteDescription(description) { calls.push(["remote", description]); },
    async createAnswer() {
      calls.push(["answer"]);
      return { type: "answer", sdp: "v=0-answer" };
    },
    async setLocalDescription(description) { calls.push(["local", description]); }
  };
  const engine = new WebRTCEngine({
    peerConnectionFactory: () => peer,
    onLocalSignal: (signal) => localSignals.push(signal)
  });

  const response = await engine.handleSignal({
    streamId: "stream-1",
    signalTypeEnum: 1,
    payload: "{\"sdp\":\"v=0-offer\"}"
  });

  assert.deepEqual(calls, [
    ["remote", { type: "offer", sdp: "v=0-offer" }],
    ["answer"],
    ["local", { type: "answer", sdp: "v=0-answer" }]
  ]);
  assert.deepEqual(response, {
    streamId: "stream-1",
    signalType: "answer",
    payload: "{\"type\":\"answer\",\"sdp\":\"v=0-answer\"}"
  });
  assert.deepEqual(localSignals, [response]);
});

test("handles answer and ICE candidate payloads", async () => {
  const calls = [];
  const peer = {
    async setRemoteDescription(description) { calls.push(["remote", description]); },
    async addIceCandidate(candidate) { calls.push(["candidate", candidate]); }
  };
  const engine = new WebRTCEngine({ peerConnectionFactory: () => peer });

  await engine.handleSignal({
    streamId: "stream-1",
    signalType: "answer",
    payload: "{\"sdp\":\"v=0-answer\"}"
  });
  await engine.handleSignal({
    streamId: "stream-1",
    signalTypeEnum: 3,
    payload: "{\"candidate\":\"candidate:1\",\"sdpMid\":\"audio\"}"
  });

  assert.deepEqual(calls, [
    ["remote", { type: "answer", sdp: "v=0-answer" }],
    ["candidate", { candidate: "candidate:1", sdpMid: "audio" }]
  ]);
});

test("attaches remote tracks to registered media surfaces", () => {
  const attached = [];
  const stream = { id: "stream-1" };
  const peer = {};
  const engine = new WebRTCEngine({
    peerConnectionFactory: () => peer,
    mediaSurfaceRegistry: {
      attachStream(componentId, attachedStream) {
        attached.push([componentId, attachedStream]);
      }
    }
  });

  engine.registerStreamSurface("stream-1", "component-1");
  engine.ensurePeer();
  peer.ontrack({ streams: [stream] });

  assert.deepEqual(attached, [["component-1", stream]]);
});
