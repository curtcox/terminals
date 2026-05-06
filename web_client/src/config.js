const DEFAULT_ENDPOINT = "ws://127.0.0.1:60738/control";

export function loadConfig({ location = globalThis.location, storage = globalThis.localStorage, injected = globalThis.TERMINALS_WEB_CLIENT_CONFIG } = {}) {
  const params = new URLSearchParams(location?.search ?? "");
  const endpoint = params.get("ws") || injected?.controlWsEndpoint || storage?.getItem("terminals.controlWsEndpoint") || DEFAULT_ENDPOINT;
  return {
    autoConnect: params.get("connect") === "1" || injected?.autoConnect === true,
    controlWsEndpoint: endpoint,
    build: {
      sha: injected?.buildSha || "dev",
      date: injected?.buildDate || new Date(0).toISOString()
    },
    production: injected?.production === true
  };
}
