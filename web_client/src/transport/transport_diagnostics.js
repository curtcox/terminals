export function describeCloseEvent(event) {
  return {
    level: event.wasClean ? "info" : "warning",
    message: `WebSocket closed (${event.code || "unknown"})`,
    detail: event.reason || ""
  };
}
