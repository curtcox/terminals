
bootstrap() {
  require_cmd go
  require_cmd flutter
  require_cmd nc

  if [[ "${CLIENT_DEVICE}" == "web-server" ]]; then
    if [[ "${SKIP_BOOTSTRAP}" == "true" && ! -d "${ROOT_DIR}/terminal_client/web" ]]; then
      fail "web support is not configured (missing terminal_client/web). Re-run without --skip-bootstrap or run: (cd terminal_client && flutter create . --platforms=web)"
    fi
  fi

  if [[ "${SKIP_BOOTSTRAP}" == "true" ]]; then
    echo "Skipping bootstrap checks/install (--skip-bootstrap)."
    return 0
  fi

  echo "Bootstrapping dependencies..."
  (
    cd "${ROOT_DIR}/terminal_server"
    go mod download
  )
  if [[ "${CLIENT_DEVICE}" != "macos" ]]; then
    (
      cd "${ROOT_DIR}/terminal_client"
      flutter pub get
    )
  fi

  if [[ "${CLIENT_DEVICE}" == "web-server" ]]; then
    (
      cd "${ROOT_DIR}/terminal_client"
      flutter config --enable-web
      if [[ ! -d "web" ]]; then
        echo "Enabling Flutter web platform support in terminal_client..."
        flutter create . --platforms=web
      fi
    )
  fi

  if [[ "${CLIENT_DEVICE}" == "macos" ]]; then
    require_cmd xcodebuild
    require_cmd pod
    rm -rf "${ROOT_DIR}/terminal_client/.dart_tool"
    (
      cd "${ROOT_DIR}/terminal_client"
      flutter pub get
    )
    ensure_macos_flutter_config
    repair_macos_pods
  fi
}

sanitize_macos_xcconfig() {
  local generated_xcconfig="${ROOT_DIR}/terminal_client/macos/Flutter/ephemeral/Flutter-Generated.xcconfig"
  if [[ -f "${generated_xcconfig}" ]]; then
    # Flutter's CocoaPods parser splits on every "="; DART_DEFINES values contain
    # base64 padding ("=="), which triggers noisy "Invalid key/value pair" output.
    # Keep the generated file intact otherwise and remove only this non-essential key.
    local sanitized_xcconfig="${generated_xcconfig}.run-local-sanitized"
    grep -Ev '^DART_DEFINES=' "${generated_xcconfig}" > "${sanitized_xcconfig}"
    mv -f "${sanitized_xcconfig}" "${generated_xcconfig}"
  fi
}

ensure_macos_flutter_config() {
  (
    cd "${ROOT_DIR}/terminal_client"
    flutter build macos --debug --config-only --no-pub "${CLIENT_DART_DEFINE_ARGS[@]}"
  )
}

repair_macos_pods() {
  sanitize_macos_xcconfig
  (
    cd "${ROOT_DIR}/terminal_client/macos"
    env -u DART_DEFINES pod install
  )
}

recover_macos_client_build() {
  echo "Attempting macOS build recovery (reset .dart_tool + pub get + config + pod install)..."
  rm -rf "${ROOT_DIR}/terminal_client/.dart_tool"
  (
    cd "${ROOT_DIR}/terminal_client"
    flutter pub get
  )
  ensure_macos_flutter_config
  repair_macos_pods
}

build_macos_client_app() {
  (
    trap - ERR
    set +E
    cd "${ROOT_DIR}/terminal_client"
    flutter build macos --debug --no-pub "${CLIENT_DART_DEFINE_ARGS[@]}"
  ) >>"${CLIENT_LOG}" 2>&1
}

start_client_macos() {
  local app_path=""
  local app_executable=""

  echo "Building macOS client..."
  if ! build_macos_client_app; then
    return 1
  fi

  app_path="$(find "${ROOT_DIR}/terminal_client/build/macos/Build/Products/Debug" -maxdepth 1 -type d -name '*.app' 2>/dev/null | head -n 1 || true)"
  if [[ -z "${app_path}" ]]; then
    app_path="$(ls -td "${HOME}/Library/Developer/Xcode/DerivedData"/Runner-*/Build/Products/Debug/*.app 2>/dev/null | head -n 1 || true)"
  fi
  if [[ -z "${app_path}" ]]; then
    echo "Unable to locate built macOS .app in Xcode DerivedData." >>"${CLIENT_LOG}"
    return 1
  fi

  app_executable="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleExecutable' "${app_path}/Contents/Info.plist" 2>/dev/null || true)"
  if [[ -z "${app_executable}" ]]; then
    app_executable="$(basename "${app_path}" .app)"
  fi

  echo "Launching macOS app: ${app_path}" >>"${CLIENT_LOG}"
  (
    trap - ERR
    set +E
    cd "${ROOT_DIR}/terminal_client"
    exec "${app_path}/Contents/MacOS/${app_executable}"
  ) >>"${CLIENT_LOG}" 2>&1 &
  CLIENT_PID=$!

  sleep "${CLIENT_STARTUP_DELAY_SECONDS}"
  if ! kill -0 "${CLIENT_PID}" >/dev/null 2>&1; then
    wait "${CLIENT_PID}" || true
    return 1
  fi

  return 0
}

collect_macos_client_diagnostics() {
  local reason="${1:-unknown}"
  rotate_log "${CLIENT_DIAG_LOG}"
  : >"${CLIENT_DIAG_LOG}"
  {
    echo "=== run-local macOS diagnostics ==="
    echo "timestamp: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
    echo "reason: ${reason}"
    echo "root: ${ROOT_DIR}"
    echo
    echo "--- client log (tail 120) ---"
    if [[ -f "${CLIENT_LOG}" ]]; then
      tail -n 120 "${CLIENT_LOG}" || true
    else
      echo "(missing)"
    fi
    echo
    echo "--- xcodebuild -version ---"
    xcodebuild -version || true
    echo
    echo "--- flutter --version ---"
    (
      cd "${ROOT_DIR}/terminal_client"
      flutter --version
    ) || true
    echo
    echo "--- flutter doctor -v (tail 120) ---"
    (
      cd "${ROOT_DIR}/terminal_client"
      flutter doctor -v
    ) 2>&1 | tail -n 120 || true
    echo
    echo "--- xcodebuild showBuildSettings (tail 200) ---"
    (
      cd "${ROOT_DIR}/terminal_client/macos"
      xcodebuild \
        -workspace Runner.xcworkspace \
        -scheme Runner \
        -configuration Debug \
        -destination 'platform=macOS' \
        -showBuildSettings
    ) 2>&1 | tail -n 200 || true
    echo
    echo "--- xcodebuild build (tail 260) ---"
    (
      cd "${ROOT_DIR}/terminal_client/macos"
      xcodebuild \
        -workspace Runner.xcworkspace \
        -scheme Runner \
        -configuration Debug \
        -destination 'platform=macOS' \
        build
    ) 2>&1 | tail -n 260 || true
  } >"${CLIENT_DIAG_LOG}" 2>&1
}

start_server() {
  rotate_log "${SERVER_LOG}"
  : >"${SERVER_LOG}"
  echo "Starting server..."
  (
    # Disarm ERR/errtrace inside the redirected subshell. Otherwise the inherited
    # trap would write diagnostics to fd 2 — which is the server log here — and a
    # tail-of-self loop filled the disk with 712 GB of duplicated lines.
    trap - ERR
    set +E
    # Hard cap on any single write stream (RLIMIT_FSIZE). Runaway output gets a
    # SIGXFSZ instead of eating the disk.
    ulimit -f "${LOG_MAX_KB}" 2>/dev/null || true
    cd "${ROOT_DIR}/terminal_server"
    TERMINALS_GRPC_PORT="${GRPC_PORT}" \
    TERMINALS_CONTROL_WS_PORT="${CONTROL_WS_PORT}" \
    TERMINALS_CONTROL_TCP_PORT="${CONTROL_TCP_PORT}" \
    TERMINALS_CONTROL_HTTP_PORT="${CONTROL_HTTP_PORT}" \
    TERMINALS_ADMIN_HTTP_PORT="${ADMIN_PORT}" \
    TERMINALS_PHOTO_FRAME_HTTP_PORT="${PHOTO_PORT}" \
    TERMINALS_BUILD_SHA="${BUILD_SHA}" \
    TERMINALS_BUILD_DATE="${BUILD_DATE}" \
    go run ./cmd/server
  ) >"${SERVER_LOG}" 2>&1 &
  SERVER_PID=$!

  local deadline=$((SECONDS + 45))
  while (( SECONDS < deadline )); do
    if [[ -f "${SERVER_LOG}" ]] && grep -Eq "control service ready" "${SERVER_LOG}"; then
      echo "Server ready on 127.0.0.1:${GRPC_PORT}"
      return 0
    fi
    if ! kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
      wait "${SERVER_PID}" || true
      fail "server exited before reaching ready state"
    fi
    sleep 1
  done

  if ! wait_for_log "${SERVER_LOG}" "control service ready" "1"; then
    fail "server did not reach ready state within timeout"
  fi

  echo "Server ready on 127.0.0.1:${GRPC_PORT}"
}

start_client() {
  rotate_log "${CLIENT_LOG}"
  : >"${CLIENT_LOG}"
  echo "Starting local client (${CLIENT_DEVICE})..."

  if [[ "${CLIENT_DEVICE}" == "macos" ]]; then
    local launch_attempt=0
    while true; do
      if start_client_macos; then
        return 0
      fi
      if [[ "${launch_attempt}" -lt "${MAX_CLIENT_RESTART_ATTEMPTS}" ]]; then
        launch_attempt=$((launch_attempt + 1))
        echo "macOS client failed to build/launch; retrying (${launch_attempt}/${MAX_CLIENT_RESTART_ATTEMPTS})..."
        recover_macos_client_build
        sleep 2
        continue
      fi
      collect_macos_client_diagnostics "macos client failed to build/launch"
      fail "client exited immediately after launch"
    done
  fi

  if [[ "${CLIENT_DEVICE}" == "web-server" && "${TEST_MODE}" != "true" ]]; then
    CLIENT_FOREGROUND="true"
    local browser_url="http://localhost:${CLIENT_WEB_PORT}"
    echo "Browser client URL: ${browser_url}"
    if [[ "${OPEN_BROWSER}" == "true" ]]; then
      (
        local deadline=$((SECONDS + 120))
        while (( SECONDS < deadline )); do
          if [[ -f "${CLIENT_LOG}" ]] && grep -Eq 'lib/main.dart is being served at http://[^ ]+:[0-9]+' "${CLIENT_LOG}"; then
            if command -v open >/dev/null 2>&1; then
              open "${browser_url}" >/dev/null 2>&1 || true
            elif command -v xdg-open >/dev/null 2>&1; then
              xdg-open "${browser_url}" >/dev/null 2>&1 || true
            fi
            exit 0
          fi
          sleep 1
        done
        exit 0
      ) &
      BROWSER_OPENER_PID=$!
    fi

    set +e
    (
      trap - ERR
      set +E
      ulimit -f "${LOG_MAX_KB}" 2>/dev/null || true
      cd "${ROOT_DIR}/terminal_client"
      flutter run \
        -d web-server \
        --web-port="${CLIENT_WEB_PORT}" \
        --web-hostname="${CLIENT_WEB_HOST}" \
        "${CLIENT_DART_DEFINE_ARGS[@]}"
    ) 2>&1 | tee "${CLIENT_LOG}"
    local client_status=${PIPESTATUS[0]}
    set -e
    if [[ "${client_status}" -ne 0 ]]; then
      fail "client exited with status ${client_status}"
    fi
    return 0
  fi

  (
    trap - ERR
    set +E
    ulimit -f "${LOG_MAX_KB}" 2>/dev/null || true
    cd "${ROOT_DIR}/terminal_client"
    if [[ "${CLIENT_DEVICE}" == "web-server" ]]; then
      flutter run \
        -d web-server \
        --web-port="${CLIENT_WEB_PORT}" \
        --web-hostname="${CLIENT_WEB_HOST}" \
        "${CLIENT_DART_DEFINE_ARGS[@]}"
    else
      flutter run -d "${CLIENT_DEVICE}" --no-pub "${CLIENT_DART_DEFINE_ARGS[@]}"
    fi
  ) >"${CLIENT_LOG}" 2>&1 &
  CLIENT_PID=$!

  sleep "${CLIENT_STARTUP_DELAY_SECONDS}"
  if ! kill -0 "${CLIENT_PID}" >/dev/null 2>&1; then
    wait "${CLIENT_PID}" || true
    if [[ "${CLIENT_DEVICE}" == "macos" ]]; then
      collect_macos_client_diagnostics "client exited immediately after launch"
    fi
    fail "client exited immediately after launch"
  fi

  if [[ "${CLIENT_DEVICE}" == "web-server" ]]; then
    if ! wait_for_log "${CLIENT_LOG}" 'lib/main.dart is being served at http://[^ ]+:[0-9]+' "120"; then
      fail "web client did not report browser URL within timeout"
    fi
    local browser_url
    browser_url="$(extract_browser_url "${CLIENT_LOG}" || true)"
    if [[ -z "${browser_url}" ]]; then
      fail "web client started but browser URL could not be determined"
    fi
    echo "Browser client URL: ${browser_url}"
  fi
}

monitor_processes() {
  echo "Server log: ${SERVER_LOG}"
  echo "Client log: ${CLIENT_LOG}"

  if [[ "${TEST_MODE}" == "true" ]]; then
    if ! kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
      wait "${SERVER_PID}" || true
      fail "server exited unexpectedly"
    fi
    if ! kill -0 "${CLIENT_PID}" >/dev/null 2>&1; then
      wait "${CLIENT_PID}" || true
      fail "client exited unexpectedly"
    fi
    echo "Test mode: startup checks passed."
    return 0
  fi

  echo "Press Ctrl+C to stop both processes."

  while true; do
    if ! kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
      wait "${SERVER_PID}" || true
      fail "server exited unexpectedly"
    fi

    if ! kill -0 "${CLIENT_PID}" >/dev/null 2>&1; then
      wait "${CLIENT_PID}" || true
      if [[ "${CLIENT_RESTART_ATTEMPTS}" -lt "${MAX_CLIENT_RESTART_ATTEMPTS}" ]] \
        && [[ "${CLIENT_DEVICE}" == "macos" ]] \
        && [[ -f "${CLIENT_LOG}" ]] \
        && grep -Eq "Xcode build system has crashed|unexpected service error" "${CLIENT_LOG}"; then
        CLIENT_RESTART_ATTEMPTS=$((CLIENT_RESTART_ATTEMPTS + 1))
        echo "Client exited due to transient Xcode build failure; retrying (${CLIENT_RESTART_ATTEMPTS}/${MAX_CLIENT_RESTART_ATTEMPTS})..."
        recover_macos_client_build
        sleep 2
        start_client
        continue
      fi
      if [[ "${CLIENT_DEVICE}" == "macos" ]]; then
        collect_macos_client_diagnostics "client exited unexpectedly after retries"
      fi
      fail "client exited unexpectedly"
    fi

    sleep 1
  done
}
