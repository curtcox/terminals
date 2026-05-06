export async function queryPermission(name) {
  if (!globalThis.navigator?.permissions?.query) return "unsupported";
  try {
    return (await navigator.permissions.query({ name })).state;
  } catch {
    return "unsupported";
  }
}
