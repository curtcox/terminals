import { create } from "@bufbuild/protobuf";
import { NodeSchema } from "../protocol/generated/terminals/ui/v1/ui_pb.js";

export function textNode(value = "Hello", id = "text-1") {
  return create(NodeSchema, { id, widget: { case: "text", value: { value, style: "", color: "" } } });
}

export function buttonNode(label = "Go", action = "tap", id = "button-1") {
  return create(NodeSchema, { id, widget: { case: "button", value: { label, action } } });
}
