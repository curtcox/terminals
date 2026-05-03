import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;
import 'package:terminal_client/diagnostics/build_metadata.dart';
import 'package:terminal_client/ui/server_driven_node_key.dart';

const String registerMetadataServerBuildShaKey = 'server_build_sha';
const String registerMetadataServerBuildDateKey = 'server_build_date';

class CommandDiagnosticsRequestIDs {
  const CommandDiagnosticsRequestIDs({
    this.runtimeStatus = '',
    this.deviceStatus = '',
    this.scenarioRegistry = '',
    this.playbackArtifacts = '',
    this.playbackMetadata = '',
  });

  final String runtimeStatus;
  final String deviceStatus;
  final String scenarioRegistry;
  final String playbackArtifacts;
  final String playbackMetadata;
}

class CommandDiagnosticsUpdate {
  const CommandDiagnosticsUpdate({
    required this.title,
    required this.data,
  });

  final String title;
  final Map<String, String> data;
}

class RegisterMetadataUpdate {
  const RegisterMetadataUpdate({
    required this.serverBuildSha,
    required this.serverBuildDate,
    required this.metadata,
  });

  final String serverBuildSha;
  final String serverBuildDate;
  final Map<String, String> metadata;

  bool get hasDiagnosticsData => metadata.isNotEmpty;
}

String statusFromConnectResponse(ConnectResponse response) {
  if (response.hasError()) {
    return 'Server error';
  }
  if (response.hasTransitionUi()) {
    return 'UI transition';
  }
  if (response.hasStartStream()) {
    return 'Stream started';
  }
  if (response.hasStopStream()) {
    return 'Stream stopped';
  }
  if (response.hasRouteStream()) {
    return 'Route updated';
  }
  if (response.hasWebrtcSignal()) {
    return 'WebRTC signal';
  }
  if (response.hasPlayAudio()) {
    return 'Play audio';
  }
  if (response.hasInstallBundle()) {
    return 'Bundle install requested';
  }
  if (response.hasRemoveBundle()) {
    return 'Bundle removal requested';
  }
  if (response.hasStartFlow()) {
    return 'Flow start requested';
  }
  if (response.hasPatchFlow()) {
    return 'Flow patch requested';
  }
  if (response.hasStopFlow()) {
    return 'Flow stop requested';
  }
  if (response.hasRequestArtifact()) {
    return 'Artifact requested';
  }
  if (response.hasBugReportAck()) {
    return 'Bug report filed';
  }
  if (response.hasUpdateUi()) {
    return 'UI patched';
  }
  if (response.hasRegisterAck()) {
    return 'Registered';
  }
  if (response.hasCommandResult()) {
    return 'Command response';
  }
  if (response.hasSetUi()) {
    return 'UI updated';
  }
  return 'Connected';
}

uiv1.Node? applyUpdateUi({
  required uiv1.Node? currentRoot,
  required uiv1.UpdateUI update,
}) {
  if (!update.hasNode()) {
    return currentRoot;
  }
  final targetID = update.componentId.trim();
  final replacement = update.node.deepCopy();
  if (targetID.isEmpty) {
    return replacement;
  }
  if (currentRoot == null) {
    return null;
  }

  final root = currentRoot.deepCopy();
  if (serverDrivenNodeId(root) == targetID) {
    return replacement;
  }
  final replaced = replaceNodeByID(
    current: root,
    targetID: targetID,
    replacement: replacement,
  );
  if (!replaced) {
    return currentRoot;
  }
  return root;
}

bool replaceNodeByID({
  required uiv1.Node current,
  required String targetID,
  required uiv1.Node replacement,
}) {
  for (var i = 0; i < current.children.length; i++) {
    final child = current.children[i];
    if (serverDrivenNodeId(child) == targetID) {
      current.children[i] = replacement.deepCopy();
      return true;
    }
    if (replaceNodeByID(
      current: child,
      targetID: targetID,
      replacement: replacement,
    )) {
      return true;
    }
  }
  return false;
}

CommandDiagnosticsUpdate? commandDiagnosticsFromResponse({
  required ConnectResponse response,
  required CommandDiagnosticsRequestIDs pendingRequestIDs,
}) {
  if (!response.hasCommandResult()) {
    return null;
  }
  final result = response.commandResult;
  if (result.data.isEmpty) {
    return null;
  }

  final title = diagnosticsTitleForCommandResult(
    result: result,
    pendingRequestIDs: pendingRequestIDs,
  );
  if (title.isEmpty) {
    return null;
  }
  return CommandDiagnosticsUpdate(
    title: title,
    data: Map<String, String>.from(result.data),
  );
}

String diagnosticsTitleForCommandResult({
  required CommandResult result,
  required CommandDiagnosticsRequestIDs pendingRequestIDs,
}) {
  final requestID = result.requestId;
  if (requestID.isNotEmpty && requestID == pendingRequestIDs.runtimeStatus) {
    return 'runtime_status';
  }
  if (requestID.isNotEmpty && requestID == pendingRequestIDs.deviceStatus) {
    return 'device_status';
  }
  if (requestID.isNotEmpty && requestID == pendingRequestIDs.scenarioRegistry) {
    return 'scenario_registry';
  }
  if (requestID.isNotEmpty &&
      requestID == pendingRequestIDs.playbackArtifacts) {
    return 'list_playback_artifacts';
  }
  if (requestID.isNotEmpty && requestID == pendingRequestIDs.playbackMetadata) {
    return 'playback_metadata';
  }

  return switch (result.notification) {
    'System query: runtime_status' => 'runtime_status',
    'System query: device_status' => 'device_status',
    'System query: scenario_registry' => 'scenario_registry',
    'System query: list_playback_artifacts' => 'list_playback_artifacts',
    'Playback metadata ready' => 'playback_metadata',
    _ => '',
  };
}

RegisterMetadataUpdate? registerMetadataFromResponse(ConnectResponse response) {
  if (!response.hasRegisterAck()) {
    return null;
  }
  final metadata = Map<String, String>.from(response.registerAck.metadata);
  return RegisterMetadataUpdate(
    serverBuildSha: normalizeBuildValue(
      metadata[registerMetadataServerBuildShaKey] ?? '',
    ),
    serverBuildDate: normalizeBuildValue(
      metadata[registerMetadataServerBuildDateKey] ?? '',
    ),
    metadata: metadata,
  );
}

String? bundleIDFromFlowPlan(iov1.FlowPlan? plan) {
  if (plan == null) {
    return null;
  }
  for (final node in plan.nodes) {
    final bundleID = (node.args['bundle_id'] ?? '').trim();
    if (bundleID.isNotEmpty) {
      return bundleID;
    }
  }
  return null;
}

String playAudioSourceLabel(iov1.PlayAudio playAudio) {
  return switch (playAudio.whichSource()) {
    iov1.PlayAudio_Source.pcmData => 'pcm_data',
    iov1.PlayAudio_Source.url => 'url',
    iov1.PlayAudio_Source.ttsText => 'tts_text',
    iov1.PlayAudio_Source.notSet => 'not_set',
  };
}

int playAudioPcmByteCount(iov1.PlayAudio playAudio) {
  if (playAudio.whichSource() != iov1.PlayAudio_Source.pcmData) {
    return 0;
  }
  return playAudio.pcmData.length;
}
