export function createBugReportPayload(state) {
  return {
    client: "web",
    connectionPhase: state.connectionPhase,
    endpoint: state.endpoint,
    diagnostics: state.diagnostics
  };
}
