export function selectChromeView(state) {
  return {
    phase: state.connectionPhase,
    endpoint: state.endpoint,
    build: state.build,
    serverMetadata: state.serverMetadata,
    diagnosticCount: state.diagnostics.length
  };
}
