const STORAGE_KEY = "terminals.controlWsEndpoint";

export class EndpointResolution {
  constructor({ location = globalThis.location, storage = globalThis.localStorage, injected = globalThis.TERMINALS_WEB_CLIENT_CONFIG } = {}) {
    this.location = location;
    this.storage = storage;
    this.injected = injected;
  }

  resolve(input) {
    const source = this.selectEndpoint(input);
    const raw = String(source.value ?? "").trim();
    if (!raw) return { endpoint: "", source: source.source, diagnostics: ["No WebSocket endpoint configured"] };
    try {
      const endpoint = normalizeEndpoint(raw);
      return { endpoint, source: source.source, diagnostics: [] };
    } catch (error) {
      return {
        endpoint: "",
        source: source.source,
        diagnostics: [`Invalid WebSocket endpoint: ${error.message}`]
      };
    }
  }

  persist(endpoint) {
    if (!endpoint) return;
    this.storage?.setItem?.(STORAGE_KEY, endpoint);
  }

  selectEndpoint(input) {
    if (String(input ?? "").trim()) return { value: input, source: "manual" };
    const params = new URLSearchParams(this.location?.search ?? "");
    const queryEndpoint = params.get("ws");
    if (queryEndpoint) return { value: queryEndpoint, source: "query" };
    if (this.injected?.controlWsEndpoint) return { value: this.injected.controlWsEndpoint, source: "config" };
    const stored = this.storage?.getItem?.(STORAGE_KEY);
    if (stored) return { value: stored, source: "storage" };
    return { value: "", source: "none" };
  }

  browserDiscoveryDiagnostic() {
    return "Browser clients cannot perform mDNS discovery directly; use an explicit WebSocket endpoint.";
  }
}

function normalizeEndpoint(raw) {
  if (/^[a-z][a-z0-9+.-]*:\/\//i.test(raw) && !raw.startsWith("ws://") && !raw.startsWith("wss://")) {
    throw new Error("endpoint must use ws:// or wss://");
  }
  const withScheme = raw.startsWith("ws://") || raw.startsWith("wss://") ? raw : `ws://${raw}`;
  const url = new URL(withScheme);
  if (url.protocol !== "ws:" && url.protocol !== "wss:") throw new Error("endpoint must use ws:// or wss://");
  return url.toString();
}
