export function selectChromeView(state) {
  const capabilities = state.capabilities;
  const screen = capabilities?.screen;
  return {
    phase: state.connectionPhase,
    endpoint: state.endpoint,
    build: state.build,
    serverMetadata: state.serverMetadata,
    diagnosticCount: state.diagnostics.length,
    capabilities: capabilities
      ? {
          deviceId: capabilities.deviceId || "",
          deviceName: capabilities.identity?.deviceName || "Browser",
          platform: capabilities.identity?.platform || "web",
          screen: screen ? `${screen.width}x${screen.height}@${screen.density || 1}` : "",
          pointer: capabilities.pointer?.type || "",
          keyboard: Boolean(capabilities.keyboard?.physical),
          touch: Boolean(capabilities.touch?.supported),
          audioOutput: Boolean(capabilities.speakers),
          audioInput: Boolean(capabilities.microphone),
          camera: Boolean(capabilities.camera)
        }
      : null
  };
}
