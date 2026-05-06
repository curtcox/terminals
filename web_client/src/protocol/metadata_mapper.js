export function normalizeServerMetadata(message) {
  return message?.serverMetadata ?? message?.metadata ?? {};
}
