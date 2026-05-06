import { create } from "@bufbuild/protobuf";
import { CarrierKind, WireEnvelopeSchema } from "../protocol/generated/terminals/control/v1/control_pb.js";
import { decodeMessage, encodeMessage } from "../protocol/codec.js";

export class EnvelopeCodec {
  constructor({ protocolVersion = 1 } = {}) {
    this.protocolVersion = protocolVersion;
    this.sequence = 0n;
    this.sessionId = "";
  }

  encodeTransportHello({ desiredDeviceId = "web-client", resumeToken = "" } = {}) {
    return encodeMessage(WireEnvelopeSchema, create(WireEnvelopeSchema, {
      protocolVersion: this.protocolVersion,
      sequence: ++this.sequence,
      payload: {
        case: "transportHello",
        value: {
          protocolVersion: this.protocolVersion,
          supportedCarriers: [CarrierKind.WEBSOCKET],
          desiredDeviceId,
          resumeToken
        }
      }
    }));
  }

  encodeConnectRequest(request) {
    return encodeMessage(WireEnvelopeSchema, create(WireEnvelopeSchema, {
      protocolVersion: this.protocolVersion,
      sessionId: this.sessionId,
      sequence: ++this.sequence,
      payload: { case: "clientMessage", value: request }
    }));
  }

  decodeConnectResponse(bytes) {
    const envelope = decodeMessage(WireEnvelopeSchema, bytes);
    const payload = envelope.payload;
    if (payload.case === "transportHelloAck") {
      this.sessionId = payload.value.sessionId ?? this.sessionId;
      return { payload: { case: "transportHelloAck", value: payload.value } };
    }
    if (payload.case === "serverMessage") return payload.value;
    if (payload.case === "transportError") return { payload: { case: "error", value: payload.value } };
    return envelope;
  }
}
