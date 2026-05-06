export function dispatchChange(element, handler) {
  element.addEventListener("change", () => handler(element.value));
}
