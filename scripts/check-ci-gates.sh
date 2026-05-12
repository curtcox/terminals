#!/usr/bin/env bash
# Probe CI gates that are not otherwise tracked by pick-next-work.py and write
# results to scripts/ci-status.json.  Running this before `make next` ensures
# failing gates surface as Priority 0 quality-debt items rather than being
# silently skipped.
#
# Currently probes:
#   android_detekt  — ./gradlew app:detektMain (style/static analysis)
#   android_lint    — ./gradlew app:lintDebug  (Android lint)
#
# Exit code: 0 if all gates pass, 1 if any gate fails, 2 if all gates were
# skipped (Android SDK not available).
#
# Usage:
#   scripts/check-ci-gates.sh               # from repo root
#   make ci-status                           # same via Make

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ANDROID_ROOT="$REPO_ROOT/android_client"
OUT="$REPO_ROOT/scripts/ci-status.json"

# ── helpers ──────────────────────────────────────────────────────────────────

timestamp() { date -u +"%Y-%m-%dT%H:%M:%SZ"; }

# Detect Android SDK availability using the same heuristic as the Makefile.
android_sdk_available() {
    [ -n "${ANDROID_SDK_ROOT:-}" ] || [ -n "${ANDROID_HOME:-}" ] || [ -f "$ANDROID_ROOT/local.properties" ]
}

# Resolve the JDK to use, mirroring the Makefile's ANDROID_JAVA_HOME logic.
resolve_java_home() {
    if [ -n "${JAVA_HOME:-}" ] && [ -x "${JAVA_HOME}/bin/java" ]; then
        echo "$JAVA_HOME"
        return
    fi
    # macOS: prefer the JBR bundled with Android Studio.
    local studio_jbr="/Applications/Android Studio.app/Contents/jbr/Contents/Home"
    if [ -x "$studio_jbr/bin/java" ]; then
        echo "$studio_jbr"
        return
    fi
    # Homebrew openjdk@17
    local brew_java
    brew_java="$(brew --prefix openjdk@17 2>/dev/null)/libexec/openjdk.jdk/Contents/Home" || true
    if [ -x "${brew_java}/bin/java" ]; then
        echo "$brew_java"
        return
    fi
    echo ""
}

run_gate() {
    local name="$1"; shift
    local cmd=("$@")
    local started; started=$(timestamp)
    local exit_code=0
    local output
    output=$("${cmd[@]}" 2>&1) || exit_code=$?
    local finished; finished=$(timestamp)
    if [ "$exit_code" -eq 0 ]; then
        echo "  $name: PASS" >&2
        printf '{"name":"%s","status":"pass","started":"%s","finished":"%s","output":""}\n' \
            "$name" "$started" "$finished"
    else
        # Summarise violations (lines containing ".kt:") for the JSON output.
        local violations
        violations=$(echo "$output" | grep -c '\.kt:' || true)
        echo "  $name: FAIL ($violations violation(s))" >&2
        # Escape output for JSON embedding (keep it short — first 20 violation lines).
        local snippet
        snippet=$(echo "$output" | grep '\.kt:' | head -20 | \
            sed 's/\\/\\\\/g; s/"/\\"/g; s/$/\\n/' | tr -d '\n')
        printf '{"name":"%s","status":"fail","violation_count":%d,"started":"%s","finished":"%s","output":"%s"}\n' \
            "$name" "$violations" "$started" "$finished" "$snippet"
    fi
    return "$exit_code"
}

# ── main ─────────────────────────────────────────────────────────────────────

echo "check-ci-gates: probing CI gates…" >&2

if ! android_sdk_available; then
    echo "  Skipping Android gates: Android SDK not configured." >&2
    cat > "$OUT" <<JSON
{"generated":"$(timestamp)","gates":[],"skipped":["android_detekt","android_lint"],"all_pass":null}
JSON
    echo "check-ci-gates: wrote $OUT (all gates skipped)" >&2
    exit 2
fi

JAVA_HOME_RESOLVED=$(resolve_java_home)
if [ -z "$JAVA_HOME_RESOLVED" ] || [ ! -x "$JAVA_HOME_RESOLVED/bin/java" ]; then
    echo "  Skipping Android gates: JDK not found. Set JAVA_HOME or install openjdk@17." >&2
    cat > "$OUT" <<JSON
{"generated":"$(timestamp)","gates":[],"skipped":["android_detekt","android_lint"],"all_pass":null}
JSON
    echo "check-ci-gates: wrote $OUT (all gates skipped — no JDK)" >&2
    exit 2
fi

export JAVA_HOME="$JAVA_HOME_RESOLVED"

all_pass=true
gates_json=()

pushd "$ANDROID_ROOT" > /dev/null

result=$(run_gate "android_detekt" ./gradlew app:detektMain --quiet) || all_pass=false
gates_json+=("$result")

result=$(run_gate "android_lint" ./gradlew app:lintDebug --quiet) || all_pass=false
gates_json+=("$result")

popd > /dev/null

# Build JSON array of gate results.
joined=$(printf ',%s' "${gates_json[@]}")
joined="[${joined:1}]"

if $all_pass; then
    cat > "$OUT" <<JSON
{"generated":"$(timestamp)","gates":$joined,"skipped":[],"all_pass":true}
JSON
    echo "check-ci-gates: all gates pass — wrote $OUT" >&2
    exit 0
else
    cat > "$OUT" <<JSON
{"generated":"$(timestamp)","gates":$joined,"skipped":[],"all_pass":false}
JSON
    echo "check-ci-gates: one or more gates FAILED — wrote $OUT" >&2
    exit 1
fi
