import test from "node:test";
import assert from "node:assert/strict";
import { ServerDrivenRenderer } from "../../src/ui/renderer.js";
import { installDomHarness, createElement } from "../../src/test_support/dom_test_harness.js";
import { buttonNode, textNode } from "../../src/test_support/fixtures.js";
import { create } from "@bufbuild/protobuf";
import { NodeSchema, ScrollDirection } from "../../src/protocol/generated/terminals/ui/v1/ui_pb.js";

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

test("renders fallback for unsupported nodes in development policy", () => {
  const restore = installDomHarness();
  const root = createElement("div");
  new ServerDrivenRenderer({ rootElement: root }).render({ id: "bad", props: {}, children: [] });
  assert.equal(root.children[0].attributes["data-widget-kind"], "unsupported");
  restore();
});
