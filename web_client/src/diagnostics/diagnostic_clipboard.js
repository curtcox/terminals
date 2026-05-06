export async function copyDiagnostics(text, { clipboard = globalThis.navigator?.clipboard } = {}) {
  if (!clipboard?.writeText) return false;
  await clipboard.writeText(text);
  return true;
}
