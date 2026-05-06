import { ScrollDirection } from "../protocol/generated/terminals/ui/v1/ui_pb.js";
import { applyPrimitiveProps, clamp01, safeCssColor } from "./primitive_props.js";
import { serverDrivenNodeId } from "./node_key.js";
import { rendererAction } from "./actions.js";

export class ServerDrivenRenderer {
  constructor({ rootElement, onAction, mediaSurfaceRegistry, imageLoader, policy } = {}) {
    this.rootElement = rootElement;
    this.onAction = onAction ?? (() => {});
    this.mediaSurfaceRegistry = mediaSurfaceRegistry;
    this.imageLoader = imageLoader;
    this.policy = policy ?? { showFallbackDiagnostics: true, onError: () => {} };
  }

  render(rootNode) {
    this.rootElement.replaceChildren(this.renderNode(rootNode, "root"));
  }

  patch(componentId, node) {
    const existing = this.rootElement.querySelector?.(`[data-component-id="${componentId}"]`);
    if (existing?.replaceWith) existing.replaceWith(this.renderNode(node, componentId));
  }

  transition(_componentId, transition) {
    this.rootElement.setAttribute("data-transition", transition?.transition ?? "");
  }

  clear() {
    this.rootElement.replaceChildren();
  }

  renderNode(node, path = "root") {
    try {
      const kind = widgetKind(node);
      if (kind === "notSet") return this.fallback("unsupported", node, path);
      const element = this.createElementForKind(kind, node, path);
      this.decorate(element, kind, node, path);
      return element;
    } catch (error) {
      this.policy.onError?.(error);
      return this.fallback("malformed", node, path, error.message);
    }
  }

  createElementForKind(kind, node, path) {
    switch (kind) {
      case "stack": return this.container("div", "stack", node, path);
      case "row": return this.container("div", "row", node, path);
      case "grid": {
        const el = this.container("div", "grid", node, path);
        el.style.gridTemplateColumns = `repeat(${Math.max(1, widgetValue(node)?.columns || 1)}, minmax(0, 1fr))`;
        return el;
      }
      case "scroll": {
        const el = this.container("div", "scroll", node, path);
        const scroll = widgetValue(node);
        const horizontal = scroll?.directionEnum === ScrollDirection.HORIZONTAL || String(scroll?.direction).toLowerCase() === "horizontal";
        el.style.overflowX = horizontal ? "auto" : "hidden";
        el.style.overflowY = horizontal ? "hidden" : "auto";
        el.setAttribute("data-scroll-direction", horizontal ? "horizontal" : "vertical");
        return el;
      }
      case "padding": {
        const el = this.container("div", "padding", node, path);
        el.style.padding = `${Math.max(0, widgetValue(node)?.all || 0)}px`;
        return el;
      }
      case "center": return this.container("div", "center", node, path);
      case "expand": return this.container("div", "expand", node, path);
      case "overlay": return this.container("div", "overlay", node, path);
      case "text": return this.text(node);
      case "image": return this.image(node);
      case "videoSurface": return this.mediaSurface("video", widgetValue(node)?.trackId, node, path);
      case "audioVisualizer": return this.mediaSurface("audio", widgetValue(node)?.streamId, node, path);
      case "canvas": return this.canvas(node);
      case "textInput": return this.textInput(node);
      case "button": return this.button(node);
      case "slider": return this.slider(node);
      case "toggle": return this.toggle(node);
      case "dropdown": return this.dropdown(node);
      case "gestureArea": return this.gestureArea(node, path);
      case "progress": return this.progress(node);
      case "fullscreen": return this.systemPlaceholder("fullscreen", node, widgetValue(node)?.enabled ? "enabled" : "disabled", path);
      case "keepAwake": return this.systemPlaceholder("keep-awake", node, widgetValue(node)?.enabled ? "enabled" : "disabled", path);
      case "brightness": return this.systemPlaceholder("brightness", node, clamp01(widgetValue(node)?.value).toFixed(2), path);
      default: return this.fallback("unsupported", node, path);
    }
  }

  container(tag, classSuffix, node, path) {
    const el = document.createElement(tag);
    el.className = `sd sd-${classSuffix}`;
    for (const [index, child] of (node.children ?? []).entries()) {
      el.appendChild(this.renderNode(child, `${path}.${index}`));
    }
    return el;
  }

  text(node) {
    const el = document.createElement("p");
    el.className = "sd sd-text";
    const text = widgetValue(node);
    el.textContent = text?.value ?? "";
    if (text?.style === "monospace") el.style.fontFamily = "monospace";
    if (text?.color) el.style.color = safeCssColor(text.color);
    return el;
  }

  image(node) {
    const image = widgetValue(node);
    if (this.imageLoader) return this.imageLoader(image?.url ?? "");
    const el = document.createElement("img");
    el.className = "sd sd-image";
    el.src = image?.url ?? "";
    el.alt = node.props?.semantic_label ?? "";
    return el;
  }

  mediaSurface(kind, id, node, path) {
    const delegated = this.mediaSurfaceRegistry?.createSurface?.(kind, serverDrivenNodeId(node, path), id ?? "");
    return delegated ?? this.systemPlaceholder(`${kind}-surface`, node, id ?? "", path);
  }

  canvas(node) {
    const el = document.createElement("canvas");
    el.className = "sd sd-canvas";
    const canvas = widgetValue(node);
    const drawOps = canvas?.drawOps ?? [];
    el.setAttribute("data-draw-op-count", String(drawOps.length));
    if (canvas?.drawOpsJson) el.setAttribute("data-legacy-draw-ops", "true");
    drawCanvasOps(el, drawOps);
    return el;
  }

  textInput(node) {
    const input = document.createElement("input");
    input.className = "sd sd-text-input";
    const textInput = widgetValue(node);
    input.placeholder = textInput?.placeholder ?? "";
    input.autofocus = Boolean(textInput?.autofocus);
    input.addEventListener("change", () => this.emit(node, "change", input.value, "text_input"));
    input.addEventListener("keydown", (event) => {
      if (event.key === "Enter") this.emit(node, "submit", input.value, "text_input");
    });
    return input;
  }

  button(node) {
    const button = document.createElement("button");
    button.className = "sd sd-button";
    button.type = "button";
    const buttonWidget = widgetValue(node);
    button.textContent = buttonWidget?.label ?? "";
    button.addEventListener("click", () => this.emit(node, buttonWidget?.action || "tap", "", "button"));
    return button;
  }

  slider(node) {
    const input = document.createElement("input");
    input.className = "sd sd-slider";
    input.type = "range";
    const slider = widgetValue(node);
    input.min = String(slider?.min ?? 0);
    input.max = String((slider?.max ?? 1) > (slider?.min ?? 0) ? slider.max : (slider?.min ?? 0) + 1);
    input.value = String(Math.min(Number(input.max), Math.max(Number(input.min), slider?.value ?? 0)));
    input.addEventListener("input", () => this.emit(node, "change", input.value, "slider"));
    return input;
  }

  toggle(node) {
    const input = document.createElement("input");
    input.className = "sd sd-toggle";
    input.type = "checkbox";
    input.checked = Boolean(widgetValue(node)?.value);
    input.addEventListener("change", () => this.emit(node, "toggle", String(Boolean(input.checked)), "toggle"));
    return input;
  }

  dropdown(node) {
    const select = document.createElement("select");
    select.className = "sd sd-dropdown";
    const dropdown = widgetValue(node);
    for (const option of dropdown?.options ?? []) {
      const child = document.createElement("option");
      child.value = option;
      child.textContent = option;
      if (option === dropdown?.value) child.selected = true;
      select.appendChild(child);
    }
    select.addEventListener("change", () => this.emit(node, "select", select.value, "dropdown"));
    return select;
  }

  gestureArea(node, path) {
    const el = this.container("div", "gesture-area", node, path);
    el.addEventListener("click", () => this.emit(node, widgetValue(node)?.action || "tap", "", "gesture_area"));
    return el;
  }

  progress(node) {
    const el = document.createElement("progress");
    el.className = "sd sd-progress";
    el.max = 1;
    el.value = clamp01(widgetValue(node)?.value);
    return el;
  }

  systemPlaceholder(kind, node, detail, path) {
    const el = this.container("div", kind, node, path);
    el.setAttribute("data-system-detail", detail);
    return el;
  }

  decorate(element, kind, node, path) {
    const componentId = serverDrivenNodeId(node, path);
    element.setAttribute("data-node-id", node?.id ?? "");
    element.setAttribute("data-widget-kind", kind);
    element.setAttribute("data-component-id", componentId);
    applyPrimitiveProps(element, node);
  }

  emit(node, action, value, fallbackId) {
    this.onAction(rendererAction(serverDrivenNodeId(node, fallbackId), action, value));
  }

  fallback(kind, node, path, detail = "") {
    const el = document.createElement("div");
    el.className = "sd sd-fallback";
    el.setAttribute("data-widget-kind", kind);
    el.setAttribute("data-component-id", serverDrivenNodeId(node, path));
    if (this.policy.showFallbackDiagnostics) el.textContent = detail || "Unsupported UI node";
    return el;
  }
}

function widgetKind(node) {
  if (node?.widget?.case) return node.widget.case;
  const kinds = [
    "stack", "row", "grid", "scroll", "padding", "center", "expand",
    "text", "image", "videoSurface", "audioVisualizer", "canvas",
    "textInput", "button", "slider", "toggle", "dropdown", "gestureArea",
    "overlay", "progress", "fullscreen", "keepAwake", "brightness"
  ];
  return kinds.find((kind) => node?.[kind] !== undefined) ?? "notSet";
}

function widgetValue(node) {
  if (node?.widget?.case) return node.widget.value;
  return node?.[widgetKind(node)];
}

function drawCanvasOps(canvasElement, drawOps) {
  const context = canvasElement.getContext?.("2d");
  if (!context) return;
  for (const drawOp of drawOps) {
    const op = drawOp?.op;
    if (!op?.case) continue;
    drawCanvasOp(context, op.case, op.value ?? {});
  }
}

function drawCanvasOp(context, kind, value) {
  switch (kind) {
    case "line":
      withStroke(context, value, () => {
        context.beginPath?.();
        context.moveTo?.(value.x1 ?? 0, value.y1 ?? 0);
        context.lineTo?.(value.x2 ?? 0, value.y2 ?? 0);
        context.stroke?.();
      });
      break;
    case "rect":
      withFillAndStroke(context, value, () => {
        if (value.fill) context.fillRect?.(value.x ?? 0, value.y ?? 0, value.width ?? 0, value.height ?? 0);
        if (value.stroke) context.strokeRect?.(value.x ?? 0, value.y ?? 0, value.width ?? 0, value.height ?? 0);
      });
      break;
    case "circle":
      withFillAndStroke(context, value, () => {
        context.beginPath?.();
        context.arc?.(value.cx ?? 0, value.cy ?? 0, Math.max(0, value.radius ?? 0), 0, Math.PI * 2);
        if (value.fill) context.fill?.();
        if (value.stroke) context.stroke?.();
      });
      break;
    case "text":
      if (value.fontSize || value.fontFamily) {
        const size = Math.max(1, value.fontSize || 16);
        context.font = `${size}px ${value.fontFamily || "sans-serif"}`;
      }
      if (value.fill) context.fillStyle = safeCssColor(value.fill);
      context.fillText?.(value.text ?? "", value.x ?? 0, value.y ?? 0);
      break;
    case "path":
      withFillAndStroke(context, value, () => {
        const path = typeof Path2D === "function" ? new Path2D(value.d ?? "") : null;
        if (path && value.fill) context.fill?.(path);
        if (path && value.stroke) context.stroke?.(path);
      });
      break;
  }
}

function withStroke(context, value, draw) {
  if (value.stroke) context.strokeStyle = safeCssColor(value.stroke);
  if (value.strokeWidth) context.lineWidth = Math.max(0, value.strokeWidth);
  draw();
}

function withFillAndStroke(context, value, draw) {
  if (value.fill) context.fillStyle = safeCssColor(value.fill);
  if (value.stroke) context.strokeStyle = safeCssColor(value.stroke);
  if (value.strokeWidth) context.lineWidth = Math.max(0, value.strokeWidth);
  draw();
}
