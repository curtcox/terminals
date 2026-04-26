#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export PATH="${ROOT_DIR}/.bin:${ROOT_DIR}/.sdk/flutter/bin:${PATH}"

INFO="${INFO:-}"
if [[ "${1:-}" == "--info" ]]; then
  INFO="1"
  shift
fi

USECASE="${1:-${USECASE:-}}"
USECASE="$(echo "${USECASE}" | tr -d '[:space:]')"

if [[ -z "${USECASE}" ]]; then
  echo "usage: make usecase-validate USECASE=<ID|all> [INFO=1]"
  echo "   or: scripts/usecase-validate.sh --info <ID|all>"
  echo "example: make usecase-validate USECASE=C1"
  exit 2
fi

metadata() {
  local id="$1"
  case "${id}" in
    B1) echo "B1|Scenario|transport input bug-report action coverage for modality parity" ;;
    B2) echo "B2|Scenario|diagnostics bug-report service cross-device subject/offline coverage" ;;
    B3) echo "B3|Scenario|diagnostics service autodetect merge test" ;;
    B4) echo "B4|Scenario|admin bug intake/list/detail and filter tests" ;;
    B5) echo "B5|Scenario|admin bug intake JSON SIP source + transcript hints test" ;;
    C1) echo "C1|Transport|internal/transport generated+wire integration tests" ;;
    C3) echo "C3|Transport|PA relay, voice start, voice stop alias tests" ;;
    C5) echo "C5|Transport|TestGeneratedSessionInternalVideoCallStartSetUIAndHangupFlow" ;;
    D1) echo "D1|Scenario|photo-frame config + heartbeat rotation tests" ;;
    M1) echo "M1|Scenario|silence classifier integration test" ;;
    M2) echo "M2|Scenario|audio monitor runtime test for dryer beep detection" ;;
    M3) echo "M3|Transport|generated+wire red alert integration tests" ;;
    M4) echo "M4|Transport|generated+wire voice stop/stand-down tests" ;;
    S1) echo "S1|Transport|generated+wire voice show-all-cameras tests" ;;
    S2) echo "S2|Transport|generated+wire focus-action routing tests" ;;
    S3) echo "S3|Transport|generated+wire multi-window audio mix tests" ;;
    P1) echo "P1|Transport|generated+wire terminal transition tests" ;;
    PL1) echo "PL1|Contract|capability message room/thread/unread acknowledgement lifecycle test" ;;
    PL8) echo "PL8|Contract|interactive session join/leave and control lifecycle capability tests" ;;
    PL20) echo "PL20|Contract|capability artifact template save/apply and artifact history tests" ;;
    T1) echo "T1|Smoke|due-timer loop; transport run_due_timers; kitchen timer package smoke test; future TAL simulation coverage" ;;
    *)
      echo "unsupported use case id: ${id}" >&2
      exit 2
      ;;
  esac
}

all_ids=(B1 B2 B3 B4 B5 C1 C3 C5 D1 M1 M2 M3 M4 S1 S2 S3 P1 PL1 PL8 PL20 T1)

run_go_test() {
  local pkg="$1"
  local regex="$2"
  echo "==> go test ${pkg} -run ${regex}"
  (cd "${ROOT_DIR}/terminal_server" && go test "${pkg}" -run "${regex}" -count=1)
}

run_app_test() {
  local name="$1"
  echo "==> go run ./cmd/term app test ${name}"
  (cd "${ROOT_DIR}/terminal_server" && go run ./cmd/term app test "${name}")
}

run_usecase() {
  local id="$1"
  case "${id}" in
    B1)
      run_go_test ./internal/transport 'TestHandleMessageInputBugReportAction(FilesReport|RespectsModalitySources)$'
      ;;
    B2)
      run_go_test ./internal/diagnostics/bugreport 'TestServiceFileCrossDeviceSubjectMarksOfflineWhenUnavailable$'
      ;;
    B3)
      run_go_test ./internal/diagnostics/bugreport 'TestServiceAutodetectMerge$'
      ;;
    B4)
      run_go_test ./internal/admin 'TestBug(IntakeAndListAndDetail|ListFilterByTag)$'
      ;;
    B5)
      run_go_test ./internal/admin 'TestBugIntakeJSONSIPSourceAndTranscriptHints$'
      ;;
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
    M2)
      run_go_test ./internal/scenario 'TestRuntimeAudioMonitorNotifiesWhenDryerBeeps$'
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
    PL1)
      run_go_test ./internal/capability 'TestMessageRoomThreadUnreadAcknowledgeLifecycle$'
      ;;
    PL8)
      run_go_test ./internal/capability 'Test(SessionJoinAndLeave|SessionAttachDetachAndControlLifecycle)$'
      ;;
    PL20)
      run_go_test ./internal/capability 'Test(ArtifactReplaceAndTemplateApply|MessageAcknowledgeUnreadAndArtifactPatch)$'
      ;;
    T1)
      run_go_test ./cmd/server 'TestRunDueTimerLoopProcessesTimers$'
      run_go_test ./internal/transport 'TestHandleMessageSystemRunDueTimers$'
      run_app_test kitchen_timer
      ;;
    *)
      echo "unsupported use case id: ${id}"
      echo "see docs/usecase-validation-matrix.md for supported IDs"
      exit 2
      ;;
  esac
}

if [[ "${INFO}" == "1" ]]; then
  if [[ "${USECASE}" == "all" ]]; then
    for id in "${all_ids[@]}"; do
      metadata "${id}"
    done
    exit 0
  fi
  metadata "${USECASE}"
  exit 0
fi

if [[ "${USECASE}" == "all" ]]; then
  for id in "${all_ids[@]}"; do
    printf "\n### Validating %s\n" "${id}"
    run_usecase "${id}"
  done
  printf "\nAll supported use-case validations passed.\n"
  exit 0
fi

run_usecase "${USECASE}"
printf "\nUse-case %s validation passed.\n" "${USECASE}"
