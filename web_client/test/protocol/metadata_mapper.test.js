import test from "node:test";
import assert from "node:assert/strict";
import { create } from "@bufbuild/protobuf";
import { RegisterAckSchema } from "../../src/protocol/generated/terminals/control/v1/control_pb.js";
import { normalizeServerMetadata } from "../../src/protocol/metadata_mapper.js";

test("normalizes typed register ack metadata before legacy map values", () => {
  const ack = create(RegisterAckSchema, {
    metadata: {
      server_build_sha: "legacy-sha",
      server_build_date: "legacy-date",
      photo_frame_asset_base_url: "https://legacy.example/assets/",
      "future.experimental_key": "preserve-but-ignore"
    },
    serverMetadata: {
      build: {
        sha: "typed-sha",
        dateRfc3339: "2026-05-03T14:00:00Z"
      },
      photoFrameAssetBaseUrl: "https://typed.example/assets/"
    }
  });

  assert.deepEqual(normalizeServerMetadata(ack), {
    build: {
      sha: "typed-sha",
      dateRfc3339: "2026-05-03T14:00:00Z"
    },
    photoFrameAssetBaseUrl: "https://typed.example/assets/",
    legacyMetadata: {
      server_build_sha: "legacy-sha",
      server_build_date: "legacy-date",
      photo_frame_asset_base_url: "https://legacy.example/assets/",
      "future.experimental_key": "preserve-but-ignore"
    }
  });
});

test("falls back to legacy register ack metadata for older servers", () => {
  const ack = create(RegisterAckSchema, {
    metadata: {
      server_build_sha: "abc1234",
      server_build_date: "2026-05-03T14:00:00Z",
      photo_frame_asset_base_url: "https://terminals.example/assets/"
    }
  });

  assert.deepEqual(normalizeServerMetadata(ack), {
    build: {
      sha: "abc1234",
      dateRfc3339: "2026-05-03T14:00:00Z"
    },
    photoFrameAssetBaseUrl: "https://terminals.example/assets/",
    legacyMetadata: {
      server_build_sha: "abc1234",
      server_build_date: "2026-05-03T14:00:00Z",
      photo_frame_asset_base_url: "https://terminals.example/assets/"
    }
  });
});
