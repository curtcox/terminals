export function requestFullscreenIfAvailable(element) {
  return element?.requestFullscreen?.() ?? Promise.resolve(false);
}
