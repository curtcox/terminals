String buildTransportDiagnosticsClipboardText({
  required String lastTransportDiagnostic,
  required List<String> recentAttempts,
}) {
  final lines = <String>['Transport Diagnostics'];
  final normalizedDiagnostic = lastTransportDiagnostic.trim();
  if (normalizedDiagnostic.isEmpty) {
    lines.add('No transport failures captured yet');
  } else {
    lines.add(normalizedDiagnostic);
  }
  if (recentAttempts.isNotEmpty) {
    lines.add('Recent Carrier Attempts');
    lines.addAll(
      recentAttempts
          .map((attempt) => attempt.trim())
          .where((attempt) => attempt.isNotEmpty),
    );
  }
  return lines.join('\n');
}

String buildControlStreamClipboardText({
  required String status,
  required String notification,
  required String transportDiagnostics,
}) {
  final lines = <String>['Control Stream: ${status.trim()}'];
  final normalizedNotification = notification.trim();
  if (normalizedNotification.isNotEmpty) {
    lines.add(normalizedNotification);
  }
  final normalizedDiagnostics = transportDiagnostics.trim();
  if (normalizedDiagnostics.isNotEmpty) {
    lines.add('Transport Diagnostics');
    lines.add(normalizedDiagnostics);
  }
  return lines.join('\n');
}
