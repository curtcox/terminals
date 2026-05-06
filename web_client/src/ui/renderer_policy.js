export class RendererPolicy {
  constructor({ showFallbackDiagnostics = true, onError = () => {} } = {}) {
    this.showFallbackDiagnostics = showFallbackDiagnostics;
    this.onError = onError;
  }
}
