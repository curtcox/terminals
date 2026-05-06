import { selectChromeView } from "../state/view_state.js";

export function createClientChrome({ rootElement, diagnosticsElement }) {
  return {
    render(state, actions = {}) {
      const view = selectChromeView(state);
      rootElement.replaceChildren();
      const form = document.createElement("form");
      const endpoint = document.createElement("input");
      endpoint.value = view.endpoint;
      endpoint.setAttribute("aria-label", "Control WebSocket endpoint");
      const connect = document.createElement("button");
      connect.type = "submit";
      connect.textContent = view.phase === "connected" ? "Reconnect" : "Connect";
      const disconnect = document.createElement("button");
      disconnect.type = "button";
      disconnect.textContent = "Disconnect";
      disconnect.addEventListener("click", () => actions.onDisconnect?.());
      form.addEventListener("submit", (event) => {
        event.preventDefault();
        actions.onConnect?.(endpoint.value);
      });
      form.append(endpoint, connect, disconnect, statusText(view));
      rootElement.appendChild(form);
      rootElement.appendChild(summaryGrid(view));

      diagnosticsElement.replaceChildren();
      for (const entry of state.diagnostics) {
        const item = document.createElement("div");
        item.className = `diagnostic diagnostic-${entry.level ?? "info"}`;
        item.textContent = entry.message ?? String(entry);
        diagnosticsElement.appendChild(item);
      }
    }
  };
}

function summaryGrid(view) {
  const dl = document.createElement("dl");
  dl.className = "chrome-summary";
  dl.append(
    summaryItem("Client", formatBuild(view.build)),
    summaryItem("Server", formatServer(view.serverMetadata)),
    summaryItem("Device", formatCapabilities(view.capabilities)),
    summaryItem("Diagnostics", String(view.diagnosticCount))
  );
  return dl;
}

function summaryItem(label, value) {
  const fragment = document.createElement("div");
  fragment.className = "chrome-summary-item";
  const term = document.createElement("dt");
  term.textContent = label;
  const detail = document.createElement("dd");
  detail.textContent = value || "unknown";
  fragment.append(term, detail);
  return fragment;
}

function formatBuild(build) {
  const sha = build?.sha || "dev";
  const date = build?.date || build?.dateRfc3339 || "";
  return date ? `${sha} ${date}` : sha;
}

function formatServer(metadata) {
  if (!metadata) return "not connected";
  return formatBuild(metadata.build);
}

function formatCapabilities(capabilities) {
  if (!capabilities) return "not probed";
  const flags = [
    capabilities.keyboard ? "keyboard" : "",
    capabilities.touch ? "touch" : "",
    capabilities.audioOutput ? "audio out" : "",
    capabilities.audioInput ? "audio in" : "",
    capabilities.camera ? "camera" : ""
  ].filter(Boolean);
  return [
    capabilities.deviceName,
    capabilities.screen,
    capabilities.pointer,
    flags.join(", ")
  ].filter(Boolean).join(" / ");
}

function statusText(view) {
  const span = document.createElement("span");
  span.className = "connection-status";
  span.textContent = view.phase;
  return span;
}
