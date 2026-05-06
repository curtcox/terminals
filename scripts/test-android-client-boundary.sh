#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

mkdir -p "${TMP_DIR}/android_client/app/src/main/java/com/curtcox/terminals/android/app"
mkdir -p "${TMP_DIR}/android_client/app/src/main/java/com/curtcox/terminals/android/ui"

cp "${ROOT_DIR}/android_client/settings.gradle.kts" "${TMP_DIR}/android_client/settings.gradle.kts"
cp "${ROOT_DIR}/android_client/build.gradle.kts" "${TMP_DIR}/android_client/build.gradle.kts"
mkdir -p "${TMP_DIR}/android_client/app"
cp "${ROOT_DIR}/android_client/app/build.gradle.kts" "${TMP_DIR}/android_client/app/build.gradle.kts"

cat >"${TMP_DIR}/android_client/app/src/main/java/com/curtcox/terminals/android/app/Ok.kt" <<'KT'
package com.curtcox.terminals.android.app
class Ok
KT

ANDROID_CLIENT_DIR="${TMP_DIR}/android_client" "${ROOT_DIR}/scripts/check-android-client-boundary.sh" >/dev/null

cat >"${TMP_DIR}/android_client/app/src/main/java/com/curtcox/terminals/android/app/Leak.kt" <<'KT'
package com.curtcox.terminals.android.app
const val forbidden = "kitchen_timer"
KT

if ANDROID_CLIENT_DIR="${TMP_DIR}/android_client" "${ROOT_DIR}/scripts/check-android-client-boundary.sh" >/dev/null 2>&1; then
  echo "ERROR: boundary scan did not catch scenario leakage"
  exit 1
fi

rm "${TMP_DIR}/android_client/app/src/main/java/com/curtcox/terminals/android/app/Leak.kt"
cat >>"${TMP_DIR}/android_client/app/build.gradle.kts" <<'KTS'
dependencies {
    implementation("com.google.firebase:firebase-messaging:24.0.0")
}
KTS

if ANDROID_CLIENT_DIR="${TMP_DIR}/android_client" "${ROOT_DIR}/scripts/check-android-client-boundary.sh" >/dev/null 2>&1; then
  echo "ERROR: boundary scan did not catch Google service dependency"
  exit 1
fi

cp "${ROOT_DIR}/android_client/app/build.gradle.kts" "${TMP_DIR}/android_client/app/build.gradle.kts"
cat >"${TMP_DIR}/android_client/app/src/main/java/com/curtcox/terminals/android/ui/RendererLeak.kt" <<'KT'
package com.curtcox.terminals.android.ui
import com.curtcox.terminals.android.connection.AndroidControlClient
class RendererLeak
KT

if ANDROID_CLIENT_DIR="${TMP_DIR}/android_client" "${ROOT_DIR}/scripts/check-android-client-boundary.sh" >/dev/null 2>&1; then
  echo "ERROR: boundary scan did not catch renderer subsystem import"
  exit 1
fi

echo "android client boundary tests passed"
