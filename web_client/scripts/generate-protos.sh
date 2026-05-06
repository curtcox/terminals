#!/usr/bin/env sh
set -eu

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
for proto in \
  "$ROOT/api/terminals/control/v1/control.proto" \
  "$ROOT/api/terminals/io/v1/io.proto" \
  "$ROOT/api/terminals/ui/v1/ui.proto" \
  "$ROOT/api/terminals/capabilities/v1/capabilities.proto"
do
  test -f "$proto"
done

if [ "${1:-}" = "--check" ]; then
  (cd "$ROOT/web_client" && npx buf generate ../api --template buf.gen.yaml)
  if ! git -C "$ROOT" diff --quiet -- web_client/src/protocol/generated; then
    echo "web client generated protobuf bindings are stale" >&2
    exit 1
  fi
  echo "web client proto inputs present"
  exit 0
fi

(cd "$ROOT/web_client" && npx buf generate ../api --template buf.gen.yaml)
