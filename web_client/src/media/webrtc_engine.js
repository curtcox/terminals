import { WebRTCSignalType } from "../protocol/generated/terminals/control/v1/control_pb.js";

export class WebRTCEngine {
  constructor({
    peerConnectionFactory = () => new RTCPeerConnection(),
    mediaSurfaceRegistry,
    onLocalSignal = () => {},
    onDiagnostic = () => {}
  } = {}) {
    this.peerConnectionFactory = peerConnectionFactory;
    this.mediaSurfaceRegistry = mediaSurfaceRegistry;
    this.onLocalSignal = onLocalSignal;
    this.onDiagnostic = onDiagnostic;
    this.peer = null;
    this.streamSurfaces = new Map();
  }

  registerStreamSurface(streamId, componentId) {
    if (streamId && componentId) this.streamSurfaces.set(streamId, componentId);
  }

  ensurePeer() {
    if (!this.peer) {
      this.peer = this.peerConnectionFactory();
      this.peer.ontrack = (event) => this.attachRemoteTrack(event);
      this.peer.onicecandidate = (event) => this.emitIceCandidate(event?.candidate);
    }
    return this.peer;
  }

  async handleSignal(signal) {
    const peer = this.ensurePeer();
    const normalized = normalizeSignal(signal);
    if (!normalized.streamId || !normalized.signalType) {
      this.onDiagnostic({ kind: "webrtc_signal_ignored", reason: "missing stream_id or signal type" });
      return null;
    }
    if (normalized.signalType === "offer") {
      const description = parseSessionDescription(normalized);
      await peer.setRemoteDescription(description);
      const answer = await peer.createAnswer();
      await peer.setLocalDescription?.(answer);
      const outbound = toProtocolSignal(normalized.streamId, "answer", answer);
      this.onLocalSignal(outbound);
      return outbound;
    }
    if (normalized.signalType === "answer") {
      return peer.setRemoteDescription(parseSessionDescription(normalized));
    }
    if (normalized.signalType === "candidate") {
      return peer.addIceCandidate(parseCandidate(normalized));
    }
    this.onDiagnostic({ kind: "webrtc_signal_ignored", reason: `unsupported signal type ${normalized.signalType}` });
    return null;
  }

  attachRemoteTrack(event) {
    const stream = event?.streams?.[0];
    const streamId = stream?.id ?? event?.track?.id ?? "";
    const componentId = this.streamSurfaces.get(streamId) ?? this.streamSurfaces.get(event?.track?.id);
    if (componentId) this.mediaSurfaceRegistry?.attachStream?.(componentId, stream);
  }

  emitIceCandidate(candidate) {
    if (!candidate) return;
    const streamId = candidate.sdpMid ?? candidate.usernameFragment ?? "";
    const outbound = toProtocolSignal(streamId, "candidate", candidate.toJSON?.() ?? candidate);
    this.onLocalSignal(outbound);
  }
}

export function normalizeSignal(signal = {}) {
  return {
    streamId: signal.streamId ?? signal.stream_id ?? "",
    signalType: signalTypeFromEnum(signal.signalTypeEnum) || normalizeLegacySignalType(signal.signalType ?? signal.signal_type ?? signal.type),
    payload: signal.payload ?? ""
  };
}

function signalTypeFromEnum(value) {
  switch (value) {
    case WebRTCSignalType.WEB_RTC_SIGNAL_TYPE_OFFER:
    case 1:
      return "offer";
    case WebRTCSignalType.WEB_RTC_SIGNAL_TYPE_ANSWER:
    case 2:
      return "answer";
    case WebRTCSignalType.WEB_RTC_SIGNAL_TYPE_ICE_CANDIDATE:
    case 3:
      return "candidate";
    default:
      return "";
  }
}

function normalizeLegacySignalType(value = "") {
  const normalized = String(value).trim().toLowerCase();
  if (normalized === "ice_candidate") return "candidate";
  return normalized;
}

function parseSessionDescription(signal) {
  const payload = parsePayload(signal.payload);
  return {
    type: signal.signalType,
    sdp: payload.sdp ?? ""
  };
}

function parseCandidate(signal) {
  const payload = parsePayload(signal.payload);
  if (typeof payload === "string") return { candidate: payload };
  return payload;
}

function parsePayload(payload) {
  if (!payload) return {};
  try {
    return JSON.parse(payload);
  } catch {
    return payload;
  }
}

function toProtocolSignal(streamId, signalType, payload) {
  return {
    streamId,
    signalType,
    payload: JSON.stringify(payload ?? {})
  };
}
