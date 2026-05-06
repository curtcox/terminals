import { EnvelopeCodec } from "./envelope_codec.js";
import { describeCloseEvent } from "./transport_diagnostics.js";

export class ControlSocket {
  constructor({ endpoint, WebSocketCtor = globalThis.WebSocket, codec = new EnvelopeCodec() } = {}) {
    this.endpoint = endpoint;
    this.WebSocketCtor = WebSocketCtor;
    this.codec = codec;
    this.socket = null;
    this.onMessage = () => {};
    this.onStateChange = () => {};
  }

  connect(endpoint = this.endpoint) {
    this.endpoint = endpoint;
    this.onStateChange("connecting");
    this.socket = new this.WebSocketCtor(endpoint);
    this.socket.binaryType = "arraybuffer";
    this.socket.onopen = () => {
      this.onStateChange("connected");
      this.socket.send(this.codec.encodeTransportHello());
    };
    this.socket.onmessage = (event) => {
      this.onMessage(this.codec.decodeConnectResponse(event.data));
    };
    this.socket.onerror = () => this.onStateChange("error", { level: "error", message: "WebSocket error" });
    this.socket.onclose = (event) => this.onStateChange("closed", describeCloseEvent(event));
    return this.socket;
  }

  sendConnectRequest(request) {
    if (!this.socket || this.socket.readyState !== this.WebSocketCtor.OPEN) return false;
    this.socket.send(this.codec.encodeConnectRequest(request));
    return true;
  }

  close() {
    if (this.socket) this.socket.close();
    this.socket = null;
  }
}
