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

function statusText(view) {
  const span = document.createElement("span");
  span.className = "connection-status";
  span.textContent = view.phase;
  return span;
}
