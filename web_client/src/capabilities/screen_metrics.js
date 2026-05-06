export function readScreenMetrics({ window = globalThis.window, document = globalThis.document } = {}) {
  const viewport = document?.documentElement ?? {};
  return {
    viewportWidth: window?.innerWidth ?? viewport.clientWidth ?? 0,
    viewportHeight: window?.innerHeight ?? viewport.clientHeight ?? 0,
    devicePixelRatio: window?.devicePixelRatio ?? 1,
    orientation: window?.screen?.orientation?.type ?? "",
    reducedMotion: window?.matchMedia?.("(prefers-reduced-motion: reduce)")?.matches ?? false,
    colorScheme: window?.matchMedia?.("(prefers-color-scheme: dark)")?.matches ? "dark" : "light",
    visibility: document?.visibilityState ?? "visible"
  };
}

export function watchScreenMetrics(callback, { window = globalThis.window, document = globalThis.document, delayMs = 100 } = {}) {
  let timer = null;
  const handler = () => {
    clearTimeout(timer);
    timer = setTimeout(() => callback(readScreenMetrics({ window, document })), delayMs);
  };
  window?.addEventListener?.("resize", handler);
  window?.screen?.orientation?.addEventListener?.("change", handler);
  document?.addEventListener?.("visibilitychange", handler);
  return () => {
    clearTimeout(timer);
    window?.removeEventListener?.("resize", handler);
    window?.screen?.orientation?.removeEventListener?.("change", handler);
    document?.removeEventListener?.("visibilitychange", handler);
  };
}
