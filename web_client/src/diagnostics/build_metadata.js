export function buildMetadata(config) {
  return {
    sha: config?.build?.sha ?? "dev",
    dateRfc3339: config?.build?.date ?? ""
  };
}
