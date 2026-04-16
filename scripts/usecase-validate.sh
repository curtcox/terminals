#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export PATH="${ROOT_DIR}/.bin:${ROOT_DIR}/.sdk/flutter/bin:${PATH}"

USECASE="${1:-${USECASE:-}}"
USECASE="$(echo "${USECASE}" | tr -d '[:space:]')"

if [[ -z "${USECASE}" ]]; then
  echo "usage: make usecase-validate USECASE=<ID|all>"
  echo "example: make usecase-validate USECASE=C1"
  exit 2
fi

run_go_test() {
  local pkg="$1"
  local regex="$2"
  echo "==> go test ${pkg} -run ${regex}"
  (cd "${ROOT_DIR}/terminal_server" && go test "${pkg}" -run "${regex}" -count=1)
}

run_usecase() {
  local id="$1"
  case "${id}" in
    C1)
      run_go_test ./internal/transport 'Test(Generated|Wire)SessionIntercom(EmitsRouteStream|StopEmitsStopStream|FanOutRelaysMediaToPeerSession)$'
      ;;
    C3)
      run_go_test ./internal/transport 'Test(Generated|Wire)Session(PASystemRelaysReceiverOverlayAndTransitions|VoicePAModeStartsPASystem|PASystemVoiceStopAliasesRelayCleanup)$'
      ;;
    C5)
      run_go_test ./internal/transport 'TestGeneratedSessionInternalVideoCallStartSetUIAndHangupFlow$'
      ;;
    D1)
      run_go_test ./cmd/server 'TestConfigurePhotoFrameUsesDirectorySlidesAndInterval$'
      run_go_test ./internal/transport 'TestHandleMessageHeartbeatRotatesPhotoFrameAfterInterval$'
      ;;
    M1)
      run_go_test ./cmd/server 'TestSilenceClassifierThroughAudioHubNotifiesAudioMonitor$'
      ;;
    M3)
      run_go_test ./internal/transport 'Test(Generated|Wire)SessionRedAlertRelaysBroadcastNotification$'
      ;;
    M4)
      run_go_test ./internal/transport 'Test(Generated|Wire)Session(VoiceStandDownStopsRedAlert|VoiceStopRedAlertStopsRedAlert)$'
      ;;
    S1)
      run_go_test ./internal/transport 'Test(Generated|Wire)Session(VoiceShowAllCamerasStartsMultiWindow|VoiceAllCamerasStartsMultiWindow)$'
      ;;
    S2)
      run_go_test ./internal/transport 'Test(Generated|Wire)SessionMultiWindowSetUI(IncludesFocusActions|AndFocusActionRouting)$'
      ;;
    S3)
      run_go_test ./internal/transport 'Test(Generated|Wire)SessionMultiWindowAudioMixAndFocusSelection$'
      ;;
    P1)
      run_go_test ./internal/transport 'Test(Generated|Wire)SessionTerminalTransitions$'
      ;;
    T1)
      run_go_test ./cmd/server 'TestRunDueTimerLoopProcessesTimers$'
      run_go_test ./internal/transport 'TestHandleMessageSystemRunDueTimers$'
      ;;
    *)
      echo "unsupported use case id: ${id}"
      echo "see docs/usecase-validation-matrix.md for supported IDs"
      exit 2
      ;;
  esac
}

if [[ "${USECASE}" == "all" ]]; then
  ids=(C1 C3 C5 D1 M1 M3 M4 S1 S2 S3 P1 T1)
  for id in "${ids[@]}"; do
    printf "\n### Validating %s\n" "${id}"
    run_usecase "${id}"
  done
  printf "\nAll supported use-case validations passed.\n"
  exit 0
fi

run_usecase "${USECASE}"
printf "\nUse-case %s validation passed.\n" "${USECASE}"
