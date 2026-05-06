const LEGACY_BUILD_SHA = "server_build_sha";
const LEGACY_BUILD_DATE = "server_build_date";
const LEGACY_PHOTO_FRAME_ASSET_BASE_URL = "photo_frame_asset_base_url";

export function normalizeServerMetadata(message) {
  const typed = message?.serverMetadata;
  const legacy = normalizeMetadataMap(message?.metadata);
  const typedBuild = typed?.build;

  return {
    build: {
      sha: nonEmpty(typedBuild?.sha) || legacy[LEGACY_BUILD_SHA] || "",
      dateRfc3339: nonEmpty(typedBuild?.dateRfc3339) || legacy[LEGACY_BUILD_DATE] || ""
    },
    photoFrameAssetBaseUrl:
      nonEmpty(typed?.photoFrameAssetBaseUrl) || legacy[LEGACY_PHOTO_FRAME_ASSET_BASE_URL] || "",
    legacyMetadata: legacy
  };
}

function nonEmpty(value) {
  return typeof value === "string" && value.trim() !== "" ? value : "";
}

function normalizeMetadataMap(metadata) {
  if (!metadata) return {};
  if (metadata instanceof Map) return Object.fromEntries(metadata.entries());
  return { ...metadata };
}
