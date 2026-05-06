export function createElement(tagName = "div") {
  return {
    tagName: tagName.toUpperCase(),
    children: [],
    attributes: {},
    style: {},
    className: "",
    textContent: "",
    value: "",
    checked: false,
    canvasCalls: [],
    append(...nodes) {
      for (const node of nodes) node.parentNode = this;
      this.children.push(...nodes);
    },
    appendChild(node) {
      node.parentNode = this;
      this.children.push(node);
      return node;
    },
    replaceChildren(...nodes) {
      for (const node of nodes) node.parentNode = this;
      this.children = nodes;
    },
    replaceWith(node) {
      if (!this.parentNode) return;
      const index = this.parentNode.children.indexOf(this);
      if (index >= 0) {
        this.parentNode.children[index] = node;
        node.parentNode = this.parentNode;
      }
    },
    setAttribute(key, value) { this.attributes[key] = String(value); },
    getAttribute(key) { return this.attributes[key]; },
    addEventListener(type, handler) { this[`on${type}`] = handler; },
    getContext(kind) {
      if (tagName !== "canvas" || kind !== "2d") return null;
      const calls = this.canvasCalls;
      return {
        beginPath: () => calls.push(["beginPath"]),
        moveTo: (x, y) => calls.push(["moveTo", x, y]),
        lineTo: (x, y) => calls.push(["lineTo", x, y]),
        stroke: () => calls.push(["stroke"]),
        fill: () => calls.push(["fill"]),
        arc: (cx, cy, radius, start, end) => calls.push(["arc", cx, cy, radius, start, end]),
        fillRect: (x, y, width, height) => calls.push(["fillRect", x, y, width, height]),
        strokeRect: (x, y, width, height) => calls.push(["strokeRect", x, y, width, height]),
        fillText: (text, x, y) => calls.push(["fillText", text, x, y]),
        set fillStyle(value) { calls.push(["fillStyle", value]); },
        set strokeStyle(value) { calls.push(["strokeStyle", value]); },
        set lineWidth(value) { calls.push(["lineWidth", value]); },
        set font(value) { calls.push(["font", value]); }
      };
    },
    querySelector(selector) {
      const attribute = selector.match(/^\[(data-[a-z-]+)="([^"]+)"\]$/);
      const match = (node) => attribute
        ? node.attributes?.[attribute[1]] === attribute[2]
        : false;
      const visit = (node) => match(node) ? node : node.children?.map(visit).find(Boolean);
      return visit(this);
    }
  };
}

export function installDomHarness() {
  const previous = globalThis.document;
  globalThis.document = {
    createElement,
    createTextNode: (text) => ({ nodeType: 3, textContent: text })
  };
  return () => { globalThis.document = previous; };
}
