export function serverDrivenNodeId(node, fallback = "") {
  return String(node?.id || node?.props?.component_id || fallback || "");
}
