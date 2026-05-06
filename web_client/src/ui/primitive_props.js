export function applyPrimitiveProps(element, node) {
  const props = node?.props ?? {};
  if (props.background) element.style.background = safeCssColor(props.background);
  if (props.width) element.style.width = cssSize(props.width);
  if (props.height) element.style.height = cssSize(props.height);
  if (props.semantic_label || props.accessibility_label) {
    element.setAttribute("aria-label", props.semantic_label || props.accessibility_label);
  }
}

export function cssSize(value) {
  const text = String(value ?? "").trim();
  if (/^\d+(\.\d+)?$/.test(text)) return `${text}px`;
  if (/^\d+(\.\d+)?(px|%|rem|em|vh|vw)$/.test(text)) return text;
  return "";
}

export function safeCssColor(value) {
  const text = String(value ?? "").trim();
  if (/^#[0-9a-fA-F]{3,8}$/.test(text)) return text;
  if (/^[a-zA-Z]+$/.test(text)) return text;
  return "";
}

export function clamp01(value) {
  const number = Number(value);
  if (!Number.isFinite(number)) return 0;
  return Math.max(0, Math.min(1, number));
}
