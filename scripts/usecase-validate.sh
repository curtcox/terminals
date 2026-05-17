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
    AA1) echo "AA1|Simulation|external automation agent triggers announcement scenario via manual API; display terminal receives announcement_audio route; YAML scenario in internal/usecasevalidation/testdata/aa1-webhook-announce.yaml" ;;
    AA2) echo "AA2|Simulation|monitoring agent arms audio monitoring via manual API; fake classifier emits dryer_beep; broadcast targeted at agent device ID" ;;
    AA3) echo "AA3|Simulation|AI agent resolves ambiguous voice command via fake LLM; LLM-decoded announce intent routes announcement_audio to a second terminal" ;;
    AA4) echo "AA4|Simulation|scheduling agent creates and cancels a timer via manual API; cancelled timer produces no 'Timer done!' broadcast; YAML scenario in internal/usecasevalidation/testdata/aa4-timer-cancel.yaml" ;;
    AA5) echo "AA5|Simulation|vision analysis agent: FakeVisionAnalyzer returns caption and labels; camera_activity sensor triggers analysis; broadcast verified with caption, labels, and agent device target" ;;
    AA6) echo "AA6|Simulation|admin scripts run over seeded sim/store/ui/bus fixture plus mutating Layer 2 message, board, artifact, canvas, and session paths" ;;
    V1) echo "V1|Simulation|voice assistant wake-and-answer: 'assistant <question>' triggers VoiceAssistantScenario; FakeLLM queried; response broadcast back to source device" ;;
    V2) echo "V2|Simulation|voice recipe request: 'assistant how do I make <dish>' triggers VoiceAssistantScenario; FakeLLM returns recipe; broadcast back to kitchen terminal" ;;
    V3) echo "V3|Simulation|voice general-knowledge query: 'assistant <question>' triggers VoiceAssistantScenario; FakeLLM returns answer; broadcast back to requesting device" ;;
    B1) echo "B1|Scenario|transport input bug-report action coverage for modality parity" ;;
    B2) echo "B2|Scenario|diagnostics bug-report service cross-device subject/offline coverage" ;;
    B3) echo "B3|Scenario|diagnostics service autodetect merge test" ;;
    B4) echo "B4|Scenario|admin bug intake/list/detail and filter tests" ;;
    B5) echo "B5|Scenario|admin bug intake JSON SIP source + transcript hints test" ;;
    C1) echo "C1|Transport|internal/transport generated+wire integration tests; internal/usecasevalidation harness-backed evidence test" ;;
    C2) echo "C2|Simulation|whole-house announcement: 1 speaker + 3 receivers; harness-backed broadcast routing and no-duplicate-delivery evidence" ;;
    C3) echo "C3|Transport|PA relay, voice start, voice stop alias tests" ;;
    C5) echo "C5|Transport|TestGeneratedSessionInternalVideoCallStartSetUIAndHangupFlow" ;;
    I3) echo "I3|Transport|capability manifest on connection: server records full hardware capability set declared by connecting device" ;;
    I4) echo "I4|Transport|device registry capability query: placement engine returns only devices matching required capability set; generated+wire variants register two devices and verify filtered routing" ;;
    I6) echo "I6|Scenario|preemption: engine suspends lower-priority scenario routes when higher-priority scenario starts and resumes them on stop" ;;
    D3) echo "D3|Transport|standby/clock mode activated by voice or manual command; server notifies requesting device" ;;
    D1) echo "D1|Scenario|photo-frame config + heartbeat rotation tests" ;;
    D2) echo "D2|Scenario|photo frame yields to higher-priority scenario (alert/call) and resumes afterward" ;;
    M1) echo "M1|Scenario|silence classifier integration test" ;;
    M2) echo "M2|Scenario|audio monitor runtime test for dryer beep detection" ;;
    M3) echo "M3|Transport|generated+wire red alert integration tests" ;;
    M4) echo "M4|Transport|generated+wire voice stop/stand-down tests" ;;
    M5) echo "M5|Simulation|camera activity watch: time-window policy gates alerts; harness-backed sensor routing and suppression evidence" ;;
    P2) echo "P2|Scenario|REPL session mobility and coexistence lifecycle test" ;;
    P3) echo "P3|Scenario|REPL ai ask/gen commands plus mutating approval-gate metadata test" ;;
    P4) echo "P4|Scenario|sticky REPL AI provider/model survives detach and reattach" ;;
    S1) echo "S1|Transport|generated+wire voice show-all-cameras tests" ;;
    S2) echo "S2|Transport|generated+wire focus-action routing tests" ;;
    S3) echo "S3|Transport|generated+wire multi-window audio mix tests" ;;
    P1) echo "P1|Transport|generated+wire terminal transition tests" ;;
    PL1) echo "PL1|Contract|capability message room/thread/unread acknowledgement lifecycle test" ;;
    PL8) echo "PL8|Contract|interactive session join/leave and control lifecycle capability tests" ;;
    PL20) echo "PL20|Contract|capability artifact template save/apply and artifact history tests" ;;
    T1) echo "T1|Simulation|due-timer loop; transport run_due_timers; kitchen timer package smoke test; voice-path fake-clock harness (timer fires via synthetic time advance)" ;;
    T2) echo "T2|Simulation|timer reminder: fake-clock advance triggers ProcessDueTimers; broadcast confirms 'Timer done!' without real elapsed time; YAML scenario in internal/usecasevalidation/testdata/t2-timer-reminder.yaml" ;;
    T3) echo "T3|Simulation|school-morning monitor: no camera activity by alert time notifies parent; activity before alert cancels notification; YAML scenarios in internal/usecasevalidation/testdata/t3-t4-school-morning.yaml and t3-activity-cancels-alert.yaml" ;;
    T4) echo "T4|Simulation|school-morning warning: bus-warning broadcast fires to child-room at configured warning time via synthetic clock advance; YAML scenario in internal/usecasevalidation/testdata/t3-t4-school-morning.yaml" ;;
    UI1) echo "UI1|Transport|idle photo-frame SetUI includes scoped corner affordance (terminal-ui plan)" ;;
    UI2) echo "UI2|Transport|corner.open toggles menu overlay claim without disturbing main" ;;
    UI3) echo "UI3|Transport|menu overlay default MIXED policy: main pointer blocked, audio live" ;;
    UI4) echo "UI4|Transport+Client|privacy toggle: proto withdrawal, server routes, post-cutover frame drop" ;;
    UI5) echo "UI5|Client|privacy mode: no VoiceAudio after withdrawal; no client-chrome privacy indicators" ;;
    UI6) echo "UI6|Transport|wake-word path activates scenario (single client)" ;;
    UI7) echo "UI7|Transport|wake-word dedupe across two clients dispatches at most one intent" ;;
    UI8) echo "UI8|Transport+Client|capability delta on rotation/resize; overlay survives orientation delta" ;;
    UI9) echo "UI9|Simulation|reconnect restores main + overlay UI state; harness-backed evidence (RECON-1)" ;;
    UI10) echo "UI10|Scenario|registry corner affordance reachability invariant for every main-layer scenario" ;;
    *)
      echo "unsupported use case id: ${id}" >&2
      exit 2
      ;;
  esac
}

all_ids=(AA1 AA2 AA3 AA4 AA5 AA6 B1 B2 B3 B4 B5 C1 C2 C3 C5 I3 I4 I6 D3 D1 D2 M1 M2 M3 M4 M5 P2 P3 P4 S1 S2 S3 P1 PL1 PL8 PL20 T1 T2 T3 T4 UI1 UI2 UI3 UI4 UI5 UI6 UI7 UI8 UI9 UI10 V1 V2 V3)

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

run_flutter_test() {
  local rel_path="$1"
  local plain_name="$2"
  echo "==> flutter test ${rel_path} --plain-name ${plain_name}"
  (cd "${ROOT_DIR}/terminal_client" && flutter test "${rel_path}" --plain-name "${plain_name}")
}

run_usecase() {
  local id="$1"
  case "${id}" in
    AA1)
      run_go_test ./internal/usecasevalidation 'TestUseCaseAA1WithEvidence$'
      run_go_test ./internal/usecasevalidation 'TestYAMLScenarioAA1WebhookAnnounce$'
      ;;
    AA2)
      run_go_test ./internal/usecasevalidation 'TestUseCaseAA2WithEvidence$'
      ;;
    AA3)
      run_go_test ./internal/usecasevalidation 'TestUseCaseAA3WithEvidence$'
      ;;
    AA4)
      run_go_test ./internal/usecasevalidation 'TestUseCaseAA4WithEvidence$'
      run_go_test ./internal/usecasevalidation 'TestYAMLScenarioAA4TimerCancel$'
      ;;
    AA5)
      run_go_test ./internal/usecasevalidation 'TestUseCaseAA5WithEvidence$'
      ;;
    AA6)
      run_go_test ./internal/admin 'TestScriptsRunCrossUsecaseSimulationFixture$'
      ;;
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
      run_go_test ./internal/usecasevalidation 'TestUseCaseC1WithEvidence$'
      ;;
    C2)
      run_go_test ./internal/usecasevalidation 'TestUseCaseC2WithEvidence$'
      ;;
    C3)
      run_go_test ./internal/transport 'Test(Generated|Wire)Session(PASystemRelaysReceiverOverlayAndTransitions|VoicePAModeStartsPASystem|PASystemVoiceStopAliasesRelayCleanup)$'
      ;;
    C5)
      run_go_test ./internal/transport 'TestGeneratedSessionInternalVideoCallStartSetUIAndHangupFlow$'
      ;;
    I3)
      run_go_test ./internal/transport 'Test(Generated|Wire)SessionCapabilityManifestReportedOnConnection$'
      ;;
    I4)
      run_go_test ./internal/transport 'Test(Generated|Wire)SessionDeviceRegistryCapabilityQueryRoutesScenarioToMatchingDevices$'
      ;;
    I6)
      run_go_test ./internal/scenario 'TestRuntime(Intercom|PA)PreemptedByRedAlertSuspendsAndResumesRoutes$'
      ;;
    D3)
      run_go_test ./internal/transport 'Test(Generated|Wire)SessionStandbyModeActivatedByVoiceCommand$'
      ;;
    D1)
      run_go_test ./cmd/server 'TestConfigurePhotoFrameUsesDirectorySlidesAndInterval$'
      run_go_test ./internal/transport 'TestHandleMessageHeartbeatRotatesPhotoFrameAfterInterval$'
      ;;
    D2)
      run_go_test ./internal/scenario 'TestRuntimePhotoFrameYields(ToHigherPriorityScenarioAndResumes|ToCallAndResumes)$'
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
    M5)
      run_go_test ./internal/usecasevalidation 'TestUseCaseM5WithEvidence$'
      ;;
    P2)
      run_go_test ./internal/replsession 'TestUseCaseP2SessionMobilityAndCoexistence$'
      ;;
    P3)
      run_go_test ./internal/repl 'TestUseCaseP3AIAssistanceAskGenerateAndMutatingGateMetadata$'
      ;;
    P4)
      run_go_test ./internal/replsession 'TestUseCaseP4StickyAISelectionSurvivesDetachReattach$'
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
      run_go_test ./internal/usecasevalidation 'TestUseCaseT1WithEvidence$'
      ;;
    T2)
      run_go_test ./internal/usecasevalidation 'TestUseCaseT2WithEvidence$'
      run_go_test ./internal/usecasevalidation 'TestYAMLScenarioT2TimerReminder$'
      ;;
    T3)
      run_go_test ./internal/usecasevalidation 'TestUseCaseT3(T4WithEvidence|ActivityCancelsAlert)$'
      run_go_test ./internal/usecasevalidation 'TestYAMLScenarioT3(ActivityCancelsAlert|T4SchoolMorning)$'
      ;;
    T4)
      run_go_test ./internal/usecasevalidation 'TestUseCaseT3T4WithEvidence$'
      run_go_test ./internal/usecasevalidation 'TestYAMLScenarioT3T4SchoolMorning$'
      ;;
    UI1)
      run_go_test ./internal/transport 'TestUseCaseUI1IdlePhotoFrameSetUIIncludesCornerAffordance$'
      ;;
    UI2)
      run_go_test ./internal/transport 'TestHandleMessageInputCornerOpenTogglesMenuOverlayAndClaim$'
      ;;
    UI3)
      run_go_test ./internal/transport 'TestHandleMessageInputMenuOverlayDefaultMixedPolicyBlocksMainPointerButKeepsAudioLive$'
      ;;
    UI4)
      run_go_test ./internal/transport 'TestHandleMessageVoiceAudioDropsPostPrivacyCutoverFrames$'
      run_go_test ./internal/transport 'TestGeneratedSessionPrivacyToggleCapabilityLossStopsAudioAndVideoRoutes$'
      run_go_test ./internal/transport 'TestGeneratedSessionPrivacyToggleExitReaddsCapabilitiesAndResumesClaims$'
      run_go_test ./internal/transport 'TestProtoRoundTripCapabilityDeltaPrivacyWithdrawalOmitsMicAndCameraFields$'
      run_flutter_test test/widget_test_media.dart 'privacy.toggle stops local capture before sending capability delta'
      ;;
    UI5)
      run_flutter_test test/widget_test_reconnect.dart 'wake-word utterance does not send VoiceAudio after privacy.toggle withdraws microphone capability'
      run_flutter_test test/widget_test_reconnect.dart 'privacy.toggle does not render persistent client-chrome privacy/capture indicator'
      ;;
    UI6)
      run_go_test ./internal/transport 'TestControlStreamVoiceAudioWakeWordDetectedActivatesScenario$'
      ;;
    UI7)
      run_go_test ./internal/transport 'TestControlStreamVoiceAudioWakeWordDedupeAcrossClientsDispatchesAtMostOneIntent$'
      ;;
    UI8)
      run_flutter_test test/widget_test_connection.dart 'deterministic metrics seam emits CapabilityDelta on rotation with fresh generation'
      run_flutter_test test/widget_test_connection.dart 'deterministic metrics seam emits CapabilityDelta on resize'
      run_go_test ./internal/transport 'TestCapabilityDeltaWhileMenuOverlayOpenPreservesMainAndOverlayActivations$'
      ;;
    UI9)
      run_go_test ./internal/transport 'TestGeneratedSessionUI_RECON_1$'
      run_go_test ./internal/usecasevalidation 'TestUseCaseUI9WithEvidence$'
      ;;
    UI10)
      run_go_test ./internal/transport 'TestWithCornerAffordance_RegistryReachabilityInvariant$'
      ;;
    V1)
      run_go_test ./internal/usecasevalidation 'TestUseCaseV1WithEvidence$'
      ;;
    V2)
      run_go_test ./internal/usecasevalidation 'TestUseCaseV2WithEvidence$'
      ;;
    V3)
      run_go_test ./internal/usecasevalidation 'TestUseCaseV3WithEvidence$'
      ;;
    *)
      echo "unsupported use case id: ${id}"
      echo "see docs/usecase-validation-matrix.md for supported IDs"
      exit 2
      ;;
  esac
}

write_usecase_result() {
  local id="$1"
  local started="$2"
  local status="$3"
  local failure="${4:-}"
  RESULT_USECASE_ID="${id}" \
    RESULT_STARTED="${started}" \
    RESULT_STATUS="${status}" \
    RESULT_FAILURE="${failure}" \
    RESULT_ROOT="${ROOT_DIR}" \
    python3 - <<'PY'
import json
import os
from datetime import datetime, timezone
from pathlib import Path

root = Path(os.environ["RESULT_ROOT"])
usecase_id = os.environ["RESULT_USECASE_ID"]
started = os.environ["RESULT_STARTED"]
status = int(os.environ["RESULT_STATUS"])
failure = os.environ.get("RESULT_FAILURE", "")
ended = datetime.now(timezone.utc)
result = {
    "run_id": str(int(ended.timestamp() * 1_000_000_000)),
    "usecase_id": usecase_id,
    "scenario_name": f"scripts/usecase-validate.sh {usecase_id}",
    "timestamp_start": started,
    "timestamp_end": ended.isoformat().replace("+00:00", "Z"),
    "pass": status == 0,
}
if status != 0:
    result["failing_assertions"] = [failure or f"use-case validation exited with status {status}"]

out = root / "artifacts" / "usecases" / usecase_id / "result.json"
out.parent.mkdir(parents=True, exist_ok=True)
out.write_text(json.dumps(result, indent=2) + "\n")
PY
}

run_and_record_usecase() {
  local id="$1"
  local started
  started="$(python3 - <<'PY'
from datetime import datetime, timezone
print(datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"))
PY
)"
  if run_usecase "${id}"; then
    write_usecase_result "${id}" "${started}" 0
    return 0
  fi
  local status="$?"
  write_usecase_result "${id}" "${started}" "${status}" "use-case validation exited with status ${status}"
  return "${status}"
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
    run_and_record_usecase "${id}"
  done
  printf "\nAll supported use-case validations passed.\n"
  exit 0
fi

run_and_record_usecase "${USECASE}"
printf "\nUse-case %s validation passed.\n" "${USECASE}"
