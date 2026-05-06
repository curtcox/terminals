import { mapRendererActionToConnectRequest } from "./protocol/ui_action_mapper.js";
import { normalizeServerMetadata } from "./protocol/metadata_mapper.js";

export class TerminalWebClientApp {
  constructor({ config, store, endpointResolution, transport, renderer, chrome }) {
    this.config = config;
    this.store = store;
    this.endpointResolution = endpointResolution;
    this.transport = transport;
    this.renderer = renderer;
    this.chrome = chrome;
  }

  mount() {
    this.store.subscribe((state) => {
      this.chrome.render(state, {
        onConnect: (endpoint) => this.connect(endpoint),
        onDisconnect: () => this.disconnect()
      });
    });
    this.transport.onMessage = (message) => this.handleServerMessage(message);
    this.transport.onStateChange = (phase, detail) => {
      this.store.dispatch({ type: "connection.phase", phase });
      if (detail) this.store.dispatch({ type: "diagnostic.add", entry: detail });
    };
  }

  connect(endpoint = this.store.getState().endpoint) {
    const resolved = this.endpointResolution.resolve(endpoint);
    for (const message of resolved.diagnostics ?? []) {
      this.store.dispatch({ type: "diagnostic.add", entry: { level: "error", message } });
    }
    if (!resolved.endpoint) return null;
    this.endpointResolution.persist?.(resolved.endpoint);
    this.store.dispatch({ type: "endpoint.selected", endpoint: resolved.endpoint });
    return this.transport.connect(resolved.endpoint);
  }

  disconnect() {
    this.transport.close();
  }

  emitRendererAction(action) {
    const request = mapRendererActionToConnectRequest(action, { deviceId: "web-client" });
    this.transport.sendConnectRequest(request);
  }

  handleServerMessage(message) {
    const payload = message?.payload;
    if (!payload?.case) return;
    const value = payload.value;
    if (payload.case === "setUi" && value.root) {
      this.store.dispatch({ type: "ui.set", root: value.root });
      this.renderer.render(value.root);
    } else if (payload.case === "updateUi" && value.componentId) {
      this.renderer.patch(value.componentId, value.node);
    } else if (payload.case === "transitionUi") {
      this.renderer.transition(value.componentId, value);
    } else if (payload.case === "registerAck") {
      this.store.dispatch({ type: "server.metadata", metadata: normalizeServerMetadata(value) });
    } else if (payload.case === "helloAck") {
      this.store.dispatch({ type: "server.metadata", metadata: value });
    }
  }
}
