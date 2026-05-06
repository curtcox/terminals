#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

cd "$ROOT/web_client"
npm test
npm run build

test -f "$ROOT/web_client/dist/index.html"
test -f "$ROOT/web_client/dist/src/main.js"

echo "web client smoke fixture ok"
