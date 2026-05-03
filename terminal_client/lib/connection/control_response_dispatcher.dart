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

class SynchronousMediaControlUpdate {
  const SynchronousMediaControlUpdate({
    this.startStreamID = '',
    this.startStreamNotification = '',
    this.stopStreamID = '',
    this.stopStreamNotification = '',
    this.routeStreamID = '',
    this.routeNotification = '',
    this.webrtcSignalNotification = '',
  });

  final String startStreamID;
  final String startStreamNotification;
  final String stopStreamID;
  final String stopStreamNotification;
  final String routeStreamID;
  final String routeNotification;
  final String webrtcSignalNotification;

  bool get shouldAcknowledgeStartStream => startStreamID.isNotEmpty;

  String get lastNotification {
    if (webrtcSignalNotification.isNotEmpty) {
      return webrtcSignalNotification;
    }
    if (routeNotification.isNotEmpty) {
      return routeNotification;
    }
    if (stopStreamNotification.isNotEmpty) {
      return stopStreamNotification;
    }
    return startStreamNotification;
  }
}

class ServerDrivenUiEventUpdate {
  const ServerDrivenUiEventUpdate({
    required this.kind,
    required this.componentId,
    required this.detail,
  });

  final String kind;
  final String componentId;
  final String detail;
}

class ServerDrivenTransitionHint {
  const ServerDrivenTransitionHint({
    required this.transition,
    required this.duration,
    required this.notification,
  });

  final String transition;
  final Duration duration;
  final String notification;
}

class ServerDrivenUiResponseUpdate {
  const ServerDrivenUiResponseUpdate({
    required this.activeRoot,
    required this.uiChanged,
    required this.events,
    this.transitionHint,
  });

  final uiv1.Node? activeRoot;
  final bool uiChanged;
  final List<ServerDrivenUiEventUpdate> events;
  final ServerDrivenTransitionHint? transitionHint;

  bool get hasUiWork =>
      uiChanged || events.isNotEmpty || transitionHint != null;
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

SynchronousMediaControlUpdate synchronousMediaControlUpdateFromResponse(
  ConnectResponse response,
) {
  var startStreamID = '';
  var startStreamNotification = '';
  if (response.hasStartStream()) {
    final start = response.startStream;
    startStreamID = start.streamId;
    if (start.kind.isNotEmpty) {
      startStreamNotification = 'Start stream: ${start.kind} '
          '(${start.streamId})';
    }
  }

  var stopStreamID = '';
  var stopStreamNotification = '';
  if (response.hasStopStream()) {
    stopStreamID = response.stopStream.streamId;
    if (stopStreamID.isNotEmpty) {
      stopStreamNotification = 'Stop stream: $stopStreamID';
    }
  }

  var routeStreamID = '';
  var routeNotification = '';
  if (response.hasRouteStream()) {
    final route = response.routeStream;
    routeStreamID = route.streamId;
    routeNotification = 'Route: ${route.sourceDeviceId} -> '
        '${route.targetDeviceId} (${route.kind})';
  }

  var webrtcSignalNotification = '';
  if (response.hasWebrtcSignal()) {
    final signal = response.webrtcSignal;
    webrtcSignalNotification =
        'WebRTC signal: ${signal.signalType} (${signal.streamId})';
  }

  return SynchronousMediaControlUpdate(
    startStreamID: startStreamID,
    startStreamNotification: startStreamNotification,
    stopStreamID: stopStreamID,
    stopStreamNotification: stopStreamNotification,
    routeStreamID: routeStreamID,
    routeNotification: routeNotification,
    webrtcSignalNotification: webrtcSignalNotification,
  );
}

ServerDrivenUiResponseUpdate? serverDrivenUiUpdateFromResponse({
  required ConnectResponse response,
  required uiv1.Node? currentRoot,
}) {
  var nextRoot = currentRoot;
  var uiChanged = false;
  final events = <ServerDrivenUiEventUpdate>[];
  ServerDrivenTransitionHint? transitionHint;

  if (response.hasSetUi() && response.setUi.hasRoot()) {
    nextRoot = response.setUi.root.deepCopy();
    uiChanged = true;
    events.add(
      ServerDrivenUiEventUpdate(
        kind: 'set_ui',
        componentId: serverDrivenNodeId(response.setUi.root),
        detail: 'root updated',
      ),
    );
  }
  if (response.hasUpdateUi()) {
    final updatedRoot = applyUpdateUi(
      currentRoot: nextRoot,
      update: response.updateUi,
    );
    if (!identical(updatedRoot, nextRoot)) {
      uiChanged = true;
    }
    nextRoot = updatedRoot;
    events.add(
      ServerDrivenUiEventUpdate(
        kind: 'update_ui',
        componentId: response.updateUi.componentId,
        detail: 'component patch',
      ),
    );
  }
  if (response.hasTransitionUi()) {
    transitionHint = transitionHintFromResponse(response.transitionUi);
    if (nextRoot != null) {
      uiChanged = true;
    }
    events.add(
      ServerDrivenUiEventUpdate(
        kind: 'transition_ui',
        componentId: serverDrivenNodeId(nextRoot ?? uiv1.Node()),
        detail: response.transitionUi.transition,
      ),
    );
  }

  final update = ServerDrivenUiResponseUpdate(
    activeRoot: nextRoot,
    uiChanged: uiChanged,
    events: events,
    transitionHint: transitionHint,
  );
  return update.hasUiWork ? update : null;
}

ServerDrivenTransitionHint transitionHintFromResponse(
  uiv1.TransitionUI transitionUi,
) {
  final transition = transitionUi.transition.trim().toLowerCase();
  final hasTransition = transition.isNotEmpty && transition != 'none';
  final defaultDuration = hasTransition ? 250 : 0;
  final durationMs =
      transitionUi.durationMs > 0 ? transitionUi.durationMs : defaultDuration;
  return ServerDrivenTransitionHint(
    transition: transition,
    duration: Duration(milliseconds: durationMs),
    notification:
        'Transition: ${transitionUi.transition} (${transitionUi.durationMs}ms)',
  );
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
  final ack = response.registerAck;
  final metadata = Map<String, String>.from(ack.metadata);
  final typedServerMetadata = ack.hasServerMetadata() ? ack.serverMetadata : null;
  final typedBuild = typedServerMetadata != null && typedServerMetadata.hasBuild()
      ? typedServerMetadata.build
      : null;
  final typedBuildSha = typedBuild?.sha ?? '';
  final typedBuildDate = typedBuild?.dateRfc3339 ?? '';
  return RegisterMetadataUpdate(
    serverBuildSha: normalizeBuildValue(
      typedBuildSha.isNotEmpty
          ? typedBuildSha
          : (metadata[registerMetadataServerBuildShaKey] ?? ''),
    ),
    serverBuildDate: normalizeBuildValue(
      typedBuildDate.isNotEmpty
          ? typedBuildDate
          : (metadata[registerMetadataServerBuildDateKey] ?? ''),
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

String firstPlaybackArtifactID(Map<String, String> data) {
  final keys = data.keys.toList()..sort();
  for (final key in keys) {
    final parts = data[key]?.split('|') ?? const <String>[];
    if (parts.isNotEmpty && parts.first.trim().isNotEmpty) {
      return parts.first.trim();
    }
  }
  return '';
}

List<String> applicationIntentsFromDiagnostics(
  Map<String, String> data, {
  String defaultIntent = 'terminal',
}) {
  final fallback = defaultIntent.trim();
  final discovered = data.keys
      .map((key) => key.trim())
      .where((key) => key.isNotEmpty && key != fallback)
      .toSet();
  final sorted = discovered.toList()..sort();
  return <String>[
    if (fallback.isNotEmpty) fallback,
    ...sorted,
  ];
}
