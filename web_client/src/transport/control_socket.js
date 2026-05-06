import { EnvelopeCodec } from "./envelope_codec.js";
import { ReconnectPolicy } from "./reconnect_policy.js";
import { describeCloseEvent } from "./transport_diagnostics.js";

export class ControlSocket {
  constructor({
    endpoint,
    WebSocketCtor = globalThis.WebSocket,
    codec = new EnvelopeCodec(),
    reconnectPolicy = new ReconnectPolicy(),
    setTimeoutFn = globalThis.setTimeout,
    clearTimeoutFn = globalThis.clearTimeout
  } = {}) {
    this.endpoint = endpoint;
    this.WebSocketCtor = WebSocketCtor;
    this.codec = codec;
    this.reconnectPolicy = reconnectPolicy;
    this.setTimeoutFn = setTimeoutFn;
    this.clearTimeoutFn = clearTimeoutFn;
    this.socket = null;
    this.reconnectTimer = null;
    this.reconnectAttempt = 0;
    this.closedByClient = false;
    this.onMessage = () => {};
    this.onStateChange = () => {};
  }

  connect(endpoint = this.endpoint) {
    this.endpoint = endpoint;
    this.closedByClient = false;
    this.clearReconnectTimer();
    this.onStateChange("connecting");
    this.socket = new this.WebSocketCtor(endpoint);
    this.socket.binaryType = "arraybuffer";
    this.socket.onopen = () => {
      this.reconnectAttempt = 0;
      this.onStateChange("connected");
      this.socket.send(this.codec.encodeTransportHello());
    };
    this.socket.onmessage = (event) => {
      this.onMessage(this.codec.decodeConnectResponse(event.data));
    };
    this.socket.onerror = () => this.onStateChange("error", { level: "error", message: "WebSocket error" });
    this.socket.onclose = (event) => {
      const diagnostic = describeCloseEvent(event);
      this.onStateChange("closed", diagnostic);
      if (!this.closedByClient && shouldReconnect(event)) this.scheduleReconnect(diagnostic);
    };
    return this.socket;
  }

  sendConnectRequest(request) {
    if (!this.socket || this.socket.readyState !== this.WebSocketCtor.OPEN) return false;
    this.socket.send(this.codec.encodeConnectRequest(request));
    return true;
  }

  close() {
    this.closedByClient = true;
    this.clearReconnectTimer();
    if (this.socket) this.socket.close();
    this.socket = null;
  }

  scheduleReconnect(diagnostic) {
    const attempt = ++this.reconnectAttempt;
    const delayMs = this.reconnectPolicy.delayForAttempt(attempt);
    this.onStateChange("reconnecting", {
      level: diagnostic?.level ?? "warn",
      message: `Reconnecting in ${delayMs}ms`,
      delayMs,
      attempt
    });
    this.reconnectTimer = this.setTimeoutFn(() => this.connect(this.endpoint), delayMs);
  }

  clearReconnectTimer() {
    if (this.reconnectTimer !== null && this.clearTimeoutFn) this.clearTimeoutFn(this.reconnectTimer);
    this.reconnectTimer = null;
  }
}

function shouldReconnect(event) {
  return event?.code !== 1000 && event?.code !== 1001;
}
