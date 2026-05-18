#!/usr/bin/env bash
# Probe CI gates that are not otherwise tracked by pick-next-work.py and write
# results to scripts/ci-status.json.  Running this before `make next` ensures
# failing gates surface as Priority 0 quality-debt items rather than being
# silently skipped.
#
# Probes (in order):
#   server_lint     — make server-lint        (gofumpt, vet, golangci)
#   proto_lint      — make proto-lint         (buf lint)
#   client_boundary — make client-boundary    (Flutter import-boundary check)
#   android_detekt  — ./gradlew app:detektMain (skipped if no Android SDK)
#   android_lint    — ./gradlew app:lintDebug  (skipped if no Android SDK)
#
# Exit code: 0 if all attempted gates pass, 1 if any gate fails, 2 if every
# gate was skipped.
#
# Usage:
#   scripts/check-ci-gates.sh               # from repo root
#   make ci-status                           # same via Make

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ANDROID_ROOT="$REPO_ROOT/android_client"
OUT="$REPO_ROOT/scripts/ci-status.json"

# ── helpers ──────────────────────────────────────────────────────────────────

timestamp() { date -u +"%Y-%m-%dT%H:%M:%SZ"; }

android_sdk_available() {
    [ -n "${ANDROID_SDK_ROOT:-}" ] || [ -n "${ANDROID_HOME:-}" ] || [ -f "$ANDROID_ROOT/local.properties" ]
}

resolve_java_home() {
    if [ -n "${JAVA_HOME:-}" ] && [ -x "${JAVA_HOME}/bin/java" ]; then
        echo "$JAVA_HOME"
        return
    fi
    local studio_jbr="/Applications/Android Studio.app/Contents/jbr/Contents/Home"
    if [ -x "$studio_jbr/bin/java" ]; then
        echo "$studio_jbr"
        return
    fi
    local brew_java
    brew_java="$(brew --prefix openjdk@17 2>/dev/null)/libexec/openjdk.jdk/Contents/Home" || true
    if [ -x "${brew_java}/bin/java" ]; then
        echo "$brew_java"
        return
    fi
    echo ""
}

# run_gate NAME VIOLATION_REGEX -- CMD ARGS...
# VIOLATION_REGEX is a grep pattern that matches file:line: violation lines
# (e.g. '\.go:[0-9]+:' or '\.kt:[0-9]+:'). Used to count and snippet failures
# in the JSON output.
run_gate() {
    local name="$1"; shift
    local violation_regex="$1"; shift
    if [ "$1" = "--" ]; then shift; fi
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
        local violations
        violations=$(echo "$output" | grep -cE "$violation_regex" || true)
        echo "  $name: FAIL ($violations violation(s))" >&2
        local snippet
        snippet=$(echo "$output" | grep -E "$violation_regex" | head -20 | \
            sed 's/\\/\\\\/g; s/"/\\"/g; s/$/\\n/' | tr -d '\n')
        # If nothing matched the regex, fall back to last 20 output lines.
        if [ -z "$snippet" ]; then
            snippet=$(echo "$output" | tail -20 | \
                sed 's/\\/\\\\/g; s/"/\\"/g; s/$/\\n/' | tr -d '\n')
        fi
        printf '{"name":"%s","status":"fail","violation_count":%d,"started":"%s","finished":"%s","output":"%s"}\n' \
            "$name" "$violations" "$started" "$finished" "$snippet"
    fi
    return "$exit_code"
}

# ── main ─────────────────────────────────────────────────────────────────────

echo "check-ci-gates: probing CI gates…" >&2

all_pass=true
any_ran=false
gates_json=()
skipped=()

# Non-Android gates: always run.
cd "$REPO_ROOT"

if result=$(run_gate "server_lint" '\.go:[0-9]+:' -- make --no-print-directory server-lint); then
    :
else
    all_pass=false
fi
gates_json+=("$result")
any_ran=true

if result=$(run_gate "proto_lint" ':[0-9]+:[0-9]+:' -- make --no-print-directory proto-lint); then
    :
else
    all_pass=false
fi
gates_json+=("$result")

if result=$(run_gate "client_boundary" '\.dart:[0-9]+' -- make --no-print-directory client-boundary); then
    :
else
    all_pass=false
fi
gates_json+=("$result")

# Android gates: skip if SDK or JDK unavailable.
if ! android_sdk_available; then
    echo "  Skipping Android gates: Android SDK not configured." >&2
    skipped+=("android_detekt" "android_lint")
else
    JAVA_HOME_RESOLVED=$(resolve_java_home)
    if [ -z "$JAVA_HOME_RESOLVED" ] || [ ! -x "$JAVA_HOME_RESOLVED/bin/java" ]; then
        echo "  Skipping Android gates: JDK not found. Set JAVA_HOME or install openjdk@17." >&2
        skipped+=("android_detekt" "android_lint")
    else
        export JAVA_HOME="$JAVA_HOME_RESOLVED"
        pushd "$ANDROID_ROOT" > /dev/null
        if result=$(run_gate "android_detekt" '\.kt:[0-9]+:' -- ./gradlew app:detektMain --quiet); then
            :
        else
            all_pass=false
        fi
        gates_json+=("$result")
        if result=$(run_gate "android_lint" '\.kt:[0-9]+:' -- ./gradlew app:lintDebug --quiet); then
            :
        else
            all_pass=false
        fi
        gates_json+=("$result")
        popd > /dev/null
    fi
fi

# Build JSON.
if [ ${#gates_json[@]} -eq 0 ]; then
    joined="[]"
else
    joined=$(printf ',%s' "${gates_json[@]}")
    joined="[${joined:1}]"
fi

if [ ${#skipped[@]} -eq 0 ]; then
    skipped_json="[]"
else
    skipped_json=$(printf ',"%s"' "${skipped[@]}")
    skipped_json="[${skipped_json:1}]"
fi

if ! $any_ran; then
    cat > "$OUT" <<JSON
{"generated":"$(timestamp)","gates":$joined,"skipped":$skipped_json,"all_pass":null}
JSON
    echo "check-ci-gates: wrote $OUT (no gates ran)" >&2
    exit 2
fi

if $all_pass; then
    cat > "$OUT" <<JSON
{"generated":"$(timestamp)","gates":$joined,"skipped":$skipped_json,"all_pass":true}
JSON
    echo "check-ci-gates: all gates pass — wrote $OUT" >&2
    exit 0
else
    cat > "$OUT" <<JSON
{"generated":"$(timestamp)","gates":$joined,"skipped":$skipped_json,"all_pass":false}
JSON
    echo "check-ci-gates: one or more gates FAILED — wrote $OUT" >&2
    exit 1
fi
