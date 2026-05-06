export class EndpointResolution {
  resolve(input) {
    const raw = String(input ?? "").trim();
    if (!raw) return { endpoint: "", diagnostics: ["No WebSocket endpoint configured"] };
    const withScheme = raw.startsWith("ws://") || raw.startsWith("wss://") ? raw : `ws://${raw}`;
    const url = new URL(withScheme);
    return { endpoint: url.toString(), diagnostics: [] };
  }

  browserDiscoveryDiagnostic() {
    return "Browser clients cannot perform mDNS discovery directly; use an explicit WebSocket endpoint.";
  }
}
