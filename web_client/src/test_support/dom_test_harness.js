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
