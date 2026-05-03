import 'package:terminal_client/connection/reliability.dart';

String buildMetadataLabel({
  required String buildDate,
  required String buildSha,
}) {
  final normalizedBuildDate = normalizeBuildValue(buildDate);
  final normalizedBuildSha = normalizeBuildValue(buildSha);
  return 'Build: $normalizedBuildDate | SHA: $normalizedBuildSha';
}

String normalizeBuildValue(String value) {
  final trimmed = value.trim();
  if (trimmed.isEmpty) {
    return 'unknown';
  }
  return trimmed;
}

String buildVersionParityNote({
  required String clientBuildDate,
  required String clientBuildSha,
  required String serverBuildDate,
  required String serverBuildSha,
}) {
  final normalizedClientSha = normalizeBuildValue(clientBuildSha);
  final normalizedServerSha = normalizeBuildValue(serverBuildSha);
  final normalizedClientDate = normalizeBuildValue(clientBuildDate);
  final normalizedServerDate = normalizeBuildValue(serverBuildDate);

  if (normalizedServerSha == 'unknown' && normalizedServerDate == 'unknown') {
    return 'Build Match: unknown (awaiting server register ack)';
  }
  if (normalizedClientSha != 'unknown' && normalizedServerSha != 'unknown') {
    if (normalizedClientSha == normalizedServerSha) {
      if (normalizedClientDate != 'unknown' &&
          normalizedServerDate != 'unknown' &&
          normalizedClientDate != normalizedServerDate) {
        return 'Build Match: same SHA, different build date';
      }
      return 'Build Match: same SHA';
    }
    return 'Build Match: different SHA';
  }
  if (normalizedClientDate != 'unknown' && normalizedServerDate != 'unknown') {
    if (normalizedClientDate == normalizedServerDate) {
      return 'Build Match: same build date';
    }
    return 'Build Match: different build date';
  }
  return 'Build Match: unknown';
}

String buildServerBuildLine({
  required String serverBuildDate,
  required String serverBuildSha,
  required bool hasRegisterAck,
}) {
  final normalizedServerSha = normalizeBuildValue(serverBuildSha);
  final normalizedServerDate = normalizeBuildValue(serverBuildDate);
  if (!hasRegisterAck &&
      normalizedServerSha == 'unknown' &&
      normalizedServerDate == 'unknown') {
    return 'Server Build: awaiting register ack';
  }
  return 'Server ${buildMetadataLabel(
    buildDate: serverBuildDate,
    buildSha: serverBuildSha,
  )}';
}

String buildWebConnectionChipLabel({
  required bool hasRegisterAck,
  required bool isConnecting,
  required bool shouldStayConnected,
}) {
  final phase = deriveConnectionPhase(
    shouldStayConnected: shouldStayConnected,
    isConnecting: isConnecting,
    hasClient: shouldStayConnected,
    hasIncoming: shouldStayConnected,
    hasRegisterAck: hasRegisterAck,
    hasRecentTransportFailure: false,
  );
  return buildConnectionPhaseLabel(phase);
}

String buildConnectionPhaseLabel(ConnectionPhase phase) {
  switch (phase) {
    case ConnectionPhase.disconnected:
      return 'Not connected';
    case ConnectionPhase.connecting:
      return 'Connecting';
    case ConnectionPhase.connectedUnregistered:
      return 'Connected (registering)';
    case ConnectionPhase.registered:
      return 'Connected';
    case ConnectionPhase.degraded:
      return 'Degraded';
  }
}
