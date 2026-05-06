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
    append(...nodes) { this.children.push(...nodes); },
    appendChild(node) { this.children.push(node); return node; },
    replaceChildren(...nodes) { this.children = nodes; },
    setAttribute(key, value) { this.attributes[key] = String(value); },
    getAttribute(key) { return this.attributes[key]; },
    addEventListener(type, handler) { this[`on${type}`] = handler; },
    querySelector(selector) {
      const match = (node) => selector.startsWith("[data-widget-kind=")
        ? node.attributes?.["data-widget-kind"] === selector.match(/"([^"]+)"/)?.[1]
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
