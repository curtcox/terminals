import test from "node:test";
import assert from "node:assert/strict";
import { ServerDrivenRenderer } from "../../src/ui/renderer.js";
import { installDomHarness, createElement } from "../../src/test_support/dom_test_harness.js";
import { buttonNode, textNode } from "../../src/test_support/fixtures.js";
import { create } from "@bufbuild/protobuf";
import { NodeSchema, ScrollDirection } from "../../src/protocol/generated/terminals/ui/v1/ui_pb.js";

function node(kind, value = {}, { id = kind, children = [], props = {} } = {}) {
  return create(NodeSchema, { id, props, children, widget: { case: kind, value } });
}

test("renders text node with stable attributes", () => {
  const restore = installDomHarness();
  const root = createElement("div");
  new ServerDrivenRenderer({ rootElement: root }).render(textNode("Hi", "greeting"));
  assert.equal(root.children[0].textContent, "Hi");
  assert.equal(root.children[0].attributes["data-widget-kind"], "text");
  assert.equal(root.children[0].attributes["data-component-id"], "greeting");
  restore();
});

test("emits configured button action", () => {
  const restore = installDomHarness();
  const actions = [];
  const root = createElement("div");
  new ServerDrivenRenderer({ rootElement: root, onAction: (action) => actions.push(action) }).render(buttonNode("Go", "launch", "go"));
  root.children[0].onclick();
  assert.deepEqual(actions[0], { componentId: "go", action: "launch", value: "" });
  restore();
});

test("renders layout primitives and scroll direction", () => {
  const restore = installDomHarness();
  const root = createElement("div");
  const tree = create(NodeSchema, {
    id: "scroll",
    children: [textNode("A", "a")],
    widget: { case: "scroll", value: { direction: "", directionEnum: ScrollDirection.HORIZONTAL } }
  });
  new ServerDrivenRenderer({ rootElement: root }).render(tree);
  assert.equal(root.children[0].attributes["data-scroll-direction"], "horizontal");
  assert.equal(root.children[0].children[0].textContent, "A");
  restore();
});

test("renders all layout and overlay primitives with stable child order", () => {
  const restore = installDomHarness();
  const root = createElement("div");
  const primitives = [
    ["stack", {}, "sd-stack"],
    ["row", {}, "sd-row"],
    ["grid", { columns: 3 }, "sd-grid"],
    ["padding", { all: 12 }, "sd-padding"],
    ["center", {}, "sd-center"],
    ["expand", {}, "sd-expand"],
    ["overlay", {}, "sd-overlay"]
  ];
  for (const [kind, value, className] of primitives) {
    new ServerDrivenRenderer({ rootElement: root }).render(node(kind, value, { children: [textNode("child", `${kind}-child`)] }));
    assert.equal(root.children[0].attributes["data-widget-kind"], kind);
    assert.match(root.children[0].className, new RegExp(className));
    assert.equal(root.children[0].children[0].textContent, "child");
  }
  new ServerDrivenRenderer({ rootElement: root }).render(node("grid", { columns: 3 }));
  assert.equal(root.children[0].style.gridTemplateColumns, "repeat(3, minmax(0, 1fr))");
  new ServerDrivenRenderer({ rootElement: root }).render(node("padding", { all: 12 }));
  assert.equal(root.children[0].style.padding, "12px");
  restore();
});

test("renders content and media primitives", () => {
  const restore = installDomHarness();
  const root = createElement("div");
  const surfaces = [];
  const mediaSurfaceRegistry = {
    createSurface(kind, componentId, streamId) {
      surfaces.push({ kind, componentId, streamId });
      const el = createElement(kind);
      el.setAttribute("data-media-id", streamId);
      return el;
    }
  };
  const images = [];
  const imageLoader = (url) => {
    images.push(url);
    const el = createElement("img");
    el.setAttribute("src", url);
    return el;
  };
  const renderer = new ServerDrivenRenderer({ rootElement: root, mediaSurfaceRegistry, imageLoader });
  renderer.render(node("image", { url: "https://example.invalid/a.png" }));
  assert.deepEqual(images, ["https://example.invalid/a.png"]);
  renderer.render(node("videoSurface", { trackId: "track-1" }, { id: "video-1" }));
  renderer.render(node("audioVisualizer", { streamId: "stream-1" }, { id: "audio-1" }));
  assert.deepEqual(surfaces, [
    { kind: "video", componentId: "video-1", streamId: "track-1" },
    { kind: "audio", componentId: "audio-1", streamId: "stream-1" }
  ]);
  renderer.render(node("canvas", { drawOps: [{ op: { case: "line", value: { x1: 0, y1: 1, x2: 2, y2: 3 } } }], drawOpsJson: "[{}]" }));
  assert.equal(root.children[0].attributes["data-draw-op-count"], "1");
  assert.equal(root.children[0].attributes["data-legacy-draw-ops"], "true");
  restore();
});

test("canvas renders typed draw operations and ignores malformed legacy payloads", () => {
  const restore = installDomHarness();
  const root = createElement("div");
  new ServerDrivenRenderer({ rootElement: root }).render(node("canvas", {
    drawOpsJson: "{bad",
    drawOps: [
      { op: { case: "line", value: { x1: 1, y1: 2, x2: 3, y2: 4, stroke: "#123456", strokeWidth: 2 } } },
      { op: { case: "rect", value: { x: 5, y: 6, width: 7, height: 8, fill: "red" } } },
      { op: { case: "circle", value: { cx: 9, cy: 10, radius: 11, stroke: "blue" } } },
      { op: { case: "text", value: { x: 12, y: 13, text: "hello", fill: "green", fontSize: 14, fontFamily: "serif" } } }
    ]
  }));
  assert.deepEqual(root.children[0].canvasCalls, [
    ["strokeStyle", "#123456"],
    ["lineWidth", 2],
    ["beginPath"],
    ["moveTo", 1, 2],
    ["lineTo", 3, 4],
    ["stroke"],
    ["fillStyle", "red"],
    ["fillRect", 5, 6, 7, 8],
    ["strokeStyle", "blue"],
    ["beginPath"],
    ["arc", 9, 10, 11, 0, Math.PI * 2],
    ["stroke"],
    ["font", "14px serif"],
    ["fillStyle", "green"],
    ["fillText", "hello", 12, 13]
  ]);
  assert.equal(root.children[0].attributes["data-legacy-draw-ops"], "true");
  restore();
});

test("input primitives emit generic actions", () => {
  const restore = installDomHarness();
  const root = createElement("div");
  const actions = [];
  const renderer = new ServerDrivenRenderer({ rootElement: root, onAction: (action) => actions.push(action) });
  renderer.render(node("textInput", { placeholder: "Name", autofocus: true }, { id: "name" }));
  root.children[0].value = "Ada";
  root.children[0].onchange();
  root.children[0].onkeydown({ key: "Enter" });
  renderer.render(node("slider", { min: 2, max: 8, value: 5 }, { id: "level" }));
  root.children[0].value = "6";
  root.children[0].oninput();
  renderer.render(node("toggle", { value: true }, { id: "flag" }));
  root.children[0].checked = false;
  root.children[0].onchange();
  renderer.render(node("dropdown", { options: ["one", "two"], value: "two" }, { id: "choice" }));
  root.children[0].value = "one";
  root.children[0].onchange();
  renderer.render(node("gestureArea", { action: "press" }, { id: "pad" }));
  root.children[0].onclick();
  assert.deepEqual(actions, [
    { componentId: "name", action: "change", value: "Ada" },
    { componentId: "name", action: "submit", value: "Ada" },
    { componentId: "level", action: "change", value: "6" },
    { componentId: "flag", action: "toggle", value: "false" },
    { componentId: "choice", action: "select", value: "one" },
    { componentId: "pad", action: "press", value: "" }
  ]);
  restore();
});

test("system primitives render safe DOM placeholders", () => {
  const restore = installDomHarness();
  const root = createElement("div");
  const renderer = new ServerDrivenRenderer({ rootElement: root });
  renderer.render(node("progress", { value: 2 }));
  assert.equal(root.children[0].value, 1);
  renderer.render(node("fullscreen", { enabled: true }));
  assert.equal(root.children[0].attributes["data-system-detail"], "enabled");
  renderer.render(node("keepAwake", { enabled: false }));
  assert.equal(root.children[0].attributes["data-system-detail"], "disabled");
  renderer.render(node("brightness", { value: 0.25 }));
  assert.equal(root.children[0].attributes["data-system-detail"], "0.25");
  restore();
});

test("patch replaces targeted component and transition stores hint", () => {
  const restore = installDomHarness();
  const root = createElement("div");
  const renderer = new ServerDrivenRenderer({ rootElement: root });
  renderer.render(node("row", {}, { id: "row", children: [textNode("before", "target")] }));
  renderer.patch("target", textNode("after", "target"));
  assert.equal(root.querySelector("[data-component-id=\"target\"]").textContent, "after");
  renderer.transition("target", { transition: "fade" });
  assert.equal(root.attributes["data-transition"], "fade");
  restore();
});

test("renders fallback for unsupported nodes in development policy", () => {
  const restore = installDomHarness();
  const root = createElement("div");
  new ServerDrivenRenderer({ rootElement: root }).render({ id: "bad", props: {}, children: [] });
  assert.equal(root.children[0].attributes["data-widget-kind"], "unsupported");
  restore();
});
