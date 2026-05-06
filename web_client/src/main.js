import { TerminalWebClientApp } from "./app.js";
import { loadConfig } from "./config.js";
import { createStore } from "./state/store.js";
import { ControlSocket } from "./transport/control_socket.js";
import { EndpointResolution } from "./transport/endpoint_resolution.js";
import { ServerDrivenRenderer } from "./ui/renderer.js";
import { RendererPolicy } from "./ui/renderer_policy.js";
import { createClientChrome } from "./diagnostics/client_chrome.js";
import { createBrowserCapabilitySnapshot } from "./capabilities/browser_capabilities.js";
import { MediaSurfaceRegistry } from "./media/media_surface_registry.js";

const config = loadConfig();
const store = createStore({
  endpoint: config.controlWsEndpoint,
  build: config.build,
  capabilities: createBrowserCapabilitySnapshot()
});

const mediaSurfaceRegistry = new MediaSurfaceRegistry();
const renderer = new ServerDrivenRenderer({
  rootElement: document.getElementById("terminal-root"),
  onAction: (action) => app.emitRendererAction(action),
  mediaSurfaceRegistry,
  policy: new RendererPolicy({ showFallbackDiagnostics: !config.production })
});

const app = new TerminalWebClientApp({
  config,
  store,
  endpointResolution: new EndpointResolution(),
  transport: new ControlSocket({ endpoint: config.controlWsEndpoint }),
  renderer,
  chrome: createClientChrome({
    rootElement: document.getElementById("client-chrome"),
    diagnosticsElement: document.getElementById("diagnostics-root")
  })
});

app.mount();
if (config.autoConnect) {
  app.connect();
}
