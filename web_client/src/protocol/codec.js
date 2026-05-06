import { create, fromBinary, toBinary } from "@bufbuild/protobuf";

export function createMessage(schema, fields = {}) {
  return create(schema, fields);
}

export function encodeMessage(schema, message) {
  return toBinary(schema, message);
}

export function decodeMessage(schema, bytes) {
  const view = bytes instanceof ArrayBuffer ? new Uint8Array(bytes) : bytes;
  return fromBinary(schema, view);
}
