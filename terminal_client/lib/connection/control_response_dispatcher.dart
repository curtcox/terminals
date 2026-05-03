import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;
import 'package:terminal_client/ui/server_driven_node_key.dart';

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
