export function monitorLifecycle(callback, { document = globalThis.document } = {}) {
  const handler = () => callback({ visibility: document.visibilityState });
  document?.addEventListener?.("visibilitychange", handler);
  return () => document?.removeEventListener?.("visibilitychange", handler);
}
