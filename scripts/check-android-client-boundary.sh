#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ANDROID_CLIENT_DIR="${ANDROID_CLIENT_DIR:-${ROOT_DIR}/android_client}"
MAIN_SRC="${ANDROID_CLIENT_DIR}/app/src/main"

if [[ ! -d "${MAIN_SRC}" ]]; then
  echo "missing Android client source directory: ${MAIN_SRC}"
  exit 1
fi

if ! command -v rg >/dev/null 2>&1; then
  echo "ERROR: rg is required for Android client boundary scanning"
  exit 1
fi

scenario_pattern='terminal_root|photo[_ -]?frame|red[_ -]?alert|kitchen[_ -]?timer|package_id|com\.example'
scenario_matches="$(
  rg --line-number --no-heading \
    --glob '!**/BuildConfig.*' \
    --regexp "${scenario_pattern}" "${MAIN_SRC}" || true
)"

if [[ -n "${scenario_matches}" ]]; then
  echo "ERROR: production Android client contains scenario or package-id tokens"
  echo
  printf '%s\n' "${scenario_matches}"
  echo
  echo "Keep Android generic; scenario behavior belongs on the server."
  exit 1
fi

google_pattern='play-services|firebase|com\.google\.android\.gms|com\.google\.firebase|nearby|play-integrity|safetynet'
google_matches="$(
  rg --line-number --no-heading --ignore-case \
    --regexp "${google_pattern}" \
    "${ANDROID_CLIENT_DIR}/settings.gradle.kts" \
    "${ANDROID_CLIENT_DIR}/build.gradle.kts" \
    "${ANDROID_CLIENT_DIR}/app/build.gradle.kts" || true
)"

if [[ -n "${google_matches}" ]]; then
  echo "ERROR: Android client declares a forbidden Google service dependency"
  echo
  printf '%s\n' "${google_matches}"
  echo
  echo "Fire OS compatibility requires avoiding Google Play Services and Firebase."
  exit 1
fi

ui_import_pattern='com\.curtcox\.terminals\.android\.(connection|discovery|media|platform)'
ui_matches="$(
  if [[ -d "${MAIN_SRC}/java/com/curtcox/terminals/android/ui" ]]; then
    rg --line-number --no-heading --glob '*.kt' \
      --regexp "${ui_import_pattern}" \
      "${MAIN_SRC}/java/com/curtcox/terminals/android/ui" || true
  fi
)"

if [[ -n "${ui_matches}" ]]; then
  echo "ERROR: Android server-driven renderer imports client subsystems"
  echo
  printf '%s\n' "${ui_matches}"
  echo
  echo "Keep android_client/ui generic: render server-driven primitives and emit ServerDrivenAction only."
  exit 1
fi

echo "android client boundary scan passed"
