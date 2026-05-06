export function createStore(initialState = {}) {
  let state = {
    connectionPhase: "idle",
    endpoint: "",
    build: { sha: "dev", date: "" },
    capabilities: null,
    currentUiRoot: null,
    serverMetadata: null,
    diagnostics: [],
    ...initialState
  };
  const listeners = new Set();

  return {
    getState: () => state,
    subscribe(listener) {
      listeners.add(listener);
      listener(state);
      return () => listeners.delete(listener);
    },
    dispatch(event) {
      state = reduceState(state, event);
      for (const listener of listeners) listener(state, event);
      return state;
    }
  };
}

export function reduceState(state, event) {
  switch (event.type) {
    case "endpoint.selected":
      return { ...state, endpoint: event.endpoint };
    case "connection.phase":
      return { ...state, connectionPhase: event.phase };
    case "ui.set":
      return { ...state, currentUiRoot: event.root };
    case "ui.clear":
      return { ...state, currentUiRoot: null };
    case "server.metadata":
      return { ...state, serverMetadata: event.metadata };
    case "capabilities.snapshot":
      return { ...state, capabilities: event.capabilities };
    case "diagnostic.add":
      return { ...state, diagnostics: [...state.diagnostics, event.entry].slice(-50) };
    default:
      return state;
  }
}
