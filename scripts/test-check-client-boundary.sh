#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHECKER="${ROOT_DIR}/scripts/check-client-boundary.sh"

fail() {
  echo "FAIL: $1" >&2
  exit 1
}

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

CLIENT_LIB="${TMP_DIR}/lib"
OUT_FILE="${TMP_DIR}/client-boundary-out"
mkdir -p "${CLIENT_LIB}/ui" "${CLIENT_LIB}/gen/terminals/ui/v1"

cat >"${CLIENT_LIB}/ui/server_driven_renderer.dart" <<'DART'
import 'package:flutter/widgets.dart';
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;
import 'package:terminal_client/ui/server_driven_action.dart';

Widget render(uiv1.Node node) => const SizedBox.shrink();
DART

cat >"${CLIENT_LIB}/ui/server_driven_action.dart" <<'DART'
class ServerDrivenAction {}
DART

CLIENT_LIB="${CLIENT_LIB}" "${CHECKER}" >/dev/null

cat >"${CLIENT_LIB}/ui/bad_import.dart" <<'DART'
import 'package:terminal_client/connection/control_client.dart';
DART

if CLIENT_LIB="${CLIENT_LIB}" "${CHECKER}" >"${OUT_FILE}" 2>&1; then
  fail "expected renderer subsystem import to fail"
fi
grep -Fq "imports client subsystems" "${OUT_FILE}" ||
  fail "expected subsystem import error message"

rm "${CLIENT_LIB}/ui/bad_import.dart"
cat >"${CLIENT_LIB}/scenario_leak.dart" <<'DART'
const scenarioName = 'photo_frame';
DART

if CLIENT_LIB="${CLIENT_LIB}" "${CHECKER}" >"${OUT_FILE}" 2>&1; then
  fail "expected scenario token to fail"
fi
grep -Fq "scenario or package-id tokens" "${OUT_FILE}" ||
  fail "expected scenario token error message"

echo "PASS: client boundary checker catches scenario and renderer import leaks"
