import { create } from "@bufbuild/protobuf";
import { ConnectRequestSchema } from "./generated/terminals/control/v1/control_pb.js";
import { InputEventSchema, UIActionSchema } from "./generated/terminals/io/v1/io_pb.js";

export function mapRendererActionToConnectRequest(action, { deviceId = "" } = {}) {
  if (!action?.componentId) throw new Error("renderer action requires componentId");
  if (!action?.action) throw new Error("renderer action requires action");
  const protoAction = create(UIActionSchema, {
    componentId: action.componentId,
    action: action.action,
    value: action.value == null ? "" : String(action.value)
  });
  const input = create(InputEventSchema, {
      deviceId,
      payload: { case: "uiAction", value: protoAction }
  });
  return create(ConnectRequestSchema, {
    payload: { case: "input", value: input }
  });
}
